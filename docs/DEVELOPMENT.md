# Development Guide

## Prerequisites

### System Requirements
- **OS**: Linux (Ubuntu 22.04+, Fedora 39+, Arch) or macOS (with Docker Desktop)
- **Go**: 1.26+
- **Node.js**: 20+ (LTS)
- **Docker**: 24+ with Compose v2
- **PostgreSQL**: 16+
- **NATS**: 2.10+ with JetStream
- **Redis**: 7+
- **Git**: 2.40+

### Install on Ubuntu/Debian
```bash
# Go
wget https://go.dev/dl/go1.26.3.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.26.3.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc

# Node.js (via nvm)
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39/install.sh | bash
nvm install 20

# Docker
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER

# PostgreSQL
sudo apt-get install postgresql-16 postgresql-client-16

# NATS
curl -sfL https://github.com/nats-io/nats-server/releases/download/v2.10.0/nats-server-v2.10.0-linux-amd64.tar.gz | tar xz
sudo mv nats-server /usr/local/bin/

# Redis
sudo apt-get install redis-server
```

### macOS (with Docker Desktop)
```bash
brew install go node@20 docker docker-compose postgresql@16 nats-server redis
brew services start postgresql@16 redis nats-server
```

---

## Project Setup

### Clone Repository
```bash
git clone https://github.com/yourorg/berth.git
cd berth
```

### Backend Setup
```bash
cd backend

# Download dependencies
go mod download

# Generate SQLC code
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
sqlc generate

# Set environment
export ENCRYPTION_KEY=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
export DATABASE_URL=postgres://berth:berth@localhost:5432/berth?sslmode=disable
export REDIS_URL=redis://localhost:6379
export NATS_URL=nats://localhost:4222
export JWT_SECRET=dev-secret-key-32-characters-long!!
export GITHUB_CLIENT_ID=your-github-client-id
export GITHUB_CLIENT_SECRET=your-github-client-secret
export FRONTEND_URL=http://localhost:3000
export MODE=api
export PORT=8080
export ENCRYPTION_KEY=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
export WORKSPACE_ROOT=/tmp/berth-workspaces
export DOCKER_HOST=unix:///var/run/docker.sock
export DOCKER_NETWORK=berth
export TRAEFIK_DOMAIN=localhost
export ENV=development
export MODE=api
export PORT=8080
```

### Frontend Setup
```bash
cd frontend

# Install dependencies
npm install

# Environment
echo "NEXT_PUBLIC_API_URL=http://api.localhost" > .env.local
echo "NEXT_PUBLIC_WS_URL=ws://api.localhost" >> .env.local
```

### Database Setup
```bash
# Create database
createdb -U berth berth

# Migrations run automatically on boot - no manual step needed
```

### Infrastructure
```bash
# Start PostgreSQL, Redis, NATS, Traefik, API, Worker, Frontend
docker compose -f docker-compose.dev.yml up -d

# Verify
docker compose -f docker-compose.dev.yml ps
```

---

## Running the Application

### Option 1: Docker Compose (Recommended)
```bash
# Start everything (infra + API + Worker + Frontend)
docker compose -f docker-compose.dev.yml up -d --build

# Frontend:          http://localhost:3000
# API (via Traefik): http://api.localhost
# Traefik dashboard: http://localhost:8080
```

### Manual Run (without docker-compose)
```bash
# Terminal 1: API
cd backend
export DATABASE_URL="postgres://berth:berth@localhost:5432/berth?sslmode=disable"
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
go run ./cmd/api

# In another terminal: Worker (same env, except MODE=worker)
export MODE=worker
go run ./cmd/worker

# In another terminal: Frontend
cd frontend
npm install
npm run dev
```

### Access URLs
| Service | URL |
|---------|-----|
| Frontend | http://localhost:3000 |
| API (via Traefik) | http://api.localhost |
| API Health | http://api.localhost/health |
| NATS Monitor | http://localhost:8222 |
| Traefik Dashboard | http://localhost:8080 |

### Development Credentials
```bash
# Dev login (auto-creates user)
curl -X GET http://localhost:8080/api/auth/dev-login
# Returns JWT token

# Or visit http://localhost:3000/login and click "Dev Login"
```

---

## Development Workflow

### Code Style
```bash
# Go formatting
go fmt ./...

# Go linting (vet + gofmt)
go vet ./...

# Frontend typechecking + lint
cd frontend && npm run lint
```

