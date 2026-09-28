# Changelog

All notable changes to Berth will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [0.5.0] - 2025-09-28 - Security Hardening & Multi-Host (Phases G1-G5)

### Phase G5: Cross-Host Platform & Container Migration
- **VXLAN Mesh Networking** - Cross-host VXLAN overlay with ARP proxy, FDB management, peer health checks
- **CRIU Checkpoint/Restore** - Full checkpoint/restore with pre-dump iterations for live migration
- **Live Migration Orchestrator** - Phase-based migration (pre-dump → final dump → transfer → restore → network setup)
- **Migration Orchestrator** - Progress tracking, cancellation, status monitoring
- **CRIU Integration** - Pre-dump iterations (3x), final dump, rsync transfer, restore on target

### Phase G4: Tenant/Worker Isolation & Multi-host Scheduling
- **Tenant Model** - Resource quotas, allowed runtimes/network modes/filesystem modes, egress policies
- **Worker Agent** - Auto-registration, heartbeats (10s), resource monitoring, status transitions
- **Distributed Scheduler** - Binpack/Spread/LeastUsed strategies with preemption support
- **Worker Health** - Heartbeat monitoring, automatic status transitions (healthy→degraded→unhealthy→offline)
- **Multi-cluster Federation** - Cross-cluster scheduling, global scheduler with cost/latency scoring
- **Cross-cluster Migration** - MigrateOut/MigrateIn for cross-cluster migration

### Phase G3: Filesystem + Capability + Syscall Hardening
- **Filesystem Modes** - BindMount (default), OverlayFS (COW), Read-only rootfs with writable paths
- **OverlayFS COW** - Per-container upper/work dirs in tmpfs, copy-on-write isolation
- **Read-only Rootfs** - Explicit writable tmpfs mounts for /tmp, /var/tmp, /home, /workspace
- **Enhanced Capability Dropping** - Empty list = drop all, explicit allowlist support
- **NoNewPrivileges** - Prevent privilege escalation via setuid/setgid/capabilities
- **Seccomp Profiles** - Default (250+ syscalls), Restricted (minimal), Unrestricted (allow all)
- **NoNewPrivileges** - Prevent privilege escalation
- **PIDs Limits** - Fork bomb prevention (default 256)
- **tmpfs Mounts** - /tmp, /var/tmp with noexec/nosuid/nodev

### Phase G2: Network Isolation (CNI + Egress)
- **CNI Network Namespaces** - Per-sandbox isolated network namespaces
- **Network Modes** - CNI (isolated), Host (shared), None (no network)
- **CNI Bridge Plugin** - Bridge with host-local IPAM, per-sandbox IPs
- **Egress Policies** - Default (allow all), Restricted (DNS/HTTP/HTTPS only), None (block all), Trusted (allow all)
- **Port Forwarding** - iptables DNAT for host→container port mapping
- **CNI Lifecycle** - ADD/DEL with proper cleanup

### Phase G1: gVisor Execution
- **Explicit Runtime Selection** - Explicit RuntimeGVisor vs RuntimeRunc, no implicit fallback
- **Runtime Validation** - runsc availability check at startup
- **Warm Pool Separation** - Separate pools per runtime (gVisor vs runc)
- **ExecutionProfile** - Encapsulates all security settings (runtime, network, filesystem, capabilities, resources)

### Added
- **Cross-host VXLAN Networking** - VXLAN mesh with ARP proxy, FDB management, peer health checks
- **CRIU Checkpoint/Restore** - Full checkpoint/restore with pre-dump iterations (3x) for live migration
- **Live Migration Orchestrator** - Phase-based migration with progress tracking
- **Migration Orchestrator** - Phase-based migration with progress tracking
- **CRIU Integration** - Pre-dump iterations (3x), final dump, rsync transfer, restore on target
- **VXLAN Mesh Networking** - Cross-host VXLAN overlay, ARP proxy, FDB management, peer health
- **CRIU Checkpoint/Restore** - Full checkpoint/restore with pre-dump iterations for live migration
- **Live Migration Orchestrator** - Phase-based migration with progress tracking
- **Migration Orchestrator** - Phase-based migration with progress tracking
- **CRIU Integration** - Pre-dump iterations (3x), final dump, rsync transfer, restore on target

