# Docker Setup

## Overview

Berth uses **Docker** (not containerd) as the container runtime. The worker manages container lifecycle via the host Docker socket.

---

## Development Setup

### Prerequisites
- Docker 24+ with Compose v2
- Linux (or macOS with Docker Desktop)

No local containerd/k8s installation needed — `docker-compose.dev.yml` provides:
- PostgreSQL 16
- Redis 7
- NATS 2.10 with JetStream
- Traefik v2.11

### Start Development Stack
```bash
docker compose -f docker-compose.dev.yml up -d --build

# Verify
docker compose -f docker-compose.dev.yml ps
# Frontend: http://localhost:3000
# API: http://api.localhost
# Traefik: http://localhost:8080
```

### Manual Run (without compose)
```bash
# Terminal 1: API
cd backend
export DATABASE_URL="postgres://berth:berth@localhost:5432/berth?sslmode=disable"
export REDIS_URL=redis://localhost:6379
export NATS_URL=nats://localhost:4222
export ENCRYPTION_KEY=0d71f78929e8b688442387dd10478006998c1fa490c42c02c627a3e5ec8a3bed
export JWT_SECRET=dev_secret_change_in_production_at_least_32_chars_long
export GITHUB_CLIENT_ID=dev_client_id
export GITHUB_CLIENT_SECRET=dev_client_secret
export FRONTEND_URL=http://localhost:3000
export WORKSPACE_ROOT=/tmp/berth-workspaces
export DOCKER_HOST=unix:///var/run/docker.sock
export DOCKER_NETWORK=berth
export TRAEFIK_DOMAIN=localhost
export ENV=development
export MODE=api
export PORT=8080
go run ./cmd/api

# Terminal 2: Worker
export MODE=worker
go run ./cmd/worker

# Terminal 3: Frontend
cd frontend && npm install && npm run dev
```

---

## Docker Compose Files

### Development (`docker-compose.dev.yml`)
```yaml
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: berth
      POSTGRES_PASSWORD: berth
      POSTGRES_DB: berth
    volumes:
      - berth_pg:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U berth -d berth"]
      interval: 5s
      timeout: 5s
      retries: 5
    networks: [berth]

  redis:
    image: redis:7-alpine
    volumes: [berth_redis:/data]
    healthcheck: {test: ["CMD", "redis-cli", "ping"], interval: 5s, timeout: 3s, retries: 5}
    networks: [berth]

  nats:
    image: nats:2.10-alpine
    command: ["--jetstream", "--store_dir", "/data/jetstream", "--http_port", "8222"]
    volumes: [berth_nats:/data/jetstream]
    healthcheck: {test: ["CMD", "wget", "-qO-", "http://localhost:8222/healthz"], interval: 5s, timeout: 3s, retries: 5}
    networks: [berth]

  traefik:
    image: traefik:v2.11
    command:
      - --api.insecure=true
      - --providers.docker=true
      - --providers.docker.exposedbydefault=false
      - --entrypoints.web.address=:80
    ports: ["80:80", "8080:8080"]
    volumes: [/var/run/docker.sock:/var/run/docker.sock:ro]
    networks: [berth]
    labels: ["traefik.enable=true", "traefik.http.routers.api.rule=Host(`api.localhost`)", "traefik.http.services.api.loadbalancer.server.port=8080"]

  api:
    build: {context: ./backend, dockerfile: Dockerfile.api}
    environment:
      MODE: api
      PORT: "8080"
      DATABASE_URL: postgres://berth:berth@postgres:5432/berth?sslmode=disable
      REDIS_URL: redis://redis:6379
      NATS_URL: nats://nats:4222
      ENCRYPTION_KEY: ${ENCRYPTION_KEY}
      JWT_SECRET: ${JWT_SECRET}
      GITHUB_CLIENT_ID: ${GITHUB_CLIENT_ID}
      GITHUB_CLIENT_SECRET: ${GITHUB_CLIENT_SECRET}
      FRONTEND_URL: http://localhost:3000
      WORKSPACE_ROOT: /workspaces
      MODEL_DIR: /models
      DOCKER_HOST: unix:///var/run/docker.sock
      DOCKER_NETWORK: berth
      TRAEFIK_DOMAIN: localhost
      ENV: development
    volumes: [/var/run/docker.sock:/var/run/docker.sock, ${BERTH_WORKSPACES:-./data/workspaces}:/workspaces, berth_models:/models]
    networks: [berth]
    depends_on: {postgres: {condition: service_healthy}, redis: {condition: service_healthy}, nats: {condition: service_healthy}}
    labels: ["traefik.enable=true", "traefik.http.routers.api.rule=Host(`api.localhost`)", "traefik.http.services.api.loadbalancer.server.port=8080"]

  worker:
    build: {context: ./backend, dockerfile: Dockerfile.worker}
    environment:
      MODE: worker
      DATABASE_URL: postgres://berth:berth@postgres:5432/berth?sslmode=disable
      REDIS_URL: redis://redis:6379
      NATS_URL: nats://nats:4222
      ENCRYPTION_KEY: ${ENCRYPTION_KEY}
      GITHUB_CLIENT_ID: ${GITHUB_CLIENT_ID}
      GITHUB_CLIENT_SECRET: ${GITHUB_CLIENT_SECRET}
      FRONTEND_URL: http://localhost:3000
      WORKSPACE_ROOT: /workspaces
      MODEL_DIR: /models
      DOCKER_HOST: unix:///var/run/docker.sock
      DOCKER_NETWORK: berth
      TRAEFIK_DOMAIN: localhost
      ENV: development
    volumes: [/var/run/docker.sock:/var/run/docker.sock, ${BERTH_WORKSPACES:-./data/workspaces}:/workspaces, berth_models:/models]
    networks: [berth]
    depends_on: {postgres: {condition: service_healthy}, redis: {condition: service_healthy}, nats: {condition: service_healthy}}

  frontend:
    build: {context: ./frontend, dockerfile: Dockerfile}
    environment: {NEXT_PUBLIC_API_URL: http://api.localhost}
    ports: ["3000:3000"]
    networks: [berth]
    depends_on: [api]

volumes: {berth_pg:, berth_redis:, berth_nats:, berth_models:}
networks: {berth: {driver: bridge}}
```

