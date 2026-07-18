#!/bin/bash
# fabric/network/organizations/registerEnroll.sh
# Register and enroll identities for all orgs using Fabric CA
#
# This script expects:
#   - CA containers are running
#   - fabric-ca-client binary is in PATH (or ../bin/)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NETWORK_DIR="$(dirname "$SCRIPT_DIR")"

# Locate fabric-ca-client binary
export PATH="${NETWORK_DIR}/bin:$PATH"

# ============================================================
# Helper: wait for CA to be ready
# ============================================================
waitForCA() {
  local ca_url="$1"
  local max_retry=10
  local counter=0
  echo "Waiting for CA at ${ca_url}..."
  while ! fabric-ca-client getcainfo -u "$ca_url" --tls.certfiles /dev/null &>/dev/null 2>&1; do
    sleep 1
    counter=$((counter + 1))
    if [ "$counter" -ge "$max_retry" ]; then
      echo "WARNING: CA at ${ca_url} not responding after ${max_retry}s, proceeding anyway"
      return 0
    fi
  done
  echo "CA at ${ca_url} is ready"
}

# ============================================================
# Create Org MSP structure
# ============================================================
createOrgMSP() {
  local org_domain="$1"
  local msp_dir="${SCRIPT_DIR}/peerOrganizations/${org_domain}/msp"
  mkdir -p "${msp_dir}"
  # NodeOUs configuration for automatic OU classification
  cat > "${msp_dir}/config.yaml" <<EOF
NodeOUs:
  Enable: true
  ClientOUIdentifier:
    Certificate: cacerts/localhost-${2}-${3}.pem
    OrganizationalUnitIdentifier: client
  PeerOUIdentifier:
    Certificate: cacerts/localhost-${2}-${3}.pem
    OrganizationalUnitIdentifier: peer
  AdminOUIdentifier:
    Certificate: cacerts/localhost-${2}-${3}.pem
    OrganizationalUnitIdentifier: admin
  OrdererOUIdentifier:
    Certificate: cacerts/localhost-${2}-${3}.pem
    OrganizationalUnitIdentifier: orderer
EOF
}