### Phase G4: Tenant/Worker Isolation & Multi-host Scheduling
- **Tenant Model** - Resource quotas, allowed configurations, egress policies
- **Worker Agent** - Auto-registration, heartbeats, resource monitoring
- **Distributed Scheduler** - Binpack/Spread/LeastUsed strategies with preemption
- **Worker Health** - Automatic status transitions based on resource usage
- **Multi-cluster Federation** - Cross-cluster scheduling, global scheduler
- **Cross-cluster Migration** - MigrateOut/MigrateIn for cross-cluster migration

### Phase G3: Filesystem + Capability + Syscall Hardening
- **Filesystem Modes** - BindMount, OverlayFS (COW), Read-only rootfs
- **OverlayFS COW** - Per-container upper/work dirs in tmpfs
- **Read-only Rootfs** - Explicit writable tmpfs mounts
- **Enhanced Capability Dropping** - Empty = drop all
- **NoNewPrivileges** - Prevent privilege escalation
- **Seccomp Profiles** - Default (250+), Restricted (minimal), Unrestricted
- **NoNewPrivileges** - Prevent privilege escalation
- **PIDs Limits** - Fork bomb prevention
- **tmpfs Mounts** - /tmp, /var/tmp with noexec/nosuid/nodev

### Phase G2: Network Isolation (CNI + Egress)
- **CNI Network Namespaces** - Per-sandbox isolated network namespaces
- **Network Modes** - CNI (isolated), Host (shared), None (no network)
- **CNI Bridge Plugin** - Bridge with host-local IPAM
- **Egress Policies** - Default, Restricted (DNS/HTTP/HTTPS), None, Trusted
- **Port Forwarding** - iptables DNAT for host→container port mapping
- **CNI Lifecycle** - ADD/DEL with proper cleanup

### Phase G1: gVisor Execution
- **Explicit Runtime Selection** - Explicit RuntimeGVisor vs RuntimeRunc
- **No Implicit Fallback** - No implicit fallback to runc for untrusted workloads
- **Runtime Validation** - runsc availability check at startup
- **Warm Pool Separation** - Separate pools by runtime
- **ExecutionProfile** - Encapsulates all security settings

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

### Phase 1: Foundation
- Go API Server - Gin framework, JWT auth, GitHub OAuth
- Worker - NATS job processing, sandbox lifecycle
- PostgreSQL - SQLC type-safe queries
- NATS - JetStream + fallback polling
- Redis - PubSub, session cache
- Runtime Detection - Node.js, Python, Go, Rust
- Frontend - Next.js 14, React 18, Tailwind CSS
- Database Schema - Users, Projects, Sandboxes, Collaborators
- Migrations - Versioned SQL migrations

---

## [0.4.0] - 2024-09-27 - Prediction Engine

### Added
- **Enhanced Analyzer** with 6 architecture types (MONOREPO, LIBRARY, CLI, API, FULL_STACK, WEB_APP)
- **Framework Detection** for 30+ frameworks across 5 languages:
  - Go: Gin, Fiber, Echo, Chi
  - Rust: Actix, Axum, Rocket, Salvo
  - Java: Spring Boot, Quarkus, Micronaut, Vert.x
  - Python: Django, FastAPI, Flask, Starlette, Quart, Tornado, Bottle
  - Node.js: Next.js, Vite, Express, NestJS, Koa, Fastify, Nuxt, Remix, SvelteKit, Astro
- **Multiple Entry Point Detection** with ambiguous entry handling
- **Docker Compose Parsing** - Full service/network/volume parsing
- **Lockfile Detection & Cache Keys** - 18 lockfile types with SHA256 hashing
- **Build Strategy Pattern** - Compose, Python, Node, Go, Rust, Java, Fallback strategies
- **Prediction Engine**:
  - Data Collector for training data
  - Model Trainer (Linear Regression, Random Forest, XGBoost)
  - Model Metrics (MSE, MAE, RMSE, R²)
  - ONNX Binary Export via protobuf
  - ONNX Runtime Inference (onnxruntime-go)
