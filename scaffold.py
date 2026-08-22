import os

base = "."
os.makedirs(base, exist_ok=True)

# 1. CI Pipeline
ci_yml = """name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  lint-backend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23.x'
      - name: golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: v1.60
          working-directory: ./backend
      - name: go vet
        run: cd backend && go vet ./...

  test-backend:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16-alpine
        env:
          POSTGRES_USER: berth
          POSTGRES_PASSWORD: berth
          POSTGRES_DB: berth_test
        ports:
          - 5432:5432
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
      redis:
        image: redis:7-alpine
        ports:
          - 6379:6379
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23.x'
      - name: Install sqlc
        run: |
          curl -L https://github.com/sqlc-dev/sqlc/releases/download/v1.27.0/sqlc_1.27.0_linux_amd64.tar.gz | tar xz -C /tmp
          sudo mv /tmp/sqlc /usr/local/bin/
      - name: sqlc generate
        run: cd backend && sqlc generate
      - name: golang-migrate
        run: |
          curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.1/migrate.linux-amd64.tar.gz | tar xz -C /tmp
          sudo mv /tmp/migrate /usr/local/bin/
      - name: Run migrations
        run: cd backend && migrate -path migrations -database "postgres://berth:berth@localhost:5432/berth_test?sslmode=disable" up
      - name: go test
        run: cd backend && go test -race -coverprofile=coverage.out ./...
        env:
          DATABASE_URL: postgres://berth:berth@localhost:5432/berth_test?sslmode=disable
          REDIS_URL: redis://localhost:6379

  lint-frontend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
      - run: cd frontend && npm ci
      - run: cd frontend && npm run lint
      - run: cd frontend && npm run build
        env:
          NEXT_PUBLIC_API_URL: http://localhost:8080

  test-ml:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
        with:
          python-version: '3.11'
      - run: cd ml && pip install -r requirements.txt
      - run: cd ml && pytest --cov=features --cov=models tests/
"""

# 2. Architecture Doc
arch_md = """# Architecture & Trust Boundaries

## Overview

Berth is a single-host ephemeral sandbox platform for research on predictive pre-warming and gVisor-based isolation.

## Trust Boundaries

```
┌─────────────────────────────────────────────────────────────┐
│  UNTRUSTED ZONE   │  GitHub OAuth, User Browser             │
├─────────────────────────────────────────────────────────────┤
│  CONTROL PLANE    │  Envoy API GW → Go API → Temporal       │
│  (Trusted)        │  PostgreSQL, Redis, NATS, MinIO       │
├─────────────────────────────────────────────────────────────┤
│  PREDICTION SVC   │  Python gRPC (localhost only)           │
│  (Semi-Trusted)   │  ONNX Runtime, no network egress         │
├─────────────────────────────────────────────────────────────┤
│  DATA PLANE       │  containerd + gVisor (runsc)          │
│  (Untrusted User  │  Per-sandbox overlayfs + 9P mount       │
│   Code Executes)  │  Cilium eBPF network policies           │
└─────────────────────────────────────────────────────────────┘
```

## Key Invariants

1. **No host root access.** The backend never mounts `/var/run/docker.sock`.
2. **No bind mounts from host to sandbox.** Filesystem access is via 9P/virtiofs or overlayfs layers.
3. **Prediction service is localhost-only.** It has no external network access.
4. **All inter-service communication is mTLS** (SPIFFE/SPIRE in production; dev uses self-signed).
5. **Audit logs are append-only.** Stored in MinIO with object lock.

## Data Flow

1. User clicks "Create Sandbox" in Next.js frontend.
2. Frontend POSTs `/api/environments` with `gitUrl`.
3. API Gateway (Envoy) validates JWT, forwards to Go API.
4. Go API calls Prediction Service (gRPC) to classify repo.
5. Orchestrator (Temporal) schedules build workflow.
6. Worker pulls base image + dependency layer, creates overlayfs.
7. containerd + runsc starts sandbox with 9P mount for live editing.
8. File edits flow: Monaco → Yjs → WebSocket → NATS → 9P → sandbox.
9. Git operations run inside the sandbox via exec (gVisor).

## Network Segmentation

- Each sandbox gets a `/30` subnet.
- Cilium L3/L4 policies block inter-sandbox traffic.
- Only the API Gateway can reach the backend.
- Only the backend can reach the prediction service (localhost:50051).
"""

