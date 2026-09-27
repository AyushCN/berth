# Backend Architecture

## Tech Stack

| Technology | Version | Purpose |
|------------|---------|---------|
| Go | 1.21+ | Core language |
| Gin | 1.9 | HTTP Framework |
| GORM/SQLC | - | Database ORM/Query Builder |
| NATS | 1.3 | Message Bus |
| PostgreSQL | 16 | Primary Database |
| Redis | 7 | Cache/PubSub |
| containerd | 2.3 | Container Runtime |
| onnxruntime-go | 1.36 | ONNX Inference |
| gRPC | 1.62 | Prediction Service |
| Protobuf | 1.36 | Serialization |

---

## Project Structure

```
backend/
├── cmd/
│   ├── api/              # API Server entry
│   │   └── main.go       # Server setup, DI, routes
│   └── worker/           # Worker entry
│       └── main.go       # Worker setup, job processing
├── internal/
│   ├── analyzer/         # Runtime/Framework detection
│   │   └── analyzer.go   # Enhanced analyzer (6 arch, 30+ frameworks)
│   ├── config/           # Configuration loading
│   ├── delivery/
│   │   ├── http/         # HTTP Handlers
│   │   │   ├── handler/  # All HTTP handlers
│   │   │   ├── middleware/ # Auth, rate limit, CORS
│   │   │   └── router.go # Route registration
│   │   └── grpc/         # gRPC Server
│   │       └── prediction_server.go
│   ├── domain/           # Core Domain Models
│   │   ├── prediction.go # Prediction, Model, TrainingData
│   │   ├── runtime_profile.go
│   │   ├── build.go
│   │   ├── sandbox.go
│   │   ├── git.go
│   │   └── ...
│   ├── infrastructure/
│   │   ├── containerd/   # Containerd client
│   │   ├── docker/       # Docker client
│   │   ├── db/           # PostgreSQL connection
│   │   ├── github/       # GitHub OAuth
│   │   ├── nats/         # NATS client
│   │   └── redis/        # Redis client
│   ├── repository/       # Data Access Layer
│   │   ├── *.go          # Repository implementations
│   │   ├── *.sql.go      # SQLC generated
│   │   └── querier.go    # Interface
│   ├── usecase/          # Business Logic
│   │   ├── analyzer.go   # Build planner, strategies
│   │   ├── prediction_service.go
│   │   ├── model_trainer.go
│   │   ├── data_collector.go
│   │   ├── build_planner.go
│   │   ├── *_strategy.go # Build strategies
│   │   └── ...
│   ├── worker/           # Worker Implementation
│   │   ├── sandbox_worker.go
│   │   └── watcher.go
│   └── integration/      # Integration Tests
└── proto/                # gRPC Protobuf Definitions
    └── prediction.proto
```

---

## Domain Models

### Core Entities

| Entity | Table | Key Fields |
|--------|-------|------------|
| `User` | `users` | id, email, username, github_id, avatar_url |
| `Organization` | `organizations` | id, name |
| `Project` | `projects` | id, name, description, owner_org_id, is_public |
| `Workspace` | `workspaces` | id, project_id, name, type (CANONICAL/FORK), owner_id, git_branch |
| `Environment` | `environments` | id, workspace_id, runtime_profile_id, state, container_id, public_url |
| `RuntimeProfile` | `runtime_profiles` | detection_evidence, language, framework, architecture, port |
| `BuildPlan` | `build_plans` | base_image, dockerfile, build_args, commands, confidence |
| `Build` | `builds` | build_plan_id, workspace_id, commit_hash, status, cache_hit, duration |

### Prediction Models
| Entity | Table | Key Fields |
|--------|-------|------------|
| `Prediction` | `predictions` | type, workspace_id, input, output, confidence, model_version |
| `Model` | `models` | name, version, type, algorithm, parameters, metrics, onnx_path, is_active |
| `TrainingData` | `training_data` | features, labels, architecture, framework, language, cache_key |

---

## Key Use Cases

### 1. Runtime Analysis (`internal/analyzer`)
```go
// EnhancedAnalyzer provides comprehensive detection
type EnhancedAnalyzer struct {
    workspaceDir string
    archDetector *ArchitectureDetector
    fwDetector   *FrameworkDetector
}

func (e *EnhancedAnalyzer) Analyze() *DetectionResult {
    // Architecture: MONOREPO, LIBRARY, CLI, API, FULL_STACK, WEB_APP
    // Framework: 30+ across Go, Rust, Java, Python, Node
    // Entry Points: Dockerfile, compose, main files
    // Lockfiles: 18 types with SHA256
    // Docker Compose: Full parse
}
```