- **gRPC Prediction Service** - 10 methods (predict, history, models, retrain, export, activate)
- **HTTP Prediction Endpoints** - REST API for all prediction types
- **Frontend Prediction UI** - Predict/Models/History tabs with metrics visualization
- **Integration Tests** - 7 tests covering analyzer, build planner, end-to-end flow

### Changed
- Enhanced `Analyzer.Analyze()` with comprehensive detection
- Updated `BuildPlanner` with strategy pattern
- Improved `Analyzer.DetectionResult` with richer metadata

### Fixed
- Analyzer panics on missing `dependencies`/`devDependencies` fields
- Linear regression NaN metrics when ssTot=0
- ONNX export now uses binary protobuf instead of JSON

---

## [0.3.0] - 2024-Q3 - Frontend IDE

### Added
- **Monaco Editor** integration with LSP support
- **xterm.js Terminal** with PTY support
- **Git Panel** - Status, branches, commit, push, pull, diff, log
- **File Operations** - CRUD, move, duplicate, search
- **Preview Proxy** - `/p/:id/*` → sandbox container
- **Collaborative Infrastructure** - Presence, cursors (infrastructure)
- **Share Links** - Project invitation via short codes
- **Change Requests** - Editor → Owner workflow

### Changed
- Migrated from Sandboxes to Environments/Workspaces
- Updated to Next.js 15 App Router
- Improved WebSocket architecture for real-time features

### Fixed
- WebSocket reconnection logic
- File tree performance with large repositories
- Terminal resize handling

---

## [0.2.0] - 2024-Q2 - Containerd Infrastructure

### Added
- **Containerd Integration** - Rootless, runc.v2
- **Layer Commit** - Commit container changes to new image
- **Tar Export** - Export images for portability
- **go-iptables** Network Setup - Port mapping, isolation
- **Warm Pool** - Exact-image container reuse
- **Dependency Caching** - pnpm, Cargo, Go modules, pip
- **Exact-Image Reuse** - Skip rebuild for unchanged images

### Changed
- Migrated from Docker API to containerd native API
- Improved worker job processing with NATS JetStream

### Fixed
- Layer commit corruption on concurrent builds
- Network cleanup on container deletion
- Warm pool eviction logic

---

## [0.1.0] - 2024-Q1 - Foundation

### Added
- **Go API Server** - Gin framework, JWT auth, GitHub OAuth
- **Worker** - NATS job processing, sandbox lifecycle
- **PostgreSQL** - SQLC type-safe queries
- **NATS** - JetStream + fallback polling
- **Redis** - PubSub, session cache
- **Runtime Detection** - Node.js, Python, Go, Rust
- **Frontend** - Next.js 14, React 18, Tailwind CSS
- **Database Schema** - Users, Projects, Sandboxes, Collaborators
- **Migrations** - Versioned SQL migrations

---

## Upgrade Guide

### v0.4.0 → v0.5.0
```bash
# 1. Pull new images
docker compose pull

# 2. Run new migrations
docker compose exec api migrate -path /migrations -database "$DATABASE_URL" up

# 3. Set new env vars
export MODEL_DIR=/var/lib/berth/models

# 4. Restart services
docker compose up -d
```

### v0.3.0 → v0.4.0
```bash
# 1. Run migrations
make migrate-up

# 2. Update frontend
cd frontend && npm install && npm run build

# 3. Restart services
docker compose up -d
```

### v0.2.0 → v0.3.0
```bash
# 1. Run migrations
make migrate-up

# 2. Update frontend
cd frontend && npm install && npm run build

# 3. Restart services
docker compose up -d
```

### v0.1.0 → v0.2.0
```bash
# 1. Install containerd
# 2. Configure rootless containerd
# 3. Update worker to use containerd
# 4. Run new migrations
make migrate-up
```

---

