#!/usr/bin/env bash
set -euo pipefail

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}Berth: Setting up rootless containerd${NC}"

LOCAL_BIN="$HOME/.local/bin"
mkdir -p "$LOCAL_BIN"

# Install rootlesskit if missing
if ! command -v rootlesskit &> /dev/null; then
    echo -e "${YELLOW}Installing rootlesskit...${NC}"
    go install github.com/rootless-containers/rootlesskit/v2/cmd/rootlesskit@latest
fi

# Install slirp4netns if missing
if ! command -v slirp4netns &> /dev/null; then
    echo -e "${YELLOW}Installing slirp4netns...${NC}"
    curl -L -o "$LOCAL_BIN/slirp4netns"         https://github.com/rootless-containers/slirp4netns/releases/download/v1.3.1/slirp4netns-x86_64
    chmod +x "$LOCAL_BIN/slirp4netns"
fi

# Install containerd-rootless-setuptool directly from nerdctl repo
if [ ! -f "$LOCAL_BIN/containerd-rootless-setuptool.sh" ]; then
    echo -e "${YELLOW}Installing containerd-rootless-setuptool...${NC}"
    curl -L -o "$LOCAL_BIN/containerd-rootless-setuptool.sh"         https://raw.githubusercontent.com/containerd/nerdctl/v2.0.0/extras/rootless/containerd-rootless-setuptool.sh
    chmod +x "$LOCAL_BIN/containerd-rootless-setuptool.sh"
fi

if [ ! -f "$LOCAL_BIN/containerd-rootless.sh" ]; then
    echo -e "${YELLOW}Installing containerd-rootless.sh...${NC}"
    curl -L -o "$LOCAL_BIN/containerd-rootless.sh"         https://raw.githubusercontent.com/containerd/nerdctl/v2.0.0/extras/rootless/containerd-rootless.sh
    chmod +x "$LOCAL_BIN/containerd-rootless.sh"
fi

# Run setup
export PATH="$(go env GOPATH)/bin:$LOCAL_BIN:${PATH}"
containerd-rootless-setuptool.sh install
containerd-rootless-setuptool.sh install-buildkit

echo -e "${GREEN}Rootless containerd installed!${NC}"
echo ""
echo "Socket path: $XDG_RUNTIME_DIR/containerd/containerd.sock"
echo ""
echo "To use in Berth, set:"
echo "  export CONTAINERD_SOCK=$XDG_RUNTIME_DIR/containerd/containerd.sock"
echo ""
echo "Start the daemon:"
echo "  containerd-rootless-setuptool.sh nsenter"
