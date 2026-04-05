#!/bin/bash
# fabric/network/scripts/envVar.sh
# Set environment variables for peer CLI operations
#
# Usage: source scripts/envVar.sh
#        setGlobals <org>     — sets env vars for the given org
#        setOrdererGlobals    — sets env vars for orderer admin operations

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NETWORK_DIR="$(dirname "$SCRIPT_DIR")"

export PATH="${NETWORK_DIR}/bin:$PATH"
export FABRIC_CFG_PATH="${NETWORK_DIR}/configtx"

export ORDERER_CA="${NETWORK_DIR}/organizations/ordererOrganizations/eventchain.com/tlsca/tlsca.eventchain.com-cert.pem"
export ORDERER_ADMIN_TLS_SIGN_CERT="${NETWORK_DIR}/organizations/ordererOrganizations/eventchain.com/orderers/orderer.eventchain.com/tls/server.crt"
export ORDERER_ADMIN_TLS_PRIVATE_KEY="${NETWORK_DIR}/organizations/ordererOrganizations/eventchain.com/orderers/orderer.eventchain.com/tls/server.key"

# When using CA-generated certs, the tlsca cert is the same as the ca-cert.pem
# We also set a fallback path using the orderer's TLS CA cert
if [ ! -f "$ORDERER_CA" ]; then
  export ORDERER_CA="${NETWORK_DIR}/organizations/ordererOrganizations/eventchain.com/orderers/orderer.eventchain.com/tls/ca.crt"
fi

# --- PlatformOrg peer TLS root cert ---
export PEER0_PLATFORM_CA="${NETWORK_DIR}/organizations/peerOrganizations/platform.eventchain.com/peers/peer0.platform.eventchain.com/tls/ca.crt"

# --- OrganizerOrg peer TLS root cert ---
export PEER0_ORGANIZER_CA="${NETWORK_DIR}/organizations/peerOrganizations/organizer.eventchain.com/peers/peer0.organizer.eventchain.com/tls/ca.crt"

# --- StudentOrg peer TLS root cert ---
export PEER0_STUDENT_CA="${NETWORK_DIR}/organizations/peerOrganizations/student.eventchain.com/peers/peer0.student.eventchain.com/tls/ca.crt"

setGlobals() {
  local ORG="$1"
  case "$ORG" in
    platform|PlatformOrg|1)
      export CORE_PEER_LOCALMSPID="PlatformMSP"
      export CORE_PEER_TLS_ROOTCERT_FILE="$PEER0_PLATFORM_CA"
      export CORE_PEER_MSPCONFIGPATH="${NETWORK_DIR}/organizations/peerOrganizations/platform.eventchain.com/users/Admin@platform.eventchain.com/msp"
      export CORE_PEER_ADDRESS=localhost:7051
      ;;
    organizer|OrganizerOrg|2)
      export CORE_PEER_LOCALMSPID="OrganizerMSP"
      export CORE_PEER_TLS_ROOTCERT_FILE="$PEER0_ORGANIZER_CA"
      export CORE_PEER_MSPCONFIGPATH="${NETWORK_DIR}/organizations/peerOrganizations/organizer.eventchain.com/users/Admin@organizer.eventchain.com/msp"
      export CORE_PEER_ADDRESS=localhost:9051
      ;;
    student|StudentOrg|3)
      export CORE_PEER_LOCALMSPID="StudentMSP"
      export CORE_PEER_TLS_ROOTCERT_FILE="$PEER0_STUDENT_CA"
      export CORE_PEER_MSPCONFIGPATH="${NETWORK_DIR}/organizations/peerOrganizations/student.eventchain.com/users/Admin@student.eventchain.com/msp"
      export CORE_PEER_ADDRESS=localhost:11051
      ;;
    *)
      echo "ERROR: Unknown organization '${ORG}'. Use: platform, organizer, or student"
      exit 1
      ;;
  esac

  export CORE_PEER_TLS_ENABLED=true
}

setOrdererGlobals() {
  export CORE_PEER_LOCALMSPID="OrdererMSP"
  export CORE_PEER_TLS_ROOTCERT_FILE="$ORDERER_CA"
  export CORE_PEER_MSPCONFIGPATH="${NETWORK_DIR}/organizations/ordererOrganizations/eventchain.com/users/Admin@eventchain.com/msp"
}

# Print current peer context (useful for debugging)
printGlobals() {
  echo "CORE_PEER_LOCALMSPID    = ${CORE_PEER_LOCALMSPID:-<not set>}"
  echo "CORE_PEER_ADDRESS       = ${CORE_PEER_ADDRESS:-<not set>}"
  echo "CORE_PEER_TLS_ENABLED   = ${CORE_PEER_TLS_ENABLED:-<not set>}"
  echo "CORE_PEER_TLS_ROOTCERT  = ${CORE_PEER_TLS_ROOTCERT_FILE:-<not set>}"
  echo "CORE_PEER_MSPCONFIGPATH = ${CORE_PEER_MSPCONFIGPATH:-<not set>}"
}
