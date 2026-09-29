# Deployment Guide

## Overview

This guide covers deploying Berth to a single Linux host (VPS/VM) using Docker Compose. Multi-node deployment is not supported.

---

## Prerequisites

### Server Requirements

| Resource | Minimum | Recommended |
|----------|---------|-------------|
| **CPU** | 2 cores | 4+ cores |
| **RAM** | 4 GB | 8+ GB |
| **Storage** | 20 GB SSD | 50+ GB NVMe |
| **OS** | Ubuntu 22.04+ / Debian 12+ | Ubuntu 24.04 LTS |
| **Kernel** | 5.15+ | 6.x |

### Required Software

```bash
# Install dependencies
apt-get update && apt-get install -y \
    docker.io docker-compose-v2 \
    postgresql-client \
    git curl jq \
    iptables iproute2

# Enable Docker
systemctl enable --now docker

# Configure user for Docker
usermod -aG docker $USER
```

---

## Environment Configuration

### Create Environment File

```bash
# /opt/berth/.env
cat > /opt/berth/.env << 'EOF'
# Database
DATABASE_URL=postgres://berth:berth@postgres:5432/berth?sslmode=disable

# Redis
REDIS_URL=redis://redis:6379

# NATS
NATS_URL=nats://nats:4222

# Security (GENERATE NEW IN PRODUCTION!)
JWT_SECRET=your-64-char-random-hex-string-here!!
ENCRYPTION_KEY=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef

# GitHub OAuth (create at https://github.com/settings/developers)
GITHUB_CLIENT_ID=your-github-client-id
GITHUB_CLIENT_SECRET=your-github-client-secret
GITHUB_CALLBACK_URL=https://api.yourdomain.com/api/auth/github/callback

# Frontend
FRONTEND_URL=https://app.yourdomain.com

# Docker
DOCKER_HOST=unix:///var/run/docker.sock
DOCKER_NETWORK=berth
TRAEFIK_DOMAIN=yourdomain.com

# Workspace
WORKSPACE_ROOT=/var/lib/berth/workspaces
EOF

chmod 600 /opt/berth/.env
```

### Generate Secrets

```bash
# JWT Secret (64 hex chars = 32 bytes)
openssl rand -hex 32

# Encryption Key (64 hex chars = 32 bytes)
openssl rand -hex 32
```

---

## Docker Compose Production

### Production Compose File

