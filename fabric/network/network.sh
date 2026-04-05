#!/bin/bash
# fabric/network/network.sh
# EventChain Fabric network management — main entry point
#
# Commands:
#   ./network.sh up             — Start CAs, generate crypto, start network
#   ./network.sh createChannel  — Create channel and join all peers
#   ./network.sh deployCC       — Deploy a chaincode
#   ./network.sh deployCCs      — Deploy all 4 EventChain chaincodes
#   ./network.sh down           — Tear down everything
#   ./network.sh restart        — down + up + createChannel

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

export PATH="${SCRIPT_DIR}/bin:$PATH"
export FABRIC_CFG_PATH="${SCRIPT_DIR}/configtx"

# Docker compose files
COMPOSE_NET="-f ${SCRIPT_DIR}/docker/docker-compose-net.yaml"
COMPOSE_CA="-f ${SCRIPT_DIR}/docker/docker-compose-ca.yaml"
DOCKER_COMPOSE="docker compose --env-file ${SCRIPT_DIR}/docker/.env"

CHANNEL_NAME="eventchain"
CC_SRC_BASE="${SCRIPT_DIR}/../chaincode"

# Parse global flags
while [[ $# -gt 0 ]]; do
  case "$1" in
    up|createChannel|deployCC|deployCCs|down|restart)
      MODE="$1"
      shift
      break
      ;;
    -h|--help)
      printHelp
      exit 0
      ;;
    *)
      echo "Unknown command: $1"
      echo "Usage: $0 {up|createChannel|deployCC|deployCCs|down|restart}"
      exit 1
      ;;
  esac
done

# Remaining args passed to subcommands
EXTRA_ARGS=("$@")

# ============================================================
# Print help
# ============================================================
printHelp() {
  echo "EventChain Fabric Network"
  echo ""
  echo "Usage: ./network.sh <command> [flags]"
  echo ""
  echo "Commands:"
  echo "  up              Start the Fabric network (CAs + peers + orderer + CouchDB)"
  echo "  createChannel   Create 'eventchain' channel and join all peers"
  echo "  deployCC        Deploy a single chaincode"
  echo "                    -ccn <name>  Chaincode name (required)"
  echo "                    -ccp <path>  Chaincode source path (required)"
  echo "                    -ccl <lang>  Language: go (default)"
  echo "                    -ccv <ver>   Version: 1.0 (default)"
  echo "                    -ccs <seq>   Sequence: 1 (default)"
  echo "  deployCCs       Deploy all 4 EventChain chaincodes"
  echo "  down            Tear down the entire network"
  echo "  restart         Tear down and restart (down + up + createChannel)"
  echo ""
}

# ============================================================
# Tear down
# ============================================================
networkDown() {
  echo ""
  echo "============================================"
  echo " Tearing down EventChain network"
  echo "============================================"

  # Stop all containers
  ${DOCKER_COMPOSE} ${COMPOSE_NET} ${COMPOSE_CA} down --volumes --remove-orphans 2>/dev/null || true

  # Remove chaincode Docker images (if any)
  docker images -a | grep "dev-peer" | awk '{print $3}' | xargs -r docker rmi -f 2>/dev/null || true

  # Remove chaincode containers
  docker ps -a | grep "dev-peer" | awk '{print $1}' | xargs -r docker rm -f 2>/dev/null || true

  # Clean up generated crypto material
  rm -rf "${SCRIPT_DIR}/organizations/peerOrganizations"
  rm -rf "${SCRIPT_DIR}/organizations/ordererOrganizations"

  # Clean up channel artifacts
  rm -rf "${SCRIPT_DIR}/channel-artifacts"

  # Clean up CA server data (but keep config files)
  for org_dir in platform organizer student ordererOrg; do
    local ca_dir="${SCRIPT_DIR}/organizations/fabric-ca/${org_dir}"
    if [ -d "$ca_dir" ]; then
      # Remove everything except the config file
      find "$ca_dir" -mindepth 1 ! -name 'fabric-ca-server-config.yaml' -exec rm -rf {} + 2>/dev/null || true
    fi
  done

  echo "Network torn down"
}

