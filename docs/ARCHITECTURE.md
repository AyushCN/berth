# Architecture

## Overview

Berth is a single-host sandbox platform with a control plane (Go API + PostgreSQL + NATS) and a worker that provisions containers via Docker.

```
┌─────────────┐     ┌──────────────┐     ┌─────────────────┐
│  Browser    │────▶│  Next.js     │────▶│  Go API         │
│  (Frontend) │     │  (UI)        │     │  (Control Plane)│
└─────────────┘     └──────────────┘     └───────┬─────────┘
                                                   │
          ┌────────────────────────────────────────┼────────────────────────┐
          ▼                                        ▼                        ▼
┌──────────────────┐                    ┌──────────────────┐      ┌──────────────────┐
│  PostgreSQL      │                    │  NATS            │      │  Docker          │
│  (State Store)   │                    │  (Message Bus)   │      │  (Container      │
│                  │                    │                  │      │   Runtime)       │
└──────────────────┘                    └──────────────────┘      └────────┬─────────┘
                                                                             │
                                                                       ┌──────▼─────────┐
                                                                       │  Sandbox       │
                                                                       │  (Container)   │
                                                                       └────────────────┘
```

---

## Core Components

### 1. API Server (`cmd/api`)
- **Framework**: Gin HTTP server
- **Auth**: JWT + GitHub OAuth 2.0
- **Endpoints**: REST + WebSocket
- **Auth Middleware**: JWT validation, per-user rate limiting (default 120/min)
- **WebSocket Hub**: Real-time terminal, file edits, presence (Redis PubSub)

### 2. Worker (`cmd/worker`)
- **Job Processing**: Consumes NATS `berth.environment.create` (create), `berth.environment.stop` (stop), `berth.environment.delete` (delete)
- **Repository Cloning**: GitHub HTTPS with shared bare cache + git -c credential header (no token on disk)
- **Runtime Analysis**: Post-clone static detection via `internal/analyzer`
- **Container Lifecycle**: Create (idempotent by name) → Install deps → Commit image → Run with file watcher
- **Readiness Check**: Polls container port for up to 60s; marks `CRASHED` if nothing listens
- **Cleanup**: Idle suspend (30min dev / 60min prod, checked every 5min), stop/start/delete via NATS; no lifetime expiry

### 3. Runtime Analyzer (`internal/analyzer`)
- **Language Detection**: Node.js, Python, Go, Rust, Java
- **Framework Detection**: Next.js, Django, FastAPI, Flask, Gin, Axum, Spring, Express, etc.
- **Package Manager**: npm, pnpm, yarn, bun, Cargo, Go modules, pip/poetry/uv, Maven/Gradle
- **Lockfile Detection**: 18+ lockfile types with SHA256 cache keys for git cache
- **Docker Compose**: Service parsing, port exposure, volume mounts

### 4. Infrastructure
- **PostgreSQL**: SQLC-generated queries, embedded migrations applied on boot
- **NATS**: JetStream for async job orchestration (`berth.environment.*`)
- **Redis**: PubSub for WebSocket real-time updates, session cache
- **Docker**: Container lifecycle via host Docker socket (host networking)

---

## Data Flow

### Environment Provisioning

```
1. Frontend → POST /api/environments (repo URL, branch)
2. API → Create workspace (git_url, git_branch) + environment (CREATED)
3. API → Publish NATS berth.environment.create
4. Worker → Consume job, clone repo (shared bare cache + git -c credential header)
5. Worker → analyzer.Analyze(workspaceDir) → RuntimeProfile
6. Worker → Docker create (idempotent by name) → install deps → commit image → run with watcher
6. Worker → Readiness: poll container port up to 60s
8. Worker → Update state (RUNNING + container_id, port, public_url) OR CRASHED + last_error
9. Frontend ← Poll /api/environments → status, files, git, logs, preview
```

### Idle Suspend

