# Berth

Ephemeral development sandboxes with containerd, repository-based runtime detection, and dependency caching.

Berth is an early-stage, single-host research prototype. Its current rootless development setup uses `runc.v2` and host networking. gVisor support is not currently the default working configuration, and this setup is not suitable for exposing untrusted workloads as a multi-tenant public service.

## Current capabilities

- Go API and worker provision sandboxes through NATS.
- The worker clones GitHub repositories and detects Node.js, Python, Go, or Rust from repository files using simple rules.
- containerd manages sandbox containers; exact-image warm containers are reused for Node.js, Python, and Go when available. Host-side dependency caching also reduces repeat setup work.
- The sandbox UI connects the file tree, editor save, terminal WebSocket, status polling, and preview link to the backend APIs.
- GitHub OAuth tokens are encrypted at rest and used for owner-authorized pushes to a `berth/<sandbox-id>` branch.
- Sandboxes receive a 24-hour expiry; the worker periodically removes expired containers and workspaces. Stop/delete requests are sent to the worker over NATS.
- The local development runtime defaults to `runc`; preview requests are proxied through the API to a host-networked port.

## Deferred

- Collaborative editing is out of demo scope. The preview/networking path is not multi-tenant safe.
- Benchmark and paper work is deferred; current checked-in measurements are research artifacts, not a published performance claim.

## Quick start

The development path requires Linux (bare metal or VM), Go, Node.js, Docker Compose, and a separately configured containerd runtime. Follow [docs/QUICKSTART.md](docs/QUICKSTART.md) for local or single-host setup. The preview path is for trusted single-host use; it is not multi-tenant safe.

```bash
make up
make migrate-up
cd backend && go run ./cmd/api
```

Run the worker separately with `MODE=worker` and the same backend configuration. Required environment variables and production notes are in the quickstart.

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