# ============================================================
# PlatformOrg
# ============================================================
createPlatformOrg() {
  local CA_PORT=7054
  local CA_URL="https://localhost:${CA_PORT}"
  local ORG_DIR="${SCRIPT_DIR}/peerOrganizations/platform.eventchain.com"
  local CA_CERT="${SCRIPT_DIR}/fabric-ca/platform/ca-cert.pem"

  echo ""
  echo "============================================"
  echo " Enrolling PlatformOrg identities"
  echo "============================================"

  # --- Enroll CA admin ---
  mkdir -p "${ORG_DIR}"
  export FABRIC_CA_CLIENT_HOME="${ORG_DIR}"

  fabric-ca-client enroll \
    -u "https://admin:adminpw@localhost:${CA_PORT}" \
    --caname ca-platform \
    --tls.certfiles "${CA_CERT}"

  createOrgMSP "platform.eventchain.com" "${CA_PORT}" "ca-platform"

  # Copy CA cert into org MSP
  mkdir -p "${ORG_DIR}/msp/cacerts"
  cp "${CA_CERT}" "${ORG_DIR}/msp/cacerts/localhost-${CA_PORT}.pem"

  mkdir -p "${ORG_DIR}/msp/tlscacerts"
  cp "${CA_CERT}" "${ORG_DIR}/msp/tlscacerts/ca.crt"

  # --- Register peer0 ---
  fabric-ca-client register \
    --caname ca-platform \
    --id.name peer0 \
    --id.secret peer0pw \
    --id.type peer \
    --tls.certfiles "${CA_CERT}"

  # --- Register org admin ---
  fabric-ca-client register \
    --caname ca-platform \
    --id.name platformadmin \
    --id.secret platformadminpw \
    --id.type admin \
    --tls.certfiles "${CA_CERT}"

  # --- Register user1 (for testing) ---
  fabric-ca-client register \
    --caname ca-platform \
    --id.name user1 \
    --id.secret user1pw \
    --id.type client \
    --tls.certfiles "${CA_CERT}"

  # --- Enroll peer0 MSP ---
  local PEER_DIR="${ORG_DIR}/peers/peer0.platform.eventchain.com"
  mkdir -p "${PEER_DIR}"

  fabric-ca-client enroll \
    -u "https://peer0:peer0pw@localhost:${CA_PORT}" \
    --caname ca-platform \
    -M "${PEER_DIR}/msp" \
    --tls.certfiles "${CA_CERT}" \
    --csr.hosts peer0.platform.eventchain.com

  cp "${ORG_DIR}/msp/config.yaml" "${PEER_DIR}/msp/config.yaml"

  # --- Enroll peer0 TLS ---
  fabric-ca-client enroll \
    -u "https://peer0:peer0pw@localhost:${CA_PORT}" \
    --caname ca-platform \
    -M "${PEER_DIR}/tls" \
    --enrollment.profile tls \
    --tls.certfiles "${CA_CERT}" \
    --csr.hosts peer0.platform.eventchain.com \
    --csr.hosts localhost

  # Rename TLS certs to standard names expected by peer config
  cp "${PEER_DIR}/tls/tlscacerts/"*  "${PEER_DIR}/tls/ca.crt"
  cp "${PEER_DIR}/tls/signcerts/"*   "${PEER_DIR}/tls/server.crt"
  cp "${PEER_DIR}/tls/keystore/"*    "${PEER_DIR}/tls/server.key"

  # --- Enroll org admin ---
  local ADMIN_DIR="${ORG_DIR}/users/Admin@platform.eventchain.com"
  mkdir -p "${ADMIN_DIR}"

  fabric-ca-client enroll \
    -u "https://platformadmin:platformadminpw@localhost:${CA_PORT}" \
    --caname ca-platform \
    -M "${ADMIN_DIR}/msp" \
    --tls.certfiles "${CA_CERT}"

  cp "${ORG_DIR}/msp/config.yaml" "${ADMIN_DIR}/msp/config.yaml"

  echo "PlatformOrg identity generation complete"
}