# ============================================================
# Start CAs
# ============================================================
startCAs() {
  echo ""
  echo "--- Starting Certificate Authorities ---"

  ${DOCKER_COMPOSE} ${COMPOSE_CA} up -d 2>&1

  # Wait for CAs to initialize and generate their TLS certs
  echo "Waiting for CAs to start..."
  sleep 3

  # Verify CAs are running
  local ca_containers=("ca_platform" "ca_organizer" "ca_student" "ca_orderer")
  for container in "${ca_containers[@]}"; do
    if ! docker ps --format '{{.Names}}' | grep -q "^${container}$"; then
      echo "ERROR: ${container} is not running"
      docker logs "$container" 2>&1 | tail -20
      exit 1
    fi
  done

  echo "All CAs running"
}

# ============================================================
# Generate crypto material
# ============================================================
generateCrypto() {
  echo ""
  echo "--- Generating crypto material via Fabric CAs ---"

  bash "${SCRIPT_DIR}/organizations/registerEnroll.sh"
}

# ============================================================
# Start network (peers + orderer + CouchDB)
# ============================================================
startNetwork() {
  echo ""
  echo "--- Starting peers, orderer, and CouchDB ---"

  ${DOCKER_COMPOSE} ${COMPOSE_NET} up -d 2>&1

  # Wait for containers to start
  sleep 3

  # Verify all containers are running
  local net_containers=(
    "orderer.eventchain.com"
    "peer0.platform.eventchain.com"
    "peer0.organizer.eventchain.com"
    "peer0.student.eventchain.com"
    "couchdb0"
    "couchdb1"
    "couchdb2"
  )
  for container in "${net_containers[@]}"; do
    if ! docker ps --format '{{.Names}}' | grep -q "^${container}$"; then
      echo "ERROR: ${container} is not running"
      docker logs "$container" 2>&1 | tail -20
      exit 1
    fi
  done

  echo "All network containers running"
}

# ============================================================
# Network up (full sequence)
# ============================================================
networkUp() {
  echo ""
  echo "============================================"
  echo " Starting EventChain Fabric Network"
  echo "============================================"
  echo ""
  echo " Organizations: PlatformOrg, OrganizerOrg, StudentOrg"
  echo " Orderer:       orderer.eventchain.com (Raft)"
  echo " Channel:       ${CHANNEL_NAME}"
  echo " State DB:      CouchDB 3.3.3"
  echo ""

  # Step 1: Start CAs
  startCAs

  # Step 2: Generate crypto material
  generateCrypto

  # Step 3: Start network
  startNetwork

  echo ""
  echo "============================================"
  echo " Network started successfully"
  echo ""
  echo " Next steps:"
  echo "   ./network.sh createChannel"
  echo "   ./network.sh deployCCs"
  echo "============================================"
}

# ============================================================
# Create channel
# ============================================================
createChannel() {
  echo ""
  echo "============================================"
  echo " Creating channel: ${CHANNEL_NAME}"
  echo "============================================"

  bash "${SCRIPT_DIR}/scripts/createChannel.sh" "$CHANNEL_NAME"
}

# ============================================================
# Deploy single chaincode
# ============================================================
deployCC() {
  bash "${SCRIPT_DIR}/scripts/deployCC.sh" "${EXTRA_ARGS[@]}"
}

# ============================================================
# Deploy all 4 EventChain chaincodes
# ============================================================
deployCCs() {
  echo ""
  echo "============================================"
  echo " Deploying all EventChain chaincodes"
  echo "============================================"

  local chaincodes=("token" "event" "prediction" "ticket")
  local sequence=1

  for cc in "${chaincodes[@]}"; do
    echo ""
    echo ">>> Deploying ${cc}-chaincode..."
    bash "${SCRIPT_DIR}/scripts/deployCC.sh" \
      -ccn "${cc}" \
      -ccp "${CC_SRC_BASE}/${cc}" \
      -ccl go \
      -ccv "1.0" \
      -ccs "$sequence" \
      -c "$CHANNEL_NAME"
  done

  echo ""
  echo "============================================"
  echo " All chaincodes deployed"
  echo ""
  echo " Deployed: token, event, prediction, ticket"
  echo " Channel:  ${CHANNEL_NAME}"
  echo " Policy:   OR(PlatformMSP.peer, OrganizerMSP.peer, StudentMSP.peer)"
  echo "============================================"
}

# ============================================================
# Dispatch
# ============================================================
case "$MODE" in
  up)
    networkUp
    ;;
  createChannel)
    createChannel
    ;;
  deployCC)
    deployCC
    ;;
  deployCCs)
    deployCCs
    ;;
  down)
    networkDown
    ;;
  restart)
    networkDown
    networkUp
    createChannel
    ;;
  *)
    printHelp
    exit 1
    ;;
esac
