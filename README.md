# Berth

Berth is a fast, isolated preview environment platform designed for students, small teams, and individual developers. It provides on-demand, zero-config sandboxes for GitHub repositories without the vendor lock-in or costs of managed cloud providers.

## The Philosophy
Berth is built on one core principle: **every feature must be true when shipped, not aspirational.** There are no mocked dashboards, no fake cloud providers, and no filesystem tricks that don't actually work on a live database. 

## 🚀 Key Features

*   **Zero-Config Deployments:** GitHub push → shallow clone → cached Nixpacks build → container → public URL via Traefik.
*   **Real-Time Logs:** Build and runtime logs streamed over WebSockets with reconnection and backoff.
*   **Encrypted Secrets:** Environment variables and secrets management, encrypted at rest.
*   **Database Addons:** One database addon type (Postgres), correctly isolated per-org network, with secure, randomly generated credentials.
*   **Team Access:** Basic team structure with invite links and a single role tier (owner/member).
*   **Portable Deployment:** A single Go binary + Docker Compose + Traefik. It runs on any Linux box with Docker installed—whether that's an AWS EC2 instance, a $10 DigitalOcean droplet, or a physical server in a room.
*   **Performance Benchmarking:** Built-in instrumentation tracks timing across the clone, detect, and build stages, proving out the speed improvements of shallow clones and BuildKit caching.

## 🛠️ Tech Stack

*   **Orchestrator:** Go + Gin
*   **Queue:** Asynq + Redis
*   **Database:** Postgres + GORM
*   **Container Runtime:** `runc` by default (`runsc` / gVisor opt-in for untrusted workloads)
*   **Build System:** Nixpacks + BuildKit layer caching
*   **Reverse Proxy:** Traefik (in-memory, label-based dynamic routing)
*   **Frontend:** Next.js + Tailwind CSS + xterm.js

## 🏃 Getting Started

### 1. Start Infrastructure Services
The project uses Docker Compose to run PostgreSQL, Redis, and the Traefik proxy.
```bash
cd frontend
docker compose up -d
```

### 2. Run the Go Backend & Worker
```bash
cd backend
go build -o server .
./server
```

### 3. Run the Next.js Frontend
In a new terminal:
```bash
cd frontend
npm run dev
```
Open `http://localhost:3000` in your browser.

### 4. Running the Benchmark Harness
To run the built-in performance benchmark harness against the core Docker engine:
```bash
cd backend
go run ./cmd/benchmark/main.go
```
