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
| Phase 4: Prediction | 🟡 In Progress | ML model and Python prediction service scaffolded |
| Phase 5: Evaluation | ✅ Done | EDEBench test harness runs to completion |

### Execution Roadmap

| # | Milestone | Status | What it adds |
|---|---|---|---|
| 1 | **Boot the existing stack** | ✅ Done | Infra + API + worker running, verified with real HTTP calls |
| 2 | **Clone-to-sandbox loop** | ✅ Done | Clone repo loop: POST repo URL → worker clones into container |
| 3 | **Runtime detection + start** | ✅ Done | Auto-detect Node/Python/Go, install deps, start the app |
| 4 | **Preview access** | ✅ Done | App reachable on host networking, DNS inside containers |
| 5 | **File editing** | ❌ Not started | Save file in browser → file lands inside sandbox |
| 6 | **Git operations** | ❌ Not started | Commit/push from inside sandbox (already scaffolded) |
| 7 | **Frontend wiring** | 🟡 Partial | Next.js UI wired for Auth, Profile, FileTree, Editor, and WebSockets |
| 8 | **The Prediction Layer**| 🟡 Partial | ML model and prediction service scaffolded |
| 9 | **Real-Time Sync**    | ❌ Not started | CRDTs |

### What Does NOT Exist Yet (Vaporware)
- ❌ Real-time collaborative editing (CRDT sync)
- ❌ Actual file saving back to the Sandbox via the frontend Code Editor (UI exists but is not wired to backend file API)

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
├── frontend/         # Next.js 15 UI
├── ml/               # Python prediction service
├── infra/            # Docker Compose for local dev
├── scripts/          # Setup and utility scripts
└── docs/             # Architecture docs + IEEE paper
```

## License

MIT — Research Prototype
