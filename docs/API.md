# API Reference

**Base URL**: `http://api.localhost` (dev) / `https://api.yourdomain.com` (prod)

All endpoints require authentication except where noted. Auth via `Authorization: Bearer <jwt>` or `berth_token` cookie (`credentials: include`).

---

## Authentication

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/auth/github` | Initiate GitHub OAuth (redirect) |
| `GET` | `/auth/github/authorize` | Alias for `/auth/github` |
| `GET` | `/auth/github/callback` | GitHub OAuth callback |
| `GET` | `/auth/dev-login` | **Dev only** — mint session for fixed dev user (not in production) |
| `POST` | `/auth/logout` | Clear session cookie |

### User
| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/user/me` | Current user profile |

---

## Organizations

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/orgs` | Create organization |
| `GET` | `/api/orgs` | List user's organizations |
| `POST` | `/api/orgs/:id/members` | Add member (role: `EDITOR`/`VIEWER`) |
| `GET` | `/api/orgs/:id/members` | List members |

---

## Projects

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/projects` | Create project |
| `GET` | `/api/projects` | List user's projects |
| `GET` | `/api/projects/:id` | Get project |
| `GET` | `/api/orgs/:id/projects` | List project for org |
| `GET` | `/api/projects/:id/sandboxes` | List environments in project (legacy key: `sandboxes`) |
| `POST` | `/api/projects/:id/share-links` | Create share link |
| `GET` | `/api/projects/:id/share-links` | List share links |
| `DELETE` | `/api/projects/:id/share-links/:linkId` | Revoke share link |

---

## Environments

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/environments` | List user's environments |
| `POST` | `/api/environments` | Create environment |
| `POST` | `/api/environments/:id/fork` | Fork environment |
| `GET` | `/api/environments/:id` | Get environment |
| `DELETE` | `/api/environments/:id` | Delete environment |
| `POST` | `/api/environments/:id/stop` | Stop environment |
| `POST` | `/api/environments/:id/restart` | Restart environment |
| `POST` | `/api/environments/:id/start` | Start environment |
| `POST` | `/api/environments/:id/exec` | Execute command in container |
| `GET` | `/api/environments/:id/logs` | Get logs (text) |
| `GET` | `/api/environments/:id/files` | List files |
| `GET` | `/api/environments/:id/files/content` | Get file content |
| `PUT` | `/api/environments/:id/files/content` | Update file content |
| `POST` | `/api/environments/:id/files/create` | Create file/directory |
| `POST` | `/api/environments/:id/files/delete` | Delete file/directory |

### Git
| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/environments/:id/git/status` | Status (branch, dirty, ahead/behind) |
| `GET` | `/api/environments/:id/git/branches` | List branches |
| `POST` | `/api/environments/:id/git/branch` | Create branch |
| `POST` | `/api/environments/:id/git/checkout` | Checkout branch |
| `POST` | `/api/environments/:id/git/pull` | Pull from remote |
| `POST` | `/api/environments/:id/git/commit` | Commit changes |
| `POST` | `/api/environments/:id/git/push` | Push to remote |
| `GET` | `/api/environments/:id/git/log` | Commit history |

---

## Change Requests

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/projects/:id/change-requests` | Create change request |
| `GET` | `/api/projects/:id/change-requests` | List change requests |
| `GET` | `/api/change-requests/:id` | Get change request |
| `PUT` | `/api/change-requests/:id` | Update change request |
| `POST` | `/api/change-requests/:id/merge` | Merge change request |
| `POST` | `/api/change-requests/:id/close` | Close change request |
| `GET` | `/api/change-requests/:id/diff` | Get diff (stub) |

---

## Share Links

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/projects/:id/share-links` | Create share link |
| `GET` | `/api/projects/:id/share-links` | List share links |
| `DELETE` | `/api/projects/:id/share-links/:linkId` | Revoke share link |
| `POST` | `/api/join` | Join via share link (requires auth) |
| `GET` | `/api/share-links/validate?code=<code>` | Validate code without consuming use |

---

## Activity

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/activity` | Record activity |
| `POST` | `/api/activity/session/start` | Start session |
| `POST` | `/api/activity/session/end` | End session |
| `POST` | `/api/environments/resume` | Resume suspended environment |
| `GET` | `/api/environments/idle` | List idle environments (stub) |

---

## WebSocket

| Endpoint | Description |
|----------|-------------|
| `GET /ws/environments/:id` | Terminal + file edits + presence |
| `GET /ws/sandbox/:id` | Legacy alias |
| `GET /ws/sandboxes/:id` | Legacy alias |

**Auth**: `?token=<jwt>` or `berth_token` cookie. Protocol: JSON `{type, payload, timestamp, userId, sandboxId}`.

---

## Preview Proxy

| Endpoint | Description |
|----------|-------------|
| `GET /p/:id/*path` | Proxy to sandbox container |
| `GET /p/:id` | Proxy root |

No auth required. Traefik routes `Host(\`<id>.localhost\`)` to this.

---

## Health

| Endpoint | Description |
|----------|-------------|
| `GET /health` | `{status, time, dev_auth, db, redis}` |

---

## Rate Limits

| Scope | Limit |
|-------|-------|
| `/api/*` (global) | 200 req/min per IP+path |
| Authenticated routes | 120 req/min per user (shared bucket) |

---

## Error Format

```json
{
  "error": "human-readable message"
}
```

HTTP status codes: `200` success, `201` created, `202` accepted, `400` bad request, `401` unauthorized, `403` forbidden, `404` not found, `409` conflict, `410` gone, `429` rate limited, `500` server error, `502` bad gateway, `503` unavailable.