**Architecture Detection Priority**: MONOREPO → LIBRARY → CLI → API → FULL_STACK → WEB_APP

**Framework Detection**: Go (gin/fiber/echo/chi), Rust (actix/axum/rocket), Java (spring/quarkus), Python (django/fastapi/flask), Node (next/vite/express/nest)

### 2. Build Planner (`internal/usecase/build_planner.go`)
```go
type BuildPlanner struct {
    strategies []BuildStrategy
}

// Strategy Priority: Compose → Python → Node → Go → Rust → Java → Fallback
type BuildStrategy interface {
    Name() string
    Detect(result *analyzer.DetectionResult) bool
    GenerateBuildPlan(ctx context.Context, result *analyzer.DetectionResult) (*domain.BuildPlan, error)
}
```

### 3. Prediction Engine (`internal/usecase/`)
| Component | Responsibility |
|-----------|----------------|
| `DataCollector` | Collect training data from profiles/builds |
| `ModelTrainer` | Train Linear/RF/XGBoost, export ONNX |
| `PredictionService` | High-level prediction API |
| `ONNXInferenceEngine` | ONNX Runtime inference |
| `ONNXModelExporter` | Export models to binary ONNX |

**Algorithms**: Linear Regression (gradient descent), Random Forest (ensemble), XGBoost (gradient boosting)

### 3. Worker (`internal/worker/sandbox_worker.go`)
```go
type SandboxWorker struct {
    repo          domain.SandboxRepository
    runtime       domain.ContainerRuntime
    natsClient    *nats.Client
    dataCollector *usecase.DataCollector
}

func (w *SandboxWorker) processPending(ctx context.Context) {
    // 1. Pop pending sandbox
    // 2. Clone repo (with Git cache)
    // 3. Analyze workspace
    // 4. Generate build plan
    // 5. Create container
    // 6. Install dependencies
    // 7. Start application
    // 8. Update state + collect training data
}
```

---

## Database Layer

### SQLC Generated Queries
```bash
# Generate after migration changes
cd backend && sqlc generate
```

### Repository Pattern
```go
type BuildRepository interface {
    CreateBuildPlan(ctx context.Context, plan *domain.BuildPlan) error
    GetBuildPlan(ctx context.Context, id uuid.UUID) (*domain.BuildPlan, error)
    UpdateBuildPlan(ctx context.Context, plan *domain.BuildPlan) error
    // ...
}
```

### Migrations
```
000001_init.up.sql          # Core tables
000002_add_github_oauth.up.sql
000003_add_share_links.up.sql
000004_collaborative_platform.up.sql
000005_prediction_engine.up.sql  # Prediction tables
```

---

## gRPC Service

### Prediction Service (`proto/prediction.proto`)
```protobuf
service PredictionService {
  rpc PredictBuildTime(PredictBuildTimeRequest) returns (PredictionResponse);
  rpc PredictImageSize(PredictImageSizeRequest) returns (PredictionResponse);
  rpc PredictCacheHit(PredictCacheHitRequest) returns (PredictionResponse);
  rpc PredictFailureRisk(PredictFailureRiskRequest) returns (PredictionResponse);
  rpc GetPredictionHistory(GetPredictionHistoryRequest) returns (GetPredictionHistoryResponse);
  rpc GetModelMetrics(GetModelMetricsRequest) returns (GetModelMetricsResponse);
  rpc RetrainModel(RetrainModelRequest) returns (RetrainModelResponse);
  rpc ExportModelONNX(ExportModelONNXRequest) returns (ExportModelONNXResponse);
  rpc ActivateModel(ActivateModelRequest) returns (ActivateModelResponse);
  rpc ListModels(ListModelsRequest) returns (ListModelsResponse);
}
```

### Generate gRPC Code
```bash
protoc --go_out=. --go-grpc_out=. proto/prediction.proto
```

---

## HTTP Handlers

### Middleware Chain
```go
router.Use(gin.Recovery())
router.Use(middleware.Logger())
router.Use(middleware.CORS(cfg.FrontendURL))

api := router.Group("/api")
api.Use(middleware.RateLimit())
api.Use(middleware.Auth(cfg.JWTSecret))      // JWT validation
api.Use(middleware.RateLimitUser())          // Per-user rate limit
```

### Key Handlers
| Handler | Endpoints |
|---------|-----------|
| `AuthHandler` | `/auth/github`, `/auth/github/callback`, `/auth/me` |
| `SandboxHandler` | `/environments*`, `/p/:id/*` (preview proxy) |
| `FileHandler` | `/environments/:id/files*` |
| `GitHandler` | `/environments/:id/git*` |
| `PredictionHandler` | `/predictions/*` |
| `ChangeRequestHandler` | `/change-requests*`, `/projects/:id/change-requests` |
| `ShareLinkHandler` | `/projects/:id/share-links*`, `/join` |
| `ActivityHandler` | `/activity*`, `/environments/idle` |

