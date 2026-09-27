# Project Status

**Berth is an early-stage, single-host research prototype.** This inventory describes the checked-in implementation, not a production readiness claim.

---

## ✅ Implemented Foundations

### Core Platform
- [x] **Go API + Worker** with PostgreSQL persistence and NATS-based job orchestration (fallback polling)
- [x] **containerd Integration** - create/start/stop/exec lifecycle, exact-image warm container reuse
- [x] **Runtime Detection** - Rule-based post-clone detection for Node.js, Python, Go, Rust, Java
- [x] **Architecture Classification** - WEB_APP, API, CLI, FULL_STACK, MONOREPO, LIBRARY
- [x] **Framework Detection** - 30+ frameworks (Next.js, Express, FastAPI, Django, Gin, Axum, Spring, etc.)
- [x] **Build Strategy Pattern** - Compose, Python, Node, Go, Rust, Java, Fallback strategies
- [x] **Dependency Caching** - pnpm store, Cargo, Go modules, pip/poetry, Maven/Gradle
- [x] **Docker Compose Support** - Parse and build multi-service compose files
- [x] **Lockfile Detection** - 18 lockfile types with SHA256 cache keys
- [x] **Entry Point Detection** - Dockerfile, docker-compose, main files with ambiguity handling

### Git & Collaboration
- [x] **GitHub OAuth** - Encrypted token storage (AES-GCM)
- [x] **Git Operations** - Clone, branch, checkout, commit, push, diff, log
- [x] **Change Requests** - Editor → Owner workflow with merge
- [x] **Share Links** - Project invitation via short codes

### Frontend (Next.js 15 + React 18)
- [x] **Authentication** - GitHub OAuth, dev login, JWT
- [x] **Dashboard** - Sandbox list, create, fork, delete
- [x] **IDE** - File tree, Monaco editor, terminal (xterm.js), Git panel
- [x] **Preview Proxy** - `/p/:id/*` → sandbox container
- [x] **Git Panel** - Status, branches, commit, push, pull, diff, log
- [x] **File Operations** - List, read, write, create, delete, move, duplicate
- [x] **Terminal** - xterm.js + WebSocket + PTY
- [x] **Presence** - User avatars, cursors (infrastructure)

### Prediction Engine (Phase 4)
- [x] **Data Collector** - Training data from profiles and builds
- [x] **Model Trainer** - Linear Regression, Random Forest, XGBoost
- [x] **Model Metrics** - MSE, MAE, RMSE, R², sample counts
- [x] **Prediction Service** - Build time, image size, cache hit, failure risk
- [x] **ONNX Export** - Binary protobuf ONNX model export
- [x] **ONNX Runtime** - onnxruntime-go DynamicAdvancedSession inference
- [x] **gRPC Service** - 10 methods (predict, history, models, retrain, export, activate)
- [x] **HTTP Handlers** - REST endpoints for all predictions
- [x] **Frontend UI** - Predict/Models/History tabs with metrics

### Infrastructure
- [x] **PostgreSQL** - SQLC-generated type-safe queries
- [x] **NATS** - JetStream + fallback polling
- [x] **Redis** - PubSub, session cache
- [x] **containerd** - Rootless, runc.v2, layer commit, tar export
- [x] **Migrations** - 5 migrations (core + prediction engine)
- [x] **Docker** - Image building, multi-stage builds

### Testing (65 Tests Passing)
- [x] **Analyzer Tests** - 25 tests (architecture, frameworks, entry points, lockfiles)
- [x] **Build Planner Tests** - 15 tests (all strategies)
- [x] **Model Trainer Tests** - 18 tests (all algorithms, ONNX export)
- [x] **Integration Tests** - 7 tests (analyzer, planner, e2e flow, 5 project types)

---

## 🚧 Current Limitations