# 3. Setup Script
setup_sh = """#!/usr/bin/env bash
set -euo pipefail

RED='\\033[0;31m'
GREEN='\\033[0;32m'
YELLOW='\\033[1;33m'
NC='\\033[0m'

echo -e "${GREEN}Berth Phase 0 Setup${NC}"

# Check OS
if [[ "$OSTYPE" != "linux-gnu"* ]]; then
    echo -e "${RED}ERROR: Berth requires Linux. Detected: $OSTYPE${NC}"
    echo "If on macOS, use a Linux VM (Parallels/UTM/Proxmox)."
    exit 1
fi

# Check Go
echo -e "${YELLOW}Checking Go...${NC}"
if ! command -v go &> /dev/null; then
    echo -e "${RED}Go is not installed. Install Go 1.23 from https://go.dev/dl/${NC}"
    exit 1
fi
GO_VER=$(go version | awk '{print $3}')
echo "  Found: $GO_VER"

# Check Docker
echo -e "${YELLOW}Checking Docker...${NC}"
if ! command -v docker &> /dev/null; then
    echo -e "${RED}Docker is not installed.${NC}"
    exit 1
fi

# Install sqlc
echo -e "${YELLOW}Installing sqlc...${NC}"
if ! command -v sqlc &> /dev/null; then
    curl -L https://github.com/sqlc-dev/sqlc/releases/download/v1.27.0/sqlc_1.27.0_linux_amd64.tar.gz | sudo tar xz -C /usr/local/bin
    echo "  sqlc installed"
else
    echo "  sqlc already installed"
fi

# Install golang-migrate
echo -e "${YELLOW}Installing golang-migrate...${NC}"
if ! command -v migrate &> /dev/null; then
    curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.1/migrate.linux-amd64.tar.gz | sudo tar xz -C /usr/local/bin
    echo "  migrate installed"
else
    echo "  migrate already installed"
fi

# Install Node.js 20 (if not present)
echo -e "${YELLOW}Checking Node.js...${NC}"
if ! command -v node &> /dev/null || [[ $(node -v | cut -d'v' -f2 | cut -d'.' -f1) -lt 20 ]]; then
    echo "  Installing Node.js 20..."
    curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
    sudo apt-get install -y nodejs
fi
node -v

# Start infrastructure
echo -e "${YELLOW}Starting infrastructure (Postgres, Redis, NATS, MinIO)...${NC}"
cd infra
docker compose up -d --wait

# Verify health
echo -e "${YELLOW}Verifying services...${NC}"
sleep 3

check_port() {
    if nc -z localhost "$1" 2>/dev/null; then
        echo -e "  ${GREEN}✓${NC} $2 on port $1"
    else
        echo -e "  ${RED}✗${NC} $2 on port $1 (not responding)"
        return 1
    fi
}

check_port 5432 "PostgreSQL"
check_port 6379 "Redis"
check_port 4222 "NATS"
check_port 9000 "MinIO"

cd ../backend
echo -e "${YELLOW}Running sqlc generate...${NC}"
sqlc generate

echo -e "${YELLOW}Running migrations...${NC}"
migrate -path migrations -database "postgres://berth:berth@localhost:5432/berth?sslmode=disable" up

echo -e "${YELLOW}Building backend...${NC}"
go build -o /tmp/berth-api ./cmd/api

echo -e "${GREEN}Phase 0 setup complete!${NC}"
echo ""
echo "Next steps:"
echo "  1. cd backend && go test ./..."
echo "  2. cd frontend && npm install && npm run dev"
echo "  3. Install containerd + gVisor (Phase 1)"
"""

# 4. Docker Compose
compose_yml = """services:
  postgres:
    image: postgres:16-alpine
    container_name: berth-postgres
    environment:
      POSTGRES_USER: berth
      POSTGRES_PASSWORD: berth
      POSTGRES_DB: berth
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U berth -d berth"]
      interval: 5s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: berth-redis
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5

  nats:
    image: nats:2.10-alpine
    container_name: berth-nats
    command: "--jetstream --store_dir /data/jetstream"
    ports:
      - "4222:4222"
      - "8222:8222"
    volumes:
      - nats_data:/data/jetstream
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8222/healthz"]
      interval: 5s
      timeout: 3s
      retries: 5

  minio:
    image: minio/minio:latest
    container_name: berth-minio
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: berthminio
      MINIO_ROOT_PASSWORD: berthminio123
    ports:
      - "9000:9000"
      - "9001:9001"
    volumes:
      - minio_data:/data
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  postgres_data:
  nats_data:
  minio_data:
"""

