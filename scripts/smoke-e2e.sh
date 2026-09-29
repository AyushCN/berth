#!/usr/bin/env bash
# End-to-end smoke test for berth.
#
#   prerequisites: postgres, redis, nats with JetStream, and a reachable
#                   docker socket, plus the api and worker already running
#                   (`docker compose -f docker-compose.dev.yml up`)
#   usage:         scripts/smoke-e2e.sh
#
# Creates a throwaway environment, exercises the whole lifecycle against the
# live stack, and cleans up after itself. Exits non-zero on the first failed
# assertion, so it is usable as a post-deploy gate.
#
# Override the API base URL, database DSN and sample repository with the
# environment variables below.

set -uo pipefail

API="${API:-http://127.0.0.1:8080}"
PSQL="${PSQL:-psql -h localhost -U berth -d berth -tAc}"
REPO="${REPO:-heroku/node-js-getting-started}"
BRANCH="${BRANCH:-main}"
# Must match WORKSPACE_ROOT as seen by the api.
WORKSPACE_ROOT="${WORKSPACE_ROOT:-/workspaces}"

# psql needs a password for the default dev credentials. Override PGPASSWORD, or
# point PSQL at a full connection string, for any other setup.
export PGPASSWORD="${PGPASSWORD:-berth}"

PASS=0
FAIL=0
ok()    { PASS=$((PASS + 1)); printf '  \033[32mPASS\033[0m %s\n' "$1"; }
bad()   { FAIL=$((FAIL + 1)); printf '  \033[31mFAIL\033[0m %s\n' "$1"; }
check() { if [[ "$2" == *"$3"* ]]; then ok "$1"; else bad "$1 (got: ${2:0:160})"; fi; }
section() { printf '\n\033[1m%s\033[0m\n' "$1"; }
db()  { $PSQL "$1" 2>/dev/null; }
jqp() { python3 -c "import sys,json;d=json.load(sys.stdin);print($1)" 2>/dev/null; }

section "preflight"
code=$(curl -s -o /dev/null -w '%{http_code}' --max-time 5 "$API/health")
if [ "$code" = "200" ]; then ok "api healthy"; else bad "api not healthy (HTTP $code)"; exit 1; fi
pgrep -x berth-worker >/dev/null && ok "worker running" || bad "worker not running"
[ -S /var/run/docker.sock ] && ok "docker socket present" || bad "no docker socket"

section "auth"
TOKEN=$(curl -s "$API/api/auth/dev-login" | jqp 'd["token"]')
[ -n "$TOKEN" ] && ok "dev-login returns a token" || { bad "no token"; exit 1; }
AUTH=(-H "Authorization: Bearer $TOKEN")
check "GET /api/user/me" "$(curl -s "${AUTH[@]}" "$API/api/user/me")" '"username"'

section "create"
CREATED=$(curl -s -X POST "${AUTH[@]}" -H 'Content-Type: application/json' \
  -d "{\"name\":\"smoke-$RANDOM\",\"git_url\":\"https://github.com/$REPO.git\",\"git_branch\":\"$BRANCH\"}" \
  "$API/api/environments")
ENV=$(echo "$CREATED" | jqp 'd["id"]')
WS=$(echo "$CREATED" | jqp 'd["workspace_id"]')
if [ -z "$ENV" ]; then bad "no environment id: $CREATED"; exit 1; fi
ok "POST /api/environments returned an id"
# The directory on disk is keyed by workspace id, so the two must differ for
# this test to exercise that path at all.
[ "$ENV" != "$WS" ] && ok "id differs from workspace_id" || bad "id == workspace_id"
check "returned id is persisted" "$(db "select count(*) from environments where id='$ENV'")" "1"
check "git_url persisted on workspace" "$(db "select git_url from workspaces where id='$WS'")" "github.com/$REPO"
check "owner membership row created" "$(db "select role from workspace_members where workspace_id='$WS'")" "OWNER"

section "provisioning"
STATE=""
for _ in $(seq 1 60); do
  STATE=$(db "select state from environments where id='$ENV'")
  case "$STATE" in RUNNING|BUILD_FAILED) break ;; esac
  sleep 3
done
check "environment reached RUNNING" "$STATE" "RUNNING"
CID=$(db "select container_id from environments where id='$ENV'")
[ -n "$CID" ] && ok "container_id persisted" || bad "no container_id"
docker ps --filter "id=$CID" --format '{{.Status}}' 2>/dev/null | grep -q Up && ok "container is up" || bad "container not running"
PORT=$(db "select port from environments where id='$ENV'")
[ -n "$PORT" ] && ok "port allocated ($PORT)" || bad "no port"

# The worker marks RUNNING as soon as the container starts, which is before the
# app is listening, so allow for boot time.
HTTP=000
for _ in $(seq 1 20); do
  HTTP=$(curl -s -o /dev/null -w '%{http_code}' --max-time 5 "http://127.0.0.1:$PORT/" 2>/dev/null)
  [ "$HTTP" = "200" ] && break
  sleep 2
done
check "application serves over HTTP" "$HTTP" "200"

