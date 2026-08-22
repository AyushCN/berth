# Berth

> Predictive ephemeral sandbox platform with gVisor isolation.

## ⚠️ Current Status: Phase 0–1 Scaffold

This project is currently in early development. It is **NOT complete**.
Please see [docs/STATUS.md](docs/STATUS.md) for a brutal, honest inventory of what works and what is vaporware.

### What Works Currently
- Rootless containerd daemon setup
- gVisor (runsc) runtime integration
- Container lifecycle (Create/Start/Stop/Exec)
- Hardened OCI Spec (namespaces, capabilities, seccomp)
- Bridge networking scaffold

### What Does NOT Exist Yet (Vaporware)
- ❌ Auth flow (GitHub OAuth)
- ❌ Real-time collaborative editing (CRDT sync)
- ❌ Prediction Service (XGBoost)
- ❌ Frontend UI (Next.js)
- ❌ End-to-end Git clone to container execution loop

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

## License

MIT — Research Prototype
