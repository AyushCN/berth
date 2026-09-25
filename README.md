# Berth

Ephemeral development sandboxes with containerd, repository-based runtime detection, and dependency caching.

Berth is an early-stage, single-host research prototype. Its current rootless development setup uses `runc.v2` and host networking. gVisor support is not currently the default working configuration, and this setup is not suitable for exposing untrusted workloads as a multi-tenant public service.

## Current capabilities

- Go API and worker provision sandboxes through NATS.
- The worker clones GitHub repositories and detects Node.js, Python, Go, or Rust from repository files using simple rules.
- containerd manages sandbox containers; a host-side dependency cache can reduce repeat setup work. Warm-pool reuse is not active in the sandbox creation path yet.
- File APIs, terminal WebSocket plumbing, GitHub OAuth, and frontend editor/file-tree components are present. Frontend integration is still in progress.
- The local development runtime defaults to `runc`; preview requests are proxied through the API to a host-networked port.

## Deferred

- Collaborative editing is out of demo scope. OAuth-backed Git push remains incomplete, and the preview/networking path is not multi-tenant safe.
- Benchmark and paper work is deferred; current checked-in measurements are research artifacts, not a published performance claim.

## Quick start

The development path currently requires Linux (bare metal or VM), Go, Node.js, Docker Compose, and a separately configured containerd runtime. See `scripts/` and `infra/docker-compose.yml` for setup details. The preview path is for trusted single-host demos; it is not multi-tenant safe.

```bash
bash scripts/setup.sh
bash scripts/setup-rootless.sh
make dev
make migrate-up
export CONTAINERD_SOCK="$XDG_RUNTIME_DIR/containerd/containerd.sock"
export JWT_SECRET="dev-secret"
cd backend && go run ./cmd/api
```

Run the worker separately with `MODE=worker` and the same backend configuration. GitHub OAuth credentials are needed for the OAuth flow; development mode can seed a local user.

## Project layout

```text
backend/   Go API, worker, containerd integration, and runtime detector
frontend/  Next.js interface
infra/     Local dependency services
scripts/   Development setup and smoke checks
docs/      Architecture and current project status
```

See [docs/STATUS.md](docs/STATUS.md) and [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for current limitations.

## License

MIT — Research Prototype
