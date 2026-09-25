# Project Status

Berth is an early-stage, single-host research prototype. This inventory describes the checked-in implementation, not a production readiness claim.

## Implemented foundations

- Go API and worker with PostgreSQL persistence and NATS-based job orchestration plus fallback polling.
- containerd create/start/stop/exec lifecycle integration. The worker can reuse exact-image warm containers for Node.js, Python, and Go; workspaces are attached before task creation and assigned containers are discarded rather than returned with stale mounts.
- Rule-based post-clone runtime detection for Node.js, Python, Go, and Rust.
- Host-side pnpm store mounting for dependency reuse.
- GitHub OAuth with encrypted token storage, owner-checked Git operations, and OAuth-backed pushes to per-sandbox branches.
- Browser file listing, read/write/delete, editor save, terminal WebSocket, preview link, and sandbox state polling are wired in the UI.
- A 24-hour expiry default and worker cleanup sweep remove expired containers/workspaces. API stop/delete requests reach the worker through NATS.
- An API preview proxy at `/p/<sandbox-id>/` that forwards to the worker's assigned host-network port; this is suitable only for a trusted single-host demo.
- Next.js UI components for authentication, file browsing, editing, and terminal access.

The frontend production build succeeds, but the complete create/edit/terminal/preview flow has not been exercised against a Linux containerd host. Local host-network preview behavior is not an isolated preview gateway.

## Current runtime and security limits

- Rootless development currently uses `runc.v2`; gVisor/runsc is not the default verified runtime.
- Containers use host networking in the rootless setup. The API proxy is not per-sandbox network isolation or a production preview gateway.
- CPU, memory, and PID limits are configured in the OCI spec when the host runs with the required cgroup privileges; enforcement is skipped in rootless mode. Do not treat this prototype as a secure multi-tenant service.
- Deployment targets one host. Multi-node operation, mTLS/SPIFFE, Cilium policies, and production gateway infrastructure are not implemented.

## Work still needed for a usable product

- Run the Linux smoke path and exercise OAuth push, warm hits, stop/delete, and expiry cleanup on a containerd host.
- Harden the preview proxy and eventually replace host-network port routing with isolated networking.
- Add idle activity tracking before claiming idle-timeout behavior; current lifecycle automation is TTL-based.
- Add API/worker systemd unit examples and operational monitoring for a VPS deployment.

## Deferred

- Collaborative editing is out of demo scope. OAuth-backed Git push remains incomplete. Existing benchmark artifacts should be treated as research material, not validated performance claims.