---

## Configuration

### Config Structure (`internal/config/config.go`)
```go
type Config struct {
    DatabaseURL       string
    RedisURL          string
    NATSURL           string
    JWTSecret         string
    GithubClientID    string
    GithubClientSecret string
    FrontendURL       string
    Port              string
    Mode              string        // "api" | "worker"
    Env               string        // "development" | "production"
    DockerHost        string
    DockerNetwork     string
    TraefikDomain     string
    GithubCallbackURL string
}
```

### Load Priority
1. Environment variables
2. `.env` file
3. Defaults

---

## Build & Run

### Commands
```bash
# Build API
go build -o bin/berth-api ./cmd/api

# Build Worker
go build -o bin/berth-worker ./cmd/worker

# Run API
ENCRYPTION_KEY=... go run ./cmd/api

# Run Worker
MODE=worker ENCRYPTION_KEY=... go run ./cmd/worker

# Generate SQLC
sqlc generate

# Generate gRPC
protoc --go_out=. --go-grpc_out=. proto/prediction.proto
```

### Environment Variables
```bash
export ENCRYPTION_KEY=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
export DATABASE_URL=postgres://berth:berth@localhost:5432/berth?sslmode=disable
export REDIS_URL=redis://localhost:6379
export NATS_URL=nats://localhost:4222
export JWT_SECRET=your-32-byte-secret-key-here!!
export GITHUB_CLIENT_ID=xxx
export GITHUB_CLIENT_SECRET=xxx
export FRONTEND_URL=http://localhost:3000
export MODEL_DIR=/tmp/berth/models
```

---

## Testing

```bash
# All tests
ENCRYPTION_KEY=... go test ./... -v

# Specific packages
go test ./internal/analyzer/... -v
go test ./internal/usecase/... -v
go test ./internal/integration/... -v -timeout 2m

# Race detection
go test -race ./...

# Coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Test Packages
| Package | Tests | Focus |
|---------|-------|-------|
| `internal/analyzer` | 25 | Architecture, frameworks, entry points, lockfiles |
| `internal/usecase` | 33 | Build planner, strategies, model trainer, data collector |
| `internal/integration` | 7 | End-to-end flow, 5 project types |

---

## Dependencies

### Direct Dependencies
```go
require (
    github.com/gin-gonic/gin v1.12.0
    github.com/google/uuid v1.6.0
    github.com/jackc/pgx/v5 v5.11.0
    github.com/nats-io/nats.go v1.53.1
    github.com/redis/go-redis/v9 v9.22.0
    github.com/containerd/containerd/v2 v2.3.5
    github.com/yalue/onnxruntime_go v1.36.0
    google.golang.org/grpc v1.62.0
    google.golang.org/protobuf v1.36.12
    gopkg.in/yaml.v3 v3.0.1
)
```

---

## Extending the Backend

### Adding New Runtime Detection
1. Add framework constants in `analyzer.go`
2. Implement `detect<Language>()` in `FrameworkDetector`
2. Add lockfile types in `detectLockfiles()`
3. Add entry points in `detectEntryPoints()`

### Adding New Build Strategy
1. Implement `BuildStrategy` interface
2. Register in `NewBuildPlanner()`
4. Add strategy-specific Dockerfile generation

### Adding New Prediction Type
1. Add `PredictionType` constant
2. Add label extraction in `getLabelKeys()`
3. Add training data collection
4. Add gRPC + HTTP endpoints
5. Update frontend prediction UI

---

## Monitoring & Debugging

### Logs
```bash
# API logs
journalctl -u berth-api -f

# Worker logs
journalctl -u berth-worker -f

# NATS logs
nats log stream
```

### Health Checks
```
GET /health                    # API health
GET /api/health               # API + DB + Redis + NATS
```

### Profiling
```bash
# pprof endpoints (if enabled)
go tool pprof http://localhost:8080/debug/pprof/heap
go tool pprof http://localhost:8080/debug/pprof/profile
```

---

## Common Issues

| Issue | Solution |
|-------|----------|
| `ENCRYPTION_KEY` not set | Export 32-byte hex key |
| `go-iptables` missing | `apt-get install libiptables-dev` |
| `libonnxruntime.so` missing | Install ONNX Runtime or disable ONNX |
| PostgreSQL connection refused | Check `DATABASE_URL`, run migrations |
| NATS connection failed | Verify `NATS_URL`, check JetStream |
| Frontend CORS errors | Check `FRONTEND_URL` matches origin |