section "files and git"
FILES=$(curl -s "${AUTH[@]}" "$API/api/environments/$ENV/files?path=.")
check "GET /files lists the checkout" "$FILES" 'README'
CONTENT=$(curl -s "${AUTH[@]}" "$API/api/environments/$ENV/files/content?path=README.md")
[ -n "$CONTENT" ] && ok "GET /files/content returns a body" || bad "empty file content"
check "GET /git/status" "$(curl -s "${AUTH[@]}" "$API/api/environments/$ENV/git/status")" '"branch"'
LOG=$(curl -s "${AUTH[@]}" "$API/api/environments/$ENV/git/log")
check "GET /git/log returns commits" "$LOG" '"commits"'
AUTHOR=$(echo "$LOG" | jqp 'd["commits"][0]["author"]')
[ -n "$AUTHOR" ] && ok "commit author populated ($AUTHOR)" || bad "commit author empty"
DATE=$(echo "$LOG" | jqp 'd["commits"][0]["date"]')
[[ "$DATE" == [0-9][0-9][0-9][0-9]-* ]] && ok "commit date is ISO-8601" || bad "commit date not ISO-8601: $DATE"

section "listing"
LIST=$(curl -s "${AUTH[@]}" "$API/api/environments")
check "GET /api/environments envelope" "$LIST" '"environments"'
echo "$LIST" | grep -q "$ENV" && ok "environment appears in the list" || bad "environment missing from list"
PROJ=$(db "select project_id from workspaces where id='$WS'")
PROJLIST=$(curl -s "${AUTH[@]}" "$API/api/projects/$PROJ/sandboxes")
echo "$PROJLIST" | grep -q "$ENV" && ok "GET /projects/:id/sandboxes returns environments" || bad "project list missing env"

section "websocket"
WSCODE=$(curl -s -o /dev/null -w '%{http_code}' --max-time 5 \
  -H "Connection: Upgrade" -H "Upgrade: websocket" \
  -H "Sec-WebSocket-Version: 13" -H "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==" \
  "$API/ws/environments/$ENV?token=$TOKEN")
check "WS handshake reaches 101" "$WSCODE" "101"

section "lifecycle"
curl -s -X POST "${AUTH[@]}" "$API/api/environments/$ENV/stop" >/dev/null
S=""
for _ in $(seq 1 20); do S=$(db "select state from environments where id='$ENV'"); [ "$S" = "STOPPED" ] && break; sleep 2; done
check "stop -> STOPPED" "$S" "STOPPED"
docker ps --filter "id=$CID" --format '{{.Status}}' 2>/dev/null | grep -qi up \
  && bad "container still up after stop" || ok "container stopped"

# start must restart the existing container, not rebuild it, so it should be
# quick. Allow generous time for the fallback rebuild path.
curl -s -X POST "${AUTH[@]}" "$API/api/environments/$ENV/start" >/dev/null
S=""
for _ in $(seq 1 30); do S=$(db "select state from environments where id='$ENV'"); [ "$S" = "RUNNING" ] && break; sleep 2; done
check "start -> RUNNING" "$S" "RUNNING"
CID2=$(db "select container_id from environments where id='$ENV'")
[ "$CID2" = "$CID" ] && ok "start reused the existing container" || bad "start rebuilt (container changed)"

curl -s -X DELETE "${AUTH[@]}" "$API/api/environments/$ENV" >/dev/null
sleep 3
check "delete -> row soft-deleted" "$(db "select deleted_at is not null from environments where id='$ENV'")" "t"
docker ps -a --filter "id=$CID" --format '{{.Names}}' 2>/dev/null | grep -q . \
  && bad "container survived delete" || ok "container removed"
[ -d "$WORKSPACE_ROOT/$WS" ] && bad "workspace dir survived delete" || ok "workspace dir removed"

section "share links"
PROJ2=$(db "select id from projects limit 1")
CODE=$(curl -s -X POST "${AUTH[@]}" -H 'Content-Type: application/json' -d '{"role":"VIEWER"}' \
  "$API/api/projects/$PROJ2/share-links" | jqp 'd["code"]')
[ -n "$CODE" ] && ok "created a share link" || bad "share link creation failed"
check "GET /share-links/validate" "$(curl -s "${AUTH[@]}" "$API/api/share-links/validate?code=$CODE")" '"valid":true'
check "validate rejects a bad code" \
  "$(curl -s -o /dev/null -w '%{http_code}' "${AUTH[@]}" "$API/api/share-links/validate?code=nope")" "404"
db "delete from share_links where code='$CODE'" >/dev/null

section "cleanup"
db "delete from workspace_members where workspace_id='$WS'" >/dev/null
db "delete from environments where id='$ENV'" >/dev/null
db "delete from workspaces where id='$WS'" >/dev/null
docker rmi -f "berth-${ENV:0:8}:latest" >/dev/null 2>&1
ok "test environment removed"

printf '\n\033[1m%d passed, %d failed\033[0m\n' "$PASS" "$FAIL"
[ "$FAIL" -eq 0 ]
