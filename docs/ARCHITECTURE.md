# Architecture

## Overview

Berth is a single-host sandbox prototype with a control plane (Go API + PostgreSQL + NATS) and a worker that provisions containers via containerd.

```
┌─────────────┐     ┌──────────────┐     ┌─────────────────┐
│  Browser    │────▶│  Next.js     │────▶│  Go API         │
│  (Frontend) │     │  (UI)        │     │  (Control Plane)│
└─────────────┘     └──────────────┘     └───────┬─────────┘
                                                  │
         ┌────────────────────────────────────────┼────────────────────────┐
         │                                        │                        │
         ▼                                        ▼                        ▼
┌──────────────────┐                    ┌──────────────────┐      ┌──────────────────┐
│  PostgreSQL      │                    │  NATS            │      │  containerd      │
│  (State Store)   │                    │  (Message Bus)   │      │  (Container      │
│                  │                    │                  │      │   Runtime)       │
└──────────────────┘                    └──────────────────┘      └────────┬─────────┘
                                                                            │
                                                                     ┌────────▼─────────┐
                                                                     │  Sandbox         │
                                                                     │  (Container)     │
                                                                     └──────────────────┘
```

---

## Core Components

### 1. API Server (`cmd/api`)
- **Framework**: Gin HTTP server
- **Auth**: JWT + GitHub OAuth
- **Endpoints**: REST + WebSocket
- **Auth Middleware**: JWT validation, rate limiting
- **WebSocket Hub**: Real-time terminal, file edits, presence

### 2. Worker (`cmd/worker`)
- **Job Processing**: Consumes NATS `berth.sandbox.create`
- **Repository Cloning**: GitHub HTTPS cloning with caching
- **Runtime Analysis**: Post-clone detection via analyzer
- **Container Lifecycle**: Create → Start → Install → Run
- **Cleanup**: Expiry sweep, stop/delete via NATS

### 3. Runtime Analyzer (`internal/analyzer`)
- **Architecture Detection**: MONOREPO, LIBRARY, CLI, API, FULL_STACK, WEB_APP
- **Framework Detection**: 30+ frameworks across 5 languages
- **Entry Points**: Dockerfile, docker-compose, main files
- **Lockfile Detection**: 18 lockfile types with SHA256 cache keys
- **Docker Compose Parsing**: Full service/network/volume parsing

### 4. Build Planner (`internal/usecase/build_planner.go`)
- **Strategy Pattern**: Compose → Python → Node → Go → Rust → Java → Fallback
- **BuildPlan Generation**: Dockerfile, build args, commands, ports
- **Strategy Detection**: Lockfiles, runtime profile, Docker Compose

### 5. Prediction Engine (`internal/usecase/`)
- **DataCollector**: Collects training data from builds/profiles
- **ModelTrainer**: Linear Regression, Random Forest, XGBoost
- **PredictionService**: Build time, image size, cache hit, failure risk
- **ONNX Export**: Binary protobuf ONNX model export
- **gRPC Service**: Prediction, model management, retraining

### 6. Infrastructure
- **containerd**: Rootless, runc.v2, layer commit, tar export
- **PostgreSQL**: SQLC-generated queries, migrations
- **NATS**: JetStream for job orchestration
- **Redis**: PubSub for WebSocket, session cache
- **Docker**: Image building, registry interaction

---

## Data Flow

### Sandbox Provisioning
```
1. Frontend → POST /api/environments (repo URL)
2. API → Create pending sandbox in PostgreSQL
3. API → Publish NATS job (berth.sandbox.create)
4. Worker → Consume job, clone repo (with Git cache)
4. Worker → Analyzer.Analyze(workspace) → RuntimeProfile
5. Worker → BuildPlanner.GenerateBuildPlan() → BuildPlan
6. Worker → containerd.CreateSandbox() → Container
7. Worker → Install deps, start app
8. Worker → Update sandbox state (RUNNING, public_url)
9. Frontend ← Poll /api/environments/:id → Status
```

### Prediction Flow
```
1. Sandbox created → DataCollector.CollectFromProfile()
2. Build completes → DataCollector.CollectFromBuild()
3. Scheduled → ModelTrainer.TrainModel() (Linear/RF/XGBoost)
4. Export → ModelTrainer.ExportONNX() → .onnx file
5. Request → PredictionService.Predict() → ONNX inference
6. Fallback → Mock prediction if ONNX unavailable
```

---

## Trust Boundaries & Security

| Boundary | Implementation | Limitation |
|----------|----------------|------------|
| **User ↔ API** | JWT + GitHub OAuth | No mTLS |
| **API ↔ Worker** | NATS (no auth in dev) | No mutual TLS |
| **Worker ↔ containerd** | Unix socket (rootless) | No remote access |
| **Sandbox Isolation** | Rootless containerd + runc.v2 | No gVisor, host networking |
| **Network** | Host network namespace | No per-sandbox isolation |
| **Secrets** | Encrypted at rest (AES-GCM) | Keys in env vars |

**Not Suitable For**: Multi-tenant public service, untrusted workloads, production deployment.

---

## Database Schema

### Core Tables
- `users` - GitHub OAuth users
- `organizations` - Workspaces/teams
- `projects` - GitHub repositories
- `workspaces` - Canonical + fork workspaces
- `environments` - Running sandboxes
- `runtime_profiles` - Detected runtime specs
- `build_plans` - Build instructions
- `builds` - Build history
- `images` - Built container images
- `change_requests` - Editor→Owner workflow

### Prediction Tables
- `predictions` - Model predictions
- `models` - Trained ML models
- `training_data` - Build data for training

---

## Configuration

### Environment Variables
| Variable | Required | Description |
|----------|----------|-------------|
| `DATABASE_URL` | Yes | PostgreSQL connection string |
| `REDIS_URL` | Yes | Redis connection string |
| `NATS_URL` | No | NATS server URL |
| `JWT_SECRET` | Yes | 32+ byte JWT signing key |
| `GITHUB_CLIENT_ID` | Yes | GitHub OAuth app ID |
| `GITHUB_CLIENT_SECRET` | Yes | GitHub OAuth secret |
| `FRONTEND_URL` | Yes | Frontend origin for CORS |
| `ENCRYPTION_KEY` | Yes | 32-byte hex AES-GCM key |
| `DOCKER_HOST` | Yes | Docker/containerd socket |
| `TRAEFIK_DOMAIN` | Yes | Preview domain |
| `MODEL_DIR` | No | ONNX model storage |

---

## Limitations

| Area | Current State |
|------|---------------|
| **Multi-host** | Not supported |
| **Network Isolation** | Host networking only |
| **Runtime Hardening** | runc.v2 only, no gVisor |
| **Multi-tenancy** | Not implemented |
| **Preview Gateway** | Host network proxy only |
| **mTLS/SPIFFE** | Not implemented |
| **Collaborative Editing** | Out of scope |
| **OAuth Git Push** | Incomplete |