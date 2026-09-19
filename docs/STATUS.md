# Project Status Inventory

This document serves as the brutal, single-source-of-truth inventory for the Berth platform. 

## ✅ Exists and Works
- **Container Lifecycle:** Fast `CreateSandbox`, `StartSandbox`, `StopSandbox`, `DeleteSandbox`, and `Exec` via containerd v2 API. Rootless execution is fully functional for arbitrary commands (e.g., `npm install`).
- **Warm Pool:** Fully functional. Tracks container dirty states, deletes dirty containers, and maintains a baseline.
- **Dependency Caching:** Fast host-side dependency caching via `pnpm` store bind mounting (`ExtraMounts`), greatly accelerating Node dependency installations across concurrent environments.
- **Job Orchestration:** NATS JetStream implemented for event-driven sandbox assignment with a 10s fallback loop.
- **OCI Spec Hardening:** PID/Mount namespaces, dropped capabilities, Seccomp profiles, and PIDs cgroup limits.
- **Networking:** Utilizes **host networking** mapped natively into the rootless containers (via `sysfs` host bind mounts) to ensure completely unimpeded external access and port mapping, completely bypassing previous bridge/netlink errors.
- **Local Dev Loop:** Rootless containerd setup script (`scripts/setup-rootless.sh`). Runs via standard `runc.v2` (temporarily downgraded from gVisor/runsc due to rootless incompatibility).
- **Benchmarking:** EDEBench test harness runs to completion against 50 parallel sandboxes.
- **Database/Redis:** Initialized via Clean Architecture with `sqlc` and `pgxpool`.
- **API Business Logic (Phase 2):** Fully implemented Usecases for Auth, Sandbox, and File operations. HTTP handlers are fully wired, tested, and working end-to-end.
## 🟡 Partial / Stubbed
- **Frontend UI:** Next.js project is partially wired. GitHub OAuth flow is functional, Profile page is implemented, and Terminal WebSocket UI is active. The File Explorer and Monaco Editor components exist and compile correctly, but are not yet wired to a live CRDT backend.
- **Prediction Service:** ML model (XGBoost/ONNX) and Python service are scaffolded. Feature extraction integration pending.

## ☁️ Cloud & Security Readiness
- **Cloud Scale:** The codebase has been audited and hardened for multi-node cloud scalability. All `localhost` hardcoding has been stripped from API routing, proxies, CORS, and WebSocket upgrader origins.
- **Security:** Credentials and keys are properly injected via environment variables. The API features a rate limiter that prevents Redis socket exhaustion on aborted requests.

## ❌ Missing (Vaporware)
- **Real-Time Sync (CRDT):** No Yjs operational transforms, no WebSocket event bus for live coding.
