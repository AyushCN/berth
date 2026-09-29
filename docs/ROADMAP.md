# Roadmap

## Current State
Single-host sandbox prototype with Docker-based container provisioning, GitHub OAuth, and collaborative IDE features.

---

## Immediate (v0.2)

- [ ] Readiness probe that waits for app to listen before `RUNNING`
- [ ] Complete OAuth-backed Git push to `berth/<id>` branch
- [ ] Idle activity tracking via WebSocket heartbeats
- [ ] Static site readiness (don't mark `CRASHED` for static sites)

## v0.3 — Beta

- [ ] TLS termination (Traefik + cert-manager)
- [ ] systemd units for API/worker
- [ ] Prometheus metrics + Grafana dashboards
- [ ] Readiness probing that handles static sites (no port)
- [ ] Complete OAuth-backed Git push to `berth/<id>` branch

## v0.4 — Production Readiness

- [ ] gVisor/runsc as default runtime
- [ ] Multi-node control plane (Raft consensus)
- [ ] Cilium network policies
- [ ] mTLS/SPIFFE for all service communication
- [ ] Multi-tenant isolation (namespace + cgroups)
- [ ] Custom domain support for previews
- [ ] Collaborative editing (Yjs/Automerge)

## Deferred / Not Planned

- Multi-node clustering
- gVisor as default (rootless runc only for now)
- CNI networking (host networking only)
- Multi-tenant SaaS model
- Custom domain previews
- Collaborative editing
- ONNX/ML predictions (removed in v0.2)
- gRPC prediction service (removed)
- Build planner strategies (removed)
- Warm pool (removed)
- gRPC prediction service (removed)