# 5. Backend go.mod
go_mod = """module github.com/swordrookie/berth

go 1.23

require (
	github.com/gin-gonic/gin v1.10.0
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/golang-migrate/migrate/v4 v4.17.1
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.6.0
	github.com/nats-io/nats.go v1.36.0
	github.com/redis/go-redis/v9 v9.6.1
	golang.org/x/crypto v0.26.0
	google.golang.org/grpc v1.65.0
	google.golang.org/protobuf v1.34.2
)

require (
	github.com/bytedance/sonic v1.11.6 // indirect
	github.com/bytedance/sonic/loader v0.1.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudwego/base64x v0.1.4 // indirect
	github.com/cloudwego/iasm v0.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/gabriel-vasile/mimetype v1.4.3 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.20.0 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.2.7 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/nats-io/nkeys v0.4.7 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.12 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	golang.org/x/arch v0.8.0 // indirect
	golang.org/x/net v0.25.0 // indirect
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/sys v0.23.0 // indirect
	golang.org/x/text v0.17.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240814211410-ddb44dafa142 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
"""

# 6. sqlc config
sqlc_yaml = """version: "2"
sql:
  - engine: "postgresql"
    queries: "queries/"
    schema: "migrations/"
    gen:
      go:
        package: "repository"
        out: "internal/repository"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_prepared_queries: false
        emit_interface: true
        emit_exact_table_names: false
        emit_empty_slices: true
        overrides:
          - db_type: "uuid"
            go_type:
              import: "github.com/google/uuid"
              type: "UUID"
          - db_type: "timestamptz"
            go_type: "time.Time"
"""

# 7. Main entrypoint
main_go = """package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/swordrookie/berth/internal/config"
	"github.com/swordrookie/berth/internal/infrastructure/db"
	"github.com/swordrookie/berth/internal/infrastructure/redis"
	"github.com/swordrookie/berth/internal/delivery/http/handler"
	"github.com/swordrookie/berth/internal/delivery/http/middleware"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Initialize infrastructure
	if err := db.Init(cfg.DatabaseURL); err != nil {
		slog.Error("failed to init database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := redis.Init(cfg.RedisURL); err != nil {
		slog.Error("failed to init redis", "error", err)
		os.Exit(1)
	}
	defer redis.Close()

	// Setup router
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())

	// Health check (no auth)
	r.GET("/health", handler.HealthCheck)

	// API routes
	api := r.Group("/api")
	api.Use(middleware.RateLimit())
	{
		api.POST("/auth/github", handler.GithubLogin)
		api.GET("/auth/github/callback", handler.GithubCallback)

		authenticated := api.Group("")
		authenticated.Use(middleware.Auth(cfg.JWTSecret))
		{
			authenticated.GET("/user/me", handler.GetMe)
			authenticated.GET("/environments", handler.ListEnvironments)
			authenticated.POST("/environments", handler.CreateEnvironment)
		}
	}

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("server started", "port", cfg.Port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	slog.Info("server exited")
}
"""

