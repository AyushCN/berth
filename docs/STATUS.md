# Project Status

**Berth is an early-stage, single-host research prototype.** This inventory describes the checked-in implementation, not a production readiness claim.

---

## ✅ Implemented Foundations

### Core Platform
- [x] **Go API + Worker** with PostgreSQL persistence and NATS-based job orchestration (fallback polling)
- [x] **Docker Integration** — create/start/stop/exec lifecycle, idempotent container creation, readiness probe
- [x] **Runtime Detection** — Rule-based post-clone detection for Node.js, Python, Go, Rust, Java
- [x] **Architecture Classification** — WEB_APP, API, CLI, FULL_STACK, MONOREPO, LIBRARY
- [x] **Framework Detection** — 30+ frameworks (Next.js, Express, FastAPI, Django, Gin, Axum, Spring, etc.)
- [x] **Dependency Caching** — pnpm store, Cargo, Go modules, pip/poetry, Maven/Gradle
- [x] **Docker Compose Support** — Parse and build multi-service compose files
- [x] **Lockfile Detection** — 18 lockfile types with SHA256 cache keys
- [x] **Entry Point Detection** — Dockerfile, docker-compose, main files with ambiguity handling

### Git & Collaboration
- [x] **GitHub OAuth** — Encrypted token storage (AES-GCM)
- [x] **Git Operations** — Clone, branch, checkout, commit, push, diff, log
- [x] **Change Requests** — Editor → Owner workflow with merge
- [x] **Share Links** — Project invitation via short codes with role/usage limits

### Frontend (Next.js 15 + React 18)
- [x] **Authentication** — GitHub OAuth, dev login, JWT
- [x] **Dashboard** — Environment list, create, fork, delete
- [x] **IDE** — File tree, Monaco editor, terminal (xterm.js), Git panel
- [x] **Preview Proxy** — `/p/:id/*` → sandbox container
- [x] **Git Panel** — Status, branches, commit, push, pull, diff, log
- [x] **File Operations** — List, read, write, create, delete
- [x] **Terminal** — xterm.js + WebSocket + PTY
- [x] **Presence** — User avatars, cursors (infrastructure)

### Security Hardening
- [x] **Container Isolation** — `--read-only`, `--cap-drop ALL`, `--init`, `--user 1000:1000`, `no-new-privileges`
- [x] **Host Networking** — Containers on host network, no cross-container communication
- [x] **Secrets** — GitHub tokens AES-256-GCM encrypted at rest; JWT signed with HS256
- [x] **Cookies** — `HttpOnly`, `SameSite=Lax`, `Secure` derived from TLS/X-Forwarded-Proto
- [x] **Rate Limiting** — 200/min per IP (global), 120/min per user (authenticated routes)
- [x] **Dev Login** (`/auth/dev-login`) — **Not registered in production**

### Infrastructure
- [x] **PostgreSQL 16** — SQLC-generated type-safe queries, embedded migrations applied on boot
- [x] **NATS 2.10** — JetStream for job orchestration
- [x] **Redis 7** — PubSub for WebSocket real-time updates, session cache
- [x] **Docker** — Container lifecycle via host Docker socket (host networking)
- [x] **Embedded Migrations** — SQL migrations in `backend/migrations/` with `go:embed`, auto-applied on boot

---

## 🚧 Current Limitations

### Runtime & Security
| Limitation | Impact |
|------------|--------|
| **Single-host only** | No multi-node, no HA |
| **No gVisor/Cilium/mTLS** | Rootless Docker only |
| **Host networking only** | No cross-container communication |
| **No mTLS/SPIFFE** | Plaintext NATS, PostgreSQL connections |
| **No multi-tenancy** | Single-tenant model |

### Preview & Networking
| Limitation | Impact |
|------------|--------|
| **Host network proxy** | `/p/:id/` → host port, no CNI |
| **No TLS termination** | HTTP only, no cert management |
| **No custom domains** | Fixed Traefik domain only |

### Git & Collaboration
| Limitation | Impact |
|------------|--------|
| **OAuth push incomplete** | Owner push to `berth/<id>` branch partial |
| **No collaborative editing** | No OT/CRDT, no presence sync |
| **No code review UI** | Change requests API only |

### Infrastructure
| Limitation | Impact |
|------------|--------|
| **No CI/CD pipeline** | Manual releases only |
| **No migration runner** | Migrations applied on boot (no rollback via `migrate` CLI) |
| **No readiness for static sites** | Static sites report `CRASHED` (no server to probe) |
| **No gVisor/CNI/mTLS** | Rootless Docker only |

---

## ✅ Current Test Coverage

- [x] **Analyzer Tests** — 25 tests (architecture, frameworks, entry points, lockfiles)
- [x] **Integration Tests** — 7 tests (analyzer, e2e flow)
- [x] **Migration Tests** — 3 tests (empty DB migrate, idempotency, version reading)
- [x] **Usecase Tests** — 18 tests (crypto, container_control, access control, worker)
- [x] **Handler Tests** — 4 tests (ws_hub, dev_login)

---

## 📋 Work Needed for Usable Product

### Immediate (Demo Ready)
- [ ] Add readiness probe that waits for actual app to listen before marking RUNNING
- [ ] Complete OAuth-backed Git push to `berth/<id>` branch
- [ ] Add idle activity tracking (WebSocket heartbeats)

### Short-term (Beta)
- [ ] Add TLS termination (Traefik + cert-manager)
- [ ] Add systemd units for API/worker
- [ ] Add Prometheus metrics + Grafana dashboards
- [ ] Add readiness probing for apps that don't listen on a port (static sites)

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

| Package | Tests | Coverage |
|---------|-------|----------|
| analyzer | 25 | ~85% |
| crypto | 4 | ~90% |
| handler | 4 | ~70% |
| integration | 7 | ~60% |
| migrations | 3 | ~80% |
| usecase | 18 | ~75% |
| worker | 2 | ~60% |

---

## 📚 Related Documentation

- [Architecture](ARCHITECTURE.md) — System architecture, components, data flow
- [API Reference](API.md) — REST endpoints, WebSocket, schemas
- [Development](DEVELOPMENT.md) — Local dev setup, workflows
- [Deployment](DEPLOYMENT.md) — Production deployment
- [Security](SECURITY.md) — Threat model, hardening