# Deployment Guide

## Overview

This guide covers deploying Berth to a single Linux host (VPS/VM) using Docker Compose. For multi-node production, see [ROADMAP.md](ROADMAP.md).

---

## Prerequisites

### Server Requirements
| Resource | Minimum | Recommended |
|----------|---------|-------------|
| **CPU** | 4 cores | 8+ cores |
| **RAM** | 8 GB | 16+ GB |
| **Storage** | 50 GB SSD | 200+ GB NVMe |
| **OS** | Ubuntu 22.04+ / Debian 12+ | Ubuntu 24.04 LTS |
| **Kernel** | 5.15+ | 6.x |

### Required Software
```bash
# Install dependencies
apt-get update && apt-get install -y \
    docker.io docker-compose \
    containerd runc \
    postgresql-client \
    git curl jq \
    iptables iproute2

# Enable Docker
systemctl enable --now docker

# Configure containerd rootless (see DOCKER.md)
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
DOCKER_NETWORK=berth_network
TRAEFIK_DOMAIN=yourdomain.com

# Model Storage
MODEL_DIR=/var/lib/berth/models
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
    environment:
      POSTGRES_DB: berth
      POSTGRES_USER: berth
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U berth"]
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
    command: redis-server --appendonly yes --maxmemory 256mb --maxmemory-policy allkeys-lru
    volumes:
      - redis_data:/data
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
    command: ["-js", "-m", "8222", "-c", "/etc/nats/nats.conf"]
    ports:
      - "4222:4222"
      - "8222:8222"
    volumes:
      - nats_data:/data
      - ./nats.conf:/etc/nats/nats.conf:ro
    healthcheck:
      test: ["CMD", "nats", "server", "check"]
      interval: 30s
      timeout: 10s
      retries: 3
    deploy:
      resources:
        limits:
          memory: 512M

  api:
    image: berth/api:latest
    build:
      context: ./backend
      dockerfile: Dockerfile.api
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgres://berth:berth@postgres:5432/berth?sslmode=disable
      - REDIS_URL=redis://redis:6379
      - NATS_URL=nats://nats:4222
      - JWT_SECRET=${JWT_SECRET}
      - GITHUB_CLIENT_ID=${GITHUB_CLIENT_ID}
      - GITHUB_CLIENT_SECRET=${GITHUB_CLIENT_SECRET}
      - GITHUB_CALLBACK_URL=https://${TRAEFIK_DOMAIN}/api/auth/github/callback
      - FRONTEND_URL=https://app.${TRAEFIK_DOMAIN}
      - ENCRYPTION_KEY=${ENCRYPTION_KEY}
      - PORT=8080
      - MODE=api
      - DOCKER_HOST=unix:///var/run/docker.sock
      - DOCKER_NETWORK=berth_network
      - TRAEFIK_DOMAIN=${TRAEFIK_DOMAIN}
      - MODEL_DIR=/var/lib/berth/models
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - berth_models:/var/lib/berth/models
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
      nats:
        condition: service_healthy
    deploy:
      resources:
        limits:
          memory: 1G
        reservations:
          memory: 512M
    restart: unless-stopped
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.api.rule=Host(`api.${TRAEFIK_DOMAIN}`)"
      - "traefik.http.routers.api.tls.certresolver=letsencrypt"
      - "traefik.http.services.api.loadbalancer.server.port=8080"

  worker:
    image: berth/worker:latest
    build:
      context: ./backend
      dockerfile: Dockerfile.worker
    environment:
      - DATABASE_URL=postgres://berth:berth@postgres:5432/berth?sslmode=disable
      - REDIS_URL=redis://redis:6379
      - NATS_URL=nats://nats:4222
      - ENCRYPTION_KEY=${ENCRYPTION_KEY}
      - MODE=worker
      - DOCKER_HOST=unix:///var/run/docker.sock
      - DOCKER_NETWORK=berth_network
      - MODEL_DIR=/var/lib/berth/models
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - berth_models:/var/lib/berth/models
    depends_on:
      - api
    deploy:
      resources:
        limits:
          memory: 2G
        reservations:
          memory: 1G
    restart: unless-stopped
    deploy:
      replicas: 2

  frontend:
    image: berth/frontend:latest
    build:
      context: ./frontend
      dockerfile: Dockerfile
    environment:
      - NEXT_PUBLIC_API_URL=https://api.${TRAEFIK_DOMAIN}
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.frontend.rule=Host(`app.${TRAEFIK_DOMAIN}`)"
      - "traefik.http.routers.frontend.tls.certresolver=letsencrypt"
      - "traefik.http.services.frontend.loadbalancer.server.port=3000"

  traefik:
    image: traefik:v3.0
    command:
      - "--api.dashboard=true"
      - "--providers.docker=true"
      - "--providers.docker.exposedbydefault=false"
      - "--entrypoints.web.address=:80"
      - "--entrypoints.websecure.address=:443"
      - "--certificatesresolvers.letsencrypt.acme.email=admin@${TRAEFIK_DOMAIN}"
      - "--certificatesresolvers.letsencrypt.acme.storage=/letsencrypt/acme.json"
      - "--certificatesresolvers.letsencrypt.acme.tlschallenge=true"
      - "--certificatesresolvers.letsencrypt.acme.httpchallenge=true"
      - "--certificatesresolvers.letsencrypt.acme.httpchallenge.entrypoint=web"
    ports:
      - "80:80"
      - "443:443"
      - "8080:8080"  # Dashboard
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - letsencrypt:/letsencrypt
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.traefik.rule=Host(`traefik.${TRAEFIK_DOMAIN}`)"
      - "traefik.http.routers.traefik.tls.certresolver=letsencrypt"
      - "traefik.http.routers.traefik.service=api@internal"

networks:
  default:
    name: berth_network

volumes:
  postgres_data:
  redis_data:
  nats_data:
  berth_models:
  letsencrypt:
```

