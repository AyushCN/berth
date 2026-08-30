#!/usr/bin/env bash
set -euo pipefail

# Build the worker (as normal user)
cd backend
go build -o bin/berth-worker ./cmd/worker
cd ..

# Run with sudo
echo "Starting Berth Worker..."
sudo -E ENCRYPTION_KEY=0000000000000000000000000000000000000000000000000000000000000000 \
        JWT_SECRET=supersecret \
        BERTH_RUNTIME=runsc \
        CONTAINERD_SOCK=/run/containerd/containerd.sock \
        ./backend/bin/berth-worker
