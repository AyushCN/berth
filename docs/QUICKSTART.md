# Quick Start

```bash
# 1. Start everything
docker compose -f docker-compose.dev.yml up -d --build

# 2. Wait ~10s, then open:
#    Frontend: http://localhost:3000
#    API: http://api.localhost
#    Traefik: http://localhost:8080

# 3. Click "Continue with GitHub" (or GET /api/auth/dev-login in dev)
```

### Prerequisites
- Docker 24+ with Compose v2
- Linux (or macOS with Docker Desktop)

### Manual env (if not using compose)
```bash
export DATABASE_URL=postgres://berth:berth@localhost:5432/berth?sslmode=disable
export REDIS_URL=redis://localhost:6379
export NATS_URL=nats://localhost:4222
export ENCRYPTION_KEY=0d71f78929e8b688442387dd10478006998c1fa490c42c02c627a3e5ec8a3bed
export JWT_SECRET=dev_secret_change_in_production_at_least_32_chars_long
export GITHUB_CLIENT_ID=dev_client_id
export GITHUB_CLIENT_SECRET=dev_client_secret
export FRONTEND_URL=http://localhost:3000
export WORKSPACE_ROOT=/tmp/berth-workspaces
export DOCKER_HOST=unix:///var/run/docker.sock
export DOCKER_NETWORK=berth
export TRAEFIK_DOMAIN=localhost
export ENV=development
export MODE=api
export PORT=8080

cd backend && go run ./cmd/api
# Terminal 2:
export MODE=worker
go run ./cmd/worker
# Terminal 3:
cd frontend && npm install && npm run dev
```

### Test with a sample repo
```bash
curl -X POST http://api.localhost/api/environments \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"test","git_url":"https://github.com/octocat/Hello-World.git","git_branch":"main"}'
```