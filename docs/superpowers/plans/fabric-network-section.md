# Fabric Network Infrastructure — Implementation Plan

> Section of the EventChain implementation plan covering all Hyperledger Fabric 2.5 network setup tasks.
> 3 organizations, Raft orderer, CouchDB state databases, CA-based identity management.

---

## Task 1: Prerequisites Check

Verify all required tools are installed and download Fabric binaries and Docker images.

- [ ] Create `fabric/network/prereqs.sh` with the contents below
- [ ] Run the script to verify the local environment is ready
- [ ] Confirm Fabric binaries are available in `fabric/network/bin/` after download

```bash
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
```

---

## Task 2: Docker Compose Files

Create the three Docker Compose files and environment file that define the entire network infrastructure.

### 2a. Environment file

- [ ] Create `fabric/network/docker/.env` with the contents below

```env
# fabric/network/docker/.env
# Docker Compose environment variables for EventChain Fabric network

COMPOSE_PROJECT_NAME=eventchain
IMAGE_TAG=2.5
CA_IMAGE_TAG=1.5
COUCH_IMAGE_TAG=3.3.3
SYS_CHANNEL=system-channel
```

### 2b. Peer + Orderer + CouchDB Compose file

- [ ] Create `fabric/network/docker/docker-compose-net.yaml` with the contents below
- [ ] Validate YAML syntax (`docker compose -f docker-compose-net.yaml config`)