# ============================================================
# OrganizerOrg
# ============================================================
createOrganizerOrg() {
  local CA_PORT=8054
  local CA_URL="https://localhost:${CA_PORT}"
  local ORG_DIR="${SCRIPT_DIR}/peerOrganizations/organizer.eventchain.com"
  local CA_CERT="${SCRIPT_DIR}/fabric-ca/organizer/ca-cert.pem"

  echo ""
  echo "============================================"
  echo " Enrolling OrganizerOrg identities"
  echo "============================================"

  mkdir -p "${ORG_DIR}"
  export FABRIC_CA_CLIENT_HOME="${ORG_DIR}"

  fabric-ca-client enroll \
    -u "https://admin:adminpw@localhost:${CA_PORT}" \
    --caname ca-organizer \
    --tls.certfiles "${CA_CERT}"

  createOrgMSP "organizer.eventchain.com" "${CA_PORT}" "ca-organizer"

  mkdir -p "${ORG_DIR}/msp/cacerts"
  cp "${CA_CERT}" "${ORG_DIR}/msp/cacerts/localhost-${CA_PORT}.pem"

  mkdir -p "${ORG_DIR}/msp/tlscacerts"
  cp "${CA_CERT}" "${ORG_DIR}/msp/tlscacerts/ca.crt"

  # --- Register peer0 ---
  fabric-ca-client register \
    --caname ca-organizer \
    --id.name peer0 \
    --id.secret peer0pw \
    --id.type peer \
    --tls.certfiles "${CA_CERT}"

  # --- Register org admin ---
  fabric-ca-client register \
    --caname ca-organizer \
    --id.name organizeradmin \
    --id.secret organizeradminpw \
    --id.type admin \
    --tls.certfiles "${CA_CERT}"

  # --- Register user1 ---
  fabric-ca-client register \
    --caname ca-organizer \
    --id.name user1 \
    --id.secret user1pw \
    --id.type client \
    --tls.certfiles "${CA_CERT}"

  # --- Enroll peer0 MSP ---
  local PEER_DIR="${ORG_DIR}/peers/peer0.organizer.eventchain.com"
  mkdir -p "${PEER_DIR}"

  fabric-ca-client enroll \
    -u "https://peer0:peer0pw@localhost:${CA_PORT}" \
    --caname ca-organizer \
    -M "${PEER_DIR}/msp" \
    --tls.certfiles "${CA_CERT}" \
    --csr.hosts peer0.organizer.eventchain.com

  cp "${ORG_DIR}/msp/config.yaml" "${PEER_DIR}/msp/config.yaml"

  # --- Enroll peer0 TLS ---
  fabric-ca-client enroll \
    -u "https://peer0:peer0pw@localhost:${CA_PORT}" \
    --caname ca-organizer \
    -M "${PEER_DIR}/tls" \
    --enrollment.profile tls \
    --tls.certfiles "${CA_CERT}" \
    --csr.hosts peer0.organizer.eventchain.com \
    --csr.hosts localhost

  cp "${PEER_DIR}/tls/tlscacerts/"*  "${PEER_DIR}/tls/ca.crt"
  cp "${PEER_DIR}/tls/signcerts/"*   "${PEER_DIR}/tls/server.crt"
  cp "${PEER_DIR}/tls/keystore/"*    "${PEER_DIR}/tls/server.key"

  # --- Enroll org admin ---
  local ADMIN_DIR="${ORG_DIR}/users/Admin@organizer.eventchain.com"
  mkdir -p "${ADMIN_DIR}"

  fabric-ca-client enroll \
    -u "https://organizeradmin:organizeradminpw@localhost:${CA_PORT}" \
    --caname ca-organizer \
    -M "${ADMIN_DIR}/msp" \
    --tls.certfiles "${CA_CERT}"

  cp "${ORG_DIR}/msp/config.yaml" "${ADMIN_DIR}/msp/config.yaml"

  echo "OrganizerOrg identity generation complete"
}

