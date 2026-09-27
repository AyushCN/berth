# API Reference

## Base URL
```
http://localhost:8080/api
```

## Authentication

### GitHub OAuth Flow
```
GET  /api/auth/github              # Initiate OAuth
GET  /api/auth/github/callback     # OAuth callback
GET  /api/auth/dev-login           # Dev login (dev only)
POST /api/auth/logout              # Logout
GET  /api/auth/me                  # Current user
```

### JWT Token
- Header: `Authorization: Bearer <token>`
- Token expires: 24 hours
- Dev token: `GET /api/auth/dev-login` (dev mode only)

---

## Sandboxes (Environments)

### List Environments
```
GET /api/environments
```
**Query Params**: `project_id` (optional)
**Response**: `Environment[]`

### Create Environment
```
POST /api/environments
```
**Body**:
```json
{
  "name": "my-sandbox",
  "git_url": "https://github.com/user/repo",
  "git_branch": "main",
  "project_id": "uuid"
}
```
**Response**: `Environment` (state: CREATED)

### Get Environment
```
GET /api/environments/:id
```
**Response**: `Environment`

### Delete Environment
```
DELETE /api/environments/:id
```

### Environment Actions
```
POST /api/environments/:id/stop      # Stop container
POST /api/environments/:id/start     # Start container
POST /api/environments/:id/restart   # Restart container
POST /api/environments/:id/fork      # Fork workspace
```

### Environment Exec
```
POST /api/environments/:id/exec
```
**Body**: `{ "command": ["sh", "-c", "ls -la"] }`
**Response**: `{ "output": "...", "exit_code": 0 }`

### Environment Logs
```
GET /api/environments/:id/logs
```
**Query**: `tail=100`, `follow=false`
**Response**: `{ "logs": "..." }`

---

## Files

### List Files
```
GET /api/environments/:id/files?path=.
```
**Response**: `FileNode[]`

### Get File Content
```
GET /api/environments/:id/files/content?path=src/main.go
```
**Response**: `{ "content": "package main...", "encoding": "utf-8" }`

### Update File Content
```
PUT /api/environments/:id/files/content?path=src/main.go
```
**Body**: `{ "content": "new content" }`

### Create File
```
POST /api/environments/:id/files/create
```
**Body**: `{ "path": "src/new.go", "is_dir": false }`

### Delete File
```
POST /api/environments/:id/files/delete
```
**Body**: `{ "path": "src/old.go" }`

---

## Git Operations

### Status
```
GET /api/environments/:id/git/status
```
**Response**: `{ "branch": "main", "dirty": true, "ahead": 2, "behind": 0 }`

### Branches
```
GET /api/environments/:id/git/branches
```
**Response**: `{ "current": "main", "branches": ["main", "feature/x"] }`

### Create Branch
```
POST /api/environments/:id/git/branch
```
**Body**: `{ "branch": "feature/new" }`

### Checkout
```
POST /api/environments/:id/git/checkout
```
**Body**: `{ "branch": "feature/new", "force": false }`

### Pull
```
POST /api/environments/:id/git/pull
```

### Commit
```
POST /api/environments/:id/git/commit
```
**Body**: `{ "message": "feat: add feature" }`

### Push
```
POST /api/environments/:id/git/push
```
**Response**: `{ "remote_branch": "berth/abc123", "push_url": "..." }`

### Log
```
GET /api/environments/:id/git/log
```
**Query**: `limit=50`
**Response**: `CommitEntry[]`

### Diff
```
GET /api/environments/:id/git/diff?file=src/main.go
```
**Response**: `{ "diff": "@@ -1,3 +1,4 @@..." }`

---

## Projects

### List Projects
```
GET /api/projects
```
**Response**: `Project[]`

### Create Project
```
POST /api/projects
```
**Body**:
```json
{
  "name": "My Project",
  "description": "Optional description",
  "owner_organization_id": "uuid",
  "is_public": false
}
```

### Get Project
```
GET /api/projects/:id
```

### Project Sandboxes
```
GET /api/projects/:id/sandboxes
```

