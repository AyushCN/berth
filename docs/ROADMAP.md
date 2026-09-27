# Roadmap

## Version History

| Version | Date | Focus |
|---------|------|-------|
| v0.1.0 | 2024-Q1 | Foundation - API, Worker, PostgreSQL, NATS |
| v0.2.0 | 2024-Q2 | Containerd Infrastructure - Layer commit, tar export, iptables |
| v0.3.0 | 2024-Q3 | Frontend IDE - Editor, Terminal, Git, Preview |
| v0.4.0 | 2024-Q4 | **Prediction Engine** - ML, ONNX, gRPC, UI |
| v0.5.0 | 2025-Q1 | **Beta** - Production hardening, A/B testing, Model Registry |
| v1.0.0 | 2025-Q2 | **GA** - Multi-node, gVisor, mTLS, Multi-tenant |

---

## v0.5.0 - Beta (2025-Q1)

### Prediction Engine Enhancements
- [ ] **ONNX Runtime Native** - Fix libonnxruntime.so loading in containers
- [ ] **Proper RF/XGBoost** - Replace mock implementations with real algorithms
- [ ] **Feature Importance** - Permutation importance + SHAP values
- [ ] **Model Registry UI** - List, compare, promote, rollback models
- [ ] **A/B Testing Framework** - Canary deployments, traffic splitting
- [ ] **Drift Detection** - Data drift monitoring, retraining triggers
- [ ] **Prediction Explanations** - SHAP waterfall plots in UI

### Production Hardening
- [ ] **gVisor Integration** - runsc as default runtime
- [ ] **CNI Networking** - Per-sandbox network isolation (Cilium/Calico)
- [ ] **TLS Everywhere** - mTLS via SPIFFE/SPIRE, cert-manager
- [ ] **Resource Enforcement** - CPU/memory limits in rootless mode
- [ ] **Idle Tracking** - WebSocket heartbeats + activity API
- [ ] **Warm Pool Intelligence** - Predictive pre-warming based on usage patterns

### Observability
- [ ] **Prometheus Metrics** - RED metrics for all services
- [ ] **Grafana Dashboards** - Sandbox, Build, Prediction, System
- [ ] **Distributed Tracing** - OpenTelemetry + Jaeger
- [ ] **Structured Logging** - JSON logs, correlation IDs
- [ ] **Alerting Rules** - PagerDuty/Slack integration

### Developer Experience
- [ ] **Custom Domains** - `*.preview.berth.dev` + custom CNAME
- [ ] **CLI Tool** - `berth` CLI for local development
- [ ] **VS Code Extension** - Connect to remote sandboxes
- [ ] **GitHub App** - Better OAuth, webhook handling
- [ ] **Template Library** - Starter templates per framework

---

## v1.0.0 - General Availability (2025-Q2)

### Multi-Node Architecture
- [ ] **Control Plane HA** - Raft consensus for API/worker coordination
- [ ] **Worker Pool** - Horizontal scaling, job scheduling
- [ ] **Shared Storage** - Ceph/Longhorn for workspace persistence
- [ ] **Service Mesh** - Istio/Linkerd for mTLS, traffic management

### Multi-Tenancy
- [ ] **Namespace Isolation** - Kubernetes namespaces per tenant
- [ ] **Resource Quotas** - CPU/memory/storage per tenant
- [ ] **Network Policies** - Cilium L3/L7 policies per tenant
- [ ] **Audit Logging** - Immutable audit trail to object storage

### Enterprise Features
- [ ] **SSO/SAML/OIDC** - Enterprise identity providers
- [ ] **RBAC** - Fine-grained permissions (org/project/resource)
- [ ] **Compliance** - SOC2, GDPR data handling
- [ ] **Private Registry** - Harbor integration, image scanning
- [ ] **Custom Runtimes** - User-provided Dockerfiles, Buildpacks

### AI/ML Platform
- [ ] **AutoML** - Automated model selection, hyperparameter tuning
- [ ] **Feature Store** - Shared feature definitions across models
- [ ] **Pipeline Orchestration** - Airflow/Kubeflow integration
- [ ] **Model Monitoring** - Performance, drift, bias detection

---

## v1.1.0+ - Platform Expansion (2025-H2)

### Language & Framework Support
- [ ] **TypeScript/Node** - Full TS support, pnpm workspaces
- [ ] **Python** - Poetry, conda, uv, pip-tools
- [ ] **Go** - Workspaces, private modules
- [ ] **Rust** - Cargo workspaces, private registries
- [ ] **Java/Kotlin** - Maven/Gradle, Spring Boot, Quarkus
- [ ] **.NET** - NuGet, MSBuild
- [ ] **PHP/Composer** - Laravel, Symfony
- [ ] **Ruby/Bundler** - Rails, Sinatra
- [ ] **Elixir/Mix** - Phoenix, Nerves

### Advanced Features
- [ ] **Database Per Sandbox** - Ephemeral Postgres/MySQL/Redis
- [ ] **Service Mesh** - Sidecar injection, mTLS
- [ ] **GitOps** - ArgoCD/Flux integration
- [ ] **Cost Attribution** - Per-sandbox/resource cost tracking
- [ ] **Time Travel** - Sandbox snapshots, point-in-time restore

---

## Milestone Definitions

| Milestone | Criteria |
|-----------|----------|
| **Alpha** | Core provisioning works on single host |
| **Beta** | Production-hardened, observable, A/B testing |
| **GA** | Multi-node, multi-tenant, enterprise-ready |
| **Platform** | Extensible, marketplace, ecosystem |

---

## Success Metrics

| Metric | Beta Target | GA Target |
|--------|-------------|-----------|
| **Sandbox Start Time** | < 30s (warm) | < 10s (warm) |
| **Build Prediction Accuracy** | R² > 0.8 | R² > 0.9 |
| **Cache Hit Rate** | > 60% | > 80% |
| **API Availability** | 99.9% | 99.99% |
| **Sandbox Density** | 50/host | 200/host |
| **Multi-tenant Isolation** | N/A | Full |