# ============================================================
# StudentOrg
# ============================================================
createStudentOrg() {
  local CA_PORT=9054
  local CA_URL="https://localhost:${CA_PORT}"
  local ORG_DIR="${SCRIPT_DIR}/peerOrganizations/student.eventchain.com"
  local CA_CERT="${SCRIPT_DIR}/fabric-ca/student/ca-cert.pem"

  echo ""
  echo "============================================"
  echo " Enrolling StudentOrg identities"
  echo "============================================"

  mkdir -p "${ORG_DIR}"
  export FABRIC_CA_CLIENT_HOME="${ORG_DIR}"

  fabric-ca-client enroll \
    -u "https://admin:adminpw@localhost:${CA_PORT}" \
    --caname ca-student \
    --tls.certfiles "${CA_CERT}"

  createOrgMSP "student.eventchain.com" "${CA_PORT}" "ca-student"

  mkdir -p "${ORG_DIR}/msp/cacerts"
  cp "${CA_CERT}" "${ORG_DIR}/msp/cacerts/localhost-${CA_PORT}.pem"

  mkdir -p "${ORG_DIR}/msp/tlscacerts"
  cp "${CA_CERT}" "${ORG_DIR}/msp/tlscacerts/ca.crt"

  # --- Register peer0 ---
  fabric-ca-client register \
    --caname ca-student \
    --id.name peer0 \
    --id.secret peer0pw \
    --id.type peer \
    --tls.certfiles "${CA_CERT}"

  # --- Register org admin ---
  fabric-ca-client register \
    --caname ca-student \
    --id.name studentadmin \
    --id.secret studentadminpw \
    --id.type admin \
    --tls.certfiles "${CA_CERT}"

  # --- Register user1 ---
  fabric-ca-client register \
    --caname ca-student \
    --id.name user1 \
    --id.secret user1pw \
    --id.type client \
    --tls.certfiles "${CA_CERT}"

  # --- Enroll peer0 MSP ---
  local PEER_DIR="${ORG_DIR}/peers/peer0.student.eventchain.com"
  mkdir -p "${PEER_DIR}"

  fabric-ca-client enroll \
    -u "https://peer0:peer0pw@localhost:${CA_PORT}" \
    --caname ca-student \
    -M "${PEER_DIR}/msp" \
    --tls.certfiles "${CA_CERT}" \
    --csr.hosts peer0.student.eventchain.com

  cp "${ORG_DIR}/msp/config.yaml" "${PEER_DIR}/msp/config.yaml"

  # --- Enroll peer0 TLS ---
  fabric-ca-client enroll \
    -u "https://peer0:peer0pw@localhost:${CA_PORT}" \
    --caname ca-student \
    -M "${PEER_DIR}/tls" \
    --enrollment.profile tls \
    --tls.certfiles "${CA_CERT}" \
    --csr.hosts peer0.student.eventchain.com \
    --csr.hosts localhost

  cp "${PEER_DIR}/tls/tlscacerts/"*  "${PEER_DIR}/tls/ca.crt"
  cp "${PEER_DIR}/tls/signcerts/"*   "${PEER_DIR}/tls/server.crt"
  cp "${PEER_DIR}/tls/keystore/"*    "${PEER_DIR}/tls/server.key"

  # --- Enroll org admin ---
  local ADMIN_DIR="${ORG_DIR}/users/Admin@student.eventchain.com"
  mkdir -p "${ADMIN_DIR}"

  fabric-ca-client enroll \
    -u "https://studentadmin:studentadminpw@localhost:${CA_PORT}" \
    --caname ca-student \
    -M "${ADMIN_DIR}/msp" \
    --tls.certfiles "${CA_CERT}"

  cp "${ORG_DIR}/msp/config.yaml" "${ADMIN_DIR}/msp/config.yaml"

  echo "StudentOrg identity generation complete"
}