```yaml
# fabric/network/docker/docker-compose-net.yaml
# EventChain Fabric network — peers, orderer, CouchDB instances

version: '3.7'

volumes:
  orderer.eventchain.com:
  peer0.platform.eventchain.com:
  peer0.organizer.eventchain.com:
  peer0.student.eventchain.com:

networks:
  eventchain_network:
    name: eventchain_network

services:

  # ============================================================
  # Orderer — Raft, single node
  # ============================================================
  orderer.eventchain.com:
    container_name: orderer.eventchain.com
    image: hyperledger/fabric-orderer:${IMAGE_TAG}
    labels:
      service: hyperledger-fabric
    environment:
      - FABRIC_LOGGING_SPEC=INFO
      - ORDERER_GENERAL_LISTENADDRESS=0.0.0.0
      - ORDERER_GENERAL_LISTENPORT=7050
      - ORDERER_GENERAL_LOCALMSPID=OrdererMSP
      - ORDERER_GENERAL_LOCALMSPDIR=/var/hyperledger/orderer/msp
      - ORDERER_GENERAL_TLS_ENABLED=true
      - ORDERER_GENERAL_TLS_PRIVATEKEY=/var/hyperledger/orderer/tls/server.key
      - ORDERER_GENERAL_TLS_CERTIFICATE=/var/hyperledger/orderer/tls/server.crt
      - ORDERER_GENERAL_TLS_ROOTCAS=[/var/hyperledger/orderer/tls/ca.crt]
      - ORDERER_GENERAL_CLUSTER_CLIENTCERTIFICATE=/var/hyperledger/orderer/tls/server.crt
      - ORDERER_GENERAL_CLUSTER_CLIENTPRIVATEKEY=/var/hyperledger/orderer/tls/server.key
      - ORDERER_GENERAL_CLUSTER_ROOTCAS=[/var/hyperledger/orderer/tls/ca.crt]
      - ORDERER_GENERAL_BOOTSTRAPMETHOD=none
      - ORDERER_CHANNELPARTICIPATION_ENABLED=true
      - ORDERER_ADMIN_TLS_ENABLED=true
      - ORDERER_ADMIN_TLS_CERTIFICATE=/var/hyperledger/orderer/tls/server.crt
      - ORDERER_ADMIN_TLS_PRIVATEKEY=/var/hyperledger/orderer/tls/server.key
      - ORDERER_ADMIN_TLS_ROOTCAS=[/var/hyperledger/orderer/tls/ca.crt]
      - ORDERER_ADMIN_TLS_CLIENTROOTCAS=[/var/hyperledger/orderer/tls/ca.crt]
      - ORDERER_ADMIN_LISTENADDRESS=0.0.0.0:7053
      - ORDERER_OPERATIONS_LISTENADDRESS=orderer.eventchain.com:9443
      - ORDERER_METRICS_PROVIDER=prometheus
    working_dir: /root
    command: orderer
    volumes:
      - ../organizations/ordererOrganizations/eventchain.com/orderers/orderer.eventchain.com/msp:/var/hyperledger/orderer/msp
      - ../organizations/ordererOrganizations/eventchain.com/orderers/orderer.eventchain.com/tls:/var/hyperledger/orderer/tls
      - orderer.eventchain.com:/var/hyperledger/production/orderer
    ports:
      - 7050:7050
      - 7053:7053
      - 9443:9443
    networks:
      - eventchain_network

  # ============================================================
  # PlatformOrg — peer0 + CouchDB
  # ============================================================
  couchdb0:
    container_name: couchdb0
    image: couchdb:${COUCH_IMAGE_TAG}
    labels:
      service: hyperledger-fabric
    environment:
      - COUCHDB_USER=admin
      - COUCHDB_PASSWORD=adminpw
    ports:
      - "5984:5984"
    networks:
      - eventchain_network

  peer0.platform.eventchain.com:
    container_name: peer0.platform.eventchain.com
    image: hyperledger/fabric-peer:${IMAGE_TAG}
    labels:
      service: hyperledger-fabric
    environment:
      - FABRIC_CFG_PATH=/etc/hyperledger/peercfg
      - FABRIC_LOGGING_SPEC=INFO
      #- FABRIC_LOGGING_SPEC=DEBUG
      - CORE_PEER_TLS_ENABLED=true
      - CORE_PEER_PROFILE_ENABLED=false
      - CORE_PEER_TLS_CERT_FILE=/etc/hyperledger/fabric/tls/server.crt
      - CORE_PEER_TLS_KEY_FILE=/etc/hyperledger/fabric/tls/server.key
      - CORE_PEER_TLS_ROOTCERT_FILE=/etc/hyperledger/fabric/tls/ca.crt
      # Peer specific
      - CORE_PEER_ID=peer0.platform.eventchain.com
      - CORE_PEER_ADDRESS=peer0.platform.eventchain.com:7051
      - CORE_PEER_LISTENADDRESS=0.0.0.0:7051
      - CORE_PEER_CHAINCODEADDRESS=peer0.platform.eventchain.com:7052
      - CORE_PEER_CHAINCODELISTENADDRESS=0.0.0.0:7052
      - CORE_PEER_GOSSIP_BOOTSTRAP=peer0.platform.eventchain.com:7051
      - CORE_PEER_GOSSIP_EXTERNALENDPOINT=peer0.platform.eventchain.com:7051
      - CORE_PEER_LOCALMSPID=PlatformMSP
      - CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/fabric/msp
      - CORE_OPERATIONS_LISTENADDRESS=peer0.platform.eventchain.com:9444
      - CORE_METRICS_PROVIDER=prometheus
      # CouchDB
      - CORE_LEDGER_STATE_STATEDATABASE=CouchDB
      - CORE_LEDGER_STATE_COUCHDBCONFIG_COUCHDBADDRESS=couchdb0:5984
      - CORE_LEDGER_STATE_COUCHDBCONFIG_USERNAME=admin
      - CORE_LEDGER_STATE_COUCHDBCONFIG_PASSWORD=adminpw
    depends_on:
      - couchdb0
    volumes:
      - ../organizations/peerOrganizations/platform.eventchain.com/peers/peer0.platform.eventchain.com:/etc/hyperledger/fabric
      - peer0.platform.eventchain.com:/var/hyperledger/production
      - ../../../config/core.yaml:/etc/hyperledger/peercfg/core.yaml
    working_dir: /root
    command: peer node start
    ports:
      - 7051:7051
      - 9444:9444
    networks:
      - eventchain_network

  # ============================================================
  # OrganizerOrg — peer0 + CouchDB
  # ============================================================
  couchdb1:
    container_name: couchdb1
    image: couchdb:${COUCH_IMAGE_TAG}
    labels:
      service: hyperledger-fabric
    environment:
      - COUCHDB_USER=admin
      - COUCHDB_PASSWORD=adminpw
    ports:
      - "7984:5984"
    networks:
      - eventchain_network

  peer0.organizer.eventchain.com:
    container_name: peer0.organizer.eventchain.com
    image: hyperledger/fabric-peer:${IMAGE_TAG}
    labels:
      service: hyperledger-fabric
    environment:
      - FABRIC_CFG_PATH=/etc/hyperledger/peercfg
      - FABRIC_LOGGING_SPEC=INFO
      #- FABRIC_LOGGING_SPEC=DEBUG
      - CORE_PEER_TLS_ENABLED=true
      - CORE_PEER_PROFILE_ENABLED=false
      - CORE_PEER_TLS_CERT_FILE=/etc/hyperledger/fabric/tls/server.crt
      - CORE_PEER_TLS_KEY_FILE=/etc/hyperledger/fabric/tls/server.key
      - CORE_PEER_TLS_ROOTCERT_FILE=/etc/hyperledger/fabric/tls/ca.crt
      # Peer specific
      - CORE_PEER_ID=peer0.organizer.eventchain.com
      - CORE_PEER_ADDRESS=peer0.organizer.eventchain.com:9051
      - CORE_PEER_LISTENADDRESS=0.0.0.0:9051
      - CORE_PEER_CHAINCODEADDRESS=peer0.organizer.eventchain.com:9052
      - CORE_PEER_CHAINCODELISTENADDRESS=0.0.0.0:9052
      - CORE_PEER_GOSSIP_BOOTSTRAP=peer0.organizer.eventchain.com:9051
      - CORE_PEER_GOSSIP_EXTERNALENDPOINT=peer0.organizer.eventchain.com:9051
      - CORE_PEER_LOCALMSPID=OrganizerMSP
      - CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/fabric/msp
      - CORE_OPERATIONS_LISTENADDRESS=peer0.organizer.eventchain.com:9445
      - CORE_METRICS_PROVIDER=prometheus
      # CouchDB
      - CORE_LEDGER_STATE_STATEDATABASE=CouchDB
      - CORE_LEDGER_STATE_COUCHDBCONFIG_COUCHDBADDRESS=couchdb1:5984
      - CORE_LEDGER_STATE_COUCHDBCONFIG_USERNAME=admin
      - CORE_LEDGER_STATE_COUCHDBCONFIG_PASSWORD=adminpw
    depends_on:
      - couchdb1
    volumes:
      - ../organizations/peerOrganizations/organizer.eventchain.com/peers/peer0.organizer.eventchain.com:/etc/hyperledger/fabric
      - peer0.organizer.eventchain.com:/var/hyperledger/production
      - ../../../config/core.yaml:/etc/hyperledger/peercfg/core.yaml
    working_dir: /root
    command: peer node start
    ports:
      - 9051:9051
      - 9445:9445
    networks:
      - eventchain_network

  # ============================================================
  # StudentOrg — peer0 + CouchDB
  # ============================================================
  couchdb2:
    container_name: couchdb2
    image: couchdb:${COUCH_IMAGE_TAG}
    labels:
      service: hyperledger-fabric
    environment:
      - COUCHDB_USER=admin
      - COUCHDB_PASSWORD=adminpw
    ports:
      - "9984:5984"
    networks:
      - eventchain_network

  peer0.student.eventchain.com:
    container_name: peer0.student.eventchain.com
    image: hyperledger/fabric-peer:${IMAGE_TAG}
    labels:
      service: hyperledger-fabric
    environment:
      - FABRIC_CFG_PATH=/etc/hyperledger/peercfg
      - FABRIC_LOGGING_SPEC=INFO
      #- FABRIC_LOGGING_SPEC=DEBUG
      - CORE_PEER_TLS_ENABLED=true
      - CORE_PEER_PROFILE_ENABLED=false
      - CORE_PEER_TLS_CERT_FILE=/etc/hyperledger/fabric/tls/server.crt
      - CORE_PEER_TLS_KEY_FILE=/etc/hyperledger/fabric/tls/server.key
      - CORE_PEER_TLS_ROOTCERT_FILE=/etc/hyperledger/fabric/tls/ca.crt
      # Peer specific
      - CORE_PEER_ID=peer0.student.eventchain.com
      - CORE_PEER_ADDRESS=peer0.student.eventchain.com:11051
      - CORE_PEER_LISTENADDRESS=0.0.0.0:11051
      - CORE_PEER_CHAINCODEADDRESS=peer0.student.eventchain.com:11052
      - CORE_PEER_CHAINCODELISTENADDRESS=0.0.0.0:11052
      - CORE_PEER_GOSSIP_BOOTSTRAP=peer0.student.eventchain.com:11051
      - CORE_PEER_GOSSIP_EXTERNALENDPOINT=peer0.student.eventchain.com:11051
      - CORE_PEER_LOCALMSPID=StudentMSP
      - CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/fabric/msp
      - CORE_OPERATIONS_LISTENADDRESS=peer0.student.eventchain.com:9446
      - CORE_METRICS_PROVIDER=prometheus
      # CouchDB
      - CORE_LEDGER_STATE_STATEDATABASE=CouchDB
      - CORE_LEDGER_STATE_COUCHDBCONFIG_COUCHDBADDRESS=couchdb2:5984
      - CORE_LEDGER_STATE_COUCHDBCONFIG_USERNAME=admin
      - CORE_LEDGER_STATE_COUCHDBCONFIG_PASSWORD=adminpw
    depends_on:
      - couchdb2
    volumes:
      - ../organizations/peerOrganizations/student.eventchain.com/peers/peer0.student.eventchain.com:/etc/hyperledger/fabric
      - peer0.student.eventchain.com:/var/hyperledger/production
      - ../../../config/core.yaml:/etc/hyperledger/peercfg/core.yaml
    working_dir: /root
    command: peer node start
    ports:
      - 11051:11051
      - 9446:9446
    networks:
      - eventchain_network
```

