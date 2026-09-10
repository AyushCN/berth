# Berth

> Predictive ephemeral sandbox platform with gVisor isolation.

## ⚠️ Current Status: Phase 0–1 Scaffold

This project is currently in early development. It is **NOT complete**.
Please see [docs/STATUS.md](docs/STATUS.md) for a brutal, honest inventory of what works and what is vaporware.

### What Works Currently
- Rootless containerd daemon setup
- gVisor (runsc) runtime integration
- Fast Container lifecycle (Create/Start/Stop/Exec) works cleanly in rootless mode
- Advanced Warm Pool with dependency caching
- Hardened OCI Spec (namespaces, capabilities, seccomp, cgroups)
- Event-driven provisioning via NATS
- Host-networking enabled (no bridge) for unblocked port-forwarding

### Honest Status

| Phase | Status | Notes |
|-------|--------|-------|
| Phase 0: Scaffold | ✅ Done | CI, schema, scripts |
| Phase 1: Isolation | ✅ Done | Runtime works, warm pool + layer caching fully implemented |
| Phase 2: Auth + API | ✅ Done | Usecases implemented, HTTP handlers fully wired and tested |
| Phase 3: Frontend | 🟡 In Progress | Next.js UI wired with GitHub OAuth, Profile, and Terminal |
| Phase 4: Prediction | ❌ Not started | No model |
| Phase 5: Evaluation | ❌ Not started | No benchmarks |

### Execution Roadmap

| # | Milestone | Status | What it adds |
|---|---|---|---|
| 1 | **Boot the existing stack** | ✅ Done | Infra + API + worker running, verified with real HTTP calls |
| 2 | **Clone-to-sandbox loop** | ❌ Not started | Port api-sandbox's proven flow: POST repo URL → worker clones into container |
| 3 | **Runtime detection + start** | ❌ Not started | Auto-detect Node/Python/Go, install deps, start the app |
| 4 | **Preview access** | ❌ Not started | Reach the running app from your browser |
| 5 | **File editing** | ❌ Not started | Save file in browser → file lands inside sandbox |
| 6 | **Git operations** | ❌ Not started | Commit/push from inside sandbox (already scaffolded) |
| 7 | **Frontend wiring** | 🟡 Partial | Next.js UI wired for Auth, Profile, and Terminal WebSockets |
| 8 | **The Prediction Layer**| ❌ Not started | The ML model predicts which repo you'll open next |
| 9 | **Real-Time Sync**    | ❌ Not started | CRDTs, Monaco editor |

### What Does NOT Exist Yet (Vaporware)
- ❌ Real-time collaborative editing (CRDT sync)
- ❌ Prediction Service (XGBoost)
- ❌ Frontend UI (Next.js)

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
# Add your GitHub OAuth credentials here:
export GITHUB_CLIENT_ID="your_client_id"
export GITHUB_CLIENT_SECRET="your_client_secret"
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
