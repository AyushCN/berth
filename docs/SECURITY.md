# Security

## Threat Model

### Assets
| Asset | Sensitivity | Protection |
|-------|-------------|------------|
| User GitHub Tokens | Critical | AES-256-GCM at rest |
| User Code/Workspaces | High | Container isolation |
| Build Artifacts | Medium | Signed images |
| Prediction Models | Low | File system permissions |
| API Keys/Secrets | Critical | Env vars, not in code |

### Trust Boundaries
```
Internet → [Traefik TLS] → [API Gateway] → [Internal Services]
                                    │
                    ┌───────────────┼───────────────┐
                    ▼               ▼               ▼
              [PostgreSQL]      [Redis]          [NATS]
                    │               │               │
                    └───────────────┼───────────────┘
                                    ▼
                              [containerd]
                                    │
                              [Sandbox Container]
                                    │
                              [User Workspace]
```

### Attacker Capabilities Assumed
- Network access to API endpoints
- Valid GitHub OAuth account
- Ability to create sandboxes
- Ability to push code to own forks
- No container escape (runc.v2)
- No host kernel exploits

---

## Authentication & Authorization

### Authentication Flow
```
1. User clicks "Login with GitHub"
2. Redirect to GitHub OAuth
3. GitHub redirects to /api/auth/github/callback
4. Exchange code for access token
5. Fetch GitHub user info
6. Create/update user in DB
6. Generate JWT (24h expiry)
7. Set HttpOnly Secure cookie
8. Redirect to frontend
```

### JWT Token
```json
{
  "sub": "user-uuid",
  "email": "user@example.com",
  "username": "githubuser",
  "iat": 1700000000,
  "exp": 1700086400
}
```

### Token Validation
```go
// middleware.Auth
func Auth(secret string) gin.HandlerFunc {
    return func(c *gin.Context) {
        token := extractToken(c)  // Header or cookie
        claims, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (interface{}, error) {
            return []byte(secret), nil
        })
        if err != nil || !claims.Valid {
            c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
            return
        }
        c.Set("userID", claims.Subject)
        c.Next()
    }
}
```

### Role-Based Access
| Resource | Owner | Editor | Viewer |
|----------|-------|--------|--------|
| Sandbox | CRUD | CRUD | Read |
| Project | CRUD | Read | Read |
| Share Link | Create/Revoke | - | - |
| Change Request | Merge/Close | Create | View |
| Predictions | Read/Write | Read | Read |

---

## Data Protection

### Encryption at Rest
```go
// AES-256-GCM for sensitive fields
func Encrypt(plaintext []byte, key []byte) ([]byte, error) {
    block, _ := aes.NewCipher(key)
    gcm, _ := cipher.NewGCM(block)
    nonce := make([]byte, gcm.NonceSize())
    rand.Read(nonce)
    return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func Decrypt(ciphertext []byte, key []byte) ([]byte, error) {
    block, _ := aes.NewCipher(key)
    gcm, _ := cipher.NewGCM(block)
    nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
    return gcm.Open(nil, nonce, ciphertext, nil)
}
```

**Encrypted Fields:**
- GitHub OAuth tokens (access_token, refresh_token)
- Git credentials in sandbox
- Any PII

### Encryption Key Management
- 32-byte key from `ENCRYPTION_KEY` env var (64 hex chars)
- Key rotation: Re-encrypt all tokens on key change
- Key never logged or exposed in API responses

---

## Network Security

### TLS Configuration
| Connection | Protocol | Verification |
|------------|----------|--------------|
| Client → Traefik | TLS 1.3 | Let's Encrypt cert |
| Traefik → API | HTTP (internal) | mTLS planned |
| API → PostgreSQL | TCP | SSL mode=disable (local) |
| API → Redis | TCP | No auth (local) |
| API → NATS | TCP | No auth (local) |
| Worker → containerd | Unix socket | Rootless user namespace |

### Firewall Rules (UFW)
```bash
ufw default deny incoming
ufw default allow outgoing
ufw allow 22/tcp      # SSH
ufw allow 80/tcp      # HTTP (ACME challenge)
ufw allow 443/tcp     # HTTPS
# Internal only (not exposed):
# ufw allow from 10.0.0.0/8 to any port 5432  # PostgreSQL
# ufw allow from 10.0.0.0/8 to any port 6379  # Redis
# ufw allow from 10.0.0.0/8 to any port 4222  # NATS
ufw enable
```

---

## Container Security

### Sandbox Isolation
| Layer | Implementation |
|-------|----------------|
| **Runtime** | containerd + runc.v2 (rootless) |
| **User Namespace** | UID/GID mapping (100000+) |
| **Filesystem** | Bind mount workspace (read-write) |
| **Network** | Host namespace (dev), CNI planned |
| **Capabilities** | Minimal (no CAP_SYS_ADMIN) |
| **Seccomp** | Default Docker profile |
| **AppArmor** | Default Docker profile |

### Container Spec
```go
spec := domain.SandboxSpec{
    BaseImage:    profile.BaseImage,
    WorkDir:      "/workspace",
    WorkspaceDir: workspaceDir,
    Cmd:          watcherCmd,  // nodemon/air/uvicorn
    MemoryLimit:  512 * 1024 * 1024,  // 512MB
    CPULimit:     1000000000,          // 1 CPU
    ExposedPort:  &profile.ExposedPort,
    Labels: map[string]string{
        "berth.language": profile.Language,
    },
}
```