# 8. Domain entity
sandbox_go = """package domain

import (
	"time"

	"github.com/google/uuid"
)

// RuntimeProfile represents the predicted runtime configuration for a sandbox.
type RuntimeProfile struct {
	Language    string // "node", "python", "go", "rust", "other"
	BaseImage   string // e.g., "node:20-alpine"
	InstallCmd  string // e.g., "npm install"
	StartCmd    string // e.g., "npm run dev"
	WorkDir     string // e.g., "/app"
	ExposedPort int    // e.g., 3000
	NeedsDB     bool   // whether a sidecar DB is predicted
	Confidence  float64
}

// SandboxState represents the lifecycle state of a sandbox.
type SandboxState string

const (
	StateIdle     SandboxState = "IDLE"
	StateBuilding SandboxState = "BUILDING"
	StateRunning  SandboxState = "RUNNING"
	StateStopped  SandboxState = "STOPPED"
	StateFailed   SandboxState = "FAILED"
)

// Sandbox is the core aggregate root for an ephemeral dev environment.
type Sandbox struct {
	ID          uuid.UUID
	ProjectID   uuid.UUID
	OwnerID     uuid.UUID
	Name        string
	GitURL      string
	GitBranch   string
	State       SandboxState
	Profile     *RuntimeProfile
	ContainerID *string // containerd container ID
	PublicURL   *string
	Port        *int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ExpiresAt   *time.Time
}

// IsActive returns true if the sandbox is running or building.
func (s *Sandbox) IsActive() bool {
	return s.State == StateBuilding || s.State == StateRunning
}

// CanEdit returns true if the sandbox is in an editable state.
func (s *Sandbox) CanEdit() bool {
	return s.State == StateRunning || s.State == StateIdle
}

// WarmPoolEntry represents a pre-warmed sandbox waiting for assignment.
type WarmPoolEntry struct {
	ID          uuid.UUID
	ProfileHash string   // hash of RuntimeProfile for matching
	ContainerID string   // containerd container ID
	CreatedAt   time.Time
	LastUsedAt  *time.Time
}

// SandboxRepository defines the interface for sandbox persistence.
// This is implemented by the PostgreSQL repository (sqlc-generated + wrapper).
type SandboxRepository interface {
	Create(ctx context.Context, s *Sandbox) error
	GetByID(ctx context.Context, id uuid.UUID) (*Sandbox, error)
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]*Sandbox, error)
	UpdateState(ctx context.Context, id uuid.UUID, state SandboxState) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// PredictionService defines the interface for the ML prediction microservice.
// Implemented by the Python gRPC service client.
type PredictionService interface {
	Predict(ctx context.Context, gitURL string, branch string) (*RuntimeProfile, error)
}

// ContainerRuntime defines the interface for containerd/gVisor operations.
// This abstracts containerd so we can mock it in tests.
type ContainerRuntime interface {
	CreateSandbox(ctx context.Context, spec SandboxSpec) (string, error) // returns containerID
	StartSandbox(ctx context.Context, containerID string) error
	StopSandbox(ctx context.Context, containerID string) error
	DeleteSandbox(ctx context.Context, containerID string) error
	Exec(ctx context.Context, containerID string, cmd []string) (string, error)
	GetLogs(ctx context.Context, containerID string, tail int) (string, error)
}

// SandboxSpec is the specification passed to the container runtime.
type SandboxSpec struct {
	ID          uuid.UUID
	BaseImage   string
	WorkDir     string
	Env         map[string]string
	MemoryLimit int64 // bytes
	CPULimit    int64 // milli-cores
	DiskLimit   int64 // bytes
	NetworkID   string
}
"""

# 9. Migration up
migration_up = """CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email TEXT UNIQUE NOT NULL,
    username TEXT UNIQUE,
    github_id TEXT UNIQUE,
    github_username TEXT,
    github_token_encrypted TEXT,
    avatar_url TEXT,
    max_sandboxes INT DEFAULT 5,
    max_builds_per_hour INT DEFAULT 10,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS organizations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS organization_members (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'MEMBER',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(organization_id, user_id)
);

CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    description TEXT,
    owner_organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    is_public BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS project_collaborators (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'COLLABORATOR',
    invited_by_user_id UUID REFERENCES users(id),
    invited_at TIMESTAMPTZ DEFAULT NOW(),
    accepted_at TIMESTAMPTZ,
    UNIQUE(project_id, user_id)
);

CREATE TABLE IF NOT EXISTS sandboxes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID REFERENCES projects(id) ON DELETE SET NULL,
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    git_url TEXT NOT NULL,
    git_branch TEXT NOT NULL DEFAULT 'main',
    state TEXT NOT NULL DEFAULT 'IDLE',
    runtime_language TEXT,
    runtime_base_image TEXT,
    runtime_port INT,
    needs_db BOOLEAN DEFAULT FALSE,
    container_id TEXT,
    public_url TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS sandbox_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    sandbox_id UUID NOT NULL REFERENCES sandboxes(id) ON DELETE CASCADE,
    message TEXT NOT NULL,
    level TEXT NOT NULL DEFAULT 'info',
    timestamp TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id UUID,
    ip_address INET,
    timestamp TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_sandboxes_project ON sandboxes(project_id);
CREATE INDEX idx_sandboxes_owner ON sandboxes(owner_id);
CREATE INDEX idx_sandboxes_state ON sandboxes(state);
CREATE INDEX idx_sandbox_logs_sandbox ON sandbox_logs(sandbox_id);
CREATE INDEX idx_audit_logs_user ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_timestamp ON audit_logs(timestamp);
"""

# 10. Migration down
migration_down = """DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS sandbox_logs;
DROP TABLE IF EXISTS sandboxes;
DROP TABLE IF EXISTS project_collaborators;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS organization_members;
DROP TABLE IF EXISTS organizations;
DROP TABLE IF EXISTS users;
"""

