
agents_md = """# Berth — Predictive Ephemeral Sandbox Platform

> A research-grade, single-host ephemeral development environment platform with predictive pre-warming, gVisor isolation, and CRDT-based collaboration. Built for reproducible systems research and IEEE publication.

---

## 1. Vision & Research Claim

**Berth** is an ephemeral sandbox platform that predicts runtime configurations from repository metadata, enabling a warm pool of pre-provisioned gVisor microVMs. It achieves **<10s cold-start p95** (vs. 30–60s for existing platforms) with **zero privileged daemon access**, while supporting real-time collaborative editing via CRDTs.

**Core Research Contributions:**
1. **Predictive Pre-warming Engine**: A gradient-boosted classifier (XGBoost) that extracts semantic features from repository manifests to predict runtime, port, and database requirements before container creation.
2. **gVisor-based Isolation Model**: True multi-tenant sandboxing using `runsc` + containerd, eliminating the Docker socket root-equivalent attack surface.
3. **CRDT-Aware File Sync**: Real-time collaborative editing with Yjs operational transforms, backed by tree-sitter AST-aware merge resolution.
4. **Reproducible Evaluation Framework**: Open-source benchmark suite (EDEBench) with statistical rigor for systems research.

**Target Venue:** IEEE CLOUD, IEEE IC2E, or ACM/USENIX Middleware.

---

## 2. Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                           USER LAYER                                │
│  Next.js 15 (App Router) + Monaco + Yjs + xterm.js + WebSocket   │
└─────────────────────────────────────────────────────────────────────┘
                                    │
┌─────────────────────────────────────────────────────────────────────┐
│                        CONTROL PLANE                                │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────┐  │
│  │   API GW     │  │   Auth Svc   │  │  Orchestrator (Temporal) │  │
│  │  (Envoy)     │  │  (OAuth2+    │  │  - Build workflows       │  │
│  │              │  │   OIDC+OPA)  │  │  - Predictive scheduling  │  │
│  └──────────────┘  └──────────────┘  └──────────────────────────┘  │
│         │                   │                   │                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────┐  │
│  │  Metadata DB │  │   Event Bus  │  │   Prediction Service     │  │
│  │ (PostgreSQL) │  │ (NATS JetSt) │  │  - Feature extractor     │  │
│  │  + sqlc      │  │              │  │  - XGBoost/ONNX Runtime  │  │
│  └──────────────┘  └──────────────┘  └──────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
                                    │
┌─────────────────────────────────────────────────────────────────────┐
│                          DATA PLANE                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────┐  │
│  │  Sandbox     │  │  Sidecar DB  │  │  Object Store (MinIO)    │  │
│  │ (gVisor/     │  │ (PostgreSQL  │  │  - Layer snapshots       │  │
│  │  runsc)      │  │  per-org)    │  │  - Dependency caches     │  │
│  │  + overlayfs │  │  + Redis     │  │  - Audit log blobs       │  │
│  └──────────────┘  └──────────────┘  └──────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 3. Technology Stack

| Layer | Technology | Rationale |
|-------|-----------|-----------|
| **Frontend** | Next.js 15 (App Router), React 19, Tailwind CSS v4, Monaco Editor, Yjs, xterm.js | App Router for SSR/Streaming; Yjs for CRDTs; Monaco for IDE |
| **State Management** | Zustand + TanStack Query (React Query) | Lightweight, typed, server-state sync |
| **Backend API** | Go 1.23, Gin (or stdlib `net/http` + chi), sqlc, golang-migrate | Type-safe SQL, versioned migrations, no ORM magic |
| **Async Workflows** | Temporal (or Cadence) | Durable execution, saga patterns, retry logic, observability |
| **Isolation** | containerd + gVisor (`runsc`) | Industry standard CRI; true sandboxing without host root |
| **Filesystem** | overlayfs + 9P/virtiofs | Layered storage for cheap forks/snapshots |
| **Message Bus** | NATS JetStream | Guaranteed delivery, pub/sub, streams, lightweight |
| **Database** | PostgreSQL 16 (metadata), Redis 7 (sessions/rate-limit) | sqlc-compatible, battle-tested |
| **Object Store** | MinIO (S3-compatible) | Layer snapshots, dependency tarballs, audit artifacts |
| **Auth** | OAuth 2.1 + PKCE + Open Policy Agent (OPA) | Modern standard, fine-grained RBAC policies |
| **Crypto** | AES-256-GCM (via `golang.org/x/crypto`) or NaCl `secretbox` | Authenticated encryption, no CFB nonsense |
| **Secrets** | HashiCorp Vault (dev mode) or AWS Secrets Manager | Dynamic credentials, rotation, audit trail |
| **Observability** | OpenTelemetry + Prometheus + Jaeger | Distributed tracing, metrics, log correlation |
| **ML/AI** | Python 3.11 service, XGBoost, ONNX Runtime, tree-sitter | Fast inference, cross-platform model deployment |
| **Networking** | Cilium (eBPF) | Per-sandbox network policies, mTLS, observability |
| **Service Mesh** | SPIFFE/SPIRE | Workload identity, mTLS between all control plane services |

---

## 4. Security Model (STRIDE)

| Threat | Component | Mitigation | Verification |
|--------|-----------|------------|--------------|
| **Spoofing** | GitHub OAuth | OAuth 2.1 + PKCE + cryptographically random state | Penetration test |
| **Tampering** | User code execution | gVisor seccomp-bpf + user namespace + no host mounts | syzkaller fuzzing |
| **Repudiation** | Audit logs | Append-only signed audit log (separate table + MinIO) | Integrity checksums |
| **Info Disclosure** | Secrets in DB | Vault envelope encryption + AES-256-GCM | Key rotation policy |
| **DoS** | Resource exhaustion | cgroup v2 limits + token-bucket rate limiting + warm pool quotas | Load test |
| **Elevation** | Container escape | **No Docker socket**; containerd CRI + gVisor ptrace sandbox | Escape attempt benchmark |

### Isolation Proof Requirements
- Run **syzkaller** inside `runsc` sandboxes for 48 hours; measure host compromise rate (target: 0).
- Measure syscall latency overhead: `runsc` vs `runc` vs native (target: <20% overhead for dev workloads).
- Network segmentation: each sandbox gets /30 subnet; Cilium eBPF policies block inter-sandbox traffic except via explicit service mesh.

---

## 5. Predictive Pre-warming Engine (AI/Data Science)

### 5.1 Problem
Cold starts dominate user experience (30–120s). Existing platforms pull base images and install dependencies *after* the user clicks "Create."

### 5.2 Solution
Predict the runtime configuration from repository metadata *before* container creation, enabling a **warm pool** of pre-provisioned sandboxes.

### 5.3 Feature Extraction
From any GitHub repo URL, extract:
```
- manifest_features:
    - package.json: dependencies count, devDependencies count, has "next", has "express", has "typescript"
    - go.mod: module path, go version, dependency count
    - requirements.txt: line count, has "fastapi", has "django", has "flask"
    - pyproject.toml: build-system, dependencies
    - Cargo.toml: has "actix", has "tokio"
    - Dockerfile: base image, exposed ports
    - docker-compose.yml: services count, DB services
    - Procfile: process types

- structural_features:
    - file extension histogram (.js, .ts, .go, .py, .rs)
    - entry point detection (main.go, index.js, app.py, manage.py)
    - test file ratio
    - config file presence (tsconfig.json, next.config.js, .env.example)

- historical_features (if available):
    - avg build time from previous runs
    - peak memory usage from previous runs
    - dependency resolution time
```

### 5.4 Model Architecture
```
Input: Feature vector (128 dims)
       │
       ▼
┌─────────────────┐
│  XGBoost Classifier │  → Runtime class (Node/Python/Go/Rust/Other)
│  (4 classes)      │
└─────────────────┘
       │
       ▼
┌─────────────────┐
│  XGBoost Regressor  │  → Predicted port (1–65535)
│  (quantile regression)│
└─────────────────┘
       │
       ▼
┌─────────────────┐
│  Binary Classifier  │  → Needs DB sidecar? (yes/no)
│  (Logistic/XGB)   │
└─────────────────┘
       │
       ▼
Output: { runtime, port, needs_db, confidence }
```

### 5.5 Warm Pool Manager
```go
type WarmPool struct {
    // Pre-created sandboxes waiting for assignment
    Available map[RuntimeProfile][]*Sandbox
    
    // When a user creates an environment:
    // 1. Predict profile from repo
    // 2. Check Available[profile] for a ready sandbox
    // 3. If match: assign (cold start ≈ 0s)
    // 4. If no match: create from scratch (fallback)
    // 5. Replenish pool asynchronously
}
```

### 5.6 Training Data
- Collect 1,000+ public GitHub repos across languages.
- Manually label runtime, port, and DB need.
- Extract features via automated parser (Go + tree-sitter).
- Store dataset in Parquet format; version with DVC.

### 5.7 Evaluation Metrics
| Metric | Target |
|--------|--------|
| Runtime prediction accuracy (F1-macro) | ≥ 0.94 |
| Port prediction MAE | ≤ 200 |
| DB need prediction F1 | ≥ 0.91 |
| Warm pool hit rate | ≥ 65% |
| False-positive pre-warm cost | ≤ 16MB RAM |
| Cold-start p95 (with prediction) | ≤ 10s |
| Cold-start p95 (baseline) | ≥ 30s |

---

## 6. CRDT-Based Collaboration

### 6.1 Real-time Sync
- **Yjs** handles operational transforms for text documents.
- Each file is a Yjs `Y.Text` document.
- Updates are persisted as binary blobs in PostgreSQL + broadcast via NATS.

### 6.2 AST-Aware Merge
- When two users edit the same function, tree-sitter parses the AST.
- Conflicting symbol renames are detected at the AST node level.
- The backend suggests resolutions (e.g., "User A renamed `foo` to `bar`; User B added parameter `baz`").

### 6.3 Evaluation
- Measure sync latency for 2, 5, 10 concurrent editors.
- Compare conflict rate: raw text OT vs AST-aware OT.

---

## 7. Project Structure

```
berth/
├── .github/
│   └── workflows/
│       └── ci.yml              # Go 1.23, Node 20, Python 3.11
├── docs/
│   ├── ARCHITECTURE.md         # System architecture
│   ├── THREAT_MODEL.md         # STRIDE analysis
│   ├── EVALUATION.md           # Benchmark protocol
│   ├── PAPER/                  # LaTeX IEEEtran source
│   │   ├── main.tex
│   │   ├── figures/
│   │   └── tables/
│   └── EDEBench/               # Dataset & benchmark harness
│       ├── dataset/
│       ├── features/
│       └── models/
├── frontend/
│   ├── app/                    # Next.js 15 App Router
│   ├── components/             # Atomic design: atoms/molecules/organisms
│   ├── hooks/                  # Custom React hooks
│   ├── lib/                    # Yjs provider, auth, fetch
│   ├── stores/                 # Zustand stores
│   └── package.json
├── backend/
│   ├── cmd/
│   │   ├── api/                # HTTP API server
│   │   ├── worker/             # Temporal worker
│   │   └── predictor/          # gRPC prediction service (calls Python)
│   ├── internal/
│   │   ├── domain/             # Entities, interfaces (Clean Architecture)
│   │   ├── usecase/            # Business logic
│   │   ├── repository/         # DB ops (sqlc-generated)
│   │   ├── delivery/           # HTTP/gRPC handlers
│   │   └── infrastructure/     # Docker client, NATS, Vault
│   ├── pkg/
│   │   ├── crypto/             # AES-256-GCM
│   │   ├── validator/          # Input validation
│   │   └── logger/             # Structured JSON logging
│   ├── migrations/             # golang-migrate SQL files
│   ├── sqlc/                   # sqlc queries and generated code
│   └── go.mod
├── ml/
│   ├── features/               # Feature extraction (Python)
│   ├── models/                 # XGBoost training notebooks
│   ├── onnx/                   # Exported ONNX models
│   └── requirements.txt
├── infra/
│   ├── containerd/             # containerd + runsc config
│   ├── cilium/                 # Cilium network policies
│   ├── temporal/               # Temporal server config
│   └── docker-compose.yml      # Local dev stack
├── scripts/
│   ├── benchmark/              # Benchmark harness (Go)
│   ├── fuzz/                   # syzkaller configs
│   └── setup.sh                # One-command dev setup
└── README.md
```

---

## 8. Development Phases

### Phase 0: Foundation (Weeks 1–2)
- [ ] Initialize repo with `berth` name.
- [ ] Set up CI: Go 1.23, Node 20, Python 3.11, lint, test, build.
- [ ] Install containerd + gVisor (`runsc`) on dev machine.
- [ ] Verify `runsc` can run a simple container without Docker socket.
- [ ] Set up PostgreSQL 16 + Redis 7 + NATS + MinIO via Docker Compose.
- [ ] Generate sqlc schema and first migration.

### Phase 1: Core Isolation (Weeks 3–4)
- [ ] Implement containerd CRI client in Go (no shell-outs).
- [ ] Create sandbox lifecycle: create → start → stop → destroy.
- [ ] Implement overlayfs layering: base image + dependency layer + writable upper.
- [ ] Implement snapshot/restore for environment forks.
- [ ] Write syzkaller fuzzing config; run 24-hour test.
- [ ] Measure syscall overhead: `runsc` vs `runc`.

### Phase 2: API & Auth (Weeks 5–6)
- [ ] Implement OAuth 2.1 + PKCE for GitHub.
- [ ] Implement OPA policies for RBAC (OWNER/ADMIN/COLLAB/VIEWER).
- [ ] Implement environment CRUD API with Clean Architecture.
- [ ] Implement file operations with FUSE/9P (not bind-mount).
- [ ] Implement WebSocket real-time sync (NATS-backed).
- [ ] Add AES-256-GCM encryption for GitHub tokens.

### Phase 3: Frontend (Weeks 7–8)
- [ ] Next.js 15 project with App Router.
- [ ] Monaco Editor with LSP support.
- [ ] File tree with Yjs CRDT integration.
- [ ] Terminal (xterm.js) connected to sandbox via WebSocket.
- [ ] Git panel: status, branch, commit, push, pull.
- [ ] Dashboard: environments, projects, collaborators.

### Phase 4: Prediction Engine (Weeks 9–10)
- [ ] Build feature extractor for 5 languages.
- [ ] Scrape 1,000 repos; label dataset.
- [ ] Train XGBoost models; export to ONNX.
- [ ] Build Go gRPC service that calls ONNX Runtime.
- [ ] Implement warm pool manager in orchestrator.
- [ ] A/B test: prediction on vs off.

### Phase 5: Evaluation (Weeks 11–12)
- [ ] Build benchmark harness: cold start, warm reload, concurrent users.
- [ ] Run against Gitpod and GitHub Codespaces (free tier baselines).
- [ ] Generate CDF plots, confidence intervals, statistical tests.
- [ ] Run security evaluation: fuzzing, escape attempts, resource exhaustion.
- [ ] Package EDEBench dataset with DOI (Zenodo).

### Phase 6: Paper & Release (Weeks 13–14)
- [ ] Write IEEE CLOUD paper (8 pages, IEEEtran).
- [ ] Create artifact repository with reproducible build.
- [ ] Submit to target venue.
- [ ] Publish open-source with comprehensive README.

---

## 9. Evaluation Methodology

### 9.1 Benchmarks
| Experiment | Baselines | Metrics | Runs |
|------------|-----------|---------|------|
| Cold start | Gitpod, Codespaces, native Docker | Time to first HTTP 200 | 30+ per config |
| Warm reload | Same as above | p50/p95/p99 save→ready | 100+ |
| Isolation overhead | Native Docker, Kata, gVisor | CPU%, memory, syscall latency | 50+ |
| Concurrent users | 1, 10, 50, 100 sandboxes | Throughput, tail latency, error rate | 10+ |
| Prediction accuracy | Manual config (oracle) | F1, MAE, hit rate | 1,000 repos |
| File sync | rsync, Unison, native bind-mount | Bytes transferred, sync latency | 50+ |

### 9.2 Statistical Rigor
- Report **95% confidence intervals** for all metrics.
- Use **Mann-Whitney U test** to prove statistical significance.
- Plot **Cumulative Distribution Functions (CDFs)** for latency.
- Host raw data and analysis scripts in **Zenodo/Figshare** repository.

### 9.3 Reproducibility Checklist
- [ ] All code open-source (GitHub).
- [ ] All configs in repo (containerd, Cilium, Temporal).
- [ ] Dataset versioned with DVC.
- [ ] One-command setup script (`scripts/setup.sh`).
- [ ] Artifact Evaluation badge from ACM/IEEE.

---

## 10. Non-Goals (What Berth Is NOT)

To prevent scope creep and architectural cowardice:

- ❌ **Not a multi-region cloud platform.** Single-host focus; distributed orchestration is future work.
- ❌ **Not a production PaaS.** No SLA guarantees; research prototype.
- ❌ **Not a replacement for GitHub Codespaces.** It is a research vehicle for predictive scheduling + isolation.
- ❌ **Not an LLM code generator.** No OpenAI API integration. The AI is systems-level (prediction, scheduling, anomaly detection).
- ❌ **Not backward-compatible with api-sandbox.** Zero code reuse. Fresh start.

---

## 11. Key Principles

1. **Security First:** If it requires host root, it doesn't ship.
2. **Measure Everything:** No feature without a benchmark. No claim without a p-value.
3. **Clean Architecture:** No god files. No global `db.DB`. Interfaces everywhere.
4. **Type Safety:** sqlc over GORM. Strict TypeScript. No `any`.
5. **Reproducibility:** One script to run the whole stack. One script to reproduce every graph.
6. **Honesty:** Document limitations. No "best-effort" hand-waving.

---

## 12. Resources & References

### Papers to Read
- [gVisor] "Sandboxing Containerized Applications with gVisor" (Google, 2018)
- [Firecracker] "Firecracker: Lightweight Virtualization for Serverless Applications" (AWS, 2020)
- [Cilium] "Cilium: BPF and XDP for Containers" (LinuxCon, 2016)
- [Temporal] "Cadence: Microservice Orchestration at Uber" (Uber, 2017)
- [Yjs] "Yjs: A Framework for Near Real-Time P2P Shared Editing" (2019)
- [XGBoost] "XGBoost: A Scalable Tree Boosting System" (KDD, 2016)
- [STRIDE] "Threat Modeling: Designing for Security" (Shostack, 2014)

### Tools
- [gVisor](https://gvisor.dev/)
- [containerd](https://containerd.io/)
- [Temporal](https://temporal.io/)
- [NATS](https://nats.io/)
- [Cilium](https://cilium.io/)
- [sqlc](https://sqlc.dev/)
- [ONNX Runtime](https://onnxruntime.ai/)
- [Yjs](https://docs.yjs.dev/)
- [tree-sitter](https://tree-sitter.github.io/)
- [OpenTelemetry](https://opentelemetry.io/)

---

*Last updated: 2026-08-22*
*Version: 1.0 — Foundation Draft*
*Target: IEEE CLOUD 2027 / IC2E 2027*
"""

with open('/mnt/agents/output/AGENTS.md', 'w') as f:
    f.write(agents_md)

print("AGENTS.md created successfully.")
print(f"File size: {len(agents_md)} bytes")



| Layer                     | Language                 | Framework                             |
| ------------------------- | ------------------------ | ------------------------------------- |
| **Backend API & Workers** | **Go 1.23**              | Gin (or stdlib), sqlc, Temporal SDK   |
| **Frontend**              | **Node.js / TypeScript** | Next.js 15, React 19                  |
| **ML Prediction Service** | **Python 3.11**          | FastAPI (gRPC), XGBoost, ONNX Runtime |