- **Idle check** (every 5min): Environments with 0 active sessions and no activity for the idle timeout → stop container → `SUSPENDED`
- **Timeout**: 30min in development, 60min in production
- **Resume**: `POST /api/environments/resume` restarts the kept container (`STARTING` → `RUNNING`)

There is no lifetime/TTL reaper for environments and no expiry warnings: the old sandbox TTL sweep was removed with the legacy model, and no `sandbox_lifetime_warnings` table exists.

### Lifecycle Operations

| Operation | Flow |
|-----------|------|
| **Start** (stopped/failed/crashed) | Publish `berth.environment.start` → worker starts the kept container → `RUNNING` (no rebuild; falls back to `CREATED` re-provision when there is no container or no bus) |
| **Stop** | Runtime present: stop directly; otherwise publish `berth.environment.stop` → worker stops → `STOPPED` (container kept for restart) |
| **Delete** | Runtime present: remove directly; otherwise publish `berth.environment.delete` → worker destroys container + workspace dir + soft-deletes |
| **Restart** | Remove container → reset state to `CREATED` → re-provision from scratch |

---

## Container Model

- **Host Networking**: Containers run with `--network host`, port bound directly on host
- **Security**: `--read-only`, `--cap-drop ALL`, `--init`, `--user 1000:1000`, `no-new-privileges`
- **Volumes**: Workspace dir bind-mounted at `/workspace`; tmpfs for `/tmp`, `/var/tmp`, `/run`
- **Seccomp**: Default profile (no custom profile; `seccomp=default` is invalid for Docker)
- **Readiness**: TCP dial on exposed port, up to 60s; `CRASHED` on timeout

---

## Data Model

### Core Tables

| Table | Purpose |
|-------|---------|
| `users` | GitHub OAuth users, encrypted GitHub token |
| `organizations` | Tenant container for projects |
| `projects` | Group environments, collaborators |
| `workspaces` | Git checkout + metadata (git_url, git_branch) |
| `environments` | Runnable instances (state, container_id, port, public_url, limits) |
| `runtime_profiles` | Detected stack per workspace (language, framework, package manager) |
| `environment_services` | Sidecar services per environment |
| `environment_events` | Activity log (file edits, terminal, state changes) |
| `change_requests` | Editor→owner review workflow per project |
| `share_links` | Join links with `VIEWER`/`EDITOR` roles, expiry, usage limits |
| `workspace_members` | Collaborator roles (OWNER, EDITOR, VIEWER) |
| `schema_migrations` | golang-migrate version tracking |

### State Machine (`environments.state`)

```
CREATED → BUILDING → RUNNING ⇄ STOPPED
              ↓          ↕
       BUILD_FAILED   SUSPENDING
              ↓          ↓
           CRASHED ← SUSPENDED
```

`STARTING` is written transiently when resuming or claiming a pending environment; `STOPPING` and `READY` exist in the enum but are never written. `DELETING` marks a delete in progress before the soft-delete (`deleted_at`). Terminal states: `BUILD_FAILED`, `CRASHED`, `DELETING`. There is no `EXPIRED` state — no TTL reaper exists.

---

## Authentication

- **GitHub OAuth 2.0** with PKCE (`S256`)
- **JWT** (HS256, 24h expiry, no refresh)
- **Session Cookie**: `berth_token` (HttpOnly, SameSite=Lax, Secure derived from TLS/X-Forwarded-Proto)
- **Dev Login** (`/api/auth/dev-login`): Unauthenticated, mints token for fixed dev UID — **not registered in production**
- **CSRF**: Relies on SameSite=Lax + CORS single-origin; no explicit CSRF tokens

---

## Frontend Architecture

- **Framework**: Next.js 15 App Router, React 18, TypeScript
- **State**: Zustand stores (`env`, `auth`, `project`)
- **API Client**: `lib/api.ts` with credentials: 'include', envelope normalization
- **Components**: `CodeEditor` (Monaco), `Terminal` (xterm.js), `GitUI`/`GitPanel`, `FileTree`, `Terminal`
- **Real-time**: WebSocket `/ws/environments/:id` (Redis PubSub fan-out)

