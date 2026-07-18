#!/bin/bash
# fabric/network/scripts/deployCC.sh
# Deploy chaincode using Fabric 2.5 lifecycle
#
# Usage: ./scripts/deployCC.sh -ccn <name> -ccp <path> [-ccl go] [-ccv 1.0] [-ccs 1]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NETWORK_DIR="$(dirname "$SCRIPT_DIR")"

# Source environment variables
. "${SCRIPT_DIR}/envVar.sh"

# Defaults
CC_NAME=""
CC_SRC_PATH=""
CC_LANGUAGE="go"
CC_VERSION="1.0"
CC_SEQUENCE=1
CC_END_POLICY="OR('PlatformMSP.peer','OrganizerMSP.peer','StudentMSP.peer')"
CC_COLL_CONFIG=""
CC_INIT_FCN=""
CHANNEL_NAME="eventchain"
DELAY=3
MAX_RETRY=5

# Parse arguments
while [[ $# -gt 0 ]]; do
  case "$1" in
    -ccn)  CC_NAME="$2";       shift 2 ;;
    -ccp)  CC_SRC_PATH="$2";   shift 2 ;;
    -ccl)  CC_LANGUAGE="$2";   shift 2 ;;
    -ccv)  CC_VERSION="$2";    shift 2 ;;
    -ccs)  CC_SEQUENCE="$2";   shift 2 ;;
    -cce)  CC_END_POLICY="$2"; shift 2 ;;
    -ccco) CC_COLL_CONFIG="$2";shift 2 ;;
    -cci)  CC_INIT_FCN="$2";   shift 2 ;;
    -c)    CHANNEL_NAME="$2";  shift 2 ;;
    -d)    DELAY="$2";         shift 2 ;;
    -r)    MAX_RETRY="$2";     shift 2 ;;
    *)
      echo "Unknown flag: $1"
      exit 1
      ;;
  esac
done

if [ -z "$CC_NAME" ] || [ -z "$CC_SRC_PATH" ]; then
  echo "Usage: $0 -ccn <name> -ccp <path> [-ccl go] [-ccv 1.0] [-ccs 1]"
  exit 1
fi

CC_LABEL="${CC_NAME}_${CC_VERSION}"

echo ""
echo "============================================"
echo " Deploying chaincode: ${CC_NAME}"
echo " Source:   ${CC_SRC_PATH}"
echo " Language: ${CC_LANGUAGE}"
echo " Version:  ${CC_VERSION}"
echo " Sequence: ${CC_SEQUENCE}"
echo " Channel:  ${CHANNEL_NAME}"
echo " Policy:   ${CC_END_POLICY}"
echo "============================================"

# ============================================================
# Step 1: Vendor Go dependencies
# ============================================================
packageChaincode() {
  echo ""
  echo "--- Packaging chaincode ---"

  local LANG="$CC_LANGUAGE"
  if [ "$CC_LANGUAGE" = "go" ]; then
    LANG="golang"
    echo "Vendoring Go dependencies..."
    pushd "$CC_SRC_PATH" > /dev/null
    GO111MODULE=on go mod vendor
    popd > /dev/null
  fi

  peer lifecycle chaincode package "${CC_NAME}.tar.gz" \
    --path "$CC_SRC_PATH" \
    --lang "$LANG" \
    --label "$CC_LABEL"

  echo "Chaincode packaged: ${CC_NAME}.tar.gz"
}

# ============================================================
# Step 2: Install on all peers
# ============================================================
installChaincode() {
  local org="$1"
  setGlobals "$org"

  echo ""
  echo "--- Installing chaincode on ${org} ---"

  local counter=0
  while true; do
    peer lifecycle chaincode install "${CC_NAME}.tar.gz" && break
    counter=$((counter + 1))
    if [ "$counter" -ge "$MAX_RETRY" ]; then
      echo "ERROR: chaincode install failed on ${org}"
      exit 1
    fi
    echo "Retrying in ${DELAY}s..."
    sleep "$DELAY"
  done
}

# ============================================================
# Step 3: Query installed and get package ID
# ============================================================
queryInstalled() {
  local org="$1"
  setGlobals "$org"

  echo ""
  echo "--- Querying installed chaincode on ${org} ---"

  peer lifecycle chaincode queryinstalled \
    --output json > query_installed.json

  PACKAGE_ID=$(jq -r --arg LABEL "$CC_LABEL" \
    '.installed_chaincodes[] | select(.label == $LABEL) | .package_id' \
    query_installed.json)

  if [ -z "$PACKAGE_ID" ]; then
    echo "ERROR: Could not find package ID for label ${CC_LABEL}"
    exit 1
  fi

  echo "Package ID: ${PACKAGE_ID}"
  rm -f query_installed.json
}

