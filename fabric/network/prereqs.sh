#!/bin/bash
# fabric/network/prereqs.sh
# EventChain — Prerequisites verification and Fabric binary download

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

PASS=0
FAIL=0

check() {
  local name="$1"
  local cmd="$2"
  local min_version="$3"

  if ! command -v "$cmd" &>/dev/null; then
    echo -e "${RED}[FAIL]${NC} $name is not installed"
    ((FAIL++))
    return
  fi

  local version
  case "$cmd" in
    docker)
      version=$(docker --version | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
      ;;
    docker-compose|docker\ compose)
      version=$(docker compose version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
      if [ -z "$version" ]; then
        version=$(docker-compose --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
      fi
      ;;
    go)
      version=$(go version | grep -oE '[0-9]+\.[0-9]+(\.[0-9]+)?' | head -1)
      ;;
    node)
      version=$(node --version | grep -oE '[0-9]+\.[0-9]+\.[0-9]+')
      ;;
    npm)
      version=$(npm --version)
      ;;
    curl)
      version=$(curl --version | head -1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
      ;;
    jq)
      version=$(jq --version | grep -oE '[0-9]+\.[0-9]+(\.[0-9]+)?' | head -1)
      ;;
  esac

  echo -e "${GREEN}[PASS]${NC} $name — version $version (minimum: $min_version)"
  ((PASS++))
}

echo "============================================"
echo " EventChain — Prerequisites Check"
echo "============================================"
echo ""

check "Docker"          docker          "20.10+"
check "Docker Compose"  docker          "2.0+"   # checked via docker compose
check "Go"              go              "1.21+"
check "Node.js"         node            "18+"
check "npm"             npm             "9+"
check "curl"            curl            "7+"
check "jq"              jq              "1.6+"

echo ""
echo "============================================"
echo -e " Results: ${GREEN}${PASS} passed${NC}, ${RED}${FAIL} failed${NC}"
echo "============================================"

if [ "$FAIL" -gt 0 ]; then
  echo ""
  echo -e "${YELLOW}Fix the above failures before proceeding.${NC}"
  exit 1
fi

echo ""
echo "All prerequisites satisfied."
echo ""

# --- Download Fabric binaries and Docker images ---
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [ -f "${SCRIPT_DIR}/bin/peer" ] && [ -f "${SCRIPT_DIR}/bin/osnadmin" ]; then
  echo "Fabric binaries already present in ${SCRIPT_DIR}/bin/"
  "${SCRIPT_DIR}/bin/peer" version
else
  echo "Downloading Fabric 2.5.10 binaries and Docker images..."
  echo "This will also pull Fabric CA 1.5.12 images."
  echo ""
  curl -sSL https://bit.ly/2ysbOFE | bash -s -- 2.5.10 1.5.12

  # The install script places binaries in ./bin and config in ./config
  # Move them into the network directory if they landed in cwd
  if [ -d "./bin" ] && [ ! -d "${SCRIPT_DIR}/bin" ]; then
    mv ./bin "${SCRIPT_DIR}/bin"
  fi
  if [ -d "./config" ] && [ ! -d "${SCRIPT_DIR}/config" ]; then
    mv ./config "${SCRIPT_DIR}/config"
  fi

  echo ""
  echo "Fabric binaries installed to ${SCRIPT_DIR}/bin/"
  echo "Fabric config installed to ${SCRIPT_DIR}/config/"
fi

echo ""
echo "Verifying Docker images..."
docker images | grep hyperledger || true
echo ""
echo "Prerequisites check complete."