# ============================================================
# OrdererOrg
# ============================================================
createOrdererOrg() {
  local CA_PORT=10054
  local CA_URL="https://localhost:${CA_PORT}"
  local ORG_DIR="${SCRIPT_DIR}/ordererOrganizations/eventchain.com"
  local CA_CERT="${SCRIPT_DIR}/fabric-ca/ordererOrg/ca-cert.pem"

  echo ""
  echo "============================================"
  echo " Enrolling OrdererOrg identities"
  echo "============================================"

  mkdir -p "${ORG_DIR}"
  export FABRIC_CA_CLIENT_HOME="${ORG_DIR}"

  fabric-ca-client enroll \
    -u "https://admin:adminpw@localhost:${CA_PORT}" \
    --caname ca-orderer \
    --tls.certfiles "${CA_CERT}"

  # Orderer org MSP config
  local MSP_DIR="${ORG_DIR}/msp"
  mkdir -p "${MSP_DIR}"
  cat > "${MSP_DIR}/config.yaml" <<EOF
NodeOUs:
  Enable: true
  ClientOUIdentifier:
    Certificate: cacerts/localhost-${CA_PORT}-ca-orderer.pem
    OrganizationalUnitIdentifier: client
  PeerOUIdentifier:
    Certificate: cacerts/localhost-${CA_PORT}-ca-orderer.pem
    OrganizationalUnitIdentifier: peer
  AdminOUIdentifier:
    Certificate: cacerts/localhost-${CA_PORT}-ca-orderer.pem
    OrganizationalUnitIdentifier: admin
  OrdererOUIdentifier:
    Certificate: cacerts/localhost-${CA_PORT}-ca-orderer.pem
    OrganizationalUnitIdentifier: orderer
EOF

  mkdir -p "${ORG_DIR}/msp/cacerts"
  cp "${CA_CERT}" "${ORG_DIR}/msp/cacerts/localhost-${CA_PORT}.pem"

  mkdir -p "${ORG_DIR}/msp/tlscacerts"
  cp "${CA_CERT}" "${ORG_DIR}/msp/tlscacerts/tlsca.eventchain.com-cert.pem"

  # --- Register orderer ---
  fabric-ca-client register \
    --caname ca-orderer \
    --id.name orderer \
    --id.secret ordererpw \
    --id.type orderer \
    --tls.certfiles "${CA_CERT}"

  # --- Register orderer admin ---
  fabric-ca-client register \
    --caname ca-orderer \
    --id.name ordererAdmin \
    --id.secret ordererAdminpw \
    --id.type admin \
    --tls.certfiles "${CA_CERT}"

  # --- Enroll orderer MSP ---
  local ORDERER_DIR="${ORG_DIR}/orderers/orderer.eventchain.com"
  mkdir -p "${ORDERER_DIR}"

  fabric-ca-client enroll \
    -u "https://orderer:ordererpw@localhost:${CA_PORT}" \
    --caname ca-orderer \
    -M "${ORDERER_DIR}/msp" \
    --tls.certfiles "${CA_CERT}" \
    --csr.hosts orderer.eventchain.com \
    --csr.hosts localhost

  cp "${ORG_DIR}/msp/config.yaml" "${ORDERER_DIR}/msp/config.yaml"

  # --- Enroll orderer TLS ---
  fabric-ca-client enroll \
    -u "https://orderer:ordererpw@localhost:${CA_PORT}" \
    --caname ca-orderer \
    -M "${ORDERER_DIR}/tls" \
    --enrollment.profile tls \
    --tls.certfiles "${CA_CERT}" \
    --csr.hosts orderer.eventchain.com \
    --csr.hosts localhost

  cp "${ORDERER_DIR}/tls/tlscacerts/"*  "${ORDERER_DIR}/tls/ca.crt"
  cp "${ORDERER_DIR}/tls/signcerts/"*   "${ORDERER_DIR}/tls/server.crt"
  cp "${ORDERER_DIR}/tls/keystore/"*    "${ORDERER_DIR}/tls/server.key"

  # --- Enroll orderer admin ---
  local ADMIN_DIR="${ORG_DIR}/users/Admin@eventchain.com"
  mkdir -p "${ADMIN_DIR}"

  fabric-ca-client enroll \
    -u "https://ordererAdmin:ordererAdminpw@localhost:${CA_PORT}" \
    --caname ca-orderer \
    -M "${ADMIN_DIR}/msp" \
    --tls.certfiles "${CA_CERT}"

  cp "${ORG_DIR}/msp/config.yaml" "${ADMIN_DIR}/msp/config.yaml"

  echo "OrdererOrg identity generation complete"
}

# ============================================================
# Main
# ============================================================
echo ""
echo "============================================"
echo " EventChain — CA-based Identity Generation"
echo "============================================"

createPlatformOrg
createOrganizerOrg
createStudentOrg
createOrdererOrg

echo ""
echo "============================================"
echo " All identities enrolled successfully"
echo "============================================"
echo ""
echo "Organization MSP directories:"
echo "  ${SCRIPT_DIR}/peerOrganizations/platform.eventchain.com/"
echo "  ${SCRIPT_DIR}/peerOrganizations/organizer.eventchain.com/"
echo "  ${SCRIPT_DIR}/peerOrganizations/student.eventchain.com/"
echo "  ${SCRIPT_DIR}/ordererOrganizations/eventchain.com/"
