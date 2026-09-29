# Contributing

## Development Workflow

1. **Fork & Clone**
   ```bash
   git clone https://github.com/yourorg/berth.git
   cd berth
   ```

2. **Feature Branch**
   ```bash
   git checkout -b feat/your-feature
   ```

3. **Conventional Commits**
   ```
   <type>(<scope>): <description>

   [optional body]
   ```
   Types: `feat`, `fix`, `refactor`, `docs`, `style`, `test`, `chore`, `ci`

3. **Code Style**
   ```bash
   # Go
   go fmt ./...
   go vet ./...

   # Frontend
   cd frontend && npm run lint
   ```

4. **Tests**
   ```bash
   # Backend
   cd backend && ENCRYPTION_KEY=... go test ./... -count=1

   # Migration tests (needs BERTH_MIGRATION_TEST_DSN)
   BERTH_MIGRATION_TEST_DSN="postgres://..." go test ./migrations/ -count=1

   # Frontend
   cd frontend && npm run lint && npm run build
   ```

4. **Push & PR**
   ```bash
   git push origin feat/your-feature
   # Open PR against main
   ```

## Code Style

- **Go**: `gofmt` + `go vet` (enforced in CI)
- **TypeScript**: `tsc --noEmit` + `next lint` (enforced in CI)
- **SQL**: `sqlc generate` after changing queries

## Architecture Principles

- **Single-host only** — no multi-node complexity
- **Docker only** — no containerd/k8s/gVisor
- **Embedded migrations** — applied on boot, no external runner
- **Pure Go** — CGO_ENABLED=0 (no onnxruntime)
- **Host networking** — containers bind directly on host ports

## Code Review Checklist

- [ ] `go fmt ./...` passes
- [ ] `go vet ./...` passes
- [ ] `cd frontend && npm run lint && npm run build` passes
- [ ] `go test ./...` passes
- [ ] No CGO dependencies added
- [ ] No `make` targets added (no Makefile in root)
- [ ] Migrations follow naming: `0000XX_description.up.sql` + `.down.sql`
- [ ] SQLC regenerates cleanly: `cd backend && sqlc generate`
- [ ] No hardcoded secrets in code
- [ ] README/docs updated if user-facing change

## CI Pipeline

`.github/workflows/ci.yml` runs on every push/PR to `main`:

- **Backend**: gofmt check → go vet → sqlc sync check → unit tests + migration tests
- **Frontend**: npm ci → tsc --noEmit → npm run build

---

## Code of Conduct

Be respectful, inclusive, and constructive. This is a research prototype — feedback and PRs welcome!