### 2c. CA Compose file

- [ ] Create `fabric/network/docker/docker-compose-ca.yaml` with the contents below
- [ ] Validate YAML syntax (`docker compose -f docker-compose-ca.yaml config`)

```yaml
# fabric/network/docker/docker-compose-ca.yaml
# EventChain Fabric network — Certificate Authorities for all orgs

version: '3.7'

networks:
  eventchain_network:
    name: eventchain_network

services:

  # ============================================================
  # Platform CA
  # ============================================================
  ca_platform:
    image: hyperledger/fabric-ca:${CA_IMAGE_TAG}
    labels:
      service: hyperledger-fabric
    environment:
      - FABRIC_CA_HOME=/etc/hyperledger/fabric-ca-server
      - FABRIC_CA_SERVER_CA_NAME=ca-platform
      - FABRIC_CA_SERVER_TLS_ENABLED=true
      - FABRIC_CA_SERVER_PORT=7054
      - FABRIC_CA_SERVER_OPERATIONS_LISTENADDRESS=0.0.0.0:17054
    ports:
      - "7054:7054"
      - "17054:17054"
    command: sh -c 'fabric-ca-server start -b admin:adminpw -d'
    volumes:
      - ../organizations/fabric-ca/platform:/etc/hyperledger/fabric-ca-server
    container_name: ca_platform
    networks:
      - eventchain_network

  # ============================================================
  # Organizer CA
  # ============================================================
  ca_organizer:
    image: hyperledger/fabric-ca:${CA_IMAGE_TAG}
    labels:
      service: hyperledger-fabric
    environment:
      - FABRIC_CA_HOME=/etc/hyperledger/fabric-ca-server
      - FABRIC_CA_SERVER_CA_NAME=ca-organizer
      - FABRIC_CA_SERVER_TLS_ENABLED=true
      - FABRIC_CA_SERVER_PORT=8054
      - FABRIC_CA_SERVER_OPERATIONS_LISTENADDRESS=0.0.0.0:18054
    ports:
      - "8054:8054"
      - "18054:18054"
    command: sh -c 'fabric-ca-server start -b admin:adminpw -d'
    volumes:
      - ../organizations/fabric-ca/organizer:/etc/hyperledger/fabric-ca-server
    container_name: ca_organizer
    networks:
      - eventchain_network

  # ============================================================
  # Student CA
  # ============================================================
  ca_student:
    image: hyperledger/fabric-ca:${CA_IMAGE_TAG}
    labels:
      service: hyperledger-fabric
    environment:
      - FABRIC_CA_HOME=/etc/hyperledger/fabric-ca-server
      - FABRIC_CA_SERVER_CA_NAME=ca-student
      - FABRIC_CA_SERVER_TLS_ENABLED=true
      - FABRIC_CA_SERVER_PORT=9054
      - FABRIC_CA_SERVER_OPERATIONS_LISTENADDRESS=0.0.0.0:19054
    ports:
      - "9054:9054"
      - "19054:19054"
    command: sh -c 'fabric-ca-server start -b admin:adminpw -d'
    volumes:
      - ../organizations/fabric-ca/student:/etc/hyperledger/fabric-ca-server
    container_name: ca_student
    networks:
      - eventchain_network

  # ============================================================
  # Orderer CA
  # ============================================================
  ca_orderer:
    image: hyperledger/fabric-ca:${CA_IMAGE_TAG}
    labels:
      service: hyperledger-fabric
    environment:
      - FABRIC_CA_HOME=/etc/hyperledger/fabric-ca-server
      - FABRIC_CA_SERVER_CA_NAME=ca-orderer
      - FABRIC_CA_SERVER_TLS_ENABLED=true
      - FABRIC_CA_SERVER_PORT=10054
      - FABRIC_CA_SERVER_OPERATIONS_LISTENADDRESS=0.0.0.0:20054
    ports:
      - "10054:10054"
      - "20054:20054"
    command: sh -c 'fabric-ca-server start -b admin:adminpw -d'
    volumes:
      - ../organizations/fabric-ca/ordererOrg:/etc/hyperledger/fabric-ca-server
    container_name: ca_orderer
    networks:
      - eventchain_network
```

