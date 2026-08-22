# Berth

> Predictive ephemeral sandbox platform with gVisor isolation.

## Quick Start

```bash
# 1. Setup (installs tools, starts infra)
bash scripts/setup.sh
bash scripts/setup-rootless.sh

# 2. Start infrastructure
make dev

# 3. Run migrations
make migrate-up

# 4. Start backend
export CONTAINERD_SOCK=$XDG_RUNTIME_DIR/containerd/containerd.sock
export JWT_SECRET="dev-secret"
cd backend && go run ./cmd/api
```

## Requirements

- Linux (bare metal or VM). macOS is not supported for local gVisor dev.
- Go 1.23+
- Node.js 20+
- Docker + Docker Compose

## Project Structure

```
berth/
├── backend/          # Go API + workers (Clean Architecture)
├── frontend/         # Next.js 15 (not yet scaffolded)
├── ml/               # Python prediction service (not yet scaffolded)
├── infra/            # Docker Compose for local dev
├── scripts/          # Setup and utility scripts
└── docs/             # Architecture docs + IEEE paper
```

## Status

- **Phase 0:** Project Scaffolding (Complete)
- **Phase 1:** gVisor Isolation Layer & Rootless Containerd (Complete)
- **Phase 2:** API + Auth + Prediction Service (Pending)

## License

MIT — Research Prototype
