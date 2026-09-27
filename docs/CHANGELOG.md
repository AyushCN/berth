# Changelog

All notable changes to Berth will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

### v0.3.0 → v0.4.0
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
| v0.4.x | Active | 2025-06-27 |
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
8. **Announce**: Discord, Twitter, Blog