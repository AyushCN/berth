# Project Status Inventory

This document serves as the brutal, single-source-of-truth inventory for the Berth platform. 

## ✅ Exists and Works
- **Container Lifecycle:** `CreateSandbox`, `StartSandbox`, `StopSandbox`, `DeleteSandbox`, and `Exec` via containerd v2 API.
- **OCI Spec Hardening:** PID/Network/Mount namespaces, dropped capabilities, and Seccomp profiles. (Note: cgroup limits are bypassed in rootless environments).
- **Networking:** Local bridge networking scaffold and port forwarding.
- **Local Dev Loop:** Rootless containerd setup script (`scripts/setup-rootless.sh`). Runs via standard `runc.v2` (temporarily downgraded from gVisor/runsc due to rootless incompatibility).
- **Benchmarking:** EDEBench test harness runs to completion against 50 parallel sandboxes.
- **Database/Redis:** Initialized via Clean Architecture with `sqlc` and `pgxpool`.
- **API Business Logic (Phase 2):** Fully implemented Usecases for Auth, Sandbox, and File operations.

## 🟡 Partial / Stubbed
- **Warm Pool:** Exists in-memory (`warm_pool.go`), but the pre-warming predictive logic is not hooked up.
- **Dependency Caching:** The overlayfs logic exists (`layer.go`), but caching is currently deferred to Phase 2.
- **API Handlers:** HTTP handlers are scaffolded and DevLogin endpoint works, but end-to-end webhook wiring is pending.

## ❌ Missing (Vaporware)
- **Prediction Service:** No ML models (XGBoost/ONNX), no feature extraction, no Python service.
- **Real-Time Sync (CRDT):** No Yjs operational transforms, no WebSocket event bus, no Monaco integration.
- **Frontend UI:** 🟡 Scaffolded. Next.js project exists, not wired to production API yet.
- **End-to-End Loop:** You cannot currently click a button to clone a repository and instantly run it in a sandbox.
