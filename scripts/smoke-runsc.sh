#!/usr/bin/env bash
set -euo pipefail

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

echo "Pulling alpine image if missing..."
ctr images pull docker.io/library/alpine:latest

echo "Running smoke test with io.containerd.runsc.v1..."
OUTPUT=$(ctr run --rm --runtime io.containerd.runsc.v1 docker.io/library/alpine:latest smoke-test echo ok)

if [ "$OUTPUT" = "ok" ]; then
    echo -e "${GREEN}SUCCESS: runsc is working correctly through containerd!${NC}"
else
    echo -e "${RED}FAILURE: Unexpected output: $OUTPUT${NC}"
    exit 1
fi