### Git Workflow
```bash
# Feature branch
git checkout -b feat/your-feature

# Commit with conventional commits
git commit -m "feat(analyzer): add Rust Axum detection"

# Push and create PR
git push origin feat/your-feature
```

### Commit Message Format
```
<type>(<scope>): <description>

[optional body]
```

### SQLC Regeneration
```bash
cd backend
sqlc generate
```
Run this after modifying any `.sql` file in `queries/`.

### Tests
```bash
# Backend unit/integration tests
cd backend && ENCRYPTION_KEY=0d71f78929e8b688442387dd10478006998c1fa490c42c02c627a3e5ec8a3bed go test ./... -count=1

# Migration tests (needs a server with CREATE DATABASE permission)
BERTH_MIGRATION_TEST_DSN="postgres://berth:berth@localhost:5432/postgres?sslmode=disable" go test ./migrations/ -count=1 -v

# Frontend
cd frontend && npm run lint && npm run build
```

---

## Architecture Overview

- **API** (`backend/cmd/api`) — Gin HTTP server, JWT + GitHub OAuth, REST + WebSocket
- **Worker** (`backend/cmd/worker`) — Consumes NATS `berth.environment.*`, provisions containers via Docker
- **PostgreSQL 16** — SQLC-generated queries, embedded migrations applied on boot
- **NATS 2.10** — JetStream for async job orchestration (`berth.environment.*`)
- **Redis 7** — PubSub for WebSocket real-time updates
- **Docker** — Container lifecycle via host Docker socket
- **Frontend** — Next.js 15 + React 18, Zustand, Monaco editor, xterm.js, Monaco editor, Git panel

### Key Commands
```bash
# Build
go build -o /tmp/berth-api ./cmd/api
go build -o /tmp/berth-worker ./cmd/worker
cd frontend && npm run build

# Run tests
cd backend && ENCRYPTION_KEY=... go test ./... -count=1
cd frontend && npm run lint && npm run build

# End-to-end smoke test (requires running stack)
./scripts/smoke-e2e.sh

# Migration tests (needs BERTH_MIGRATION_TEST_DSN)
BERTH_MIGRATION_TEST_DSN="postgres://berth:berth@localhost:5432/postgres?sslmode=disable" go test ./migrations/ -count=1 -v
```

---

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | ✅ | — | Postgres DSN |
| `REDIS_URL` | ✅ | — | Redis URL |
| `NATS_URL` | ✅ | — | NATS JetStream URL |
| `ENCRYPTION_KEY` | ✅ | — | 32 raw bytes or 64 hex chars (GitHub token encryption) |
| `JWT_SECRET` | ✅ | — | ≥32 chars (JWT signing) |
| `GITHUB_CLIENT_ID` | ✅ | — | GitHub OAuth client ID |
| `GITHUB_CLIENT_SECRET` | ✅ | — | GitHub OAuth client secret |
| `FRONTEND_URL` | ✅ | `http://localhost:3000` | CORS origin + OAuth redirect |
| `WORKSPACE_ROOT` | — | `~/.local/state/berth/workspaces` | Host path for workspace checkouts |
| `DOCKER_HOST` | — | `unix:///var/run/docker.sock` | Docker socket |
| `DOCKER_NETWORK` | — | `berth` | Docker network name |
| `TRAEFIK_DOMAIN` | — | `""` | Traefik base domain |
| `ENV` | — | `development` | `development` or `production` (`dev-login` not registered in production) |
| `MODE` | — | `api` | `api` or `worker` |
| `PORT` | — | `8080` | API port |
| `RATE_LIMIT_REQUESTS_PER_MINUTE` | — | `200` | Per-IP+path limit for `/api` |
| `RATE_LIMIT_AUTHENTICATED_PER_MINUTE` | — | `120` | Shared per-user bucket for authenticated routes |
| `READINESS_TIMEOUT` | — | `60s` | How long provisioning waits for the app to listen |

`MODEL_DIR` is still read by `config` but nothing consumes it (leftover from the removed prediction stack).

---

## Known Limitations

- Single-host only; no multi-node support
- No readiness probing for apps that don't listen on a port (static sites report `CRASHED`)
- No gVisor/Cilium/mTLS; rootless Docker only
- Preview URLs work on `*.localhost` via Traefik; not routable externally without DNS
- No collaborative editing (presence only)
- Migrations require manual `psql` for down (000009, 000010 are irreversible)
- No CI/CD pipeline