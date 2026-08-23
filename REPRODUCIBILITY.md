# Reproducing Berth Benchmarks

## Requirements
- Linux host with containerd + gVisor installed
- Go 1.23, Node.js 20, Python 3.11
- Docker + Docker Compose

## Quick Start
```bash
make dev          # Start Postgres, Redis, NATS, MinIO
make migrate-up   # Run DB migrations
make build        # Compile API + Worker binaries

# Terminal 1: Prediction service
python3 ml/predictor.py 50052

# Terminal 2: Worker
sudo MODE=worker ./backend/bin/berth-worker

# Terminal 3: API
MODE=api JWT_SECRET=dev ./backend/bin/berth-api

# Terminal 4: Benchmark
./backend/bin/bench -c 10 -n 30
```

## Without Containerd (Frontend/API review only)
If you are running on macOS, Windows, or a Linux machine without `containerd`/`runsc`, you can run the API in a mock mode to review the control-plane functionality.

```bash
MOCK_CONTAINERD=1 MODE=api JWT_SECRET=dev go run ./cmd/api
```