---

## Task 3: Channel Configuration (configtx.yaml)

- [ ] Create `fabric/network/configtx/configtx.yaml` with the contents below
- [ ] Validate with `configtxgen -inspectBlock` after genesis block generation

```yaml
# fabric/network/configtx/configtx.yaml
# EventChain — Channel configuration for 3 orgs + Raft orderer

---
Organizations:

  - &OrdererOrg
    Name: OrdererOrg
    ID: OrdererMSP
    MSPDir: ../organizations/ordererOrganizations/eventchain.com/msp
    Policies:
      Readers:
        Type: Signature
        Rule: "OR('OrdererMSP.member')"
      Writers:
        Type: Signature
        Rule: "OR('OrdererMSP.member')"
      Admins:
        Type: Signature
        Rule: "OR('OrdererMSP.admin')"
    OrdererEndpoints:
      - orderer.eventchain.com:7050

  - &PlatformOrg
    Name: PlatformOrg
    ID: PlatformMSP
    MSPDir: ../organizations/peerOrganizations/platform.eventchain.com/msp
    Policies:
      Readers:
        Type: Signature
        Rule: "OR('PlatformMSP.admin', 'PlatformMSP.peer', 'PlatformMSP.client')"
      Writers:
        Type: Signature
        Rule: "OR('PlatformMSP.admin', 'PlatformMSP.client')"
      Admins:
        Type: Signature
        Rule: "OR('PlatformMSP.admin')"
      Endorsement:
        Type: Signature
        Rule: "OR('PlatformMSP.peer')"
    AnchorPeers:
      - Host: peer0.platform.eventchain.com
        Port: 7051

  - &OrganizerOrg
    Name: OrganizerOrg
    ID: OrganizerMSP
    MSPDir: ../organizations/peerOrganizations/organizer.eventchain.com/msp
    Policies:
      Readers:
        Type: Signature
        Rule: "OR('OrganizerMSP.admin', 'OrganizerMSP.peer', 'OrganizerMSP.client')"
      Writers:
        Type: Signature
        Rule: "OR('OrganizerMSP.admin', 'OrganizerMSP.client')"
      Admins:
        Type: Signature
        Rule: "OR('OrganizerMSP.admin')"
      Endorsement:
        Type: Signature
        Rule: "OR('OrganizerMSP.peer')"
    AnchorPeers:
      - Host: peer0.organizer.eventchain.com
        Port: 9051

  - &StudentOrg
    Name: StudentOrg
    ID: StudentMSP
    MSPDir: ../organizations/peerOrganizations/student.eventchain.com/msp
    Policies:
      Readers:
        Type: Signature
        Rule: "OR('StudentMSP.admin', 'StudentMSP.peer', 'StudentMSP.client')"
      Writers:
        Type: Signature
        Rule: "OR('StudentMSP.admin', 'StudentMSP.client')"
      Admins:
        Type: Signature
        Rule: "OR('StudentMSP.admin')"
      Endorsement:
        Type: Signature
        Rule: "OR('StudentMSP.peer')"
    AnchorPeers:
      - Host: peer0.student.eventchain.com
        Port: 11051

Capabilities:
  Channel: &ChannelCapabilities
    V2_0: true
  Orderer: &OrdererCapabilities
    V2_0: true
  Application: &ApplicationCapabilities
    V2_5: true

Application: &ApplicationDefaults
  Organizations:
  Policies:
    Readers:
      Type: ImplicitMeta
      Rule: "ANY Readers"
    Writers:
      Type: ImplicitMeta
      Rule: "ANY Writers"
    Admins:
      Type: ImplicitMeta
      Rule: "MAJORITY Admins"
    LifecycleEndorsement:
      Type: ImplicitMeta
      Rule: "MAJORITY Endorsement"
    Endorsement:
      Type: ImplicitMeta
      Rule: "MAJORITY Endorsement"
  Capabilities:
    <<: *ApplicationCapabilities

Orderer: &OrdererDefaults
  OrdererType: etcdraft
  Addresses:
    - orderer.eventchain.com:7050
  EtcdRaft:
    Consenters:
      - Host: orderer.eventchain.com
        Port: 7050
        ClientTLSCert: ../organizations/ordererOrganizations/eventchain.com/orderers/orderer.eventchain.com/tls/server.crt
        ServerTLSCert: ../organizations/ordererOrganizations/eventchain.com/orderers/orderer.eventchain.com/tls/server.crt
  BatchTimeout: 2s
  BatchSize:
    MaxMessageCount: 10
    AbsoluteMaxBytes: 99 MB
    PreferredMaxBytes: 512 KB
  Organizations:
  Policies:
    Readers:
      Type: ImplicitMeta
      Rule: "ANY Readers"
    Writers:
      Type: ImplicitMeta
      Rule: "ANY Writers"
    Admins:
      Type: ImplicitMeta
      Rule: "MAJORITY Admins"
    BlockValidation:
      Type: ImplicitMeta
      Rule: "ANY Writers"

Channel: &ChannelDefaults
  Policies:
    Readers:
      Type: ImplicitMeta
      Rule: "ANY Readers"
    Writers:
      Type: ImplicitMeta
      Rule: "ANY Writers"
    Admins:
      Type: ImplicitMeta
      Rule: "MAJORITY Admins"
  Capabilities:
    <<: *ChannelCapabilities

Profiles:

  EventChainGenesis:
    <<: *ChannelDefaults
    Orderer:
      <<: *OrdererDefaults
      Organizations:
        - *OrdererOrg
      Capabilities: *OrdererCapabilities
    Application:
      <<: *ApplicationDefaults
      Organizations:
        - *PlatformOrg
        - *OrganizerOrg
        - *StudentOrg
      Capabilities: *ApplicationCapabilities
```