# 11. sqlc queries
queries_sql = """-- name: CreateSandbox :one
INSERT INTO sandboxes (
    project_id, owner_id, name, git_url, git_branch, state,
    runtime_language, runtime_base_image, runtime_port, needs_db
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING *;

-- name: GetSandboxByID :one
SELECT * FROM sandboxes WHERE id = $1;

-- name: ListSandboxesByProject :many
SELECT * FROM sandboxes WHERE project_id = $1 ORDER BY created_at DESC;

-- name: UpdateSandboxState :exec
UPDATE sandboxes SET state = $2, updated_at = NOW() WHERE id = $1;

-- name: UpdateSandboxContainer :exec
UPDATE sandboxes SET container_id = $2, public_url = $3, updated_at = NOW() WHERE id = $1;

-- name: DeleteSandbox :exec
DELETE FROM sandboxes WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (email, username, github_id, github_username, avatar_url)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByGithubID :one
SELECT * FROM users WHERE github_id = $1;
"""

# 12. Config package stub
config_go = """package config

import (
	"fmt"
	"os"
)

// Config holds all application configuration.
type Config struct {
	Env          string
	Port         string
	DatabaseURL  string
	RedisURL     string
	JWTSecret    string
	GithubClientID     string
	GithubClientSecret string
	PredictionAddr     string // gRPC address for Python prediction service
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://berth:berth@localhost:5432/berth?sslmode=disable"
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return &Config{
		Env:                getEnv("ENV", "development"),
		Port:               port,
		DatabaseURL:        dbURL,
		RedisURL:           redisURL,
		JWTSecret:          jwtSecret,
		GithubClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		GithubClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		PredictionAddr:     getEnv("PREDICTION_ADDR", "localhost:50051"),
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
"""

# 13. DB infrastructure stub
db_go = """package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var pool *pgxpool.Pool

// Init initializes the PostgreSQL connection pool.
func Init(dsn string) error {
	var err error
	for i := 1; i <= 10; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		pool, err = pgxpool.New(ctx, dsn)
		cancel()
		if err == nil {
			break
		}
		slog.Warn("database not ready", "attempt", i, "error", err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("failed to connect to database after 10 attempts: %w", err)
	}

	pool.Config().MaxConns = 100
	pool.Config().MinConns = 25
	pool.Config().MaxConnLifetime = 5 * time.Minute

	slog.Info("database connection established")
	return nil
}

// Pool returns the connection pool.
func Pool() *pgxpool.Pool {
	return pool
}

// Close closes the connection pool.
func Close() {
	if pool != nil {
		pool.Close()
	}
}
"""

# 14. Redis infrastructure stub
redis_go = """package redis

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

var client *redis.Client

// Init initializes the Redis client.
func Init(url string) error {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return fmt.Errorf("failed to parse redis url: %w", err)
	}
	client = redis.NewClient(opt)

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to ping redis: %w", err)
	}

	slog.Info("redis connection established")
	return nil
}

// Client returns the Redis client.
func Client() *redis.Client {
	return client
}

// Close closes the Redis client.
func Close() {
	if client != nil {
		client.Close()
	}
}
"""

# 15. Middleware stubs
middleware_logger_go = """package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"log/slog"
)

func Logger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		slog.Info("request",
			"method", param.Method,
			"path", param.Path,
			"status", param.StatusCode,
			"latency", param.Latency,
			"client_ip", param.ClientIP,
		)
		return ""
	})
}
"""

middleware_cors_go = """package middleware

import "github.com/gin-gonic/gin"

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
"""

middleware_auth_go = """package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func Auth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
			return
		}

		token, err := jwt.Parse(parts[1], func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			c.Set("userId", claims["userId"])
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			return
		}

		c.Next()
	}
}
"""

middleware_ratelimit_go = """package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/swordrookie/berth/internal/infrastructure/redis"
	"log/slog"
)

func RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		client := redis.Client()
		if client == nil {
			c.Next()
			return
		}

		key := fmt.Sprintf("rate_limit:%s:%s", c.ClientIP(), c.Request.URL.Path)
		ctx := context.Background()

		count, err := client.Incr(ctx, key).Result()
		if err != nil {
			slog.Error("redis rate limit error", "error", err)
			c.Next()
			return
		}

		if count == 1 {
			client.Expire(ctx, key, time.Minute)
		}

		if count > 200 {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}

		c.Next()
	}
}
"""

