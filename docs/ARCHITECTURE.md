# Architecture & Trust Boundaries

## Overview

Berth is a single-host ephemeral sandbox platform for research on predictive pre-warming and gVisor-based isolation.

## Trust Boundaries

```
┌─────────────────────────────────────────────────────────────┐
│  UNTRUSTED ZONE   │  GitHub OAuth, User Browser             │
├─────────────────────────────────────────────────────────────┤
│  CONTROL PLANE    │  Envoy API GW → Go API                                │
│  (Trusted)        │  PostgreSQL, Redis, NATS (Job Orchestration), MinIO   │
├─────────────────────────────────────────────────────────────┤
│  PREDICTION SVC   │  Python gRPC (localhost only)           │
│  (Semi-Trusted)   │  ONNX Runtime, no network egress         │
├─────────────────────────────────────────────────────────────┤
│  DATA PLANE       │  containerd + runc (v2)               │
│  (Untrusted User  │  Rootless containerd for local dev    │
│   Code Executes)  │  Per-sandbox overlayfs / bind mounts  │
│                   │  Host networking (in rootless mode)   │
└─────────────────────────────────────────────────────────────┘
```

## Key Invariants

1. **No host root access.** The backend never mounts `/var/run/docker.sock`.
2. **No bind mounts from host to sandbox.** Filesystem access is via 9P/virtiofs or overlayfs layers. *(Note: Currently using bind mounts for Phase 1).*
3. **Prediction service is localhost-only.** It has no external network access.
4. **All inter-service communication is mTLS** (SPIFFE/SPIRE in production; dev uses self-signed).
5. **Audit logs are append-only.** Stored in MinIO with object lock.
6. **Resource Limits (cgroups).** Cgroup limits (PIDs, Memory, CPU) are strictly enforced. *(Note: Explicitly disabled in rootless mode due to permission constraints. Run in privileged mode for evaluation).*

## Data Flow

1. User clicks "Create Sandbox" in Next.js frontend.
2. Frontend POSTs `/api/environments` with `gitUrl`.
3. API Gateway (Envoy) validates JWT, forwards to Go API.
4. Go API calls Prediction Service (gRPC) to classify repo.
5. API publishes a sandbox creation event to NATS JetStream (`berth.sandbox.create`).
6. Worker pulls job from NATS, provisions the warm container or creates a new one, and restores host-side dependency layer cache.
7. containerd + runsc starts sandbox with bind mount for live editing (Phase 1).
8. File edits flow: Monaco → Yjs → WebSocket → NATS → Bind Mount → sandbox.
9. Git operations run inside the sandbox via exec (gVisor).

## Network Segmentation

- By default, each sandbox gets a `/30` subnet and Cilium L3/L4 policies block inter-sandbox traffic.
- *(Note: In local rootless dev mode, containers use host networking to bypass restricted user namespaces, so network segmentation and port mapping are not isolated.)*
- Only the API Gateway can reach the backend.
- Only the backend can reach the prediction service (localhost:50051).