---

## Task 4: CA Server Configurations

Create `fabric-ca-server-config.yaml` for each organization's CA. These are placed in the CA volume-mount directories so the CA server reads them on startup.

### 4a. Platform CA config

- [ ] Create `fabric/network/organizations/fabric-ca/platform/fabric-ca-server-config.yaml`

```yaml
# fabric/network/organizations/fabric-ca/platform/fabric-ca-server-config.yaml
# Fabric CA server configuration for PlatformOrg

version: 1.5.12

port: 7054

debug: false

crlsizelimit: 512000

tls:
  enabled: true
  certfile:
  keyfile:
  clientauth:
    type: noclientcert
    certfiles:

ca:
  name: ca-platform

csr:
  cn: ca.platform.eventchain.com
  keyrequest:
    algo: ecdsa
    size: 256
  names:
    - C: CN
      ST: Zhejiang
      L: Hangzhou
      O: PlatformOrg
      OU:
  hosts:
    - localhost
    - ca.platform.eventchain.com
    - ca_platform

registry:
  maxenrollments: -1
  identities:
    - name: admin
      pass: adminpw
      type: client
      affiliation: ""
      attrs:
        hf.Registrar.Roles: "*"
        hf.Registrar.DelegateRoles: "*"
        hf.Revoker: true
        hf.IntermediateCA: true
        hf.GenCRL: true
        hf.Registrar.Attributes: "*"
        hf.AffiliationMgr: true

db:
  type: sqlite3
  datasource: fabric-ca-server.db
  tls:
    enabled: false

affiliations:
  platform:
    - admin
    - peer

signing:
  default:
    usage:
      - digital signature
    expiry: 8760h
  profiles:
    ca:
      usage:
        - cert sign
        - crl sign
      expiry: 43800h
      caconstraint:
        isca: true
        maxpathlen: 0
    tls:
      usage:
        - signing
        - key encipherment
        - server auth
        - client auth
        - key agreement
      expiry: 8760h

bccsp:
  default: SW
  sw:
    hash: SHA2
    security: 256

operations:
  listenAddress: 0.0.0.0:17054
  tls:
    enabled: false

metrics:
  provider: prometheus
```

### 4b. Organizer CA config

- [ ] Create `fabric/network/organizations/fabric-ca/organizer/fabric-ca-server-config.yaml`

```yaml
# fabric/network/organizations/fabric-ca/organizer/fabric-ca-server-config.yaml
# Fabric CA server configuration for OrganizerOrg

version: 1.5.12

port: 8054

debug: false

crlsizelimit: 512000

tls:
  enabled: true
  certfile:
  keyfile:
  clientauth:
    type: noclientcert
    certfiles:

ca:
  name: ca-organizer

csr:
  cn: ca.organizer.eventchain.com
  keyrequest:
    algo: ecdsa
    size: 256
  names:
    - C: CN
      ST: Zhejiang
      L: Hangzhou
      O: OrganizerOrg
      OU:
  hosts:
    - localhost
    - ca.organizer.eventchain.com
    - ca_organizer

registry:
  maxenrollments: -1
  identities:
    - name: admin
      pass: adminpw
      type: client
      affiliation: ""
      attrs:
        hf.Registrar.Roles: "*"
        hf.Registrar.DelegateRoles: "*"
        hf.Revoker: true
        hf.IntermediateCA: true
        hf.GenCRL: true
        hf.Registrar.Attributes: "*"
        hf.AffiliationMgr: true

db:
  type: sqlite3
  datasource: fabric-ca-server.db
  tls:
    enabled: false

affiliations:
  organizer:
    - admin
    - peer

signing:
  default:
    usage:
      - digital signature
    expiry: 8760h
  profiles:
    ca:
      usage:
        - cert sign
        - crl sign
      expiry: 43800h
      caconstraint:
        isca: true
        maxpathlen: 0
    tls:
      usage:
        - signing
        - key encipherment
        - server auth
        - client auth
        - key agreement
      expiry: 8760h

bccsp:
  default: SW
  sw:
    hash: SHA2
    security: 256

operations:
  listenAddress: 0.0.0.0:18054
  tls:
    enabled: false

metrics:
  provider: prometheus
```