### NATS Config (`/opt/berth/nats.conf`)
```
listen: 0.0.0.0:4222
http: 8222

jetstream {
  max_memory: 1GB
  max_file: 10GB
  store_dir: "/data/jetstream"
}

authorization {
  timeout: 10
}
```

---

## Deployment Steps

### 1. Prepare Server
```bash
# Create deployment user
useradd -m -s /bin/bash berth
usermod -aG docker berth

# Create directories
mkdir -p /opt/berth /var/lib/berth/models
chown -R berth:berth /opt/berth /var/lib/berth

# Copy config
cp .env /opt/berth/
cp nats.conf /opt/berth/
cp docker-compose.prod.yml /opt/berth/
chown berth:berth /opt/berth/*
```

### 2. Build Images
```bash
# On build server or CI/CD
cd /path/to/berth

# Build images
docker build -f backend/Dockerfile.api -t berth/api:v0.4.0 ./backend
docker build -f backend/Dockerfile.worker -t berth/worker:v0.4.0 ./backend
docker build -t berth/frontend:v0.4.0 ./frontend

# Push to registry (if using remote)
docker push your-registry/berth/api:v0.4.0
docker push your-registry/berth/worker:v0.4.0
docker push your-registry/berth/frontend:v0.4.0
```

### 3. Deploy
```bash
# On target server
cd /opt/berth

# Pull images (if using registry)
docker compose -f docker-compose.prod.yml pull

# Start services
docker compose -f docker-compose.prod.yml up -d

# Run migrations
docker compose -f docker-compose.prod.yml exec api migrate -path /migrations -database "$DATABASE_URL" up

# Check status
docker compose -f docker-compose.prod.yml ps
```

### 4. Verify Deployment
```bash
# Check all services healthy
docker compose -f docker-compose.prod.yml ps

# Check API health
curl -f https://api.yourdomain.com/health

# Check frontend
curl -f https://app.yourdomain.com

# Check Traefik dashboard
open https://traefik.yourdomain.com

# Check logs
docker compose -f docker-compose.prod.yml logs -f api
```

---

## SSL/TLS with Traefik

### Automatic Let's Encrypt
Traefik automatically provisions certificates via ACME HTTP-01 challenge.

**Requirements:**
- Domain A record pointing to server IP
- Ports 80/443 open on firewall
- Valid email in `traefik` config

### Custom Certificates (Optional)
```yaml
# In traefik dynamic config
tls:
  certificates:
    - certFile: /certs/cert.pem
      keyFile: /certs/key.pem
```

---

## Backup & Restore