### Image Security
- **Base Images**: Official Alpine/Debian slim
- **Multi-stage Builds**: Build → Runtime separation
- **No Root**: Containers run as non-root (UID 1000)
- **Read-only Rootfs**: Where possible
- **No New Privileges**: `security_opt: no-new-privileges`

---

## API Security

### Rate Limiting
```go
// middleware.RateLimit
func RateLimit() gin.HandlerFunc {
    return ratelimit.RateLimiter(ratelimit.Config{
        Rate:  100,  // requests
        Per:   time.Minute,
        Key:   func(c *gin.Context) string { return c.ClientIP() },
    })
}

// Per-user
func RateLimitUser() gin.HandlerFunc {
    return ratelimit.RateLimiter(ratelimit.Config{
        Rate:  50,
        Per:   time.Minute,
        Key:   func(c *gin.Context) string { 
            return c.GetString("userID") 
        },
    })
}
```

### Input Validation
```go
// Struct tags for validation
type CreateSandboxRequest struct {
    Name     string `json:"name" binding:"required,min=1,max=100"`
    GitURL   string `json:"git_url" binding:"required,url,github"`
    GitBranch string `json:"git_branch" binding:"omitempty,alphanumdash,max=100"`
    ProjectID string `json:"project_id" binding:"omitempty,uuid"`
}

// URL validation
func validateGitURL(raw string) error {
    u, err := url.Parse(raw)
    if err != nil || u.Scheme != "https" || u.Host != "github.com" {
        return errors.New("only https://github.com URLs allowed")
    }
    if strings.Contains(u.Path, "..") || strings.HasPrefix(u.Path, "-") {
        return errors.New("invalid path")
    }
    return nil
}
```

### CORS
```go
func CORS(frontendURL string) gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("Access-Control-Allow-Origin", frontendURL)
        c.Header("Access-Control-Allow-Credentials", "true")
        c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
        c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return
        }
        c.Next()
    }
}
```

---

## Secrets Management

### Environment Variables
```bash
# Never commit these!
JWT_SECRET=64-char-hex-string
ENCRYPTION_KEY=64-char-hex-string
GITHUB_CLIENT_SECRET=github-oauth-secret
DATABASE_URL=postgres://user:pass@host:5432/db
```

### Secret Generation
```bash
# JWT Secret (32 bytes = 64 hex chars)
openssl rand -hex 32

# Encryption Key (32 bytes)
openssl rand -hex 32
```

### Key Rotation
```bash
# 1. Generate new key
NEW_KEY=$(openssl rand -hex 32)

# 2. Re-encrypt all tokens with new key
# (run migration script)

# 3. Update env var and restart
export ENCRYPTION_KEY=$NEW_KEY
systemctl restart berth-api berth-worker
```

---

## Incident Response

### Security Incident Checklist
1. **Detect** - Alert from monitoring/logs
2. **Contain** - Revoke tokens, disable accounts
3. **Investigate** - Check logs, audit trail
4. **Remediate** - Patch, rotate keys, redeploy
4. **Notify** - Users, authorities if needed
5. **Post-mortem** - Document, improve

### Compromise Indicators
| Indicator | Action |
|-----------|--------|
| Unknown IP in auth logs | Revoke session, force re-auth |
| High failed login rate | Block IP, alert |
| Unexpected sandbox creation | Audit user, check OAuth token |
| Large egress traffic | Isolate sandbox, investigate |
| Unknown container image | Block image, scan |

### Key Compromise Response
```bash
# 1. Rotate all keys immediately
NEW_JWT=$(openssl rand -hex 32)
NEW_ENC=$(openssl rand -hex 32)

# 2. Update all services
# 3. Force logout all users (invalidate JWTs)
# 4. Re-encrypt stored tokens with new ENCRYPTION_KEY
# 5. Rotate GitHub OAuth secret
# 6. Audit all recent access
```

---

## Compliance Considerations

### Data Handling
| Data Type | Retention | Deletion |
|-----------|-----------|----------|
| User PII | Account lifetime | On account delete |
| GitHub Tokens | Session + 30 days | On revoke |
| Sandbox Logs | 30 days | Auto-purge |
| Build Artifacts | 90 days | Auto-purge |
| Predictions | 1 year | Auto-purge |
| Models | Indefinite | Manual |

### GDPR Compliance
- **Right to Access**: `/api/user/me` returns all user data
- **Right to Deletion**: `DELETE /api/user/me` cascades
- **Data Portability**: JSON export via API
- **Consent**: Explicit OAuth consent

---

## Security Checklist (Pre-deployment)

- [ ] All secrets in env vars (not code)
- [ ] `ENCRYPTION_KEY` is 32-byte random
- [ ] `JWT_SECRET` is 32-byte random
- [ ] PostgreSQL password != default
- [ ] Redis has password (if remote)
- [ ] NATS has auth (if remote)
- [ ] Traefik TLS enabled (Let's Encrypt)
- [ ] Firewall allows only 80/443/22
- [ ] Docker `no-new-privileges` enabled
- [ ] Containers run as non-root
- [ ] Resource limits set on all containers
- [ ] Logging excludes sensitive fields
- [ ] Error messages don't leak internals
- [ ] Rate limiting enabled on all endpoints
- [ ] CORS restricted to frontend domain
- [ ] Security headers (HSTS, CSP, etc.)
- [ ] Dependency scanning enabled (Dependabot)
- [ ] Container image scanning (Trivy/Snyk)