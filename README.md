# Berth

**Ephemeral development environments from GitHub repositories.**

Berth provisions sandboxed containers on-demand from GitHub repositories. It auto-detects the runtime (Node.js, Python, Go, Rust, Java), installs dependencies, starts the application, and gives you a live preview URL, file editor, terminal, and Git integration — all in the browser.

---

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌──────────────┐
│  Browser    │────▶│  Next.js     │────▶│  Go API       │
│  (Frontend) │     │  (UI)        │     │  (Control)    │
└─────────────┘     └──────────────┘     └──────┬───────┘
                                                 │
                      ┌──────────────────────────┼────────────────────────┐
                      │                          │                        │
                      ▼                          ▼                        ▼
             ┌─────────────────┐        ┌─────────────────┐      ┌─────────────────┐
             │  PostgreSQL     │        │  NATS           │      │  Docker         │
             │  (State Store)  │        │  (Message Bus)  │      │  (Container     │
             │                 │        │                 │      │   Runtime)      │
             └─────────────────┘        └─────────────────┘      └────────┬────────┘
                                                                           │
                                                                  ┌────────▼─────────┐
                                                                  │  Environment     │
                                                                  │  (Container)     │
                                                                  └──────────────────┘
```

- **Go API** (`backend/cmd/api`) — Gin HTTP server, JWT + GitHub OAuth (web flow + PKCE), REST + WebSocket, rate limiting
- **Worker** (`backend/cmd/worker`) — Consumes NATS `berth.environment.*`, clones repos, provisions containers via Docker
- **PostgreSQL 16** — SQLC-generated queries, embedded migrations applied on boot by both binaries
- **NATS 2.10** — JetStream for async job orchestration (`berth.environment.create/stop/start/delete`)
- **Redis 7** — PubSub for WebSocket real-time updates
- **Docker** — Container lifecycle via host Docker socket
- **Docker Compose** (dev) — Postgres, Redis, NATS, Traefik, API, Worker, Frontend

---

## Features

| Feature | Description |
|---------|-------------|
| **Runtime Detection** | Auto-detects Node.js, Python, Go, Rust, Java with framework support (Next.js, Django, Gin, Axum, Spring, etc.) |
| **Dependency Caching** | npm/pnpm/yarn/bun, Cargo, Go modules, pip/poetry, Maven/Gradle — shared git and package caches |
| **Git Integration** | OAuth-backed Git operations: status, diff, commit, push, branch management |
| **Collaborative IDE** | Monaco editor, xterm terminal, Git panel, live preview |
| **GitHub OAuth** | GitHub OAuth web flow with PKCE (S256); tokens AES-256-GCM encrypted at rest |
| **Idle Suspend** | No active sessions for 30min (dev) / 60min (prod) → container stopped, resumable |
| **Readiness Check** | Polls the app port (default 60s); records `CRASHED` with reason if nothing listens |
| **Fork & Share** | Fork any environment; share links with `VIEWER` / `EDITOR` roles, expiry, usage limits |

---

## Quick Start

### Prerequisites

- Linux (or macOS with Docker Desktop)
- Go 1.26+
- Node.js 20+
- Docker 24+ with Compose v2

Infra (Postgres 16, Redis 7, NATS 2.10 with JetStream, Traefik v2) is provided by `docker-compose.dev.yml` — no local installs needed.

### Development Setup

```bash
# 1. Start everything (Postgres, Redis, NATS, Traefik, API, Worker, Frontend)
docker compose -f docker-compose.dev.yml up -d --build

# 2. Wait for services to be healthy (~10s), then:
#    Frontend:          http://localhost:3000
#    API (via Traefik): http://api.localhost
#    Traefik dashboard: http://localhost:8080

# 3. Open http://localhost:3000 → "Continue with GitHub"
#    (or GET /api/auth/dev-login in non-production for the fixed dev user)
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

# Terminal 2: Worker (same env, except MODE=worker; ENCRYPTION_KEY must match)
export MODE=worker
go run ./cmd/worker

# Terminal 3: Frontend
cd frontend
npm install
npm run dev
```

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | ✅ | — | Postgres DSN |
| `REDIS_URL` | ✅ | — | Redis URL |
| `NATS_URL` | ✅ | — | NATS JetStream URL |
| `ENCRYPTION_KEY` | ✅ | — | 32 raw bytes or 64 hex chars; must match on API and worker |
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

## Project Structure

```
berth/
├── backend/                    # Go API, Worker
│   ├── cmd/
│   │   ├── api/               # API server entry point
│   │   └── worker/            # Worker entry point
│   ├── internal/
│   │   ├── analyzer/          # Runtime & framework detection
│   │   ├── domain/            # Core domain models
│   │   ├── usecase/           # Business logic
│   │   ├── delivery/          # HTTP handlers, WebSocket
│   │   ├── infrastructure/    # Docker, NATS, Redis, DB
│   │   ├── repository/        # SQLC repositories
│   │   ├── worker/            # Environment worker
│   │   └── integration/       # Integration tests
│   ├── migrations/            # SQL migrations (embedded, applied on boot)
│   └── go.mod                # Go 1.26.3
├── frontend/                   # Next.js 15 + React 18
│   ├── app/                   # App Router pages
│   ├── components/            # React components (editor, terminal, git, file-tree)
│   ├── stores/                # Zustand state management
│   └── lib/                   # API client
├── docker-compose.dev.yml
├── docker-compose.prod.yml
└── scripts/                   # smoke-e2e.sh, setup scripts
```

---

## Key Commands

```bash
# Build
cd backend && go build -o /tmp/berth-api ./cmd/api
cd backend && go build -o /tmp/berth-worker ./cmd/worker
cd frontend && npm run build

# Unit / integration tests (all packages)
cd backend && ENCRYPTION_KEY=0d71f78929e8b688442387dd10478006998c1fa490c42c02c627a3e5ec8a3bed go test ./... -count=1

# Migration tests (needs a server the tests may create databases in)
cd backend && BERTH_MIGRATION_TEST_DSN="postgres://berth:berth@localhost:5432/postgres?sslmode=disable" go test ./migrations/ -count=1 -v

# Frontend checks
cd frontend && npm run lint && npm run build

# End-to-end smoke test (requires a running stack; cleans up after itself)
./scripts/smoke-e2e.sh
```

---

## Documentation

| Document | Description |
|----------|-------------|
| [Architecture](docs/ARCHITECTURE.md) | System architecture, components, data flow |
| [API Reference](docs/API.md) | REST endpoints, WebSocket, schemas |
| [Development](docs/DEVELOPMENT.md) | Local dev setup, workflows |
| [Deployment](docs/DEPLOYMENT.md) | Production deployment |
| [Security](docs/SECURITY.md) | Threat model, hardening |
| [Status](docs/STATUS.md) | Current implementation status |

---

## Security Notice

**This is a research prototype, not production-ready.**

- Single-host only, no multi-node support
- Docker containers with `--read-only`, `--cap-drop ALL`, `--init`, `no-new-privileges`
- No mTLS/SPIFFE, no Cilium policies
- `/api/auth/dev-login` mints a session for a fixed user and is only registered outside production
- Do not expose to untrusted users or public internet without enabling all security features

See [Security](docs/SECURITY.md) for full threat model.
