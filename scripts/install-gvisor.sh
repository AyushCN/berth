#!/usr/bin/env bash
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}Berth: Installing containerd + gVisor${NC}"

# --- Install containerd ---
echo -e "${YELLOW}Installing containerd...${NC}"
if ! command -v containerd &> /dev/null; then
    sudo apt-get update
    sudo apt-get install -y containerd
    echo "  containerd installed"
else
    echo "  containerd already installed"
fi

# --- Install runsc (gVisor) ---
echo -e "${YELLOW}Installing runsc (gVisor)...${NC}"
RUNSC_URL=$(curl -s https://api.github.com/repos/google/gvisor/releases/latest | grep browser_download_url | grep runsc | head -n1 | cut -d'"' -f4)
if [ -z "$RUNSC_URL" ]; then
    echo -e "${RED}Failed to fetch latest runsc release URL${NC}"
    exit 1
fi

curl -L -o /tmp/runsc "$RUNSC_URL"
sudo chmod +x /tmp/runsc
sudo mv /tmp/runsc /usr/local/bin/runsc
sudo /usr/local/bin/runsc install
echo "  runsc installed"

# --- Configure containerd runtime class ---
echo -e "${YELLOW}Configuring containerd runtime class...${NC}"
CONTAINERD_CONFIG="/etc/containerd/config.toml"

if [ ! -f "$CONTAINERD_CONFIG" ]; then
    sudo mkdir -p /etc/containerd
    sudo containerd config default | sudo tee "$CONTAINERD_CONFIG" > /dev/null
fi

# Add runsc runtime if not present
if ! grep -q "io.containerd.runsc.v1" "$CONTAINERD_CONFIG"; then
    sudo tee -a "$CONTAINERD_CONFIG" > /dev/null <<'EOF'

[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runsc]
  runtime_type = "io.containerd.runsc.v1"
EOF
    echo "  runsc runtime class added to containerd"
else
    echo "  runsc runtime class already configured"
fi

# --- Restart containerd ---
echo -e "${YELLOW}Restarting containerd...${NC}"
sudo systemctl restart containerd
sudo systemctl enable containerd

# --- Verify ---
echo -e "${YELLOW}Verifying installation...${NC}"
if containerd --version; then
    echo -e "  ${GREEN}✓${NC} containerd OK"
else
    echo -e "  ${RED}✗${NC} containerd failed"
    exit 1
fi

if runsc --version; then
    echo -e "  ${GREEN}✓${NC} runsc OK"
else
    echo -e "  ${RED}✗${NC} runsc failed"
    exit 1
fi

if sudo ctr version &> /dev/null; then
    echo -e "  ${GREEN}✓${NC} ctr CLI OK"
else
    echo -e "  ${YELLOW}!${NC} ctr requires root (expected)"
fi

echo -e "${GREEN}Installation complete!${NC}"
echo ""
echo "Next steps:"
echo "  1. Run 'sudo ctr run --rm --runtime io.containerd.runsc.v1 docker.io/library/alpine:latest test echo hello'"
echo "  2. Run './scripts/setup-rootless.sh' for rootless dev mode"
