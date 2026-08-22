#!/usr/bin/env bash
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}Berth Phase 0 Setup${NC}"

# Check OS
if [[ "$OSTYPE" != "linux-gnu"* ]]; then
    echo -e "${RED}ERROR: Berth requires Linux. Detected: $OSTYPE${NC}"
    echo "If on macOS, use a Linux VM (Parallels/UTM/Proxmox)."
    exit 1
fi

# Check Go
echo -e "${YELLOW}Checking Go...${NC}"
if ! command -v go &> /dev/null; then
    echo -e "${RED}Go is not installed. Install Go 1.23 from https://go.dev/dl/${NC}"
    exit 1
fi
GO_VER=$(go version | awk '{print $3}')
echo "  Found: $GO_VER"

# Check Docker
echo -e "${YELLOW}Checking Docker...${NC}"
if ! command -v docker &> /dev/null; then
    echo -e "${RED}Docker is not installed.${NC}"
    exit 1
fi

# Install sqlc
echo -e "${YELLOW}Installing sqlc...${NC}"
if ! command -v sqlc &> /dev/null; then
    go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.27.0
    export PATH="$(go env GOPATH)/bin:$PATH"
    echo "  sqlc installed"
else
    echo "  sqlc already installed"
fi

# Install golang-migrate
echo -e "${YELLOW}Installing golang-migrate...${NC}"
if ! command -v migrate &> /dev/null; then
    go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.17.1
    export PATH="$(go env GOPATH)/bin:$PATH"
    echo "  migrate installed"
else
    echo "  migrate already installed"
fi

# Install Node.js 20 (if not present)
echo -e "${YELLOW}Checking Node.js...${NC}"
if ! command -v node &> /dev/null || [[ $(node -v | cut -d'v' -f2 | cut -d'.' -f1) -lt 20 ]]; then
    echo "  Installing Node.js 20..."
    echo "  Please install Node.js 20+ manually or via NVM."
fi
node -v

# Start infrastructure
echo -e "${YELLOW}Starting infrastructure (Postgres, Redis, NATS, MinIO)...${NC}"
cd infra
docker compose up -d --wait

# Verify health
echo -e "${YELLOW}Verifying services...${NC}"
sleep 3

check_port() {
    if nc -z localhost "$1" 2>/dev/null; then
        echo -e "  ${GREEN}✓${NC} $2 on port $1"
    else
        echo -e "  ${RED}✗${NC} $2 on port $1 (not responding)"
        return 1
    fi
}

check_port 5432 "PostgreSQL"
check_port 6379 "Redis"
check_port 4222 "NATS"
check_port 9000 "MinIO"

cd ../backend
echo -e "${YELLOW}Running sqlc generate...${NC}"
sqlc generate

echo -e "${YELLOW}Running migrations...${NC}"
migrate -path migrations -database "postgres://berth:berth@localhost:5432/berth?sslmode=disable" up

echo -e "${YELLOW}Building backend...${NC}"
go mod tidy
go build -o /tmp/berth-api ./cmd/api

echo -e "${GREEN}Phase 0 setup complete!${NC}"
echo ""
echo "Next steps:"
echo "  1. cd backend && go test ./..."
echo "  2. cd frontend && npm install && npm run dev"
echo "  3. Install containerd + gVisor (Phase 1)"
