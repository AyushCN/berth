# Docker & Containerd Setup

## Overview

Berth uses **containerd** as the container runtime with **rootless** configuration for development. Docker is used for image building and local development infrastructure.

---

## Containerd Configuration

### Rootless Setup (Development)

```bash
# Install containerd
sudo apt-get update && sudo apt-get install -y containerd

# Enable rootless
mkdir -p ~/.config/containerd
containerd config default | sed 's/root = "\/var\/lib\/containerd"/root = "\/home\/$USER\/.local\/share\/containerd"/' > ~/.config/containerd/config.toml

# Start containerd
systemctl --user enable --now containerd
```

### Containerd Config (`~/.config/containerd/config.toml`)
```toml
version = 2
root = "/home/user/.local/share/containerd"
state = "/run/user/1000/containerd"

[grpc]
  address = "/run/user/1000/containerd/containerd.sock"

[metrics]
  address = "127.0.0.1:1338"

[plugins."io.containerd.grpc.v1.cri"]
  sandbox_image = "registry.k8s.io/pause:3.9"
  enable_unprivileged_ports = true
  enable_unprivileged_icmp = true

[plugins."io.containerd.grpc.v1.cri".containerd]
  snapshotter = "native"
  default_runtime_name = "runc"
  
[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
  runtime_type = "io.containerd.runc.v2"
  [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
    SystemdCgroup = false
    BinaryName = "runc"
```

### Verify Installation
```bash
# Check containerd
ctr --address /run/user/1000/containerd/containerd.sock version

# Check runc
runc --version

# Test container
ctr run --rm docker.io/library/alpine:latest test echo hello
```

---

## Docker Infrastructure

### Development Docker Compose (`docker-compose.dev.yml`)
```yaml
version: '3.8'

services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: berth
      POSTGRES_USER: berth
      POSTGRES_PASSWORD: berth
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U berth"]
      interval: 5s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5

  nats:
    image: nats:2.10-alpine
    ports:
      - "4222:4222"
      - "8222:8222"
    command: ["-js", "-m", "8222"]
    healthcheck:
      test: ["CMD", "nats", "server", "check"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  postgres_data:
  redis_data:
```

### Start Infrastructure
```bash
# Start all services
docker compose -f docker-compose.dev.yml up -d

# Check status
docker compose -f docker-compose.dev.yml ps

# View logs
docker compose -f docker-compose.dev.yml logs -f

# Stop
docker compose -f docker-compose.dev.yml down

# Stop with volumes (clean slate)
docker compose -f docker-compose.dev.yml down -v
```

---

## Containerd Integration in Berth

### Runtime Interface
```go
// internal/domain/sandbox.go
type ContainerRuntime interface {
    CreateSandbox(ctx context.Context, spec SandboxSpec) (string, error)
    StartSandbox(ctx context.Context, containerID string) error
    StopSandbox(ctx context.Context, containerID string) error
    DeleteSandbox(ctx context.Context, containerID string) error
    Exec(ctx context.Context, containerID string, cmd []string) (string, error)
    ExecPTY(ctx context.Context, containerID string, cmd []string) (io.WriteCloser, io.Reader, func() error, error)
}
```

### Implementation (`internal/infrastructure/containerd/`)
```go
// runtime.go - Main client
func NewDockerRuntime(dockerHost, network, traefikDomain string) (domain.ContainerRuntime, error)

// layer.go - Layer operations
func (r *DockerRuntime) CommitLayer(ctx context.Context, containerID, imageName string) error
func (r *DockerRuntime) ExportLayer(ctx context.Context, imageName string) (io.ReadCloser, error)

// network.go - Network setup
func (r *DockerRuntime) SetupNetwork(ctx context.Context) error
```

### Key Features
| Feature | Implementation |
|---------|----------------|
| **Layer Commit** | `containerd` diff + commit to new image |
| **Tar Export** | `ctr image export` for image portability |
| **Network** | `go-iptables` for port mapping |
| **Warm Pool** | Exact-image reuse via image digest |
| **Rootless** | User namespace mapping |

---

## Image Building

### Multi-stage Dockerfiles

