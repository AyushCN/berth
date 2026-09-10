# Project Status Inventory

This document serves as the brutal, single-source-of-truth inventory for the Berth platform. 

## ✅ Exists and Works
- **Container Lifecycle:** Fast `CreateSandbox`, `StartSandbox`, `StopSandbox`, `DeleteSandbox`, and `Exec` via containerd v2 API. Rootless execution is fully functional for arbitrary commands (e.g., `npm install`).
- **Warm Pool:** Fully functional. Tracks container dirty states, deletes dirty containers, and maintains a baseline.
- **Dependency Caching:** Fast host-side layer caching via lockfile hashing (package-lock.json / go.sum) implemented in worker. Rootless IO pipes and `cio.WithFIFODir` properly route dependency manager logs.
- **Job Orchestration:** NATS JetStream implemented for event-driven sandbox assignment with a 10s fallback loop.
- **OCI Spec Hardening:** PID/Mount namespaces, dropped capabilities, Seccomp profiles, and PIDs cgroup limits.
- **Networking:** Utilizes **host networking** mapped natively into the rootless containers (via `sysfs` host bind mounts) to ensure completely unimpeded external access and port mapping, completely bypassing previous bridge/netlink errors.
- **Local Dev Loop:** Rootless containerd setup script (`scripts/setup-rootless.sh`). Runs via standard `runc.v2` (temporarily downgraded from gVisor/runsc due to rootless incompatibility).
- **Benchmarking:** EDEBench test harness runs to completion against 50 parallel sandboxes.
- **Database/Redis:** Initialized via Clean Architecture with `sqlc` and `pgxpool`.
- **API Business Logic (Phase 2):** Fully implemented Usecases for Auth, Sandbox, and File operations. HTTP handlers are fully wired, tested, and working end-to-end.
## 🟡 Partial / Stubbed
- **Frontend UI:** Next.js project is partially wired. GitHub OAuth flow is functional, Profile page is implemented, and Terminal WebSocket UI is active. File Explorer and Monaco Editor are pending.

## ❌ Missing (Vaporware)
- **Prediction Service:** No ML models (XGBoost/ONNX), no feature extraction, no Python service.
- **Real-Time Sync (CRDT):** No Yjs operational transforms, no WebSocket event bus, no Monaco integration.
- **File Explorer / Code Editor:** The frontend does not yet support editing files inside the sandbox.