```yaml
# /opt/berth/docker-compose.prod.yml
version: '3.8'

services:
  postgres:
    image: postgres:16-alpine
    restart: unless-stopped
    environment:
      POSTGRES_USER: berth
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      POSTGRES_DB: berth
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "127.0.0.1:5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U berth -d berth"]
      interval: 10s
      timeout: 5s
      retries: 5
    deploy:
      resources:
        limits:
          memory: 1G
        reservations:
          memory: 512M

  redis:
    image: redis:7-alpine
    restart: unless-stopped
    command: ["redis-server", "--appendonly", "yes", "--maxmemory", "256mb", "--maxmemory-policy", "allkeys-lru"]
    volumes:
      - redis_data:/data
    ports:
      - "127.0.0.1:6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
    deploy:
      resources:
        limits:
          memory: 512M

  nats:
    image: nats:2.10-alpine
    restart: unless-stopped
    command: ["-js", "-m", "8222", "-sd", "/data/jetstream"]
    volumes:
      - nats_data:/data/jetstream
    ports:
      - "127.0.0.1:4222:4222"
      - "127.0.0.1:8222:8222"
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8222/healthz"]
      interval: 30s
      timeout: 10s
      retries: 3
    deploy:
      resources:
        limits:
          memory: 512M

  api:
    image: ghcr.io/yourorg/berth-api:latest
    restart: unless-stopped
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
      FRONTEND_URL: ${FRONTEND_URL}
      WORKSPACE_ROOT: /workspaces
      DOCKER_HOST: unix:///var/run/docker.sock
      DOCKER_NETWORK: berth
      TRAEFIK_DOMAIN: ${TRAEFIK_DOMAIN}
      ENV: production
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - berth_workspaces:/workspaces
    networks:
      - berth
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
      nats:
        condition: service_healthy
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.api.rule=Host(`api.${TRAEFIK_DOMAIN}`)"
      - "traefik.http.services.api.loadbalancer.server.port=8080"

  worker:
    image: ghcr.io/yourorg/berth-worker:latest
    restart: unless-stopped
    environment:
      MODE: worker
      DATABASE_URL: postgres://berth:berth@postgres:5432/berth?sslmode=disable
      REDIS_URL: redis://redis:6379
      NATS_URL: nats://nats:4222
      ENCRYPTION_KEY: ${ENCRYPTION_KEY}
      GITHUB_CLIENT_ID: ${GITHUB_CLIENT_ID}
      GITHUB_CLIENT_SECRET: ${GITHUB_CLIENT_SECRET}
      FRONTEND_URL: ${FRONTEND_URL}
      WORKSPACE_ROOT: /workspaces
      DOCKER_HOST: unix:///var/run/docker.sock
      DOCKER_NETWORK: berth
      TRAEFIK_DOMAIN: ${TRAEFIK_DOMAIN}
      ENV: production
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - berth_workspaces:/workspaces
    networks:
      - berth
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
      nats:
        condition: service_healthy

  frontend:
    image: ghcr.io/yourorg/berth-frontend:latest
    restart: unless-stopped
    environment:
      NEXT_PUBLIC_API_URL: https://api.${TRAEFIK_DOMAIN}
      NEXT_PUBLIC_WS_URL: wss://api.${TRAEFIK_DOMAIN}
    ports:
      - "127.0.0.1:3000:3000"
    networks:
      - berth
    depends_on:
      - api

  traefik:
    image: traefik:v2.11
    restart: unless-stopped
    command:
      - --api.insecure=false
      - --providers.docker=true
      - --providers.docker.exposedbydefault=false
      - --entrypoints.web.address=:80
      - --entrypoints.websecure.address=:443
      - --certificatesresolvers.myresolver.acme.tlschallenge=true
      - --certificatesresolvers.myresolver.acme.email=${ACME_EMAIL}
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - letsencrypt:/letsencrypt
    networks:
      - berth

volumes:
  postgres_data:
  redis_data:
  nats_data:
  berth_workspaces:
  letsencrypt:

networks:
  berth:
    driver: bridge
```

---

## SSL/TLS with Let's Encrypt

Add to `.env`:

```bash
ACME_EMAIL=admin@yourdomain.com
```

Traefik will automatically obtain certificates via ACME TLS challenge on port 80.

---

## Deployment Steps

```bash
# 1. Prepare host
mkdir -p /opt/berth
cd /opt/berth

# 2. Create .env with production values
# (use the template above, replace all placeholder values)

# 2. Deploy
docker compose -f docker-compose.prod.yml up -d --build

# 3. Verify
docker compose -f docker-compose.prod.yml ps
curl https://api.yourdomain.com/health
```

---

## Backup & Restore

### Backup
```bash
# Database
docker compose exec -T postgres pg_dump -U berth berth | gzip > backup-$(date +%F).sql.gz

# Workspaces
tar -czf workspaces-$(date +%F).tar.gz /var/lib/berth/workspaces
```

### Restore
```bash
# Database
gunzip -c backup-2024-01-15.sql.gz | docker compose exec -T postgres psql -U berth -d berth

# Workspaces
tar -xzf workspaces-2024-01-15.tar.gz -C /var/lib/berth/
```

---

## Health Checks

```bash
# API health
curl -f https://api.yourdomain.com/health

# Database
docker compose exec postgres pg_isready -U berth -d berth

# NATS
curl -f http://localhost:8222/healthz

# Redis
docker compose exec redis redis-cli ping
```

---

## Rolling Updates

```bash
# Pull new images
docker compose -f docker-compose.prod.yml pull

# Restart with zero-downtime (Traefik handles draining)
docker compose -f docker-compose.prod.yml up -d --build --force-recreate api worker

# Verify
docker compose -f docker-compose.prod.yml ps
```

---

## Security Notes

- **Never commit `.env` to version control** — use a secrets manager in CI/CD
- **Rotate `JWT_SECRET` and `ENCRYPTION_KEY` annually**
- **Use strong `POSTGRES_PASSWORD`** (32+ chars)
- **Restrict Docker socket access** — only the `worker` container needs it
- **Firewall** — only expose ports 80/443; keep 5432, 6379, 4222, 8080 internal