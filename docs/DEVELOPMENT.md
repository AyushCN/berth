# Development Guide

## Prerequisites

### System Requirements
- **OS**: Linux (Ubuntu 22.04+, Fedora 39+, Arch) or macOS (with Docker Desktop)
- **Go**: 1.21+
- **Node.js**: 20+ (LTS)
- **Docker**: 24+ with Compose v2
- **containerd**: 1.7+ (rootless)
- **PostgreSQL**: 16+
- **NATS**: 2.10+
- **Redis**: 7+
- **Git**: 2.40+

### Install on Ubuntu/Debian
```bash
# Go
wget https://go.dev/dl/go1.22.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc

# Node.js (via nvm)
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39/install.sh | bash
nvm install 20

# Docker
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER

# containerd rootless
# See DOCKER.md

# PostgreSQL
sudo apt-get install postgresql-16 postgresql-client-16

# NATS
curl -sfL https://github.com/nats-io/nats-server/releases/download/v2.10.0/nats-server-v2.10.0-linux-amd64.tar.gz | tar xz
sudo mv nats-server /usr/local/bin/

# Redis
sudo apt-get install redis-server
```

### macOS (with Docker Desktop)
```bash
brew install go node@20 docker docker-compose postgresql@16 nats-server redis
brew services start postgresql@16 redis nats-server
```

---

## Project Setup

### Clone Repository
```bash
git clone https://github.com/yourorg/berth.git
cd berth
```

### Backend Setup
```bash
cd backend

# Download dependencies
go mod download

# Generate SQLC code
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
sqlc generate

# Generate gRPC code
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
protoc --go_out=. --go-grpc_out=. proto/prediction.proto

# Set environment
export ENCRYPTION_KEY=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
export DATABASE_URL=postgres://berth:berth@localhost:5432/berth?sslmode=disable
export REDIS_URL=redis://localhost:6379
export NATS_URL=nats://localhost:4222
export JWT_SECRET=dev-secret-key-32-characters-long!!
export GITHUB_CLIENT_ID=your-github-client-id
export GITHUB_CLIENT_SECRET=your-github-client-secret
export FRONTEND_URL=http://localhost:3000
export MODE=api
export PORT=8080
export MODEL_DIR=/tmp/berth/models
```

### Frontend Setup
```bash
cd frontend

# Install dependencies
npm install

# Environment
echo "NEXT_PUBLIC_API_URL=http://localhost:8080" > .env.local
```

### Database Setup
```bash
# Create database
createdb -U berth berth

# Run migrations
cd backend
make migrate-up

# Or manually
psql -U berth -d berth -f migrations/000001_init.up.sql
psql -U berth -d berth -f migrations/000002_add_github_oauth.up.sql
psql -U berth -d berth -f migrations/000003_add_share_links.up.sql
psql -U berth -d berth -f migrations/000004_collaborative_platform.up.sql
psql -U berth -d berth -f migrations/000005_prediction_engine.up.sql
```

### Infrastructure
```bash
# Start PostgreSQL, Redis, NATS
make up
# or
docker compose -f docker-compose.dev.yml up -d

# Verify
docker compose -f docker-compose.dev.yml ps
```

---

## Running the Application

### Option 1: Make Commands
```bash
# Start everything (infra + API + Worker + Frontend)
make dev

# Or individually:
make up              # Start infra only
make migrate-up      # Run migrations
cd backend && go run ./cmd/api    # API server
MODE=worker go run ./cmd/worker   # Worker
cd frontend && npm run dev        # Frontend
```

### Access URLs
| Service | URL |
|---------|-----|
| Frontend | http://localhost:3000 |
| API | http://localhost:8080 |
| API Health | http://localhost:8080/health |
| NATS Monitor | http://localhost:8222 |
| Traefik Dashboard | http://localhost:8080 (if enabled) |

### Development Credentials
```bash
# Dev login (auto-creates user)
curl -X GET http://localhost:8080/api/auth/dev-login
# Returns JWT token

# Or visit http://localhost:3000/login and click "Dev Login"
```

---

## Development Workflow

### Code Style
```bash
# Go formatting
go fmt ./...
goimports -w .

# Go linting
golangci-lint run ./...

# Frontend formatting
cd frontend && npm run lint
cd frontend && npm run format
```

### Git Workflow
```bash
# Feature branch
git checkout -b feat/your-feature

# Commit with conventional commits
git commit -m "feat(analyzer): add Rust Axum detection"

# Push and create PR
git push origin feat/your-feature
```

### Commit Message Format
```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`, `perf`

---

## Running Tests

### Backend Tests
```bash
# All tests
cd backend
ENCRYPTION_KEY=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef go test ./... -v

# Specific packages
go test ./internal/analyzer/... -v
go test ./internal/usecase/... -v
go test ./internal/integration/... -v -timeout 2m

# With race detector
go test -race ./...

# Coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Frontend Tests
```bash
cd frontend

# Unit tests
npm test

# Type checking
npm run type-check

# E2E (Playwright)
npm run test:e2e
```

### Integration Tests
```bash
# Requires Docker for testcontainers
cd backend
ENCRYPTION_KEY=... go test ./internal/integration/... -v -timeout 5m
```

---

## Code Generation

### SQLC (Database)
```bash
# After modifying SQL migrations
cd backend
sqlc generate
```

### gRPC (Protocol Buffers)
```bash
# After modifying proto files
cd backend
protoc --go_out=. --go-grpc_out=. proto/prediction.proto
```

### OpenAPI/Swagger (Planned)
```bash
# Generate from Gin routes
# swag init -g cmd/api/main.go
```

---

## Debugging

### Backend Debugging (Delve)
```bash
# Install dlv
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug API
dlv debug ./cmd/api --headless --listen=:2345 --api-version=2 --accept-multiclient