#### API Server
```dockerfile
# Dockerfile.api
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /berth-api ./cmd/api

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /berth-api /berth-api
EXPOSE 8080
ENTRYPOINT ["/berth-api"]
```

#### Worker
```dockerfile
# Dockerfile.worker
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /berth-worker ./cmd/worker

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata git
COPY --from=builder /berth-worker /berth-worker
ENTRYPOINT ["/berth-worker"]
```

### Build Commands
```bash
# Build API
docker build -f Dockerfile.api -t berth/api:latest .

# Build Worker
docker build -f Dockerfile.worker -t berth/worker:latest .

# Build all
make build
```

---

## Development Workflow

### Local Development with Docker

```bash
# Start infrastructure only
docker compose -f docker-compose.dev.yml up -d

# Run API locally (hot reload with air)
cd backend && air

# Run Worker locally
MODE=worker go run ./cmd/worker

# Frontend
cd frontend && npm run dev
```

### Full Docker Stack (Production-like)

```yaml
# docker-compose.prod.yml
version: '3.8'

services:
  api:
    build:
      context: ./backend
      dockerfile: Dockerfile.api
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgres://berth:berth@postgres:5432/berth
      - REDIS_URL=redis://redis:6379
      - NATS_URL=nats://nats:4222
      - JWT_SECRET=${JWT_SECRET}
      - GITHUB_CLIENT_ID=${GITHUB_CLIENT_ID}
      - GITHUB_CLIENT_SECRET=${GITHUB_CLIENT_SECRET}
      - FRONTEND_URL=https://app.example.com
      - ENCRYPTION_KEY=${ENCRYPTION_KEY}
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
      nats:
        condition: service_healthy

  worker:
    build:
      context: ./backend
      dockerfile: Dockerfile.worker
    environment:
      - DATABASE_URL=postgres://berth:berth@postgres:5432/berth
      - REDIS_URL=redis://redis:6379
      - NATS_URL=nats://nats:4222
      - ENCRYPTION_KEY=${ENCRYPTION_KEY}
      - MODE=worker
    depends_on:
      - api

  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
    ports:
      - "3000:3000"
    environment:
      - NEXT_PUBLIC_API_URL=https://api.example.com

  postgres:
    image: postgres:16-alpine
    # ... (same as dev)

  redis:
    image: redis:7-alpine
    # ...

  nats:
    image: nats:2.10-alpine
    # ...
```

---

## Containerd Operations

### Common Commands
```bash
# List containers
ctr -n k8s.io containers list

# List images
ctr images list

# Pull image
ctr images pull docker.io/library/node:20-alpine

# Create container
ctr run --rm -t docker.io/library/alpine:latest test sh

# Exec into container
ctr task exec --exec-id exec-1 -t <container-id> sh

# Checkpoint/Restore (experimental)
ctr checkpoint <container-id>
ctr restore <checkpoint>
```

### Logs
```bash
# Container logs
ctr -n k8s.io tasks ls
ctr -n k8s.io task logs <task-id>

# System logs
journalctl -u containerd -f
```

---

## Networking

### Port Mapping (Development)
| Service | Host Port | Container Port |
|---------|-----------|----------------|
| PostgreSQL | 5432 | 5432 |
| Redis | 6379 | 6379 |
| NATS | 4222, 8222 | 4222, 8222 |
| API | 8080 | 8080 |
| Frontend | 3000 | 3000 |

### Production Networking
- **Traefik** - Reverse proxy with Let's Encrypt
- **CNI** - Cilium/Calico for pod networking
- **Service Mesh** - Istio/Linkerd for mTLS

---

## Troubleshooting

| Issue | Solution |
|-------|----------|
| `permission denied` on socket | `sudo chown $USER:$USER /run/user/1000/containerd/containerd.sock` |
| `image not found` | `ctr images pull <image>` |
| `no space left` | `ctr images prune`, `docker system prune` |
| `runc not found` | `apt-get install runc` |
| `iptables` permission | `sudo usermod -aG docker $USER` |

### Cleanup
```bash
# Remove all containers
ctr containers delete $(ctr containers list -q)

# Remove all images
ctr images remove $(ctr images list -q)

# Prune containerd
ctr content prune

# Full Docker cleanup
docker system prune -a --volumes
```