---

## Dockerfiles

### API (`backend/Dockerfile.api`)
```dockerfile
# Build stage
FROM golang:1.26-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o berth-api ./cmd/api

# Runtime stage
FROM alpine:3.20
RUN apk add --no-cache ca-certificates git curl bash
WORKDIR /app
COPY --from=builder /app/berth-api /usr/local/bin/
RUN mkdir -p /app/workspaces
EXPOSE 8080
ENTRYPOINT ["berth-api"]
```

### Worker (`backend/Dockerfile.worker`)
```dockerfile
# Build stage
FROM golang:1.26-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o berth-worker ./cmd/worker

# Runtime stage
FROM alpine:3.20
RUN apk add --no-cache ca-certificates git curl bash docker-cli
WORKDIR /app
COPY --from=builder /app/berth-worker /usr/local/bin/
RUN mkdir -p /app/workspaces
ENTRYPOINT ["berth-worker"]
```

### Frontend (`frontend/Dockerfile`)
```dockerfile
# Build stage
FROM node:20-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
ARG NEXT_PUBLIC_API_URL=http://api.localhost
ARG NEXT_PUBLIC_WS_URL=ws://api.localhost
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL
ENV NEXT_PUBLIC_WS_URL=$NEXT_PUBLIC_WS_URL
RUN npm run build

# Runtime stage
FROM node:20-alpine
WORKDIR /app
ENV NODE_ENV=production
COPY --from=builder /app/public ./public
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
EXPOSE 3000
ENV PORT=3000
ENV HOSTNAME=0.0.0.0
CMD ["node", "server.js"]
```

---

## Key Points

| Aspect | Detail |
|--------|--------|
| **Runtime** | Docker (not containerd) |
| **Networking** | Host networking (`--network host`) |
| **Security** | `--read-only`, `--cap-drop ALL`, `--init`, `--user 1000:1000`, `no-new-privileges` |
| **Registry** | Images built locally; no external registry needed for dev |
| **CGO** | Disabled (`CGO_ENABLED=0`) — pure Go binaries |
| **Base Images** | `golang:1.26-alpine` (build), `alpine:3.20` (runtime) |

---

## Common Commands

```bash
# Start dev stack
docker compose -f docker-compose.dev.yml up -d --build

# View logs
docker compose -f docker-compose.dev.yml logs -f api
docker compose -f docker-compose.dev.yml logs -f worker

# Stop
docker compose -f docker-compose.dev.yml down

# Clean slate
docker compose -f docker-compose.dev.yml down -v

# Build images manually
docker build -f backend/Dockerfile.api -t berth-api ./backend
docker build -f backend/Dockerfile.worker -t berth-worker ./backend
docker build -f frontend/Dockerfile -t berth-frontend ./frontend

# Rebuild single service
docker compose -f docker-compose.dev.yml up -d --build --force-recreate api
```

---

## Production Notes

See [DEPLOYMENT.md](DEPLOYMENT.md) for production deployment with Traefik, Let's Encrypt, and production-grade resource limits.