# Debug Worker
dlv debug ./cmd/worker --headless --listen=:2346 -- -MODE=worker
```

### VS Code Launch Config
```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug API",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}/backend/cmd/api",
      "env": {
        "ENCRYPTION_KEY": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
        "DATABASE_URL": "postgres://berth:berth@localhost:5432/berth?sslmode=disable"
      }
    },
    {
      "name": "Debug Worker",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}/backend/cmd/worker",
      "env": {
        "MODE": "worker",
        "ENCRYPTION_KEY": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
      }
    }
  ]
}
```

### Frontend Debugging
```bash
# Next.js dev server with debugging
cd frontend
npm run dev

# In VS Code: Debug > Add Configuration > Next.js
```

### Logs
```bash
# API logs
cd backend && go run ./cmd/api 2>&1 | jq .

# Worker logs
MODE=worker go run ./cmd/worker 2>&1 | jq .

# NATS logs
nats log stream

# Database logs
tail -f /var/log/postgresql/postgresql-16-main.log
```

---

## Hot Reloading

### Backend (Air)
```bash
# Install air
go install github.com/air-verse/air@latest

# Config (.air.toml)
root = "."
testdata_dir = "tmp"
tmp_dir = "tmp"

[build]
  cmd = "go build -o ./tmp/api ./cmd/api"
  bin = "./tmp/api"
  full_bin = ""
  watch = true
  poll = false
  poll_interval = 1000
  delay = 1000
  exclude_dir = ["assets", "tmp", "vendor", "frontend"]
  include_ext = ["go", "tpl", "tmpl", "html"]
  exclude_file = []
  exclude_regex = ["_test.go"]
  include_dir = []
  log = "build-errors.log"
  follow = false

# Run
cd backend && air
```

### Frontend (Next.js Fast Refresh)
```bash
# Automatic with `npm run dev`
# Fast Refresh enabled by default
```

---

## Database Management

### Migrations
```bash
# Create new migration
cd backend
sqlc generate  # After adding SQL files to migrations/

# Or create manually
cat > migrations/000006_new_feature.up.sql << 'EOF'
-- Your SQL here
EOF

cat > migrations/000006_new_feature.down.sql << 'EOF'
-- Rollback SQL
EOF
```

### Seed Data
```bash
# Insert dev user
psql -U berth -d berth << 'EOF'
INSERT INTO users (id, email, username, github_id, github_username, avatar_url)
VALUES ('00000000-0000-0000-0000-000000000001', 'dev@berth.local', 'dev_user', '0', 'dev_user', '')
ON CONFLICT DO NOTHING;
EOF
```

### Query Database
```bash
# Interactive
psql -U berth -d berth

# Query
psql -U berth -d berth -c "SELECT * FROM environments WHERE state = 'RUNNING';"

# JSON output
psql -U berth -d berth -c "SELECT json_agg(t) FROM (SELECT * FROM projects) t;"
```

---

## Common Issues

| Issue | Solution |
|-------|----------|
| `go: module not found` | `go mod tidy` |
| `sqlc: command not found` | `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest` |
| `protoc: command not found` | `apt-get install protobuf-compiler` |
| `ENCRYPTION_KEY not set` | Export 32-byte hex key |
| `go-iptables` missing | `apt-get install libiptables-dev` |
| `libonnxruntime.so` missing | Install ONNX Runtime or disable ONNX |
| Port 8080 in use | `lsof -i :8080` then kill |
| DB connection refused | Check PostgreSQL running, `DATABASE_URL` correct |
| NATS connection failed | Check NATS running, `NATS_URL` correct |
| Frontend CORS error | Check `FRONTEND_URL` matches origin |

---

## IDE Setup

### VS Code Extensions
```json
{
  "recommendations": [
    "golang.go",
    "bradlc.vscode-tailwindcss",
    "esbenp.prettier-vscode",
    "dbaeumer.vscode-eslint",
    "formulahendry.auto-rename-tag",
    "github.copilot",
    "golang.go",
    "ms-vscode.go",
    "redhat.vscode-yaml"
  ]
}
```

### Go Settings
```json
{
  "go.lintTool": "golangci-lint",
  "go.formatTool": "goimports",
  "go.testFlags": ["-v"],
  "go.testEnvVars": {
    "ENCRYPTION_KEY": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
  }
}
```

---

## Performance Profiling

```bash
# CPU Profile
go test -cpuprofile=cpu.prof -bench=. ./internal/analyzer
go tool pprof cpu.prof

# Memory Profile
go test -memprofile=mem.prof -bench=. ./internal/usecase
go tool pprof mem.prof

# HTTP Profile (if pprof enabled)
go tool pprof http://localhost:8080/debug/pprof/heap
go tool pprof http://localhost:8080/debug/pprof/profile
```

---

## Useful Commands Cheatsheet

```bash
# Quick test run
make test

# Build all
make build

# Clean build artifacts
make clean

# Regenerate all code
sqlc generate
protoc --go_out=. --go-grpc_out=. proto/prediction.proto

# Database reset
make migrate-down && make migrate-up

# Full clean rebuild
make clean && make build && make migrate-up
```