### Database Backup
```bash
# Daily backup script
cat > /opt/berth/backup.sh << 'EOF'
#!/bin/bash
DATE=$(date +%Y%m%d_%H%M%S)
docker exec berth_postgres pg_dump -U berth berth | gzip > /backups/berth_${DATE}.sql.gz
# Keep last 30 days
find /backups -name "berth_*.sql.gz" -mtime +30 -delete
EOF

chmod +x /opt/berth/backup.sh

# Cron: daily at 2 AM
echo "0 2 * * * root /opt/berth/backup.sh" >> /etc/crontab
```

### Restore
```bash
# Restore from backup
gunzip -c /backups/berth_20240115_020000.sql.gz | docker exec -i berth_postgres psql -U berth berth
```

### Model Backup
```bash
# Backup ONNX models
tar -czf /backups/models_${DATE}.tar.gz /var/lib/berth/models
```

---

## Monitoring

### Health Checks
```bash
# Service health
docker compose -f docker-compose.prod.yml ps --format "table {{.Name}}\t{{.Status}}\t{{.Health}}"

# API health
curl -s https://api.yourdomain.com/health | jq .

# Database connections
docker exec berth_postgres psql -U berth -c "SELECT count(*) FROM pg_stat_activity;"
```

### Logs
```bash
# All services
docker compose -f docker-compose.prod.yml logs -f --tail=100

# Specific service
docker compose -f docker-compose.prod.yml logs -f api --tail=100

# Follow with timestamps
docker compose -f docker-compose.prod.yml logs -f -t api
```

### Prometheus Metrics (Future)
```yaml
# Add to docker-compose
prometheus:
  image: prom/prometheus
  volumes:
    - ./prometheus.yml:/etc/prometheus/prometheus.yml
  ports:
    - "9090:9090"
```

---

## Scaling

### Horizontal Worker Scaling
```bash
# Scale workers
docker compose -f docker-compose.prod.yml up -d --scale worker=4

# Or update compose file
deploy:
  replicas: 4
```

### Resource Limits
```yaml
deploy:
  resources:
    limits:
      cpus: '2'
      memory: 2G
    reservations:
      cpus: '1'
      memory: 1G
```

---

## Updates & Rollbacks

### Rolling Update
```bash
# Pull new images
docker compose -f docker-compose.prod.yml pull

# Rolling restart (zero-downtime for API)
docker compose -f docker-compose.prod.yml up -d --no-deps api

# Worker rolling update
docker compose -f docker-compose.prod.yml up -d --no-deps --scale worker=2 worker
# Wait, then scale down old
```

### Rollback
```bash
# Tag current as backup
docker tag berth/api:latest berth/api:backup-$(date +%Y%m%d)

# Deploy previous version
docker tag berth/api:v0.3.0 berth/api:latest
docker compose -f docker-compose.prod.yml up -d --no-deps api
```

---

## Security Hardening

### Firewall (UFW)
```bash
ufw default deny incoming
ufw default allow outgoing
ufw allow 22/tcp    # SSH
ufw allow 80/tcp    # HTTP
ufw allow 443/tcp   # HTTPS
ufw enable
```

### Docker Security
```yaml
# In docker-compose
security_opt:
  - no-new-privileges:true
read_only: true
tmpfs:
  - /tmp
  - /run
cap_drop:
  - ALL
cap_add:
  - CHOWN
  - DAC_OVERRIDE
  - SETGID
  - SETUID
```

### Secrets Management
```bash
# Use Docker secrets (Swarm mode)
echo "secret-value" | docker secret create jwt_secret -

# Or use HashiCorp Vault / AWS Secrets Manager
```

---

## Troubleshooting

| Issue | Check |
|-------|---------|
| API 502 | `docker logs api`, check DB/NATS connectivity |
| Worker not starting | `docker logs worker`, check Docker socket access |
| Traefik certs failing | Check port 80/443 open, DNS A record |
| DB connection refused | `docker exec postgres pg_isready -U berth` |
| Worker OOM | Increase memory limit, check for leaks |
| NATS JetStream full | Increase `max_file`, check disk space |

---

## Maintenance Windows

| Task | Frequency | Window |
|------|-----------|--------|
| OS Updates | Monthly | Sunday 03:00 |
| Docker Image Updates | Bi-weekly | Sunday 04:00 |
| DB Vacuum | Weekly | Sunday 02:00 |
| Log Rotation | Daily | Automatic |
| Backup Verification | Weekly | Monday 09:00 |