# ============================================================
# Step 4: Approve for each org
# ============================================================
approveForOrg() {
  local org="$1"
  setGlobals "$org"

  echo ""
  echo "--- Approving chaincode for ${org} ---"

  local init_flag=""
  if [ -n "$CC_INIT_FCN" ]; then
    init_flag="--init-required"
  fi

  local coll_flag=""
  if [ -n "$CC_COLL_CONFIG" ]; then
    coll_flag="--collections-config ${CC_COLL_CONFIG}"
  fi

  peer lifecycle chaincode approveformyorg \
    -o localhost:7050 \
    --ordererTLSHostnameOverride orderer.eventchain.com \
    --tls \
    --cafile "$ORDERER_CA" \
    --channelID "$CHANNEL_NAME" \
    --name "$CC_NAME" \
    --version "$CC_VERSION" \
    --package-id "$PACKAGE_ID" \
    --sequence "$CC_SEQUENCE" \
    --signature-policy "$CC_END_POLICY" \
    $init_flag \
    $coll_flag

  echo "Chaincode approved by ${org}"
}

# ============================================================
# Step 5: Check commit readiness
# ============================================================
checkCommitReadiness() {
  echo ""
  echo "--- Checking commit readiness ---"

  local init_flag=""
  if [ -n "$CC_INIT_FCN" ]; then
    init_flag="--init-required"
  fi

  local coll_flag=""
  if [ -n "$CC_COLL_CONFIG" ]; then
    coll_flag="--collections-config ${CC_COLL_CONFIG}"
  fi

  setGlobals platform

  peer lifecycle chaincode checkcommitreadiness \
    --channelID "$CHANNEL_NAME" \
    --name "$CC_NAME" \
    --version "$CC_VERSION" \
    --sequence "$CC_SEQUENCE" \
    --signature-policy "$CC_END_POLICY" \
    --output json \
    $init_flag \
    $coll_flag
}

# ============================================================
# Step 6: Commit chaincode definition
# ============================================================
commitChaincode() {
  echo ""
  echo "--- Committing chaincode definition ---"

  local init_flag=""
  if [ -n "$CC_INIT_FCN" ]; then
    init_flag="--init-required"
  fi

  local coll_flag=""
  if [ -n "$CC_COLL_CONFIG" ]; then
    coll_flag="--collections-config ${CC_COLL_CONFIG}"
  fi

  setGlobals platform

  peer lifecycle chaincode commit \
    -o localhost:7050 \
    --ordererTLSHostnameOverride orderer.eventchain.com \
    --tls \
    --cafile "$ORDERER_CA" \
    --channelID "$CHANNEL_NAME" \
    --name "$CC_NAME" \
    --version "$CC_VERSION" \
    --sequence "$CC_SEQUENCE" \
    --signature-policy "$CC_END_POLICY" \
    --peerAddresses localhost:7051 \
    --tlsRootCertFiles "$PEER0_PLATFORM_CA" \
    --peerAddresses localhost:9051 \
    --tlsRootCertFiles "$PEER0_ORGANIZER_CA" \
    --peerAddresses localhost:11051 \
    --tlsRootCertFiles "$PEER0_STUDENT_CA" \
    $init_flag \
    $coll_flag

  echo "Chaincode definition committed"
}

# ============================================================
# Step 7: Query committed
# ============================================================
queryCommitted() {
  echo ""
  echo "--- Querying committed chaincode ---"

  setGlobals platform

  peer lifecycle chaincode querycommitted \
    --channelID "$CHANNEL_NAME" \
    --name "$CC_NAME" \
    --output json
}

# ============================================================
# Step 8 (optional): Init chaincode
# ============================================================
initChaincode() {
  if [ -z "$CC_INIT_FCN" ]; then
    return
  fi

  echo ""
  echo "--- Initializing chaincode with ${CC_INIT_FCN} ---"

  setGlobals platform

  peer chaincode invoke \
    -o localhost:7050 \
    --ordererTLSHostnameOverride orderer.eventchain.com \
    --tls \
    --cafile "$ORDERER_CA" \
    -C "$CHANNEL_NAME" \
    -n "$CC_NAME" \
    --isInit \
    --peerAddresses localhost:7051 \
    --tlsRootCertFiles "$PEER0_PLATFORM_CA" \
    --peerAddresses localhost:9051 \
    --tlsRootCertFiles "$PEER0_ORGANIZER_CA" \
    --peerAddresses localhost:11051 \
    --tlsRootCertFiles "$PEER0_STUDENT_CA" \
    -c "{\"function\":\"${CC_INIT_FCN}\",\"Args\":[]}"

  echo "Chaincode initialized"
}

# ============================================================
# Main
# ============================================================
packageChaincode
installChaincode platform
installChaincode organizer
installChaincode student
queryInstalled platform
approveForOrg platform
approveForOrg organizer
approveForOrg student
checkCommitReadiness
commitChaincode
queryCommitted
initChaincode

# Clean up package
rm -f "${CC_NAME}.tar.gz"

echo ""
echo "============================================"
echo " Chaincode '${CC_NAME}' deployed successfully"
echo "============================================"
