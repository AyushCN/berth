# Changelog

## [Unreleased]

### Removed
- **ML/Prediction Stack** — ONNX inference, model trainer, prediction service, build planner, all per-language build strategies, gRPC prediction service, protobufs, frontend predictions page
- **Warm Pool** — never functional; three bugs deep (wrong project lookup, nil workspace_id, reaper collision)
- **Legacy Sandbox Model** — `sandboxes` table, `sandbox_logs`, `sandbox_changes`, `sandbox_activities`, `audit_logs`, `SandboxRepository`, `SandboxUsecase`, `SandboxHandler`
- **ContainerRuntime Legacy Names** — `CreateSandbox`->`Create`, `StartSandbox`->`Start`, `StopSandbox`->`Stop`, `DeleteSandbox`->`Remove`
- **Per-User Quotas** — `max_sandboxes`, `max_builds_per_hour` (never enforced)
- **ONNX Runtime** — removed `onnxruntime_go` dependency, enabling CGO_ENABLED=0 builds
- **gRPC** — removed prediction gRPC server and protobufs
- **Frontend Dead Routes** — `/predictions` page, `/env/[id]` route, `components/git-panel`, `components/file-tree/`, `components/terminal/`, `components/code-editor/`, `components/presence-bar`, `components/environment-list`, `berth-page-reference.tsx`

### Fixed
- **Stop -> Start** — was impossible (container name collision); `Create` now clears existing container
- **Start rebuilt** — now publishes `berth.environment.start` to resume existing container; falls back to rebuild only if no container/no bus
- **IDOR on file endpoints** — added `resolveAccess` usecase with owner/collaborator checks
- **Dev-login in production** — route no longer registered when `ENV=production`
- **Auth cookies** — unified `Secure` flag derived from TLS/`X-Forwarded-Proto`; `SameSite=Lax` explicit
- **Readiness check** — worker polls app port up to 60s; marks `CRASHED` with reason if nothing listens
- **Idle suspend** — 30min dev / 60min prod (was broken; no reaper existed)
- **File/git endpoints** — now resolve workspace dir by `workspace_id` not `environment_id` (fixes dir-not-found for new envs)
- **Share links** — added `GET /api/share-links/validate` (was 404, broke `/join/<code>` flow)
- **Git log** — fixed `--oneline -20` passed as single argv; now returns real author/date
- **Env var `MODEL_DIR`** — marked dead config (prediction stack removed)
- **CGO** — removed; pure-Go builds with `CGO_ENABLED=0` (Dockerfiles updated)
- **Docs** — README, ARCHITECTURE.md, DEVELOPMENT.md, DEPLOYMENT.md, STATUS.md, API.md, SECURITY.md, QUICKSTART.md, ROADMAP.md, CONTRIBUTING.md rewritten to match reality

### Added
- **Readiness check** — worker polls container port up to 60s; marks `CRASHED` with reason if nothing listens
- **Migration runner** — `migrations.Migrate(dsn)` runs on boot; embedded SQL with `go:embed`
- **Migration 000009** — drops legacy `sandboxes`, `sandbox_logs`, `sandbox_changes`, `sandbox_activities`, `audit_logs`
- **Migration 000010** — drops prediction tables (`builds`, `build_plans`, `models`, `predictions`, `training_data`, `images`)
- **Migration 000011** — drops `users.max_sandboxes`, `max_builds_per_hour`
- **Readiness timeout** — configurable via `READINESS_TIMEOUT` (default 60s)
- **Rate limit config** — `RATE_LIMIT_REQUESTS_PER_MINUTE` (default 200), `RATE_LIMIT_AUTHENTICATED_PER_MINUTE` (default 120)
- **Auth cookie helper** — single `setAuthCookie`/`clearAuthCookie` with request-derived `Secure` flag
- **Access control** — `resolveAccess` shared by file + git endpoints (owner/collaborator)
- **Frontend** — removed `/predictions` page, dead routes, dead client methods; `gofmt` normalization

### Fixed
- **Migration 000006** — added missing `detection_evidence` to INSERT; fixed `metadata`->`payload` in down migration
- **Container name collision** — `Create` now force-removes existing container by name
- `git log --oneline -20` — was passed as single argv; now split
- `dev-login` cookie — no longer `Secure` over plain HTTP
- `dev-login` route — not registered when `ENV=production`
- `WarmPool` — deleted (was non-functional)
- `WORKSPACE_ROOT` default — now `~/.local/state/berth/workspaces` (was hardcoded `/home/swordrookie/...`)
- `MODEL_DIR` — documented as dead config (prediction stack removed)

---

## [0.1.0] - 2024-XX-XX

### Added
- Initial sandbox platform with Docker-based provisioning
- GitHub OAuth + JWT auth
- Runtime detection (Node, Python, Go, Rust, Java)
- Git integration (clone, branch, commit, push)
- Monaco editor + xterm terminal
- Share links with roles/usage limits
- Change request workflow
- Embedded migrations (000001-000008)
- Postgres + NATS + Redis + Docker
- Next.js 15 + React 18 frontend