### Runtime & Security
| Limitation | Impact |
|------------|--------|
| **Rootless runc.v2 only** | No gVisor/runsc, no hardware isolation |
| **Host networking** | No per-sandbox network isolation |
| **No resource enforcement** | CPU/memory limits skipped in rootless |
| **No mTLS/SPIFFE** | Plaintext NATS, PostgreSQL connections |
| **Single host only** | No multi-node, no HA |

### Preview & Networking
| Limitation | Impact |
|------------|--------|
| **Host network proxy** | `/p/:id/` → host port, no isolation |
| **No TLS termination** | HTTP only, no cert management |
| **No custom domains** | Fixed Traefik domain only |

### Git & Collaboration
| Limitation | Impact |
|------------|--------|
| **OAuth push incomplete** | Owner push to `berth/<id>` branch partial |
| **No collaborative editing** | No OT/CRDT, no presence sync |
| **No code review UI** | Change requests API only |

### Prediction Engine
| Limitation | Impact |
|------------|--------|
| **Mock ONNX inference** | onnxruntime-go needs libonnxruntime.so |
| **Linear models only** | RF/XGBoost use simplified implementations |
| **No feature importance** | No SHAP/permutation importance |
| **No A/B testing** | No canary deployments, traffic splitting |
| **Single model per type** | No model registry/versioning UI |

---

## 📋 Work Needed for Usable Product

### Immediate (Demo Ready)
- [ ] Run Linux smoke test (OAuth push, warm hits, stop/delete, expiry)
- [ ] Fix ONNX Runtime native library loading
- [ ] Complete OAuth-backed Git push to `berth/<id>` branch
- [ ] Add idle activity tracking (WebSocket heartbeats)
- [ ] Warm pool pre-seeding on project creation

### Short-term (Beta)
- [ ] Replace host-network preview with isolated networking (CNI)
- [ ] Add TLS termination (Traefik + cert-manager)
- [ ] Add systemd units for API/worker
- [ ] Add Prometheus metrics + Grafana dashboards
- [ ] Implement model registry UI (compare, promote, rollback)
- [ ] Add feature importance (permutation/SHAP)
- [ ] Implement A/B testing framework

### Medium-term (Production Readiness)
- [ ] gVisor/runsc as default runtime
- [ ] Multi-node control plane (Raft consensus)
- [ ] Cilium network policies
- [ ] mTLS/SPIFFE for all service communication
- [ ] Multi-tenant isolation (namespace + cgroups)
- [ ] Custom domain support for previews
- [ ] Collaborative editing (Yjs/Automerge)

---

## 📊 Test Coverage Summary

```
backend/internal/analyzer           25 tests  ✅
backend/internal/usecase            33 tests  ✅
backend/internal/integration         7 tests  ✅
backend/internal/infrastructure     (cached)  ✅
frontend                            (build)   ✅
Total: 65 tests passing
```

---

## 📈 Recent Changes (v0.4.0)

### Phase 4: Prediction Engine
- Enhanced analyzer with 6 architecture types + 30+ frameworks
- Build strategy pattern (7 strategies)
- Data collector + model trainer (3 algorithms)
- ONNX binary export via protobuf
- ONNX Runtime inference (onnxruntime-go)
- gRPC + HTTP prediction endpoints
- Frontend Predict/Models/History tabs

### Phase 3: Frontend IDE
- Monaco editor with LSP
- xterm.js terminal with PTY
- Git panel (status, branch, diff, log)
- File operations (CRUD, move, duplicate)
- Preview proxy

### Phase 2: Containerd Infrastructure
- Layer commit + tar export
- go-iptables network setup
- Warm pool reuse
- Exact-image container reuse

---

## 📅 Roadmap

See [ROADMAP.md](ROADMAP.md) for detailed timeline.

---

## 📝 Notes

- All benchmarks are research artifacts, not validated claims
- OAuth Git push remains incomplete
- Collaborative editing is out of demo scope
- Current configuration is **not suitable for untrusted multi-tenant workloads**