### 4c. Student CA config

- [ ] Create `fabric/network/organizations/fabric-ca/student/fabric-ca-server-config.yaml`

```yaml
# fabric/network/organizations/fabric-ca/student/fabric-ca-server-config.yaml
# Fabric CA server configuration for StudentOrg

version: 1.5.12

port: 9054

debug: false

crlsizelimit: 512000

tls:
  enabled: true
  certfile:
  keyfile:
  clientauth:
    type: noclientcert
    certfiles:

ca:
  name: ca-student

csr:
  cn: ca.student.eventchain.com
  keyrequest:
    algo: ecdsa
    size: 256
  names:
    - C: CN
      ST: Zhejiang
      L: Hangzhou
      O: StudentOrg
      OU:
  hosts:
    - localhost
    - ca.student.eventchain.com
    - ca_student

registry:
  maxenrollments: -1
  identities:
    - name: admin
      pass: adminpw
      type: client
      affiliation: ""
      attrs:
        hf.Registrar.Roles: "*"
        hf.Registrar.DelegateRoles: "*"
        hf.Revoker: true
        hf.IntermediateCA: true
        hf.GenCRL: true
        hf.Registrar.Attributes: "*"
        hf.AffiliationMgr: true

db:
  type: sqlite3
  datasource: fabric-ca-server.db
  tls:
    enabled: false

affiliations:
  student:
    - admin
    - peer

signing:
  default:
    usage:
      - digital signature
    expiry: 8760h
  profiles:
    ca:
      usage:
        - cert sign
        - crl sign
      expiry: 43800h
      caconstraint:
        isca: true
        maxpathlen: 0
    tls:
      usage:
        - signing
        - key encipherment
        - server auth
        - client auth
        - key agreement
      expiry: 8760h

bccsp:
  default: SW
  sw:
    hash: SHA2
    security: 256

operations:
  listenAddress: 0.0.0.0:19054
  tls:
    enabled: false

metrics:
  provider: prometheus
```

### 4d. Orderer CA config

- [ ] Create `fabric/network/organizations/fabric-ca/ordererOrg/fabric-ca-server-config.yaml`

```yaml
# fabric/network/organizations/fabric-ca/ordererOrg/fabric-ca-server-config.yaml
# Fabric CA server configuration for OrdererOrg

version: 1.5.12

port: 10054

debug: false

crlsizelimit: 512000

tls:
  enabled: true
  certfile:
  keyfile:
  clientauth:
    type: noclientcert
    certfiles:

ca:
  name: ca-orderer

csr:
  cn: ca.orderer.eventchain.com
  keyrequest:
    algo: ecdsa
    size: 256
  names:
    - C: CN
      ST: Zhejiang
      L: Hangzhou
      O: OrdererOrg
      OU:
  hosts:
    - localhost
    - ca.orderer.eventchain.com
    - ca_orderer

registry:
  maxenrollments: -1
  identities:
    - name: admin
      pass: adminpw
      type: client
      affiliation: ""
      attrs:
        hf.Registrar.Roles: "*"
        hf.Registrar.DelegateRoles: "*"
        hf.Revoker: true
        hf.IntermediateCA: true
        hf.GenCRL: true
        hf.Registrar.Attributes: "*"
        hf.AffiliationMgr: true

db:
  type: sqlite3
  datasource: fabric-ca-server.db
  tls:
    enabled: false

affiliations:
  orderer:
    - admin

signing:
  default:
    usage:
      - digital signature
    expiry: 8760h
  profiles:
    ca:
      usage:
        - cert sign
        - crl sign
      expiry: 43800h
      caconstraint:
        isca: true
        maxpathlen: 0
    tls:
      usage:
        - signing
        - key encipherment
        - server auth
        - client auth
        - key agreement
      expiry: 8760h

bccsp:
  default: SW
  sw:
    hash: SHA2
    security: 256

operations:
  listenAddress: 0.0.0.0:20054
  tls:
    enabled: false

metrics:
  provider: prometheus
```

---

## Task 5: Crypto Material Generation (CA-based)

Register and enroll all identities using Fabric CA rather than cryptogen. This script is called by `network.sh` during `up`.

- [ ] Create `fabric/network/organizations/registerEnroll.sh` with the contents below
- [ ] Verify the script generates the correct MSP directory structure for all 4 orgs

```bash
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
    Certificate: cacerts/localhost-${2}.pem
    OrganizationalUnitIdentifier: client
  PeerOUIdentifier:
    Certificate: cacerts/localhost-${2}.pem
    OrganizationalUnitIdentifier: peer
  AdminOUIdentifier:
    Certificate: cacerts/localhost-${2}.pem
    OrganizationalUnitIdentifier: admin
  OrdererOUIdentifier:
    Certificate: cacerts/localhost-${2}.pem
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

  createOrgMSP "platform.eventchain.com" "${CA_PORT}"

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

  createOrgMSP "organizer.eventchain.com" "${CA_PORT}"

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

  createOrgMSP "student.eventchain.com" "${CA_PORT}"

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
    Certificate: cacerts/localhost-${CA_PORT}.pem
    OrganizationalUnitIdentifier: client
  PeerOUIdentifier:
    Certificate: cacerts/localhost-${CA_PORT}.pem
    OrganizationalUnitIdentifier: peer
  AdminOUIdentifier:
    Certificate: cacerts/localhost-${CA_PORT}.pem
    OrganizationalUnitIdentifier: admin
  OrdererOUIdentifier:
    Certificate: cacerts/localhost-${CA_PORT}.pem
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
```

