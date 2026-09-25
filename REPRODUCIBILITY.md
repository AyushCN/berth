# Benchmark Notes

The repository includes a small Go benchmark command at `backend/cmd/bench`. The previous EDEBench dataset and paper materials have been removed/deferred; no current benchmark result should be read as a validated cold-versus-warm comparison or a performance claim.

## Requirements

- Linux host with containerd configured for the runtime under evaluation (`runc` is the local development default; `runsc` requires a separate working setup).
- Go toolchain matching `backend/go.mod`.
- Docker Compose for the local PostgreSQL, Redis, and NATS services.

## Local control-plane setup

```bash
make dev
make migrate-up
make build
```

Start the worker and API with the environment settings described in the README. Run `./backend/bin/bench -c 10 -n 30` only against a configured runtime and treat its output as a local measurement until a repeatable methodology and results are documented.

## Control-plane-only review

The API can be run in mock containerd mode for control-plane review on systems without a local containerd runtime:

```bash
MOCK_CONTAINERD=1 MODE=api JWT_SECRET=dev go run ./cmd/api
```