---

## Configuration Reference

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | ✅ | — | Postgres DSN |
| `REDIS_URL` | ✅ | — | Redis URL |
| `NATS_URL` | ✅ | — | NATS JetStream URL |
| `ENCRYPTION_KEY` | ✅ | — | 32 bytes raw or 64 hex chars |
| `JWT_SECRET` | ✅ | — | ≥32 chars |
| `GITHUB_CLIENT_ID` | ✅ | — | GitHub OAuth client ID |
| `GITHUB_CLIENT_SECRET` | ✅ | — | GitHub OAuth client secret |
| `FRONTEND_URL` | ✅ | `http://localhost:3000` | CORS origin + OAuth redirect |
| `WORKSPACE_ROOT` | — | `~/.local/state/berth/workspaces` | Host path for workspaces (`/workspaces` in compose) |
| `DOCKER_HOST` | — | `unix:///var/run/docker.sock` | Docker socket |
| `DOCKER_NETWORK` | — | `berth` | Docker network |
| `TRAEFIK_DOMAIN` | — | `""` | Traefik base domain |
| `ENV` | — | `development` | `development` or `production` |
| `MODE` | — | `api` | `api` or `worker` |
| `PORT` | — | `8080` | API port |
| `RATE_LIMIT_REQUESTS_PER_MINUTE` | — | `200` | Per-IP+path limit for `/api` |
| `RATE_LIMIT_AUTHENTICATED_PER_MINUTE` | — | `120` | Shared per-user bucket for authenticated routes |
| `READINESS_TIMEOUT` | — | `60s` | How long provisioning waits for the app to listen |

`MODEL_DIR` is still read by `config` but nothing consumes it (leftover from the removed prediction stack).

---

## Migrations

- **Embedded**: SQL migrations in `backend/migrations/` with `go:embed`
- **Auto-run**: `migrations.Migrate(dsn)` called on every boot by both API and Worker
- **No manual step**: Fresh deploy provisions itself; second boot is no-op
- **Irreversible**: Migration 000009 (drop legacy sandboxes) and 000010 (drop prediction tables) are explicit `RAISE EXCEPTION` on down — restore from backup instead

---

## Testing

```bash
# Backend unit/integration tests
cd backend && ENCRYPTION_KEY=... go test ./... -count=1

# Migration tests (needs server with CREATE DATABASE)
BERTH_MIGRATION_TEST_DSN="postgres://..." go test ./backend/migrations/... -v

# Frontend
cd frontend && npm run lint && npm run build

# End-to-end smoke test (requires running stack)
cd berth && WORKSPACE_ROOT=/tmp/ws ./scripts/smoke-e2e.sh
```

---

## Security Model

- **Isolation**: `--read-only`, `--cap-drop ALL`, `--init`, `--user 1000:1000`, `no-new-privileges` (host Docker socket, so a compromised worker host is game over — single-host prototype, not a tenant boundary)
- **Network**: Host networking only; no cross-container communication
- **Secrets**: GitHub tokens AES-256-GCM encrypted at rest; JWT signed with HS256
- **Cookies**: `HttpOnly`, `SameSite=Lax`, `Secure` derived from request (TLS or `X-Forwarded-Proto`)
- **CORS**: Single origin (`FRONTEND_URL`), credentials allowed
- **Rate Limiting**: 200/min per IP (global), 120/min per user (authenticated routes)
- **Dev Login** (`/auth/dev-login`): **Not registered in production**; mints token for fixed UID

---

## Known Limitations

- Single-host only; no multi-node clustering
- No readiness probing for apps that don't listen on a port (static sites report `CRASHED`)
- No gVisor/Cilium/mTLS; rootless Docker only
- Preview URLs work on `*.localhost` via Traefik; not routable externally without DNS
- No collaborative editing (presence only)
- Migrations require manual `psql` for down (000009, 000010 are irreversible)
- No CI/CD pipeline
