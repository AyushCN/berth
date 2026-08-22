# Architecture & Trust Boundaries

## Overview

Berth is a single-host ephemeral sandbox platform for research on predictive pre-warming and gVisor-based isolation.

## Trust Boundaries

```
┌─────────────────────────────────────────────────────────────┐
│  UNTRUSTED ZONE   │  GitHub OAuth, User Browser             │
├─────────────────────────────────────────────────────────────┤
│  CONTROL PLANE    │  Envoy API GW → Go API → Temporal       │
│  (Trusted)        │  PostgreSQL, Redis, NATS, MinIO       │
├─────────────────────────────────────────────────────────────┤
│  PREDICTION SVC   │  Python gRPC (localhost only)           │
│  (Semi-Trusted)   │  ONNX Runtime, no network egress         │
├─────────────────────────────────────────────────────────────┤
│  DATA PLANE       │  containerd + gVisor (runsc)          │
│  (Untrusted User  │  Rootless containerd for local dev    │
│   Code Executes)  │  Per-sandbox overlayfs + 9P mount       │
│                   │  Cilium eBPF network policies           │
└─────────────────────────────────────────────────────────────┘
```

## Key Invariants

1. **No host root access.** The backend never mounts `/var/run/docker.sock`.
2. **No bind mounts from host to sandbox.** Filesystem access is via 9P/virtiofs or overlayfs layers.
3. **Prediction service is localhost-only.** It has no external network access.
4. **All inter-service communication is mTLS** (SPIFFE/SPIRE in production; dev uses self-signed).
5. **Audit logs are append-only.** Stored in MinIO with object lock.

## Data Flow

1. User clicks "Create Sandbox" in Next.js frontend.
2. Frontend POSTs `/api/environments` with `gitUrl`.
3. API Gateway (Envoy) validates JWT, forwards to Go API.
4. Go API calls Prediction Service (gRPC) to classify repo.
5. Orchestrator (Temporal) schedules build workflow.
6. Worker pulls base image + dependency layer, creates overlayfs.
7. containerd + runsc starts sandbox with 9P mount for live editing.
8. File edits flow: Monaco → Yjs → WebSocket → NATS → 9P → sandbox.
9. Git operations run inside the sandbox via exec (gVisor).

## Network Segmentation

- Each sandbox gets a `/30` subnet.
- Cilium L3/L4 policies block inter-sandbox traffic.
- Only the API Gateway can reach the backend.
- Only the backend can reach the prediction service (localhost:50051).