---

## Organizations

### List Organizations
```
GET /api/orgs
```

### Create Organization
```
POST /api/orgs
```
**Body**: `{ "name": "My Org" }`

### Organization Members
```
GET /api/orgs/:id/members
POST /api/orgs/:id/members
```
**Body**: `{ "user_id": "uuid", "role": "EDITOR" }`

---

## Share Links

### Create Share Link
```
POST /api/projects/:id/share-links
```
**Body**: `{ "role": "EDITOR", "expires_at": "2024-12-31T23:59:59Z", "max_uses": 10 }`

### List Share Links
```
GET /api/projects/:id/share-links
```

### Revoke Share Link
```
DELETE /api/projects/:id/share-links/:linkId
```

### Join via Share Link
```
POST /api/join
```
**Body**: `{ "code": "abc123" }`

---

## Change Requests

### Create Change Request
```
POST /api/projects/:id/change-requests
```
**Body**: `{ "title": "Fix bug", "description": "..." }`

### List Change Requests
```
GET /api/projects/:id/change-requests
```

### Get Change Request
```
GET /api/change-requests/:id
```

### Update Change Request
```
PUT /api/change-requests/:id
```
**Body**: `{ "title": "...", "description": "..." }`

### Merge Change Request
```
POST /api/change-requests/:id/merge
```

### Close Change Request
```
POST /api/change-requests/:id/close
```

### Get Diff
```
GET /api/change-requests/:id/diff
```

---

## Predictions

### Predict Build Time
```
POST /api/predictions/build-time
```
**Body**:
```json
{
  "workspace_id": "uuid",
  "features": {
    "language": "node",
    "framework": "next",
    "architecture": "WEB_APP",
    "exposed_port": 3000,
    "num_lockfiles": 1
  }
}
```

### Predict Image Size
```
POST /api/predictions/image-size
```

### Predict Cache Hit
```
POST /api/predictions/cache-hit
```

### Predict Failure Risk
```
POST /api/predictions/failure-risk
```

### Get Prediction History
```
GET /api/predictions/history?workspace_id=uuid&type=BUILD_TIME&limit=50&offset=0
```

### Get Model Metrics
```
GET /api/predictions/models/metrics?type=BUILD_TIME
```
**Response**:
```json
{
  "mse": 12345.67,
  "mae": 89.23,
  "rmse": 111.11,
  "r2": 0.92,
  "samples": 150
}
```

### Retrain Model
```
POST /api/predictions/models/retrain
```
**Body**: `{ "type": "BUILD_TIME", "algorithm": "random_forest" }`

### Export Model ONNX
```
POST /api/predictions/models/export
```
**Body**: `{ "model_id": "uuid" }`
**Response**: `{ "onnx_path": "/tmp/berth/models/..." }`

### Activate Model
```
POST /api/predictions/models/activate
```
**Body**: `{ "model_id": "uuid" }`

---

## Activity & Warm Pool

### Record Activity
```
POST /api/activity
```
**Body**: `{ "environment_id": "uuid", "activity_type": "EDIT" }`

### Session Start/End
```
POST /api/activity/session/start
POST /api/activity/session/end
```
**Body**: `{ "environment_id": "uuid" }`

### Resume Environment
```
POST /api/environments/resume
```
**Body**: `{ "environment_id": "uuid" }`

### Get Idle Environments
```
GET /api/environments/idle
```

---

## WebSocket

### Sandbox Connection
```
GET /ws/sandbox/:id
```
**Auth**: JWT via query param `?token=...` or cookie

### Messages
```json
// Client → Server
{ "type": "terminal_input", "data": "ls\n" }
{ "type": "file_edit", "path": "main.go", "content": "..." }
{ "type": "cursor", "x": 10, "y": 5 }

// Server → Client
{ "type": "terminal_output", "data": "file1.go file2.go\n" }
{ "type": "file_changed", "path": "main.go", "content": "..." }
{ "type": "presence", "users": [{ "id": "...", "name": "...", "cursor": { "x": 10, "y": 5 } }] }
{ "type": "state_change", "state": "RUNNING" }
```

