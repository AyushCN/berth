#!/usr/bin/env bash
set -euo pipefail

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}Berth: Setting up root containerd with runsc${NC}"

# Check for root
if [ "$EUID" -ne 0 ]; then
  echo -e "${RED}Please run this script with sudo or as root.${NC}"
  exit 1
fi

echo -e "${YELLOW}Configuring /etc/containerd/config.toml for runsc...${NC}"

# Ensure config directory exists
mkdir -p /etc/containerd

# Generate default config if it doesn't exist or is empty
if [ ! -s /etc/containerd/config.toml ] || grep -q 'disabled_plugins = \["cri"\]' /etc/containerd/config.toml; then
  echo -e "${YELLOW}Generating default containerd config...${NC}"
  containerd config default > /etc/containerd/config.toml
fi

# Ensure binaries are available in root PATH
echo -e "${YELLOW}Symlinking runsc binaries to /usr/local/bin...${NC}"
ln -sf /home/swordrookie/.local/bin/runsc /usr/local/bin/runsc
ln -sf /home/swordrookie/.local/bin/containerd-shim-runsc-v1 /usr/local/bin/containerd-shim-runsc-v1
ln -sf /home/swordrookie/.local/bin/runsc-real /usr/local/bin/runsc-real

# Add runsc to containerd if not present
if ! grep -q "io.containerd.runsc.v1" /etc/containerd/config.toml; then
  echo '
[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runsc]
  runtime_type = "io.containerd.runsc.v1"' >> /etc/containerd/config.toml
  echo -e "${GREEN}Added runsc to config.toml${NC}"
else
  echo -e "${YELLOW}runsc already present in config.toml${NC}"
fi

echo -e "${YELLOW}Restarting containerd service...${NC}"
systemctl restart containerd

echo -e "${GREEN}Root containerd with runsc configured!${NC}"
echo ""
echo "Socket path: /run/containerd/containerd.sock"
echo ""
echo "Next steps:"
echo "1. Verify runsc works via containerd:"
echo "   sudo bash scripts/smoke-runsc.sh"
echo "2. Start the Berth worker:"
echo "   sudo -E BERTH_RUNTIME=runsc CONTAINERD_SOCK=/run/containerd/containerd.sock go run ./cmd/worker"
