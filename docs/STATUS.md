# Project Status

Berth is an early-stage, single-host research prototype. This inventory describes the checked-in implementation, not a production readiness claim.

## Implemented foundations

- Go API and worker with PostgreSQL persistence and NATS-based job orchestration plus fallback polling.
- containerd create/start/stop/exec lifecycle integration. A warm-pool manager exists, but the normal create path does not currently reuse its containers.
- Rule-based post-clone runtime detection for Node.js, Python, Go, and Rust.
- Host-side pnpm store mounting for dependency reuse.
- GitHub OAuth and sandbox/file API paths.
- An API preview proxy at `/p/<sandbox-id>/` that forwards to the worker's assigned host-network port; this is suitable only for a trusted single-host demo.
- Next.js UI components for authentication, file browsing, editing, and terminal access.

These pieces do not yet form a fully verified, one-command end-to-end product flow. In particular, portions of the frontend are not wired to backend APIs and local host-network preview behavior is not an isolated preview gateway.

## Current runtime and security limits

- Rootless development currently uses `runc.v2`; gVisor/runsc is not the default verified runtime.
- Containers use host networking in the rootless setup. The API proxy is not per-sandbox network isolation or a production preview gateway.
- Some OCI hardening is configured, but resource enforcement depends on host/runtime capabilities. Do not treat this prototype as a secure multi-tenant service.
- Deployment targets one host. Multi-node operation, mTLS/SPIFFE, Cilium policies, and production gateway infrastructure are not implemented.

## Work still needed for a usable product

- Complete frontend-to-API wiring and verify the clone, edit, terminal, and preview flow end to end.
- Harden the existing preview proxy and eventually replace host-network port routing with isolated networking.
- Finish Git commit/push with securely handled OAuth credentials.
- Provide a reproducible local deployment and document operational requirements.

## Deferred

- Collaborative editing is out of demo scope. OAuth-backed Git push remains incomplete. Existing benchmark artifacts should be treated as research material, not validated performance claims.