# 16. Handler stubs
handler_health_go = """package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/swordrookie/berth/internal/infrastructure/db"
	"github.com/swordrookie/berth/internal/infrastructure/redis"
)

func HealthCheck(c *gin.Context) {
	ctx := c.Request.Context()

	dbOK := true
	if err := db.Pool().Ping(ctx); err != nil {
		dbOK = false
	}

	redisOK := true
	if redis.Client() != nil {
		if err := redis.Client().Ping(ctx).Err(); err != nil {
			redisOK = false
		}
	}

	status := http.StatusOK
	if !dbOK || !redisOK {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, gin.H{
		"status": "ok",
		"db":     dbOK,
		"redis":  redisOK,
	})
}
"""

handler_auth_go = """package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func GithubLogin(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "github login not yet implemented"})
}

func GithubCallback(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "github callback not yet implemented"})
}
"""

handler_user_go = """package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetMe(c *gin.Context) {
	userID, _ := c.Get("userId")
	c.JSON(http.StatusOK, gin.H{
		"id": userID,
		"message": "user profile endpoint stub",
	})
}
"""

handler_sandbox_go = """package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func ListEnvironments(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"sandboxes": []any{}})
}

func CreateEnvironment(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "create environment not yet implemented"})
}
"""

# 17. Makefile
makefile = """.PHONY: all build test lint migrate sqlc clean dev

all: sqlc build

build:
	cd backend && go build -o bin/berth-api ./cmd/api

test:
	cd backend && go test -race -v ./...

lint:
	cd backend && golangci-lint run ./...

migrate-up:
	cd backend && migrate -path migrations -database "postgres://berth:berth@localhost:5432/berth?sslmode=disable" up

migrate-down:
	cd backend && migrate -path migrations -database "postgres://berth:berth@localhost:5432/berth?sslmode=disable" down

sqlc:
	cd backend && sqlc generate

dev:
	cd infra && docker compose up -d
	@echo "Infrastructure started. Run 'make migrate-up' then 'cd backend && go run ./cmd/api'"

clean:
	cd infra && docker compose down -v
	rm -rf backend/bin/
"""

# 18. README
readme = """# Berth

> Predictive ephemeral sandbox platform with gVisor isolation.

## Quick Start

```bash
# 1. Setup (installs tools, starts infra)
bash scripts/setup.sh

# 2. Start infrastructure
make dev

# 3. Run migrations
make migrate-up

# 4. Start backend
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
"""

# Write all files
files = {
    ".github/workflows/ci.yml": ci_yml,
    "docs/ARCHITECTURE.md": arch_md,
    "scripts/setup.sh": setup_sh,
    "infra/docker-compose.yml": compose_yml,
    "backend/go.mod": go_mod,
    "backend/sqlc.yaml": sqlc_yaml,
    "backend/cmd/api/main.go": main_go,
    "backend/internal/domain/sandbox.go": sandbox_go,
    "backend/internal/config/config.go": config_go,
    "backend/internal/infrastructure/db/db.go": db_go,
    "backend/internal/infrastructure/redis/redis.go": redis_go,
    "backend/internal/delivery/http/middleware/logger.go": middleware_logger_go,
    "backend/internal/delivery/http/middleware/cors.go": middleware_cors_go,
    "backend/internal/delivery/http/middleware/auth.go": middleware_auth_go,
    "backend/internal/delivery/http/middleware/ratelimit.go": middleware_ratelimit_go,
    "backend/internal/delivery/http/handler/health.go": handler_health_go,
    "backend/internal/delivery/http/handler/auth.go": handler_auth_go,
    "backend/internal/delivery/http/handler/user.go": handler_user_go,
    "backend/internal/delivery/http/handler/sandbox.go": handler_sandbox_go,
    "backend/migrations/000001_init_schema.up.sql": migration_up,
    "backend/migrations/000001_init_schema.down.sql": migration_down,
    "backend/queries/sandbox.sql": queries_sql,
    "Makefile": makefile,
    "README.md": readme,
}

for path, content in files.items():
    full_path = os.path.join(base, path)
    os.makedirs(os.path.dirname(full_path), exist_ok=True)
    with open(full_path, "w") as f:
        f.write(content)

# Make setup.sh executable
os.chmod(os.path.join(base, "scripts/setup.sh"), 0o755)

print("Phase 0 scaffold generated.")
print(f"Total files: {len(files)}")
print(f"Output directory: {base}")
