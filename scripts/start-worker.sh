#!/usr/bin/env bash
set -euo pipefail

# Build the worker (as normal user)
cd backend
go build -o bin/berth-worker ./cmd/worker
cd ..

# Run with sudo
echo "Starting Berth Worker..."
if [ -z "${JWT_SECRET:-}" ] || [ -z "${ENCRYPTION_KEY:-}" ]; then
  echo "Error: JWT_SECRET and ENCRYPTION_KEY must be set in the environment"
  exit 1
fi

sudo -E ENCRYPTION_KEY="$ENCRYPTION_KEY" \
        JWT_SECRET="$JWT_SECRET" \
        BERTH_RUNTIME=runsc \
        CONTAINERD_SOCK=/run/containerd/containerd.sock \
        ./backend/bin/berth-worker