---

## Task 6: Helper Scripts

### 6a. Environment variables script

- [ ] Create `fabric/network/scripts/envVar.sh` with the contents below

```bash
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
```

### 6b. Channel creation script

- [ ] Create `fabric/network/scripts/createChannel.sh` with the contents below

```bash
#!/bin/bash
# fabric/network/scripts/createChannel.sh
# Create the 'eventchain' channel and join all peers
#
# Uses osnadmin CLI for channel participation API (Fabric 2.5 pattern — no system channel)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NETWORK_DIR="$(dirname "$SCRIPT_DIR")"

CHANNEL_NAME="${1:-eventchain}"
DELAY="${2:-3}"
MAX_RETRY="${3:-5}"

# Source environment variables
. "${SCRIPT_DIR}/envVar.sh"

export FABRIC_CFG_PATH="${NETWORK_DIR}/configtx"

# ============================================================
# Create channel genesis block using configtxgen
# ============================================================
createChannelGenesisBlock() {
  echo ""
  echo "Generating channel genesis block '${CHANNEL_NAME}.block'..."

  configtxgen \
    -profile EventChainGenesis \
    -outputBlock "${NETWORK_DIR}/channel-artifacts/${CHANNEL_NAME}.block" \
    -channelID "$CHANNEL_NAME"

  echo "Genesis block created: ${NETWORK_DIR}/channel-artifacts/${CHANNEL_NAME}.block"
}

# ============================================================
# Join orderer to channel via osnadmin
# ============================================================
joinOrdererToChannel() {
  echo ""
  echo "Joining orderer to channel '${CHANNEL_NAME}'..."

  osnadmin channel join \
    --channelID "$CHANNEL_NAME" \
    --config-block "${NETWORK_DIR}/channel-artifacts/${CHANNEL_NAME}.block" \
    -o localhost:7053 \
    --ca-file "$ORDERER_CA" \
    --client-cert "$ORDERER_ADMIN_TLS_SIGN_CERT" \
    --client-key "$ORDERER_ADMIN_TLS_PRIVATE_KEY"

  echo "Orderer joined channel '${CHANNEL_NAME}'"

  # Verify
  osnadmin channel list \
    -o localhost:7053 \
    --ca-file "$ORDERER_CA" \
    --client-cert "$ORDERER_ADMIN_TLS_SIGN_CERT" \
    --client-key "$ORDERER_ADMIN_TLS_PRIVATE_KEY"
}

# ============================================================
# Join a peer to the channel
# ============================================================
joinPeerToChannel() {
  local org="$1"
  setGlobals "$org"

  local counter=0
  while true; do
    peer channel join \
      -b "${NETWORK_DIR}/channel-artifacts/${CHANNEL_NAME}.block" && break

    counter=$((counter + 1))
    if [ "$counter" -ge "$MAX_RETRY" ]; then
      echo "ERROR: peer channel join failed for ${org} after ${MAX_RETRY} attempts"
      exit 1
    fi
    echo "Retrying in ${DELAY}s... (attempt ${counter}/${MAX_RETRY})"
    sleep "$DELAY"
  done

  echo "Peer for ${org} joined channel '${CHANNEL_NAME}'"
}

# ============================================================
# Set anchor peer for an org
# ============================================================
setAnchorPeer() {
  local org="$1"
  setGlobals "$org"

  echo "Setting anchor peer for ${org}..."

  # Fetch current channel config
  peer channel fetch config "${NETWORK_DIR}/channel-artifacts/config_block.pb" \
    -o localhost:7050 \
    --ordererTLSHostnameOverride orderer.eventchain.com \
    -c "$CHANNEL_NAME" \
    --tls \
    --cafile "$ORDERER_CA"

  cd "${NETWORK_DIR}/channel-artifacts"

  # Decode to JSON
  configtxlator proto_decode \
    --input config_block.pb \
    --type common.Block \
    --output config_block.json

  # Extract config
  jq '.data.data[0].payload.data.config' config_block.json > config.json

  # Determine anchor peer host and port
  local anchor_host anchor_port
  case "$org" in
    platform)
      anchor_host="peer0.platform.eventchain.com"
      anchor_port=7051
      ;;
    organizer)
      anchor_host="peer0.organizer.eventchain.com"
      anchor_port=9051
      ;;
    student)
      anchor_host="peer0.student.eventchain.com"
      anchor_port=11051
      ;;
  esac

  local msp_id
  msp_id="${CORE_PEER_LOCALMSPID}"

  # Modify config to add anchor peer
  jq --arg MSP "$msp_id" --arg HOST "$anchor_host" --argjson PORT "$anchor_port" \
    '.channel_group.groups.Application.groups[$MSP].values += {
      "AnchorPeers": {
        "mod_policy": "Admins",
        "value": {
          "anchor_peers": [{"host": $HOST, "port": $PORT}]
        },
        "version": "0"
      }
    }' config.json > modified_config.json

  # Encode original and modified configs
  configtxlator proto_encode \
    --input config.json \
    --type common.Config \
    --output config.pb

  configtxlator proto_encode \
    --input modified_config.json \
    --type common.Config \
    --output modified_config.pb

  # Compute update delta
  configtxlator compute_update \
    --channel_id "$CHANNEL_NAME" \
    --original config.pb \
    --updated modified_config.pb \
    --output config_update.pb

  configtxlator proto_decode \
    --input config_update.pb \
    --type common.ConfigUpdate \
    --output config_update.json

  # Wrap in envelope
  echo '{"payload":{"header":{"channel_header":{"channel_id":"'"$CHANNEL_NAME"'","type":2}},"data":{"config_update":'$(cat config_update.json)'}}}' | \
    jq . > config_update_in_envelope.json

  configtxlator proto_encode \
    --input config_update_in_envelope.json \
    --type common.Envelope \
    --output config_update_in_envelope.pb

  # Submit config update
  peer channel update \
    -f config_update_in_envelope.pb \
    -c "$CHANNEL_NAME" \
    -o localhost:7050 \
    --ordererTLSHostnameOverride orderer.eventchain.com \
    --tls \
    --cafile "$ORDERER_CA"

  echo "Anchor peer set for ${org}"

  # Clean up temp files
  rm -f config_block.pb config_block.json config.json modified_config.json \
        config.pb modified_config.pb config_update.pb config_update.json \
        config_update_in_envelope.json config_update_in_envelope.pb

  cd "$NETWORK_DIR"
}

# ============================================================
# Main
# ============================================================
echo ""
echo "============================================"
echo " EventChain — Channel Setup"
echo " Channel: ${CHANNEL_NAME}"
echo "============================================"

mkdir -p "${NETWORK_DIR}/channel-artifacts"

# Step 1: Generate genesis block
createChannelGenesisBlock

# Step 2: Join orderer
joinOrdererToChannel

# Step 3: Join all peers
echo ""
echo "Joining all peers to channel..."
joinPeerToChannel platform
joinPeerToChannel organizer
joinPeerToChannel student

# Step 4: Set anchor peers for each org
echo ""
echo "Setting anchor peers..."
setAnchorPeer platform
setAnchorPeer organizer
setAnchorPeer student

echo ""
echo "============================================"
echo " Channel '${CHANNEL_NAME}' ready"
echo " All 3 peers joined, anchor peers configured"
echo "============================================"
```