---

## Data Types

### Environment
```typescript
interface Environment {
  id: string;
  workspace_id: string;
  runtime_profile_id?: string;
  name: string;
  state: 'CREATED' | 'BUILDING' | 'BUILD_FAILED' | 'READY' | 
         'STARTING' | 'RUNNING' | 'STOPPING' | 'STOPPED' | 
         'SUSPENDING' | 'SUSPENDED' | 'CRASHED' | 'DELETING';
  container_id?: string;
  image_id?: string;
  public_url?: string;
  port: number;
  memory_limit: number;
  cpu_limit: number;
  last_activity_at?: string;
  active_sessions: number;
  suspended_at?: string;
  last_error?: string;
  restart_count: number;
  created_at: string;
  updated_at: string;
}
```

### RuntimeProfile
```typescript
interface RuntimeProfile {
  id: string;
  project_id?: string;
  workspace_id?: string;
  detection_evidence: Record<string, any>;
  language: string;
  version?: string;
  framework?: string;
  package_manager?: string;
  architecture: string;  // WEB_APP, API, CLI, FULL_STACK, MONOREPO, LIBRARY
  entrypoint?: string;
  build_command?: string;
  start_command?: string;
  port: number;
  dockerfile_source: 'USER_PROVIDED' | 'GENERATED' | 'COMPOSE';
  dockerfile_content?: string;
  requires_database: boolean;
  requires_redis: boolean;
  confidence: number;
  status: 'DETECTED' | 'CONFIRMED' | 'OVERRIDDEN';
  created_at: string;
  updated_at: string;
}
```

### Prediction
```typescript
interface Prediction {
  id: string;
  type: 'BUILD_TIME' | 'IMAGE_SIZE' | 'CACHE_HIT' | 'FAILURE_RISK';
  workspace_id: string;
  input: Record<string, any>;
  output: Record<string, any>;
  confidence: number;
  model_version: string;
  created_at: string;
}
```

### Model
```typescript
interface Model {
  id: string;
  name: string;
  version: string;
  type: 'BUILD_TIME' | 'IMAGE_SIZE' | 'CACHE_HIT' | 'FAILURE_RISK';
  algorithm: 'linear_regression' | 'random_forest' | 'xgboost';
  parameters: Record<string, any>;
  metrics: {
    mse: number;
    mae: number;
    rmse: number;
    r2: number;
    samples: number;
  };
  onnx_path?: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}
```

### BuildPlan
```typescript
interface BuildPlan {
  id: string;
  runtime_profile_id: string;
  base_image: string;
  dockerfile: string;
  build_args: Record<string, string>;
  install_command?: string;
  build_command?: string;
  start_command: string;
  working_dir: string;
  port: number;
  confidence: number;
  status: 'PENDING' | 'READY' | 'FAILED';
}
```

---

## Error Responses

### Standard Error Format
```json
{
  "error": "Human readable message",
  "code": "ERROR_CODE",
  "details": {}
}
```

### Common HTTP Status Codes
| Code | Meaning |
|------|---------|
| 200 | Success |
| 201 | Created |
| 400 | Bad Request |
| 401 | Unauthorized |
| 403 | Forbidden |
| 404 | Not Found |
| 409 | Conflict |
| 422 | Validation Error |
| 429 | Rate Limited |
| 500 | Internal Error |

---

## Rate Limiting

| Endpoint | Limit |
|----------|-------|
| Auth | 10/min |
| API | 100/min per user |
| WebSocket | 50 msg/sec |
| Predictions | 20/min |

---

## Webhooks (Planned)

```json
{
  "event": "sandbox.state_changed",
  "timestamp": "2024-01-15T10:30:00Z",
  "payload": {
    "environment_id": "uuid",
    "old_state": "BUILDING",
    "new_state": "RUNNING"
  }
}
```

Events: `sandbox.created`, `sandbox.state_changed`, `sandbox.deleted`, `build.started`, `build.finished`, `change_request.created`, `change_request.merged`