## Deprecations

| Feature | Deprecated In | Removed In | Replacement |
|---------|---------------|------------|-------------|
| `Sandbox` type | v0.3.0 | v0.4.0 | `Environment` + `Workspace` |
| `analyzer.Analyze()` basic | v0.4.0 | - | `EnhancedAnalyzer.Analyze()` |
| Docker API client | v0.2.0 | v0.3.0 | containerd native API |
| Single sandbox per project | v0.3.0 | - | Multiple workspaces per project |

---

## Migration Notes

### Database Migrations
```bash
# Check current version
docker exec berth_postgres psql -U berth -c "SELECT version FROM schema_migrations;"

# Run pending migrations
docker compose exec api migrate -path /migrations -database "$DATABASE_URL" up

# Rollback (if needed)
docker compose exec api migrate -path /migrations -database "$DATABASE_URL" down 1
```

### Breaking Changes

#### v0.5.0
- `ExecutionProfile` struct encapsulates all security settings
- `SandboxSpec` now requires `ExecutionProfile` field
- NetworkMode defaults to CNI for untrusted workloads

#### v0.4.0
- `analyzer.DetectionResult` struct expanded with new fields
- `BuildPlan` now requires `Confidence` field
- Prediction endpoints require authentication

#### v0.3.0
- `Sandbox` → `Environment` + `Workspace` (database migration required)
- WebSocket protocol changed (added presence messages)
- Project API response format changed

---

## Deprecations

| Feature | Deprecated In | Removed In | Replacement |
|---------|---------------|------------|-------------|
| `Sandbox` type | v0.3.0 | v0.4.0 | `Environment` + `Workspace` |
| `analyzer.Analyze()` basic | v0.4.0 | - | `EnhancedAnalyzer.Analyze()` |
| Docker API client | v0.2.0 | v0.3.0 | containerd native API |
| Single sandbox per project | v0.3.0 | - | Multiple workspaces per project |

---

## Migration Notes

### Database Migrations
```bash
# Check current version
docker exec berth_postgres psql -U berth -c "SELECT version FROM schema_migrations;"

# Run pending migrations
docker compose exec api migrate -path /migrations -database "$DATABASE_URL" up

# Rollback (if needed)
docker compose exec api migrate -path /migrations -database "$DATABASE_URL" down 1
```

### Breaking Changes

#### v0.5.0
- `ExecutionProfile` struct encapsulates all security settings
- `SandboxSpec` now requires `ExecutionProfile` field
- NetworkMode defaults to CNI for untrusted workloads

#### v0.4.0
- `analyzer.DetectionResult` struct expanded with new fields
- `BuildPlan` now requires `Confidence` field
- Prediction endpoints require authentication

#### v0.3.0
- `Sandbox` → `Environment` + `Workspace` (database migration required)
- WebSocket protocol changed (added presence messages)
- Project API response format changed

---

## Security Advisories

| Version | CVE | Severity | Fixed In |
|---------|-----|----------|----------|
| < v0.2.0 | CVE-2024-XXXXX | Medium | v0.2.0 (containerd update) |
| < v0.3.0 | CVE-2024-YYYYY | Low | v0.3.0 (input validation) |

---

## Support Policy

| Version | Status | Supported Until |
|---------|--------|-----------------|
| v0.5.x | Active | 2025-12-28 |
| v0.4.x | Maintenance | 2025-06-27 |
| v0.3.x | Maintenance | 2024-12-27 |
| v0.2.x | EOL | 2024-09-27 |
| v0.1.x | EOL | 2024-06-27 |

---

## Release Process

1. **Create Release Branch**: `release/v0.x.y`
2. **Update Version**: `go run ./cmd/version` or manual
3. **Update Changelog**: Move "Unreleased" to version
4. **Run Full Test Suite**: `make test && make lint`
5. **Build Images**: `docker build -t berth/api:v0.x.y ...`
6. **Tag Release**: `git tag v0.x.y`
7. **Publish**: GitHub Release + Docker Hub
7. **Announce**: Discord, Twitter, Blog