### 6c. Chaincode deployment script

- [ ] Create `fabric/network/scripts/deployCC.sh` with the contents below

```bash
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

  if [ "$CC_LANGUAGE" = "go" ]; then
    echo "Vendoring Go dependencies..."
    pushd "$CC_SRC_PATH" > /dev/null
    GO111MODULE=on go mod vendor
    popd > /dev/null
  fi

  peer lifecycle chaincode package "${CC_NAME}.tar.gz" \
    --path "$CC_SRC_PATH" \
    --lang "$CC_LANGUAGE" \
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

  setGlobals platform

  peer lifecycle chaincode checkcommitreadiness \
    --channelID "$CHANNEL_NAME" \
    --name "$CC_NAME" \
    --version "$CC_VERSION" \
    --sequence "$CC_SEQUENCE" \
    --signature-policy "$CC_END_POLICY" \
    --output json \
    $init_flag
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
```

---

## Task 7: Network Management Script (network.sh)

The main entry point for all network operations.

- [ ] Create `fabric/network/network.sh` with the contents below
- [ ] `chmod +x` on network.sh and all scripts in `scripts/`
- [ ] Test `./network.sh up`, `./network.sh createChannel`, `./network.sh down`

```bash
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
```

---

## Port Map Reference

Quick reference for all ports used in the network. Useful when debugging connectivity.

| Container | Service Port | Operations Port | Notes |
|-----------|-------------|----------------|-------|
| orderer.eventchain.com | 7050 | 9443 | Admin API on 7053 |
| peer0.platform.eventchain.com | 7051 | 9444 | Chaincode on 7052 |
| peer0.organizer.eventchain.com | 9051 | 9445 | Chaincode on 9052 |
| peer0.student.eventchain.com | 11051 | 9446 | Chaincode on 11052 |
| couchdb0 (PlatformOrg) | 5984 | — | admin/adminpw |
| couchdb1 (OrganizerOrg) | 7984 | — | admin/adminpw |
| couchdb2 (StudentOrg) | 9984 | — | admin/adminpw |
| ca_platform | 7054 | 17054 | |
| ca_organizer | 8054 | 18054 | |
| ca_student | 9054 | 19054 | |
| ca_orderer | 10054 | 20054 | |

---

## File Checklist

All files to create for the complete Fabric network infrastructure:

| # | File | Task |
|---|------|------|
| 1 | `fabric/network/prereqs.sh` | Task 1 |
| 2 | `fabric/network/docker/.env` | Task 2a |
| 3 | `fabric/network/docker/docker-compose-net.yaml` | Task 2b |
| 4 | `fabric/network/docker/docker-compose-ca.yaml` | Task 2c |
| 5 | `fabric/network/configtx/configtx.yaml` | Task 3 |
| 6 | `fabric/network/organizations/fabric-ca/platform/fabric-ca-server-config.yaml` | Task 4a |
| 7 | `fabric/network/organizations/fabric-ca/organizer/fabric-ca-server-config.yaml` | Task 4b |
| 8 | `fabric/network/organizations/fabric-ca/student/fabric-ca-server-config.yaml` | Task 4c |
| 9 | `fabric/network/organizations/fabric-ca/ordererOrg/fabric-ca-server-config.yaml` | Task 4d |
| 10 | `fabric/network/organizations/registerEnroll.sh` | Task 5 |
| 11 | `fabric/network/scripts/envVar.sh` | Task 6a |
| 12 | `fabric/network/scripts/createChannel.sh` | Task 6b |
| 13 | `fabric/network/scripts/deployCC.sh` | Task 6c |
| 14 | `fabric/network/network.sh` | Task 7 |

After creating all files, set executable permissions:
```bash
chmod +x fabric/network/network.sh
chmod +x fabric/network/prereqs.sh
chmod +x fabric/network/organizations/registerEnroll.sh
chmod +x fabric/network/scripts/*.sh
```
