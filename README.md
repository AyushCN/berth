# Berth

**Ephemeral development sandboxes with containerd, repository-based runtime detection, dependency caching, and ML-powered build predictions.**

Berth is a research prototype for ephemeral development environments. It provisions sandboxed containers on-demand from GitHub repositories, automatically detects the runtime/framework, caches dependencies, and uses ML to predict build times, cache hits, and failure risks.

---

## 🏗️ Architecture Overview

```
┌─────────────┐     ┌──────────────┐     ┌──────────────┐
│  Browser    │────▶│  Next.js     │────▶│  Go API      │
│  (Frontend) │     │  (UI)        │     │  (Control)   │
└─────────────┘     └──────────────┘     └──────┬───────┘
                                                 │
                    ┌────────────────────────────┼────────────────────────────┐
                    ▼                            ▼                            ▼
           ┌──────────────────┐          ┌──────────────────┐          ┌──────────────────┐
           │  PostgreSQL      │          │  NATS          │          │  containerd      │
           │  (Persistence)   │          │  (Message Bus) │          │  (Container      │
           │                  │          │                │          │   Runtime)       │
           └──────────────────┘          └──────────────────┘          └────────┬─────────┘
                                                                               │
                                                                        ┌────────▼─────────┐
                                                                        │  Sandbox         │
                                                                        │  (Container)     │
                                                                        └──────────────────┘
```

---

## ✨ Key Features

| Feature | Description |
|---------|-------------|
| **Runtime Detection** | Auto-detects Node.js, Python, Go, Rust, Java with 30+ framework support (Next.js, Django, Gin, Axum, Spring, etc.) |
| **Architecture Classification** | Detects WEB_APP, API, CLI, FULL_STACK, MONOREPO, LIBRARY |
| **Build Strategy Pattern** | Pluggable strategies per language (Python, Node, Go, Rust, Java, Docker Compose, Fallback) |
| **Dependency Caching** | pnpm/npm/yarn/bun, Cargo, Go modules, pip/poetry, Maven/Gradle |
| **Docker Compose Support** | Parses and builds multi-service compose files |
| **ML Predictions** | Build time, image size, cache hit probability, failure risk |
| **ONNX Export** | Export trained models to ONNX for production inference |
| **Git Integration** | OAuth-backed Git operations, branch management, diff, push |
| **Collaborative IDE** | File editor, terminal, Git panel, preview proxy |
| **Warm Pool** | Pre-warmed containers for instant startup |

---

## 📁 Project Structure

```
berth/
├── backend/                    # Go API, Worker, Prediction Engine
│   ├── cmd/
│   │   ├── api/               # API server entry point
│   │   └── worker/            # Worker entry point
│   ├── internal/
│   │   ├── analyzer/          # Runtime & framework detection
│   │   ├── usecase/           # Business logic (Build Planner, ML)
│   │   ├── delivery/          # HTTP/gRPC handlers
│   │   ├── infrastructure/    # containerd, Docker, NATS, Redis
│   │   ├── domain/            # Core domain models
│   │   ├── repository/        # PostgreSQL repositories
│   │   ├── worker/            # Sandbox worker implementation
│   │   └── integration/       # Integration tests
│   ├── migrations/            # SQL migrations
│   ├── proto/                 # gRPC protobuf definitions
│   └── go.mod
├── frontend/                   # Next.js 15 + React 18
│   ├── app/                   # App Router pages
│   ├── components/            # React components (editor, terminal, git, file-tree)
│   ├── stores/                # Zustand state management
│   └── lib/                   # API client
├── docs/                       # Documentation
├── infra/                      # Local dev infrastructure (docker-compose)
├── scripts/                    # Development scripts
└── Makefile
```

---

## 🚀 Quick Start

### Prerequisites
- Linux (bare metal or VM with nested virtualization)
- Go 1.21+
- Node.js 20+
- Docker + Docker Compose
- containerd (rootless)
- PostgreSQL 16+
- NATS
- Redis

### Development Setup

```bash
# Start infrastructure
make up

# Run migrations
make migrate-up

# Start API server
cd backend && go run ./cmd/api

# In another terminal, start worker
MODE=worker go run ./cmd/worker

# Start frontend
cd frontend && npm run dev
```

### Environment Variables

```bash
# Backend
DATABASE_URL=postgres://berth:berth@localhost:5432/berth?sslmode=disable
REDIS_URL=redis://localhost:6379
NATS_URL=nats://localhost:4222
JWT_SECRET=your-32-byte-secret-key-here!!
GITHUB_CLIENT_ID=your-github-oauth-client-id
GITHUB_CLIENT_SECRET=your-github-oauth-secret
FRONTEND_URL=http://localhost:3000
ENCRYPTION_KEY=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
MODE=api
PORT=8080
DOCKER_HOST=unix:///run/user/1000/docker.sock
TRAEFIK_DOMAIN=localhost
MODEL_DIR=/tmp/berth/models
```

---

## 📚 Documentation

| Document | Description |
|----------|-------------|
| [Architecture](docs/ARCHITECTURE.md) | System architecture, trust boundaries, data flow |
| [Status](docs/STATUS.md) | Current implementation status, limitations |
| [API Reference](docs/API.md) | REST/gRPC endpoints, schemas |
| [Frontend](docs/FRONTEND.md) | Next.js app structure, components, state |
| [Backend](docs/BACKEND.md) | Go services, domain models, use cases |
| [Docker](docs/DOCKER.md) | Container setup, containerd, Docker |
| [Deployment](docs/DEPLOYMENT.md) | Production deployment guide |
| [Development](docs/DEVELOPMENT.md) | Local development workflow |
| [Testing](docs/TESTING.md) | Test strategy, running tests |
| [Security](docs/SECURITY.md) | Security model, threat model |
| [Contributing](docs/CONTRIBUTING.md) | Contribution guidelines |
| [Changelog](docs/CHANGELOG.md) | Version history |

---

## 🧪 Testing

```bash
# Run all backend tests
make test

# Run specific package tests
cd backend && ENCRYPTION_KEY=... go test ./internal/analyzer/... -v
cd backend && ENCRYPTION_KEY=... go test ./internal/usecase/... -v
cd backend && ENCRYPTION_KEY=... go test ./internal/integration/... -v

# Run frontend tests
cd frontend && npm test

# Run linter
make lint
```

---

## 🔒 Security Notice

**This is a research prototype, not production-ready.**

- Single-host only, no multi-node support
- Rootless containerd with runc.v2 (not gVisor)
- Host networking - no network isolation between sandboxes
- No mTLS/SPIFFE, no Cilium policies
- Do not expose to untrusted users or public internet

See [SECURITY.md](docs/SECURITY.md) for full threat model.

---

## 📄 License

MIT License - Research Prototype

---

## 🤝 Contributing

See [CONTRIBUTING.md](docs/CONTRIBUTING.md) for guidelines.

---

## 🔗 Links

- [Project Specification](BERTH_FULL_SPEC.md)
- [Architecture Diagram](docs/ARCHITECTURE.md)
- [Current Status](docs/STATUS.md)
- [API Documentation](docs/API.md)