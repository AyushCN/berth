#!/usr/bin/env bash
set -euo pipefail

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}Berth: Setting up rootless containerd${NC}"

# Install rootlesskit and slirp4netns if missing
if ! command -v rootlesskit &> /dev/null; then
    echo -e "${YELLOW}Installing rootlesskit...${NC}"
    go install github.com/rootless-containers/rootlesskit/v2/cmd/rootlesskit@latest
fi

if ! command -v slirp4netns &> /dev/null; then
    echo -e "${YELLOW}Installing slirp4netns...${NC}"
    mkdir -p ~/.local/bin
    curl -L https://github.com/rootless-containers/slirp4netns/releases/download/v1.3.1/slirp4netns-x86_64 -o ~/.local/bin/slirp4netns
    chmod +x ~/.local/bin/slirp4netns
fi

# Install containerd-rootless-setuptool
if ! command -v containerd-rootless-setuptool.sh &> /dev/null; then
    echo -e "${YELLOW}Installing containerd-rootless-setuptool...${NC}"
    mkdir -p ~/.local/bin
    curl -L https://github.com/containerd/nerdctl/releases/download/v2.0.0/nerdctl-2.0.0-linux-amd64.tar.gz | tar xz -C ~/.local/bin/
    chmod +x ~/.local/bin/containerd-rootless-setuptool.sh
    chmod +x ~/.local/bin/containerd-rootless.sh
fi

# Run setup
export PATH="$(go env GOPATH)/bin:$HOME/.local/bin:${PATH}"
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
