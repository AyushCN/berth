# Architecture and Trust Boundaries

## Overview

Berth is currently a single-host sandbox prototype. The control plane is a Go API backed by PostgreSQL and NATS. A worker consumes sandbox jobs, clones the repository, detects a runtime from repository files, and asks containerd to create and start a sandbox.

```text
Browser → Next.js frontend → Go API → PostgreSQL
                                  └── NATS → Go worker → containerd → sandbox
```

## Current implementation

- Runtime detection happens after cloning and uses file-based rules in `backend/internal/analyzer`.
- The local configuration defaults to rootless containerd with `runc.v2`; gVisor/runsc is not the default verified path.
- Workspace files are bind-mounted so API edits can reach the running container.
- Rootless local networking uses the host network namespace. Preview ports therefore do not have per-sandbox network isolation.
- NATS carries provisioning and terminal traffic. PostgreSQL stores sandbox and account state.

## Security and deployment limits

This setup is intended for development and research on a single host. It does not currently provide network isolation between sandboxes, a dedicated public preview gateway, verified multi-tenant isolation, production mTLS/SPIFFE, Cilium policy enforcement, or a multi-node control plane. Runtime hardening and resource limits are host/runtime dependent. Do not expose this configuration to untrusted users as a production multi-tenant service.

## Provisioning flow

1. The frontend submits a repository URL to the Go API.
2. The API stores a pending sandbox and publishes a NATS job.
3. The worker validates the URL, clones the repository, and detects the runtime from its files.
4. The worker creates and starts the container, installs dependencies, and starts the detected application command.
5. The worker records the container and preview port. Current local preview behavior relies on host networking and is not a stable public routing solution.

File editing and terminal components exist, but frontend wiring remains incomplete. OAuth-backed Git push and collaborative editing are outside the demo scope.
