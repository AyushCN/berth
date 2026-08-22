# Project Status Inventory

This document serves as the brutal, single-source-of-truth inventory for the Berth platform. 

## ✅ Exists and Works
- **Container Lifecycle:** `CreateSandbox`, `StartSandbox`, `StopSandbox`, `DeleteSandbox`, and `Exec` via containerd v2 API.
- **OCI Spec Hardening:** PID/Network/Mount namespaces, Cgroup limits, dropped capabilities, and Seccomp profiles.
- **Networking:** Local bridge networking scaffold and port forwarding.
- **Local Dev Loop:** Rootless containerd setup script (`scripts/setup-rootless.sh`).
- **Database/Redis:** Initialized via Clean Architecture with `sqlc` and `pgxpool`.

## 🟡 Partial / Stubbed
- **Warm Pool:** Exists in-memory (`warm_pool.go`), but the pre-warming predictive logic is not hooked up.
- **Dependency Caching:** The overlayfs logic exists (`layer.go`), but caching is currently deferred to Phase 2.

## ❌ Missing (Vaporware)
- **Prediction Service:** No ML models (XGBoost/ONNX), no feature extraction, no Python service.
- **Real-Time Sync (CRDT):** No Yjs operational transforms, no WebSocket event bus, no Monaco integration.
- **Auth Flow:** Handlers for GitHub OAuth are `501 Not Implemented`.
- **Frontend UI:** The Next.js 15 application has not been scaffolded.
- **End-to-End Loop:** You cannot currently click a button to clone a repository and instantly run it in a sandbox.
