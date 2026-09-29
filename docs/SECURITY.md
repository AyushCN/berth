# Security

## Threat Model

### Scope
Berth is a **single-host, single-tenant** sandbox platform for development environments. It is **not** designed for:
- Multi-tenant SaaS hosting
- Untrusted workloads from the public internet
- Multi-node clusters

### Trust Boundaries

| Boundary | Implementation | Limitation |
|----------|----------------|------------|
| User ↔ API | JWT + GitHub OAuth (PKCE S256) | No mTLS |
| API ↔ Worker | NATS (no auth in dev) | No mutual TLS |
| Worker ↔ Docker | Unix socket (rootless) | Host socket = host access |
| Container Isolation | Rootless Docker + seccomp | Host networking, no gVisor |
| Network | Host network namespace | No per-container isolation |
| Secrets | Encrypted at rest (AES-GCM) | Keys in env vars |

---

## Authentication & Authorization

### GitHub OAuth 2.0 (PKCE S256)
- **Authorization Code Grant** with `S256` code challenge
- Scopes: `read:user`, `user:email`, `repo`
- Tokens encrypted at rest with AES-256-GCM (`ENCRYPTION_KEY`)

### JWT Sessions
- **Algorithm**: HS256
- **Expiry**: 24 hours
- **Claims**: `userId` (UUID), `exp`
- **No refresh tokens** — re-auth via GitHub on expiry

### Cookie Security
- **Name**: `berth_token`
- **Attributes**: `HttpOnly`, `SameSite=Lax`, `Secure` (derived from TLS / `X-Forwarded-Proto`)
- **No `Secure` over plain HTTP** — dev login would silently fail

### Dev Login
- **Route**: `GET /api/auth/dev-login`
- **Mints**: Token for fixed UID `00000000-0000-0000-0000-000000000001`
- **Scope**: **Only registered when `ENV != "production"`**
- **Signing**: Uses production `JWT_SECRET` — **do not run with `ENV=development` in production**

---

## Container Security

### Runtime Flags
```bash
docker run \
  --read-only \
  --cap-drop ALL \
  --init \
  --user 1000:1000 \
  --security-opt no-new-privileges \
  --security-opt seccomp=default \
  --pids-limit 256 \
  --memory 512m \
  --cpus 1.0 \
  --network host \
  --tmpfs /tmp:rw,noexec,nosuid,size=100m \
  --tmpfs /var/tmp:rw,noexec,nosuid,size=100m \
  --tmpfs /run:rw,noexec,nosuid,size=10m
```

### Host Docker Socket
- **Worker mounts `/var/run/docker.sock`** — full Docker API access
- **Compromise = host compromise** — single-host prototype, not a tenant boundary
- **No gVisor / runsc** — rootless Docker only

### Network
- **Host networking only** — containers bind directly on host ports
- **No CNI / gVisor / Cilium** — no per-sandbox network isolation
- **Traefik** on host ports 80/443 routes `*.localhost` to sandboxes

---

## Secrets Management

### At Rest
- **GitHub OAuth tokens** — AES-256-GCM encrypted with `ENCRYPTION_KEY`
- **JWT signing** — HS256 with `JWT_SECRET`
- **Keys** — Must be 32 bytes raw or 64 hex chars; validated on boot

### In Transit
- **TLS** — Traefik terminates TLS (Let's Encrypt) at edge
- **Internal** — No mTLS; NATS/PostgreSQL/Redis in plaintext on private network

### Key Rotation
- **`JWT_SECRET`** — Rotate annually; invalidates all sessions
- **`ENCRYPTION_KEY`** — Rotate annually; re-encrypt tokens with `scripts/rotate-encryption.sh`
- **GitHub OAuth secret** — Rotate via GitHub settings

---

## Rate Limiting

| Bucket | Scope | Limit | Action |
|--------|-------|-------|--------|
| `/api/*` | Per IP + path | 200/min | 429 |
| Authenticated routes | Per user (single bucket) | 120/min | 429 |

Frontend polling (~36/min) is below the 120/min ceiling.

---

## Known Gaps

| Gap | Severity | Mitigation |
|-----|----------|------------|
| **No mTLS** | Plaintext internal traffic | Run on private network/VPC |
| **Host Docker socket** | Worker compromise = host compromise | Single-host only; no untrusted workloads |
| **No gVisor** | Container breakout risk | Rootless + seccomp + caps drop |
| **No multi-node** | Single point of failure | Single-host prototype |
| **No mTLS/SPIFFE** | Internal traffic unencrypted | Private VPC only |
| **No Cilium/CNI** | No network policies | Host networking only |

---

## Hardening Checklist (Production)

- [ ] Generate unique `JWT_SECRET` (64 hex chars)
- [ ] Generate unique `ENCRYPTION_KEY` (64 hex chars)
- [ ] Set strong `POSTGRES_PASSWORD` (32+ chars)
- [ ] Configure `ACME_EMAIL` for Let's Encrypt
- [ ] Set `ENV=production` (disables `/auth/dev-login`)
- [ ] Configure Traefik with `acme.email`
- [ ] Firewall: only 80/443 exposed; 5432/6379/4222/8080 internal
- [ ] Disable `dev-login` in production (enforced by `ENV=production`)
- [ ] Rotate `JWT_SECRET` / `ENCRYPTION_KEY` annually
- [ ] Monitor `docker compose ps` for unhealthy services
- [ ] Audit `docker compose exec` access

---

## Incident Response

### Compromised Worker
1. Revoke `JWT_SECRET` (invalidates all sessions)
2. Rotate `ENCRYPTION_KEY` (re-encrypt tokens)
3. Rotate GitHub OAuth secret
4. Audit `docker ps -a` for unexpected containers

### Leaked `ENCRYPTION_KEY`
1. Generate new key
2. Re-encrypt all `github_token_encrypted` in `users` table
3. Deploy new key; old tokens become unreadable

### Database Leak
- GitHub tokens are AES-256-GCM encrypted
- JWTs are signed, not encrypted (claims readable)
- `ENCRYPTION_KEY` rotation re-protects tokens