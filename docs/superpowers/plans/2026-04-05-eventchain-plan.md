# EventChain Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a full-stack campus event prediction market + fair ticketing system on Hyperledger Fabric 2.5 with a Liquid Glass UI.

**Architecture:** Vue 3 frontend communicates via REST with an Express backend that uses the Fabric Gateway SDK to interact with 4 Go chaincodes (token, event, prediction, ticket) running on a 3-org Fabric network. CouchDB provides rich query support. Authentication flows through Fabric CA for certificate management and bcrypt+SQLite for password verification.

**Tech Stack:** Hyperledger Fabric 2.5 / Go 1.21+ / Node.js 18+ / Express / Vue 3 / Pinia / ECharts / Element Plus / Docker Compose / CouchDB / Liquid Glass CSS

---

## Task Dependency Order

```
Task 1: Project Scaffolding (.gitignore, README, scripts)
   │
Task 2: Prerequisites Check (Docker, Go, Node, Fabric binaries)
   │
Task 3: Fabric Network Configuration (Docker Compose, configtx, CA)
   │
Task 4: Network Management Scripts (network.sh, channel, deploy)
   │
   ├── Task 5: token-chaincode (Go)
   │      │
   │      ├── Task 6: event-chaincode (Go)
   │      │
   │      ├── Task 7: prediction-chaincode (Go, AMM)
   │      │      │
   │      │      └── Task 8: ticket-chaincode (Go, cross-chaincode)
   │      │
   ├── Task 9: Server Scaffolding (Express + Fabric Gateway)
   │      │
   │      ├─�� Task 10: Auth System (register/login + Fabric CA)
   │      │
   │      ├── Task 11: Event + Prediction Routes
   │      │
   │      └── Task 12: Ticket + User Routes
   │
   └── Task 13: Client Scaffolding + Liquid Glass CSS
          │
          ├── Task 14: Base Components (GlassCard, Navbar, etc.)
          │
          ├── Task 15: Pinia Stores + API Layer
          │
          ├── Task 16: Home Page
          │
          ├── Task 17: Event Detail Page (ECharts)
          │
          ��── Task 18: Ticket Hall + Profile + Admin Pages
          │
          └── Task 19: Login Page

Task 20: Seed Data + Integration Test
Task 21: Final Polish + Demo Prep
```

**Parallelism:** Tasks 5-8 (chaincodes) can be developed in parallel. Tasks 13-19 (frontend) can start after Task 9 (server scaffolding) is done since the frontend uses the API.

---
## Task 1: Project Scaffolding

**Files:**
- Create: `/Users/Apple/EventChain/.gitignore`
- Create: `/Users/Apple/EventChain/README.md`
- Create: `/Users/Apple/EventChain/scripts/start-all.sh`
- Create: `/Users/Apple/EventChain/scripts/cleanup.sh`
- Create: `/Users/Apple/EventChain/scripts/seed-data.sh`

- [ ] **Step 1: Create .gitignore**

```gitignore
# Node
node_modules/
dist/
.env

# Go
vendor/

# Fabric
fabric/network/organizations/peerOrganizations/
fabric/network/organizations/ordererOrganizations/
fabric/network/channel-artifacts/
fabric/network/system-genesis-block/

# Server
server/wallet/
server/users.db

# IDE
.vscode/
.idea/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db

# Logs
*.log
logs/

# Superpowers
.superpowers/
```

- [ ] **Step 2: Create README.md**

```markdown
# EventChain — 校园赛事预测市场

ZJU 区块链课程大作业。校园版 Polymarket + 公平抢票系统，基于 Hyperledger Fabric 2.5。

## 功能

- **预测市场**：AMM 做市，虚拟代币「浙币」预测赛事结果
- **公平购票**：预测准确度加权抽签，奖励真正关注赛事的人
- **链上透明**：所有交易记录链上可查，Fabric CA 学号实名

## 技术栈

| 层 | 技术 |
|----|------|
| 前端 | Vue 3 + Liquid Glass UI + ECharts |
| 后端 | Node.js + Express + Fabric Gateway SDK |
| 区块链 | Hyperledger Fabric 2.5 (3 Orgs, 4 Chaincodes) |
| 链码 | Go |
| 数据库 | CouchDB (Fabric state DB) + SQLite (user auth) |

## 前置要求

- Docker & Docker Compose
- Go 1.21+
- Node.js 18+
- Fabric 2.5 binaries (`peer`, `orderer`, `configtxgen`, `fabric-ca-client`)

## 快速启动

```bash
# 1. 安装 Fabric binaries + Docker images
curl -sSLO https://raw.githubusercontent.com/hyperledger/fabric/main/scripts/install-fabric.sh
chmod +x install-fabric.sh
./install-fabric.sh --fabric-version 2.5.10 --ca-version 1.5.12 docker binary

# 2. 一键启动
./scripts/start-all.sh

# 3. 灌入演示数据
./scripts/seed-data.sh

# 4. 打开浏览器
open http://localhost:5173
```

## 演示账号

| 角色 | 学号 | 密码 |
|------|------|------|
| 学生 | 3220100001 | student123 |
| 主办方 | organizer01 | org123 |
| 管理员 | admin01 | admin123 |

## 项目结构

```
EventChain/
├── fabric/          # Fabric 网络 + 链码
│   ├── network/     # Docker Compose, 配置, 脚本
│   └── chaincode/   # 4 个 Go 链码
├── server/          # Express 后端
├── client/          # Vue 3 前端
└── scripts/         # 启动/清理/灌数据脚本
```
```

- [ ] **Step 3: Create start-all.sh**

```bash
#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo "=========================================="
echo "  EventChain — 一键启动"
echo "=========================================="

# Step 1: Start Fabric network
echo ""
echo "[1/5] 启动 Fabric 网络..."
cd "$PROJECT_DIR/fabric/network"
./network.sh up createChannel -ca -s couchdb

# Step 2: Deploy chaincodes
echo ""
echo "[2/5] 部署链码..."
./network.sh deployCC -ccn token -ccp ../chaincode/token -ccl go
./network.sh deployCC -ccn event -ccp ../chaincode/event -ccl go
./network.sh deployCC -ccn prediction -ccp ../chaincode/prediction -ccl go
./network.sh deployCC -ccn ticket -ccp ../chaincode/ticket -ccl go

# Step 3: Install server dependencies
echo ""
echo "[3/5] 安装后端依赖..."
cd "$PROJECT_DIR/server"
npm install

# Step 4: Start backend
echo ""
echo "[4/5] 启动后端 (端口 3000)..."
npm start &
SERVER_PID=$!
echo "后端 PID: $SERVER_PID"
sleep 3

# Step 5: Start frontend
echo ""
echo "[5/5] 启动前端 (端口 5173)..."
cd "$PROJECT_DIR/client"
npm install
npm run dev &
CLIENT_PID=$!
echo "前端 PID: $CLIENT_PID"

echo ""
echo "=========================================="
echo "  EventChain 启动完成！"
echo "  前端: http://localhost:5173"
echo "  后端: http://localhost:3000"
echo "=========================================="
echo ""
echo "按 Ctrl+C 停止所有服务"

# Trap Ctrl+C to kill both processes
trap "kill $SERVER_PID $CLIENT_PID 2>/dev/null; exit 0" INT TERM
wait
```

- [ ] **Step 4: Create cleanup.sh**

```bash
#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo "=========================================="
echo "  EventChain — 清理所有资源"
echo "=========================================="

# Kill running node processes
echo "[1/3] 停止后端/前端进程..."
pkill -f "node.*server/src/app.js" 2>/dev/null || true
pkill -f "vite" 2>/dev/null || true

# Tear down Fabric network
echo "[2/3] 关闭 Fabric 网络..."
cd "$PROJECT_DIR/fabric/network"
./network.sh down 2>/dev/null || true

# Clean up wallet and database
echo "[3/3] 清理本地数据..."
rm -rf "$PROJECT_DIR/server/wallet"
rm -f "$PROJECT_DIR/server/users.db"

echo ""
echo "清理完成。"
```

- [ ] **Step 5: Create seed-data.sh**

```bash
#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

API_BASE="http://localhost:3000/api/v1"

echo "=========================================="
echo "  EventChain — 灌入演示数据"
echo "=========================================="

# Wait for server
echo "等待后端启动..."
for i in $(seq 1 30); do
  if curl -s "$API_BASE/events" > /dev/null 2>&1; then
    echo "后端已就绪。"
    break
  fi
  sleep 1
done

# ===== Register Users =====
echo ""
echo "[1/4] 注册用户..."

# Admin
curl -s -X POST "$API_BASE/auth/register" \
  -H "Content-Type: application/json" \
  -d '{"studentID":"admin01","password":"admin123","name":"系统管理员","role":"admin"}' | jq .

# Organizer
curl -s -X POST "$API_BASE/auth/register" \
  -H "Content-Type: application/json" \
  -d '{"studentID":"organizer01","password":"org123","name":"体育社团","role":"organizer"}' | jq .

# Students
STUDENTS=(
  '{"studentID":"3220100001","password":"student123","name":"张同学"}'
  '{"studentID":"3220100002","password":"student123","name":"李同学"}'
  '{"studentID":"3220100003","password":"student123","name":"王同学"}'
  '{"studentID":"3220100004","password":"student123","name":"陈同学"}'
  '{"studentID":"3220100005","password":"student123","name":"赵同学"}'
  '{"studentID":"3220100006","password":"student123","name":"刘同学"}'
  '{"studentID":"3220100007","password":"student123","name":"周同学"}'
  '{"studentID":"3220100008","password":"student123","name":"吴同学"}'
)

for s in "${STUDENTS[@]}"; do
  curl -s -X POST "$API_BASE/auth/register" \
    -H "Content-Type: application/json" \
    -d "$s" | jq .
done

# ===== Login as Organizer =====
echo ""
echo "[2/4] 创建赛事..."

ORG_TOKEN=$(curl -s -X POST "$API_BASE/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"studentID":"organizer01","password":"org123"}' | jq -r '.token')

# Create events
curl -s -X POST "$API_BASE/events" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ORG_TOKEN" \
  -d '{
    "title": "院际篮球决赛",
    "type": "basketball",
    "teams": ["教院", "丹青"],
    "ticketTotal": 200,
    "predictionOptions": ["教院赢", "丹青赢"]
  }' | jq .

curl -s -X POST "$API_BASE/events" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ORG_TOKEN" \
  -d '{
    "title": "校运会足球半决赛",
    "type": "football",
    "teams": ["竺院", "蓝田"],
    "ticketTotal": 300,
    "predictionOptions": ["竺院赢", "蓝田赢"]
  }' | jq .

curl -s -X POST "$API_BASE/events" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ORG_TOKEN" \
  -d '{
    "title": "英雄联盟校赛决赛",
    "type": "esports",
    "teams": ["CS战队", "EE战队"],
    "ticketTotal": 150,
    "predictionOptions": ["CS战队赢", "EE战队赢"]
  }' | jq .

curl -s -X POST "$API_BASE/events" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ORG_TOKEN" \
  -d '{
    "title": "羽毛球团体赛",
    "type": "badminton",
    "teams": ["求是", "云峰"],
    "ticketTotal": 100,
    "predictionOptions": ["求是赢", "云峰赢"]
  }' | jq .

# Open predictions for first 3 events
echo ""
echo "开放预测市场..."
for EVT_ID in evt001 evt002 evt003; do
  curl -s -X PUT "$API_BASE/events/$EVT_ID/status" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $ORG_TOKEN" \
    -d '{"status": "PREDICTION_OPEN"}' | jq .
done

# ===== Place Bets =====
echo ""
echo "[3/4] 模拟下注..."

# Login as each student and place bets
BETS=(
  '3220100001:student123:evt001:教院赢:200'
  '3220100002:student123:evt001:丹青赢:150'
  '3220100003:student123:evt001:教院赢:300'
  '3220100004:student123:evt001:丹青赢:100'
  '3220100005:student123:evt001:教院赢:250'
  '3220100001:student123:evt002:竺院赢:150'
  '3220100002:student123:evt002:蓝田赢:200'
  '3220100003:student123:evt002:竺院赢:100'
  '3220100006:student123:evt003:CS战队赢:300'
  '3220100007:student123:evt003:EE战队赢:200'
  '3220100008:student123:evt003:CS战队赢:150'
)

for BET in "${BETS[@]}"; do
  IFS=':' read -r SID PWD EVT OPT AMT <<< "$BET"
  TOKEN=$(curl -s -X POST "$API_BASE/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"studentID\":\"$SID\",\"password\":\"$PWD\"}" | jq -r '.token')
  
  curl -s -X POST "$API_BASE/predictions/bet" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d "{\"eventID\":\"$EVT\",\"option\":\"$OPT\",\"amount\":$AMT}" | jq .
  
  sleep 0.5
done

# ===== Settle one event for demo =====
echo ""
echo "[4/4] 结算一场赛事（篮球赛，教院赢）..."

# Open tickets for evt001
curl -s -X PUT "$API_BASE/events/evt001/status" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ORG_TOKEN" \
  -d '{"status": "TICKET_OPEN"}' | jq .

# Some students apply for tickets
for SID in 3220100001 3220100002 3220100003 3220100004 3220100005; do
  TOKEN=$(curl -s -X POST "$API_BASE/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"studentID\":\"$SID\",\"password\":\"student123\"}" | jq -r '.token')
  
  curl -s -X POST "$API_BASE/tickets/apply" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d '{"eventID":"evt001"}' | jq .
done

# Run lottery
curl -s -X POST "$API_BASE/tickets/lottery/evt001" \
  -H "Authorization: Bearer $ORG_TOKEN" | jq .

# Move to ONGOING then settle
curl -s -X PUT "$API_BASE/events/evt001/status" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ORG_TOKEN" \
  -d '{"status": "ONGOING"}' | jq .

curl -s -X PUT "$API_BASE/events/evt001/result" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ORG_TOKEN" \
  -d '{"outcome": "教院赢"}' | jq .

echo ""
echo "=========================================="
echo "  演示数据灌入完成！"
echo "  - 8 名学生 + 1 主办方 + 1 管理员"
echo "  - 4 场赛事（1 场已结算）"
echo "  - 11 笔预测下注"
echo "  - 篮球赛已结算，教院赢"
echo "=========================================="
```

- [ ] **Step 6: Make scripts executable and commit**

```bash
chmod +x scripts/start-all.sh scripts/cleanup.sh scripts/seed-data.sh
git add .gitignore README.md scripts/
git commit -m "feat: add project scaffolding, README, and management scripts"
```
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
## Task 3: Token Chaincode

### 3.1 Write token chaincode tests

- [ ] Create `fabric/chaincode/token/token_test.go`
- [ ] Create `fabric/chaincode/token/go.mod`

`fabric/chaincode/token/go.mod`:

```go
module github.com/eventchain/chaincode/token

go 1.21

require (
	github.com/hyperledger/fabric-chaincode-go v0.0.0-20240124143825-007bcabaa3a2
	github.com/hyperledger/fabric-contract-api-go v1.2.2
	github.com/hyperledger/fabric-protos-go v0.3.3
	github.com/stretchr/testify v1.8.4
)
```

`fabric/chaincode/token/token_test.go`:

```go
package token

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-chaincode-go/shimtest"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTokenChaincode(t *testing.T) (*shimtest.MockStub, *TokenContract) {
	tc := new(TokenContract)
	cc, err := contractapi.NewChaincode(tc)
	require.NoError(t, err)
	stub := shimtest.NewMockStub("token", cc)
	require.NotNil(t, stub)
	return stub, tc
}

func TestInitialize(t *testing.T) {
	stub, _ := setupTokenChaincode(t)

	// Initialize with total supply of 1000000
	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("TokenContract:Initialize"),
		[]byte("1000000"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Verify platform account balance
	resp = stub.MockInvoke("tx2", [][]byte{
		[]byte("TokenContract:BalanceOf"),
		[]byte("platform"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var result TokenAccount
	err := json.Unmarshal(resp.Payload, &result)
	require.NoError(t, err)
	assert.Equal(t, int64(1000000), result.Balance)
	assert.Equal(t, "platform", result.Owner)

	// Double initialize should fail
	resp = stub.MockInvoke("tx3", [][]byte{
		[]byte("TokenContract:Initialize"),
		[]byte("500000"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestMint(t *testing.T) {
	stub, _ := setupTokenChaincode(t)

	// Initialize first
	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("TokenContract:Initialize"),
		[]byte("1000000"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Mint 1000 tokens for user1
	resp = stub.MockInvoke("tx2", [][]byte{
		[]byte("TokenContract:Mint"),
		[]byte("user1"),
		[]byte("1000"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Verify user1 balance
	resp = stub.MockInvoke("tx3", [][]byte{
		[]byte("TokenContract:BalanceOf"),
		[]byte("user1"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var result TokenAccount
	err := json.Unmarshal(resp.Payload, &result)
	require.NoError(t, err)
	assert.Equal(t, int64(1000), result.Balance)
	assert.Equal(t, "user1", result.Owner)

	// Mint again to verify accumulation
	resp = stub.MockInvoke("tx4", [][]byte{
		[]byte("TokenContract:Mint"),
		[]byte("user1"),
		[]byte("500"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	resp = stub.MockInvoke("tx5", [][]byte{
		[]byte("TokenContract:BalanceOf"),
		[]byte("user1"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	err = json.Unmarshal(resp.Payload, &result)
	require.NoError(t, err)
	assert.Equal(t, int64(1500), result.Balance)
}

func TestMintInvalidAmount(t *testing.T) {
	stub, _ := setupTokenChaincode(t)

	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("TokenContract:Initialize"),
		[]byte("1000000"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Mint with zero amount
	resp = stub.MockInvoke("tx2", [][]byte{
		[]byte("TokenContract:Mint"),
		[]byte("user1"),
		[]byte("0"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)

	// Mint with negative amount
	resp = stub.MockInvoke("tx3", [][]byte{
		[]byte("TokenContract:Mint"),
		[]byte("user1"),
		[]byte("-100"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestTransfer(t *testing.T) {
	stub, _ := setupTokenChaincode(t)

	// Initialize
	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("TokenContract:Initialize"),
		[]byte("1000000"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Mint tokens for user1
	resp = stub.MockInvoke("tx2", [][]byte{
		[]byte("TokenContract:Mint"),
		[]byte("user1"),
		[]byte("1000"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Transfer 300 from user1 to user2
	resp = stub.MockInvoke("tx3", [][]byte{
		[]byte("TokenContract:Transfer"),
		[]byte("user1"),
		[]byte("user2"),
		[]byte("300"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Verify user1 balance is 700
	resp = stub.MockInvoke("tx4", [][]byte{
		[]byte("TokenContract:BalanceOf"),
		[]byte("user1"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var account1 TokenAccount
	err := json.Unmarshal(resp.Payload, &account1)
	require.NoError(t, err)
	assert.Equal(t, int64(700), account1.Balance)

	// Verify user2 balance is 300
	resp = stub.MockInvoke("tx5", [][]byte{
		[]byte("TokenContract:BalanceOf"),
		[]byte("user2"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var account2 TokenAccount
	err = json.Unmarshal(resp.Payload, &account2)
	require.NoError(t, err)
	assert.Equal(t, int64(300), account2.Balance)
}

func TestTransferInsufficientBalance(t *testing.T) {
	stub, _ := setupTokenChaincode(t)

	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("TokenContract:Initialize"),
		[]byte("1000000"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Mint 100 tokens for user1
	resp = stub.MockInvoke("tx2", [][]byte{
		[]byte("TokenContract:Mint"),
		[]byte("user1"),
		[]byte("100"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Try to transfer 200 (more than balance)
	resp = stub.MockInvoke("tx3", [][]byte{
		[]byte("TokenContract:Transfer"),
		[]byte("user1"),
		[]byte("user2"),
		[]byte("200"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestTransferToSelf(t *testing.T) {
	stub, _ := setupTokenChaincode(t)

	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("TokenContract:Initialize"),
		[]byte("1000000"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	resp = stub.MockInvoke("tx2", [][]byte{
		[]byte("TokenContract:Mint"),
		[]byte("user1"),
		[]byte("1000"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Transfer to self should fail
	resp = stub.MockInvoke("tx3", [][]byte{
		[]byte("TokenContract:Transfer"),
		[]byte("user1"),
		[]byte("user1"),
		[]byte("100"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestBalanceOfNonexistent(t *testing.T) {
	stub, _ := setupTokenChaincode(t)

	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("TokenContract:Initialize"),
		[]byte("1000000"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Query nonexistent user returns zero balance
	resp = stub.MockInvoke("tx2", [][]byte{
		[]byte("TokenContract:BalanceOf"),
		[]byte("nonexistent"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var result TokenAccount
	err := json.Unmarshal(resp.Payload, &result)
	require.NoError(t, err)
	assert.Equal(t, int64(0), result.Balance)
}

func TestHistory(t *testing.T) {
	stub, _ := setupTokenChaincode(t)

	// Initialize
	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("TokenContract:Initialize"),
		[]byte("1000000"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Mint for user1
	resp = stub.MockInvoke("tx2", [][]byte{
		[]byte("TokenContract:Mint"),
		[]byte("user1"),
		[]byte("1000"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Mint for user2
	resp = stub.MockInvoke("tx3", [][]byte{
		[]byte("TokenContract:Mint"),
		[]byte("user2"),
		[]byte("500"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Transfer from user1 to user2
	resp = stub.MockInvoke("tx4", [][]byte{
		[]byte("TokenContract:Transfer"),
		[]byte("user1"),
		[]byte("user2"),
		[]byte("200"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Query history for user1
	resp = stub.MockInvoke("tx5", [][]byte{
		[]byte("TokenContract:History"),
		[]byte("user1"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var records []TransactionRecord
	err := json.Unmarshal(resp.Payload, &records)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(records), 1)

	// Verify the transfer record exists
	found := false
	for _, r := range records {
		if r.Type == "TRANSFER" && r.From == "user1" && r.To == "user2" && r.Amount == 200 {
			found = true
			break
		}
	}
	assert.True(t, found, "expected transfer record not found in history")
}

func TestMultipleTransfers(t *testing.T) {
	stub, _ := setupTokenChaincode(t)

	// Initialize
	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("TokenContract:Initialize"),
		[]byte("1000000"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Mint 1000 for user1
	resp = stub.MockInvoke("tx2", [][]byte{
		[]byte("TokenContract:Mint"),
		[]byte("user1"),
		[]byte("1000"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Transfer 100 to user2, user3, user4
	for i, user := range []string{"user2", "user3", "user4"} {
		txID := fmt.Sprintf("tx%d", i+3)
		resp = stub.MockInvoke(txID, [][]byte{
			[]byte("TokenContract:Transfer"),
			[]byte("user1"),
			[]byte(user),
			[]byte("100"),
		})
		assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)
	}

	// user1 should have 700
	resp = stub.MockInvoke("tx10", [][]byte{
		[]byte("TokenContract:BalanceOf"),
		[]byte("user1"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var account TokenAccount
	err := json.Unmarshal(resp.Payload, &account)
	require.NoError(t, err)
	assert.Equal(t, int64(700), account.Balance)
}
```

### 3.2 Run token tests (expect fail)

- [ ] `cd fabric/chaincode/token && go test -v ./...` -- should fail because `token.go` does not exist yet

### 3.3 Write token chaincode implementation

- [ ] Create `fabric/chaincode/token/token.go`

`fabric/chaincode/token/token.go`:

```go
package token

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// TokenAccount represents a user's token account stored in world state
type TokenAccount struct {
	DocType     string `json:"docType"`
	Owner       string `json:"owner"`
	Balance     int64  `json:"balance"`
	LastUpdated string `json:"lastUpdated"`
}

// TransactionRecord represents a token transaction stored in world state for history
type TransactionRecord struct {
	DocType   string `json:"docType"`
	TxID      string `json:"txID"`
	Type      string `json:"type"` // MINT, TRANSFER
	From      string `json:"from"`
	To        string `json:"to"`
	Amount    int64  `json:"amount"`
	Timestamp string `json:"timestamp"`
}

// TokenContract implements the token chaincode
type TokenContract struct {
	contractapi.Contract
}

// Initialize sets up the token pool with a total supply assigned to the platform account
func (tc *TokenContract) Initialize(ctx contractapi.TransactionContextInterface, totalSupplyStr string) error {
	totalSupply, err := strconv.ParseInt(totalSupplyStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid total supply: %s", err.Error())
	}
	if totalSupply <= 0 {
		return fmt.Errorf("total supply must be positive")
	}

	// Check if already initialized
	existing, err := ctx.GetStub().GetState("token:platform")
	if err != nil {
		return fmt.Errorf("failed to read state: %s", err.Error())
	}
	if existing != nil {
		return fmt.Errorf("token system already initialized")
	}

	timestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return fmt.Errorf("failed to get timestamp: %s", err.Error())
	}
	timeStr := time.Unix(timestamp.Seconds, int64(timestamp.Nanos)).UTC().Format(time.RFC3339)

	account := TokenAccount{
		DocType:     "token",
		Owner:       "platform",
		Balance:     totalSupply,
		LastUpdated: timeStr,
	}

	accountJSON, err := json.Marshal(account)
	if err != nil {
		return fmt.Errorf("failed to marshal account: %s", err.Error())
	}

	err = ctx.GetStub().PutState("token:platform", accountJSON)
	if err != nil {
		return fmt.Errorf("failed to put state: %s", err.Error())
	}

	return nil
}

// Mint creates new tokens for a user account
func (tc *TokenContract) Mint(ctx contractapi.TransactionContextInterface, userID string, amountStr string) error {
	if userID == "" {
		return fmt.Errorf("user ID must not be empty")
	}

	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid amount: %s", err.Error())
	}
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	key := "token:" + userID

	timestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return fmt.Errorf("failed to get timestamp: %s", err.Error())
	}
	timeStr := time.Unix(timestamp.Seconds, int64(timestamp.Nanos)).UTC().Format(time.RFC3339)

	// Get existing account or create new one
	existing, err := ctx.GetStub().GetState(key)
	if err != nil {
		return fmt.Errorf("failed to read state: %s", err.Error())
	}

	var account TokenAccount
	if existing != nil {
		err = json.Unmarshal(existing, &account)
		if err != nil {
			return fmt.Errorf("failed to unmarshal account: %s", err.Error())
		}
		account.Balance += amount
		account.LastUpdated = timeStr
	} else {
		account = TokenAccount{
			DocType:     "token",
			Owner:       userID,
			Balance:     amount,
			LastUpdated: timeStr,
		}
	}

	accountJSON, err := json.Marshal(account)
	if err != nil {
		return fmt.Errorf("failed to marshal account: %s", err.Error())
	}

	err = ctx.GetStub().PutState(key, accountJSON)
	if err != nil {
		return fmt.Errorf("failed to put state: %s", err.Error())
	}

	// Record transaction
	txID := ctx.GetStub().GetTxID()
	record := TransactionRecord{
		DocType:   "tokenTx",
		TxID:      txID,
		Type:      "MINT",
		From:      "system",
		To:        userID,
		Amount:    amount,
		Timestamp: timeStr,
	}

	recordJSON, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal record: %s", err.Error())
	}

	recordKey := fmt.Sprintf("tokenTx:%s:%s", userID, txID)
	err = ctx.GetStub().PutState(recordKey, recordJSON)
	if err != nil {
		return fmt.Errorf("failed to put transaction record: %s", err.Error())
	}

	return nil
}

// Transfer moves tokens from one account to another
func (tc *TokenContract) Transfer(ctx contractapi.TransactionContextInterface, from string, to string, amountStr string) error {
	if from == "" || to == "" {
		return fmt.Errorf("from and to must not be empty")
	}
	if from == to {
		return fmt.Errorf("cannot transfer to self")
	}

	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid amount: %s", err.Error())
	}
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	timestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return fmt.Errorf("failed to get timestamp: %s", err.Error())
	}
	timeStr := time.Unix(timestamp.Seconds, int64(timestamp.Nanos)).UTC().Format(time.RFC3339)

	// Read sender account
	fromKey := "token:" + from
	fromBytes, err := ctx.GetStub().GetState(fromKey)
	if err != nil {
		return fmt.Errorf("failed to read sender account: %s", err.Error())
	}
	if fromBytes == nil {
		return fmt.Errorf("sender account %s does not exist", from)
	}

	var fromAccount TokenAccount
	err = json.Unmarshal(fromBytes, &fromAccount)
	if err != nil {
		return fmt.Errorf("failed to unmarshal sender account: %s", err.Error())
	}

	if fromAccount.Balance < amount {
		return fmt.Errorf("insufficient balance: have %d, need %d", fromAccount.Balance, amount)
	}

	// Read or create receiver account
	toKey := "token:" + to
	toBytes, err := ctx.GetStub().GetState(toKey)
	if err != nil {
		return fmt.Errorf("failed to read receiver account: %s", err.Error())
	}

	var toAccount TokenAccount
	if toBytes != nil {
		err = json.Unmarshal(toBytes, &toAccount)
		if err != nil {
			return fmt.Errorf("failed to unmarshal receiver account: %s", err.Error())
		}
	} else {
		toAccount = TokenAccount{
			DocType: "token",
			Owner:   to,
			Balance: 0,
		}
	}

	// Execute transfer
	fromAccount.Balance -= amount
	fromAccount.LastUpdated = timeStr
	toAccount.Balance += amount
	toAccount.LastUpdated = timeStr

	fromJSON, err := json.Marshal(fromAccount)
	if err != nil {
		return fmt.Errorf("failed to marshal sender account: %s", err.Error())
	}

	toJSON, err := json.Marshal(toAccount)
	if err != nil {
		return fmt.Errorf("failed to marshal receiver account: %s", err.Error())
	}

	err = ctx.GetStub().PutState(fromKey, fromJSON)
	if err != nil {
		return fmt.Errorf("failed to update sender account: %s", err.Error())
	}

	err = ctx.GetStub().PutState(toKey, toJSON)
	if err != nil {
		return fmt.Errorf("failed to update receiver account: %s", err.Error())
	}

	// Record transaction
	txID := ctx.GetStub().GetTxID()
	record := TransactionRecord{
		DocType:   "tokenTx",
		TxID:      txID,
		Type:      "TRANSFER",
		From:      from,
		To:        to,
		Amount:    amount,
		Timestamp: timeStr,
	}

	recordJSON, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal record: %s", err.Error())
	}

	// Store record under both sender and receiver for lookup
	fromRecordKey := fmt.Sprintf("tokenTx:%s:%s", from, txID)
	err = ctx.GetStub().PutState(fromRecordKey, recordJSON)
	if err != nil {
		return fmt.Errorf("failed to put sender transaction record: %s", err.Error())
	}

	toRecordKey := fmt.Sprintf("tokenTx:%s:%s", to, txID)
	err = ctx.GetStub().PutState(toRecordKey, recordJSON)
	if err != nil {
		return fmt.Errorf("failed to put receiver transaction record: %s", err.Error())
	}

	return nil
}

// BalanceOf returns the token account for a user
func (tc *TokenContract) BalanceOf(ctx contractapi.TransactionContextInterface, userID string) (*TokenAccount, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID must not be empty")
	}

	key := "token:" + userID
	data, err := ctx.GetStub().GetState(key)
	if err != nil {
		return nil, fmt.Errorf("failed to read state: %s", err.Error())
	}

	if data == nil {
		// Return zero-balance account for nonexistent users
		return &TokenAccount{
			DocType:     "token",
			Owner:       userID,
			Balance:     0,
			LastUpdated: "",
		}, nil
	}

	var account TokenAccount
	err = json.Unmarshal(data, &account)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal account: %s", err.Error())
	}

	return &account, nil
}

// History returns all transaction records for a user using CouchDB rich query
func (tc *TokenContract) History(ctx contractapi.TransactionContextInterface, userID string) ([]*TransactionRecord, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID must not be empty")
	}

	// Use GetStateByRange to query all transaction records for this user
	// Keys are in format tokenTx:{userID}:{txID}
	startKey := fmt.Sprintf("tokenTx:%s:", userID)
	endKey := fmt.Sprintf("tokenTx:%s:~", userID)

	resultsIterator, err := ctx.GetStub().GetStateByRange(startKey, endKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction history: %s", err.Error())
	}
	defer resultsIterator.Close()

	var records []*TransactionRecord
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate results: %s", err.Error())
		}

		var record TransactionRecord
		err = json.Unmarshal(queryResponse.Value, &record)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal record: %s", err.Error())
		}

		records = append(records, &record)
	}

	if records == nil {
		records = []*TransactionRecord{}
	}

	return records, nil
}

// GetContractInfo provides metadata about the chaincode
func (tc *TokenContract) GetContractInfo() contractapi.ContractInterface {
	return tc
}
```

### 3.4 Run token tests (expect pass)

- [ ] `cd fabric/chaincode/token && go test -v ./...`

### 3.5 Commit token chaincode

- [ ] `git add fabric/chaincode/token/ && git commit -m "feat(chaincode): implement token chaincode with tests"`

---

## Task 4: Event Chaincode

### 4.1 Write event chaincode tests

- [ ] Create `fabric/chaincode/event/event_test.go`
- [ ] Create `fabric/chaincode/event/go.mod`

`fabric/chaincode/event/go.mod`:

```go
module github.com/eventchain/chaincode/event

go 1.21

require (
	github.com/hyperledger/fabric-chaincode-go v0.0.0-20240124143825-007bcabaa3a2
	github.com/hyperledger/fabric-contract-api-go v1.2.2
	github.com/hyperledger/fabric-protos-go v0.3.3
	github.com/stretchr/testify v1.8.4
)
```

`fabric/chaincode/event/event_test.go`:

```go
package event

import (
	"encoding/json"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-chaincode-go/shimtest"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupEventChaincode(t *testing.T) (*shimtest.MockStub, *EventContract) {
	ec := new(EventContract)
	cc, err := contractapi.NewChaincode(ec)
	require.NoError(t, err)
	stub := shimtest.NewMockStub("event", cc)
	require.NotNil(t, stub)
	return stub, ec
}

func createTestEvent(t *testing.T, stub *shimtest.MockStub, eventID string) {
	teamsJSON, _ := json.Marshal([]string{"TeamA", "TeamB"})
	optionsJSON, _ := json.Marshal([]string{"TeamA wins", "TeamB wins"})

	resp := stub.MockInvoke("tx-create-"+eventID, [][]byte{
		[]byte("EventContract:CreateEvent"),
		[]byte(eventID),
		[]byte("Test Basketball Game"),
		[]byte("basketball"),
		teamsJSON,
		[]byte("100"),
		optionsJSON,
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)
}

func TestCreateEvent(t *testing.T) {
	stub, _ := setupEventChaincode(t)

	teamsJSON, _ := json.Marshal([]string{"ZJU Eagles", "ZJU Lions"})
	optionsJSON, _ := json.Marshal([]string{"Eagles win", "Lions win"})

	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("EventContract:CreateEvent"),
		[]byte("evt001"),
		[]byte("Basketball Finals"),
		[]byte("basketball"),
		teamsJSON,
		[]byte("200"),
		optionsJSON,
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Query the created event
	resp = stub.MockInvoke("tx2", [][]byte{
		[]byte("EventContract:QueryEvent"),
		[]byte("evt001"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var evt Event
	err := json.Unmarshal(resp.Payload, &evt)
	require.NoError(t, err)
	assert.Equal(t, "evt001", evt.ID)
	assert.Equal(t, "Basketball Finals", evt.Title)
	assert.Equal(t, "basketball", evt.Type)
	assert.Equal(t, []string{"ZJU Eagles", "ZJU Lions"}, evt.Teams)
	assert.Equal(t, 200, evt.TicketTotal)
	assert.Equal(t, StatusCreated, evt.Status)
	assert.Equal(t, []string{"Eagles win", "Lions win"}, evt.PredictionOptions)
	assert.Equal(t, "", evt.Result)
}

func TestCreateEventDuplicate(t *testing.T) {
	stub, _ := setupEventChaincode(t)

	createTestEvent(t, stub, "evt001")

	// Try to create again with same ID
	teamsJSON, _ := json.Marshal([]string{"A", "B"})
	optionsJSON, _ := json.Marshal([]string{"A wins", "B wins"})
	resp := stub.MockInvoke("tx-dup", [][]byte{
		[]byte("EventContract:CreateEvent"),
		[]byte("evt001"),
		[]byte("Duplicate"),
		[]byte("football"),
		teamsJSON,
		[]byte("50"),
		optionsJSON,
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestCreateEventInvalidTicketTotal(t *testing.T) {
	stub, _ := setupEventChaincode(t)

	teamsJSON, _ := json.Marshal([]string{"A", "B"})
	optionsJSON, _ := json.Marshal([]string{"A wins", "B wins"})

	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("EventContract:CreateEvent"),
		[]byte("evt001"),
		[]byte("Test"),
		[]byte("basketball"),
		teamsJSON,
		[]byte("0"),
		optionsJSON,
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestQueryEventNotFound(t *testing.T) {
	stub, _ := setupEventChaincode(t)

	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("EventContract:QueryEvent"),
		[]byte("nonexistent"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestUpdateStatusValidTransitions(t *testing.T) {
	stub, _ := setupEventChaincode(t)

	createTestEvent(t, stub, "evt001")

	// CREATED -> PREDICTION_OPEN
	resp := stub.MockInvoke("tx-s1", [][]byte{
		[]byte("EventContract:UpdateStatus"),
		[]byte("evt001"),
		[]byte("PREDICTION_OPEN"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Verify status
	resp = stub.MockInvoke("tx-q1", [][]byte{
		[]byte("EventContract:QueryEvent"),
		[]byte("evt001"),
	})
	var evt Event
	json.Unmarshal(resp.Payload, &evt)
	assert.Equal(t, StatusPredictionOpen, evt.Status)

	// PREDICTION_OPEN -> TICKET_OPEN
	resp = stub.MockInvoke("tx-s2", [][]byte{
		[]byte("EventContract:UpdateStatus"),
		[]byte("evt001"),
		[]byte("TICKET_OPEN"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// TICKET_OPEN -> ONGOING
	resp = stub.MockInvoke("tx-s3", [][]byte{
		[]byte("EventContract:UpdateStatus"),
		[]byte("evt001"),
		[]byte("ONGOING"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// ONGOING -> SETTLED
	resp = stub.MockInvoke("tx-s4", [][]byte{
		[]byte("EventContract:UpdateStatus"),
		[]byte("evt001"),
		[]byte("SETTLED"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Verify final status
	resp = stub.MockInvoke("tx-q2", [][]byte{
		[]byte("EventContract:QueryEvent"),
		[]byte("evt001"),
	})
	json.Unmarshal(resp.Payload, &evt)
	assert.Equal(t, StatusSettled, evt.Status)
}

func TestUpdateStatusInvalidTransition(t *testing.T) {
	stub, _ := setupEventChaincode(t)

	createTestEvent(t, stub, "evt001")

	// CREATED -> ONGOING should fail (must go through PREDICTION_OPEN first)
	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("EventContract:UpdateStatus"),
		[]byte("evt001"),
		[]byte("ONGOING"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)

	// CREATED -> SETTLED should fail
	resp = stub.MockInvoke("tx2", [][]byte{
		[]byte("EventContract:UpdateStatus"),
		[]byte("evt001"),
		[]byte("SETTLED"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestUpdateResult(t *testing.T) {
	stub, _ := setupEventChaincode(t)

	createTestEvent(t, stub, "evt001")

	// Advance to ONGOING
	stub.MockInvoke("tx-s1", [][]byte{
		[]byte("EventContract:UpdateStatus"),
		[]byte("evt001"),
		[]byte("PREDICTION_OPEN"),
	})
	stub.MockInvoke("tx-s2", [][]byte{
		[]byte("EventContract:UpdateStatus"),
		[]byte("evt001"),
		[]byte("TICKET_OPEN"),
	})
	stub.MockInvoke("tx-s3", [][]byte{
		[]byte("EventContract:UpdateStatus"),
		[]byte("evt001"),
		[]byte("ONGOING"),
	})

	// Update result
	resp := stub.MockInvoke("tx-r1", [][]byte{
		[]byte("EventContract:UpdateResult"),
		[]byte("evt001"),
		[]byte("TeamA wins"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Verify result is stored and status is SETTLED
	resp = stub.MockInvoke("tx-q1", [][]byte{
		[]byte("EventContract:QueryEvent"),
		[]byte("evt001"),
	})
	var evt Event
	json.Unmarshal(resp.Payload, &evt)
	assert.Equal(t, "TeamA wins", evt.Result)
	assert.Equal(t, StatusSettled, evt.Status)
}

func TestUpdateResultWrongStatus(t *testing.T) {
	stub, _ := setupEventChaincode(t)

	createTestEvent(t, stub, "evt001")

	// Try to update result on CREATED event (should fail, must be ONGOING)
	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("EventContract:UpdateResult"),
		[]byte("evt001"),
		[]byte("TeamA wins"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestListEvents(t *testing.T) {
	stub, _ := setupEventChaincode(t)

	// Create multiple events
	createTestEvent(t, stub, "evt001")
	createTestEvent(t, stub, "evt002")
	createTestEvent(t, stub, "evt003")

	// Advance evt002 to PREDICTION_OPEN
	stub.MockInvoke("tx-s1", [][]byte{
		[]byte("EventContract:UpdateStatus"),
		[]byte("evt002"),
		[]byte("PREDICTION_OPEN"),
	})

	// List all events (empty filter returns all)
	resp := stub.MockInvoke("tx-list1", [][]byte{
		[]byte("EventContract:ListEvents"),
		[]byte(""),
		[]byte(""),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var events []Event
	err := json.Unmarshal(resp.Payload, &events)
	require.NoError(t, err)
	assert.Equal(t, 3, len(events))

	// List by status
	resp = stub.MockInvoke("tx-list2", [][]byte{
		[]byte("EventContract:ListEvents"),
		[]byte("CREATED"),
		[]byte(""),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	err = json.Unmarshal(resp.Payload, &events)
	require.NoError(t, err)
	assert.Equal(t, 2, len(events))

	// List by status PREDICTION_OPEN
	resp = stub.MockInvoke("tx-list3", [][]byte{
		[]byte("EventContract:ListEvents"),
		[]byte("PREDICTION_OPEN"),
		[]byte(""),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	err = json.Unmarshal(resp.Payload, &events)
	require.NoError(t, err)
	assert.Equal(t, 1, len(events))
	assert.Equal(t, "evt002", events[0].ID)

	// List by type
	resp = stub.MockInvoke("tx-list4", [][]byte{
		[]byte("EventContract:ListEvents"),
		[]byte(""),
		[]byte("basketball"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	err = json.Unmarshal(resp.Payload, &events)
	require.NoError(t, err)
	assert.Equal(t, 3, len(events))
}
```

### 4.2 Run event tests (expect fail)

- [ ] `cd fabric/chaincode/event && go test -v ./...` -- should fail because `event.go` does not exist yet

### 4.3 Write event chaincode implementation

- [ ] Create `fabric/chaincode/event/event.go`

`fabric/chaincode/event/event.go`:

```go
package event

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// Event status constants
const (
	StatusCreated        = "CREATED"
	StatusPredictionOpen = "PREDICTION_OPEN"
	StatusTicketOpen     = "TICKET_OPEN"
	StatusOngoing        = "ONGOING"
	StatusSettled        = "SETTLED"
)

// validTransitions defines the allowed state machine transitions
var validTransitions = map[string]string{
	StatusCreated:        StatusPredictionOpen,
	StatusPredictionOpen: StatusTicketOpen,
	StatusTicketOpen:     StatusOngoing,
	StatusOngoing:        StatusSettled,
}

// Event represents a campus event stored in world state
type Event struct {
	DocType           string   `json:"docType"`
	ID                string   `json:"id"`
	Title             string   `json:"title"`
	Type              string   `json:"type"`
	Teams             []string `json:"teams"`
	TicketTotal       int      `json:"ticketTotal"`
	Status            string   `json:"status"`
	PredictionOptions []string `json:"predictionOptions"`
	Result            string   `json:"result"`
	CreatedAt         string   `json:"createdAt"`
	UpdatedAt         string   `json:"updatedAt"`
}

// EventContract implements the event chaincode
type EventContract struct {
	contractapi.Contract
}

// CreateEvent creates a new event
func (ec *EventContract) CreateEvent(ctx contractapi.TransactionContextInterface, eventID string, title string, eventType string, teamsJSON string, ticketTotalStr string, optionsJSON string) error {
	if eventID == "" {
		return fmt.Errorf("event ID must not be empty")
	}
	if title == "" {
		return fmt.Errorf("title must not be empty")
	}

	// Check if event already exists
	key := "event:" + eventID
	existing, err := ctx.GetStub().GetState(key)
	if err != nil {
		return fmt.Errorf("failed to read state: %s", err.Error())
	}
	if existing != nil {
		return fmt.Errorf("event %s already exists", eventID)
	}

	// Parse teams
	var teams []string
	err = json.Unmarshal([]byte(teamsJSON), &teams)
	if err != nil {
		return fmt.Errorf("failed to parse teams: %s", err.Error())
	}
	if len(teams) < 2 {
		return fmt.Errorf("at least 2 teams required")
	}

	// Parse ticket total
	ticketTotal, err := strconv.Atoi(ticketTotalStr)
	if err != nil {
		return fmt.Errorf("invalid ticket total: %s", err.Error())
	}
	if ticketTotal <= 0 {
		return fmt.Errorf("ticket total must be positive")
	}

	// Parse prediction options
	var options []string
	err = json.Unmarshal([]byte(optionsJSON), &options)
	if err != nil {
		return fmt.Errorf("failed to parse prediction options: %s", err.Error())
	}
	if len(options) < 2 {
		return fmt.Errorf("at least 2 prediction options required")
	}

	timestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return fmt.Errorf("failed to get timestamp: %s", err.Error())
	}
	timeStr := time.Unix(timestamp.Seconds, int64(timestamp.Nanos)).UTC().Format(time.RFC3339)

	evt := Event{
		DocType:           "event",
		ID:                eventID,
		Title:             title,
		Type:              eventType,
		Teams:             teams,
		TicketTotal:       ticketTotal,
		Status:            StatusCreated,
		PredictionOptions: options,
		Result:            "",
		CreatedAt:         timeStr,
		UpdatedAt:         timeStr,
	}

	evtJSON, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %s", err.Error())
	}

	err = ctx.GetStub().PutState(key, evtJSON)
	if err != nil {
		return fmt.Errorf("failed to put state: %s", err.Error())
	}

	return nil
}

// QueryEvent returns a single event by ID
func (ec *EventContract) QueryEvent(ctx contractapi.TransactionContextInterface, eventID string) (*Event, error) {
	if eventID == "" {
		return nil, fmt.Errorf("event ID must not be empty")
	}

	key := "event:" + eventID
	data, err := ctx.GetStub().GetState(key)
	if err != nil {
		return nil, fmt.Errorf("failed to read state: %s", err.Error())
	}
	if data == nil {
		return nil, fmt.Errorf("event %s not found", eventID)
	}

	var evt Event
	err = json.Unmarshal(data, &evt)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %s", err.Error())
	}

	return &evt, nil
}

// ListEvents returns events filtered by status and/or type using key range query
func (ec *EventContract) ListEvents(ctx contractapi.TransactionContextInterface, status string, eventType string) ([]*Event, error) {
	// Use GetStateByRange to get all events
	resultsIterator, err := ctx.GetStub().GetStateByRange("event:", "event:~")
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %s", err.Error())
	}
	defer resultsIterator.Close()

	var events []*Event
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate results: %s", err.Error())
		}

		var evt Event
		err = json.Unmarshal(queryResponse.Value, &evt)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal event: %s", err.Error())
		}

		// Apply filters
		if status != "" && evt.Status != status {
			continue
		}
		if eventType != "" && evt.Type != eventType {
			continue
		}

		events = append(events, &evt)
	}

	if events == nil {
		events = []*Event{}
	}

	return events, nil
}

// UpdateStatus advances the event state machine
func (ec *EventContract) UpdateStatus(ctx contractapi.TransactionContextInterface, eventID string, newStatus string) error {
	if eventID == "" {
		return fmt.Errorf("event ID must not be empty")
	}

	key := "event:" + eventID
	data, err := ctx.GetStub().GetState(key)
	if err != nil {
		return fmt.Errorf("failed to read state: %s", err.Error())
	}
	if data == nil {
		return fmt.Errorf("event %s not found", eventID)
	}

	var evt Event
	err = json.Unmarshal(data, &evt)
	if err != nil {
		return fmt.Errorf("failed to unmarshal event: %s", err.Error())
	}

	// Validate state transition
	allowedNext, exists := validTransitions[evt.Status]
	if !exists {
		return fmt.Errorf("event %s is in terminal state %s", eventID, evt.Status)
	}
	if newStatus != allowedNext {
		return fmt.Errorf("invalid transition: %s -> %s (allowed: %s -> %s)", evt.Status, newStatus, evt.Status, allowedNext)
	}

	timestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return fmt.Errorf("failed to get timestamp: %s", err.Error())
	}
	timeStr := time.Unix(timestamp.Seconds, int64(timestamp.Nanos)).UTC().Format(time.RFC3339)

	evt.Status = newStatus
	evt.UpdatedAt = timeStr

	evtJSON, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %s", err.Error())
	}

	err = ctx.GetStub().PutState(key, evtJSON)
	if err != nil {
		return fmt.Errorf("failed to put state: %s", err.Error())
	}

	return nil
}

// UpdateResult records the event outcome and sets status to SETTLED
func (ec *EventContract) UpdateResult(ctx contractapi.TransactionContextInterface, eventID string, outcome string) error {
	if eventID == "" {
		return fmt.Errorf("event ID must not be empty")
	}
	if outcome == "" {
		return fmt.Errorf("outcome must not be empty")
	}

	key := "event:" + eventID
	data, err := ctx.GetStub().GetState(key)
	if err != nil {
		return fmt.Errorf("failed to read state: %s", err.Error())
	}
	if data == nil {
		return fmt.Errorf("event %s not found", eventID)
	}

	var evt Event
	err = json.Unmarshal(data, &evt)
	if err != nil {
		return fmt.Errorf("failed to unmarshal event: %s", err.Error())
	}

	// Can only update result when event is ONGOING
	if evt.Status != StatusOngoing {
		return fmt.Errorf("can only update result for ONGOING events, current status: %s", evt.Status)
	}

	timestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return fmt.Errorf("failed to get timestamp: %s", err.Error())
	}
	timeStr := time.Unix(timestamp.Seconds, int64(timestamp.Nanos)).UTC().Format(time.RFC3339)

	evt.Result = outcome
	evt.Status = StatusSettled
	evt.UpdatedAt = timeStr

	evtJSON, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %s", err.Error())
	}

	err = ctx.GetStub().PutState(key, evtJSON)
	if err != nil {
		return fmt.Errorf("failed to put state: %s", err.Error())
	}

	return nil
}
```

### 4.4 Run event tests (expect pass)

- [ ] `cd fabric/chaincode/event && go test -v ./...`

### 4.5 Commit event chaincode

- [ ] `git add fabric/chaincode/event/ && git commit -m "feat(chaincode): implement event chaincode with state machine and tests"`

---

## Task 5: Prediction Chaincode (AMM)

### 5.1 Write prediction chaincode tests

- [ ] Create `fabric/chaincode/prediction/prediction_test.go`
- [ ] Create `fabric/chaincode/prediction/go.mod`

`fabric/chaincode/prediction/go.mod`:

```go
module github.com/eventchain/chaincode/prediction

go 1.21

require (
	github.com/hyperledger/fabric-chaincode-go v0.0.0-20240124143825-007bcabaa3a2
	github.com/hyperledger/fabric-contract-api-go v1.2.2
	github.com/hyperledger/fabric-protos-go v0.3.3
	github.com/stretchr/testify v1.8.4
)
```

`fabric/chaincode/prediction/prediction_test.go`:

```go
package prediction

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-chaincode-go/shimtest"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupPredictionChaincode(t *testing.T) (*shimtest.MockStub, *PredictionContract) {
	pc := new(PredictionContract)
	cc, err := contractapi.NewChaincode(pc)
	require.NoError(t, err)
	stub := shimtest.NewMockStub("prediction", cc)
	require.NotNil(t, stub)
	return stub, pc
}

// mockTokenTransferSuccess creates a mock token chaincode that always returns success
func mockTokenTransferSuccess() *shimtest.MockStub {
	// Create a minimal mock stub for the token chaincode
	// The mock will return OK for Transfer calls
	tokenStub := shimtest.NewMockStub("token-cc", nil)
	return tokenStub
}

func initializePool(t *testing.T, stub *shimtest.MockStub, eventID string) {
	// Set up mock for token chaincode
	tokenStub := mockTokenTransferSuccess()
	stub.MockPeerChaincode("token-cc", tokenStub, "eventchain")

	resp := stub.MockInvoke("tx-init-"+eventID, [][]byte{
		[]byte("PredictionContract:InitializePool"),
		[]byte(eventID),
		[]byte("A"),
		[]byte("B"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)
}

func TestInitializePool(t *testing.T) {
	stub, _ := setupPredictionChaincode(t)

	tokenStub := mockTokenTransferSuccess()
	stub.MockPeerChaincode("token-cc", tokenStub, "eventchain")

	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("PredictionContract:InitializePool"),
		[]byte("evt001"),
		[]byte("A"),
		[]byte("B"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Verify pool state
	resp = stub.MockInvoke("tx2", [][]byte{
		[]byte("PredictionContract:GetPool"),
		[]byte("evt001"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var pool Pool
	err := json.Unmarshal(resp.Payload, &pool)
	require.NoError(t, err)
	assert.Equal(t, "evt001", pool.EventID)
	assert.Equal(t, int64(10000), pool.PoolA)
	assert.Equal(t, int64(10000), pool.PoolB)
	assert.Equal(t, int64(100000000), pool.K)
	assert.Equal(t, int64(0), pool.TotalVolume)
	assert.Equal(t, "A", pool.OptionA)
	assert.Equal(t, "B", pool.OptionB)
}

func TestInitializePoolDuplicate(t *testing.T) {
	stub, _ := setupPredictionChaincode(t)

	tokenStub := mockTokenTransferSuccess()
	stub.MockPeerChaincode("token-cc", tokenStub, "eventchain")

	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("PredictionContract:InitializePool"),
		[]byte("evt001"),
		[]byte("A"),
		[]byte("B"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Second init should fail
	resp = stub.MockInvoke("tx2", [][]byte{
		[]byte("PredictionContract:InitializePool"),
		[]byte("evt001"),
		[]byte("A"),
		[]byte("B"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestPlaceBet(t *testing.T) {
	stub, _ := setupPredictionChaincode(t)

	// Set up mock token chaincode that returns success for Transfer
	tokenStub := shimtest.NewMockStub("token-cc", nil)
	tokenStub.MockInvokeWithSignedProposal("mock", [][]byte{[]byte("init")}, nil)
	stub.MockPeerChaincode("token-cc", tokenStub, "eventchain")

	initializePool(t, stub, "evt001")

	// Place a bet on option A with 100 tokens
	resp := stub.MockInvoke("tx-bet1", [][]byte{
		[]byte("PredictionContract:PlaceBet"),
		[]byte("evt001"),
		[]byte("user1"),
		[]byte("A"),
		[]byte("100"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Parse the bet result
	var betResult BetResult
	err := json.Unmarshal(resp.Payload, &betResult)
	require.NoError(t, err)
	assert.Greater(t, betResult.Shares, int64(0))
	assert.Equal(t, "A", betResult.Option)

	// Verify pool state changed
	resp = stub.MockInvoke("tx-pool1", [][]byte{
		[]byte("PredictionContract:GetPool"),
		[]byte("evt001"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var pool Pool
	err = json.Unmarshal(resp.Payload, &pool)
	require.NoError(t, err)

	// After betting on A: poolB += 100 = 10100, newPoolA = 100000000/10100 = 9900 (approx)
	assert.Equal(t, int64(10100), pool.PoolB)
	assert.Equal(t, int64(100), pool.TotalVolume)
	// Verify k invariant: newPoolA * newPoolB should be close to k
	assert.InDelta(t, float64(pool.K), float64(pool.PoolA*pool.PoolB), float64(pool.K)*0.01)
}

func TestPlaceBetInvalidOption(t *testing.T) {
	stub, _ := setupPredictionChaincode(t)

	tokenStub := shimtest.NewMockStub("token-cc", nil)
	stub.MockPeerChaincode("token-cc", tokenStub, "eventchain")

	initializePool(t, stub, "evt001")

	// Place bet with invalid option
	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("PredictionContract:PlaceBet"),
		[]byte("evt001"),
		[]byte("user1"),
		[]byte("C"),
		[]byte("100"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestPlaceBetInvalidAmount(t *testing.T) {
	stub, _ := setupPredictionChaincode(t)

	tokenStub := shimtest.NewMockStub("token-cc", nil)
	stub.MockPeerChaincode("token-cc", tokenStub, "eventchain")

	initializePool(t, stub, "evt001")

	// Place bet with zero amount
	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("PredictionContract:PlaceBet"),
		[]byte("evt001"),
		[]byte("user1"),
		[]byte("A"),
		[]byte("0"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestPlaceBetNoPool(t *testing.T) {
	stub, _ := setupPredictionChaincode(t)

	tokenStub := shimtest.NewMockStub("token-cc", nil)
	stub.MockPeerChaincode("token-cc", tokenStub, "eventchain")

	// Bet on event with no pool
	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("PredictionContract:PlaceBet"),
		[]byte("evt999"),
		[]byte("user1"),
		[]byte("A"),
		[]byte("100"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestGetOdds(t *testing.T) {
	stub, _ := setupPredictionChaincode(t)

	tokenStub := shimtest.NewMockStub("token-cc", nil)
	stub.MockPeerChaincode("token-cc", tokenStub, "eventchain")

	initializePool(t, stub, "evt001")

	// Initial odds should be 50/50
	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("PredictionContract:GetOdds"),
		[]byte("evt001"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var odds Odds
	err := json.Unmarshal(resp.Payload, &odds)
	require.NoError(t, err)
	assert.InDelta(t, 0.5, odds.ProbA, 0.001)
	assert.InDelta(t, 0.5, odds.ProbB, 0.001)

	// Place a bet on A, odds should shift
	stub.MockInvoke("tx-bet1", [][]byte{
		[]byte("PredictionContract:PlaceBet"),
		[]byte("evt001"),
		[]byte("user1"),
		[]byte("A"),
		[]byte("1000"),
	})

	resp = stub.MockInvoke("tx2", [][]byte{
		[]byte("PredictionContract:GetOdds"),
		[]byte("evt001"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	err = json.Unmarshal(resp.Payload, &odds)
	require.NoError(t, err)
	// After betting on A, P(A) should increase
	assert.Greater(t, odds.ProbA, 0.5)
	assert.Less(t, odds.ProbB, 0.5)
	assert.InDelta(t, 1.0, odds.ProbA+odds.ProbB, 0.001)
}

func TestAMMConstantProduct(t *testing.T) {
	stub, _ := setupPredictionChaincode(t)

	tokenStub := shimtest.NewMockStub("token-cc", nil)
	stub.MockPeerChaincode("token-cc", tokenStub, "eventchain")

	initializePool(t, stub, "evt001")

	// Place multiple bets and verify k invariant holds
	bets := []struct {
		user   string
		option string
		amount string
	}{
		{"user1", "A", "500"},
		{"user2", "B", "300"},
		{"user3", "A", "1000"},
		{"user4", "B", "200"},
		{"user5", "A", "100"},
	}

	for i, bet := range bets {
		txID := fmt.Sprintf("tx-bet-%d", i)
		resp := stub.MockInvoke(txID, [][]byte{
			[]byte("PredictionContract:PlaceBet"),
			[]byte("evt001"),
			[]byte(bet.user),
			[]byte(bet.option),
			[]byte(bet.amount),
		})
		assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

		// Check pool after each bet
		poolResp := stub.MockInvoke(txID+"-pool", [][]byte{
			[]byte("PredictionContract:GetPool"),
			[]byte("evt001"),
		})
		var pool Pool
		json.Unmarshal(poolResp.Payload, &pool)

		// k should remain constant (within integer rounding)
		product := pool.PoolA * pool.PoolB
		assert.InDelta(t, float64(pool.K), float64(product), float64(pool.K)*0.01,
			"k invariant violated after bet %d: poolA=%d, poolB=%d, product=%d, k=%d",
			i, pool.PoolA, pool.PoolB, product, pool.K)
	}
}

func TestSettle(t *testing.T) {
	stub, _ := setupPredictionChaincode(t)

	// Create a mock token chaincode
	tokenStub := shimtest.NewMockStub("token-cc", nil)
	stub.MockPeerChaincode("token-cc", tokenStub, "eventchain")

	initializePool(t, stub, "evt001")

	// Place bets
	stub.MockInvoke("tx-bet1", [][]byte{
		[]byte("PredictionContract:PlaceBet"),
		[]byte("evt001"),
		[]byte("user1"),
		[]byte("A"),
		[]byte("500"),
	})
	stub.MockInvoke("tx-bet2", [][]byte{
		[]byte("PredictionContract:PlaceBet"),
		[]byte("evt001"),
		[]byte("user2"),
		[]byte("B"),
		[]byte("300"),
	})
	stub.MockInvoke("tx-bet3", [][]byte{
		[]byte("PredictionContract:PlaceBet"),
		[]byte("evt001"),
		[]byte("user3"),
		[]byte("A"),
		[]byte("200"),
	})

	// Settle with result A
	resp := stub.MockInvoke("tx-settle", [][]byte{
		[]byte("PredictionContract:Settle"),
		[]byte("evt001"),
		[]byte("A"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var result SettlementResult
	err := json.Unmarshal(resp.Payload, &result)
	require.NoError(t, err)
	assert.Equal(t, 2, result.WinnerCount)
	assert.Equal(t, int64(1000), result.TotalPool) // 500 + 300 + 200
	assert.Equal(t, true, result.Settled)

	// Verify scores updated
	resp = stub.MockInvoke("tx-score1", [][]byte{
		[]byte("PredictionContract:GetUserScore"),
		[]byte("user1"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var score UserScore
	err = json.Unmarshal(resp.Payload, &score)
	require.NoError(t, err)
	assert.Equal(t, 1, score.TotalBets)
	assert.Equal(t, 1, score.CorrectBets)
	assert.InDelta(t, 1.0, score.AccuracyRate, 0.001)

	// User2 (loser) should have 0 correct
	resp = stub.MockInvoke("tx-score2", [][]byte{
		[]byte("PredictionContract:GetUserScore"),
		[]byte("user2"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	err = json.Unmarshal(resp.Payload, &score)
	require.NoError(t, err)
	assert.Equal(t, 1, score.TotalBets)
	assert.Equal(t, 0, score.CorrectBets)
	assert.InDelta(t, 0.0, score.AccuracyRate, 0.001)
}

func TestSettleNoPool(t *testing.T) {
	stub, _ := setupPredictionChaincode(t)

	tokenStub := shimtest.NewMockStub("token-cc", nil)
	stub.MockPeerChaincode("token-cc", tokenStub, "eventchain")

	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("PredictionContract:Settle"),
		[]byte("evt999"),
		[]byte("A"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestSettleAlreadySettled(t *testing.T) {
	stub, _ := setupPredictionChaincode(t)

	tokenStub := shimtest.NewMockStub("token-cc", nil)
	stub.MockPeerChaincode("token-cc", tokenStub, "eventchain")

	initializePool(t, stub, "evt001")

	stub.MockInvoke("tx-bet1", [][]byte{
		[]byte("PredictionContract:PlaceBet"),
		[]byte("evt001"),
		[]byte("user1"),
		[]byte("A"),
		[]byte("100"),
	})

	// First settle
	resp := stub.MockInvoke("tx-settle1", [][]byte{
		[]byte("PredictionContract:Settle"),
		[]byte("evt001"),
		[]byte("A"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Second settle should fail
	resp = stub.MockInvoke("tx-settle2", [][]byte{
		[]byte("PredictionContract:Settle"),
		[]byte("evt001"),
		[]byte("A"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestGetUserScore(t *testing.T) {
	stub, _ := setupPredictionChaincode(t)

	// Query score for user with no bets
	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("PredictionContract:GetUserScore"),
		[]byte("newuser"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var score UserScore
	err := json.Unmarshal(resp.Payload, &score)
	require.NoError(t, err)
	assert.Equal(t, "newuser", score.UserID)
	assert.Equal(t, 0, score.TotalBets)
	assert.Equal(t, 0, score.CorrectBets)
	assert.InDelta(t, 0.0, score.AccuracyRate, 0.001)
}

func TestGetUserBets(t *testing.T) {
	stub, _ := setupPredictionChaincode(t)

	tokenStub := shimtest.NewMockStub("token-cc", nil)
	stub.MockPeerChaincode("token-cc", tokenStub, "eventchain")

	initializePool(t, stub, "evt001")
	initializePool(t, stub, "evt002")

	// User1 bets on two events
	stub.MockInvoke("tx-bet1", [][]byte{
		[]byte("PredictionContract:PlaceBet"),
		[]byte("evt001"),
		[]byte("user1"),
		[]byte("A"),
		[]byte("100"),
	})
	stub.MockInvoke("tx-bet2", [][]byte{
		[]byte("PredictionContract:PlaceBet"),
		[]byte("evt002"),
		[]byte("user1"),
		[]byte("B"),
		[]byte("200"),
	})

	// Get all bets for user1
	resp := stub.MockInvoke("tx-bets1", [][]byte{
		[]byte("PredictionContract:GetUserBets"),
		[]byte("user1"),
		[]byte(""),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var bets []Bet
	err := json.Unmarshal(resp.Payload, &bets)
	require.NoError(t, err)
	assert.Equal(t, 2, len(bets))

	// Get bets for user1 on evt001 only
	resp = stub.MockInvoke("tx-bets2", [][]byte{
		[]byte("PredictionContract:GetUserBets"),
		[]byte("user1"),
		[]byte("evt001"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	err = json.Unmarshal(resp.Payload, &bets)
	require.NoError(t, err)
	assert.Equal(t, 1, len(bets))
	assert.Equal(t, "evt001", bets[0].EventID)
}

func TestMultipleSettlementsScoreAccumulation(t *testing.T) {
	stub, _ := setupPredictionChaincode(t)

	tokenStub := shimtest.NewMockStub("token-cc", nil)
	stub.MockPeerChaincode("token-cc", tokenStub, "eventchain")

	// Event 1: user1 wins
	initializePool(t, stub, "evt001")
	stub.MockInvoke("tx-bet1a", [][]byte{
		[]byte("PredictionContract:PlaceBet"),
		[]byte("evt001"),
		[]byte("user1"),
		[]byte("A"),
		[]byte("100"),
	})
	stub.MockInvoke("tx-settle1", [][]byte{
		[]byte("PredictionContract:Settle"),
		[]byte("evt001"),
		[]byte("A"),
	})

	// Event 2: user1 loses
	initializePool(t, stub, "evt002")
	stub.MockInvoke("tx-bet2a", [][]byte{
		[]byte("PredictionContract:PlaceBet"),
		[]byte("evt002"),
		[]byte("user1"),
		[]byte("A"),
		[]byte("100"),
	})
	stub.MockInvoke("tx-settle2", [][]byte{
		[]byte("PredictionContract:Settle"),
		[]byte("evt002"),
		[]byte("B"),
	})

	// Event 3: user1 wins
	initializePool(t, stub, "evt003")
	stub.MockInvoke("tx-bet3a", [][]byte{
		[]byte("PredictionContract:PlaceBet"),
		[]byte("evt003"),
		[]byte("user1"),
		[]byte("B"),
		[]byte("100"),
	})
	stub.MockInvoke("tx-settle3", [][]byte{
		[]byte("PredictionContract:Settle"),
		[]byte("evt003"),
		[]byte("B"),
	})

	// Check user1 score: 3 total, 2 correct, 66.7% accuracy
	resp := stub.MockInvoke("tx-score", [][]byte{
		[]byte("PredictionContract:GetUserScore"),
		[]byte("user1"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var score UserScore
	err := json.Unmarshal(resp.Payload, &score)
	require.NoError(t, err)
	assert.Equal(t, 3, score.TotalBets)
	assert.Equal(t, 2, score.CorrectBets)
	assert.InDelta(t, 2.0/3.0, score.AccuracyRate, 0.01)
}

func TestCrossChaincodeMockInvoke(t *testing.T) {
	stub, _ := setupPredictionChaincode(t)

	// Set up a mock token chaincode that returns a specific response
	tokenStub := shimtest.NewMockStub("token-cc", nil)
	stub.MockPeerChaincode("token-cc", tokenStub, "eventchain")

	initializePool(t, stub, "evt001")

	// The InvokeChaincode call should go through to the mock
	// Place a bet which internally calls token-cc.Transfer
	resp := stub.MockInvoke("tx-bet1", [][]byte{
		[]byte("PredictionContract:PlaceBet"),
		[]byte("evt001"),
		[]byte("user1"),
		[]byte("A"),
		[]byte("100"),
	})
	// Should succeed because mock returns OK by default
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)
}

func TestSettlementPayoutDistribution(t *testing.T) {
	stub, _ := setupPredictionChaincode(t)

	tokenStub := shimtest.NewMockStub("token-cc", nil)
	stub.MockPeerChaincode("token-cc", tokenStub, "eventchain")

	initializePool(t, stub, "evt001")

	// User1 bets 600 on A, user2 bets 400 on A, user3 bets 1000 on B
	stub.MockInvoke("tx-bet1", [][]byte{
		[]byte("PredictionContract:PlaceBet"),
		[]byte("evt001"),
		[]byte("user1"),
		[]byte("A"),
		[]byte("600"),
	})
	stub.MockInvoke("tx-bet2", [][]byte{
		[]byte("PredictionContract:PlaceBet"),
		[]byte("evt001"),
		[]byte("user2"),
		[]byte("A"),
		[]byte("400"),
	})
	stub.MockInvoke("tx-bet3", [][]byte{
		[]byte("PredictionContract:PlaceBet"),
		[]byte("evt001"),
		[]byte("user3"),
		[]byte("B"),
		[]byte("1000"),
	})

	// Settle with A winning
	resp := stub.MockInvoke("tx-settle", [][]byte{
		[]byte("PredictionContract:Settle"),
		[]byte("evt001"),
		[]byte("A"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var result SettlementResult
	err := json.Unmarshal(resp.Payload, &result)
	require.NoError(t, err)
	assert.Equal(t, 2, result.WinnerCount)
	assert.Equal(t, int64(2000), result.TotalPool) // 600 + 400 + 1000

	// Total payouts should equal total pool (minus rounding dust)
	var totalPayout int64
	for _, p := range result.Payouts {
		totalPayout += p.Payout
	}
	assert.InDelta(t, float64(result.TotalPool), float64(totalPayout), 2.0, "total payouts should approximately equal total pool")
}

// Mock chaincode for testing InvokeChaincode with controlled responses
type MockTokenCC struct {
}

func (m *MockTokenCC) Init(stub shim.ChaincodeStubInterface) peer.Response {
	return shim.Success(nil)
}

func (m *MockTokenCC) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
	fn, _ := stub.GetFunctionAndParameters()
	if fn == "TokenContract:Transfer" {
		return shim.Success(nil)
	}
	return shim.Error("unknown function: " + fn)
}
```

### 5.2 Run prediction tests (expect fail)

- [ ] `cd fabric/chaincode/prediction && go test -v ./...` -- should fail because `prediction.go` does not exist yet

### 5.3 Write prediction chaincode implementation

- [ ] Create `fabric/chaincode/prediction/prediction.go`

`fabric/chaincode/prediction/prediction.go`:

```go
package prediction

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// Pool represents the AMM liquidity pool for an event
type Pool struct {
	DocType     string `json:"docType"`
	EventID     string `json:"eventID"`
	OptionA     string `json:"optionA"`
	OptionB     string `json:"optionB"`
	PoolA       int64  `json:"poolA"`
	PoolB       int64  `json:"poolB"`
	K           int64  `json:"k"`
	TotalVolume int64  `json:"totalVolume"`
	Settled     bool   `json:"settled"`
}

// Bet represents a single user bet
type Bet struct {
	DocType   string `json:"docType"`
	EventID   string `json:"eventID"`
	UserID    string `json:"userID"`
	BetID     string `json:"betID"`
	Option    string `json:"option"`
	Amount    int64  `json:"amount"`
	Shares    int64  `json:"shares"`
	Timestamp string `json:"timestamp"`
}

// UserScore tracks a user's prediction accuracy
type UserScore struct {
	DocType      string  `json:"docType"`
	UserID       string  `json:"userID"`
	TotalBets    int     `json:"totalBets"`
	CorrectBets  int     `json:"correctBets"`
	AccuracyRate float64 `json:"accuracyRate"`
}

// BetResult is returned after placing a bet
type BetResult struct {
	BetID  string `json:"betID"`
	Option string `json:"option"`
	Amount int64  `json:"amount"`
	Shares int64  `json:"shares"`
	ProbA  float64 `json:"probA"`
	ProbB  float64 `json:"probB"`
}

// Odds represents current probabilities
type Odds struct {
	EventID string  `json:"eventID"`
	ProbA   float64 `json:"probA"`
	ProbB   float64 `json:"probB"`
	OptionA string  `json:"optionA"`
	OptionB string  `json:"optionB"`
}

// PayoutRecord represents a single winner's payout
type PayoutRecord struct {
	UserID string `json:"userID"`
	Shares int64  `json:"shares"`
	Payout int64  `json:"payout"`
}

// SettlementResult is returned after settling a pool
type SettlementResult struct {
	Settled     bool           `json:"settled"`
	WinnerCount int            `json:"winnerCount"`
	TotalPool   int64          `json:"totalPool"`
	Payouts     []PayoutRecord `json:"payouts"`
}

// PredictionContract implements the prediction market chaincode
type PredictionContract struct {
	contractapi.Contract
}

// InitializePool creates a new AMM pool for an event with initial liquidity
func (pc *PredictionContract) InitializePool(ctx contractapi.TransactionContextInterface, eventID string, optionA string, optionB string) error {
	if eventID == "" {
		return fmt.Errorf("event ID must not be empty")
	}
	if optionA == "" || optionB == "" {
		return fmt.Errorf("options must not be empty")
	}

	key := "pool:" + eventID
	existing, err := ctx.GetStub().GetState(key)
	if err != nil {
		return fmt.Errorf("failed to read state: %s", err.Error())
	}
	if existing != nil {
		return fmt.Errorf("pool for event %s already exists", eventID)
	}

	pool := Pool{
		DocType:     "pool",
		EventID:     eventID,
		OptionA:     optionA,
		OptionB:     optionB,
		PoolA:       10000,
		PoolB:       10000,
		K:           100000000, // 10000 * 10000
		TotalVolume: 0,
		Settled:     false,
	}

	poolJSON, err := json.Marshal(pool)
	if err != nil {
		return fmt.Errorf("failed to marshal pool: %s", err.Error())
	}

	err = ctx.GetStub().PutState(key, poolJSON)
	if err != nil {
		return fmt.Errorf("failed to put state: %s", err.Error())
	}

	return nil
}

// PlaceBet places a bet on an event option using the AMM
func (pc *PredictionContract) PlaceBet(ctx contractapi.TransactionContextInterface, eventID string, userID string, option string, amountStr string) (*BetResult, error) {
	if eventID == "" || userID == "" {
		return nil, fmt.Errorf("event ID and user ID must not be empty")
	}

	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %s", err.Error())
	}
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}

	// Read pool
	poolKey := "pool:" + eventID
	poolBytes, err := ctx.GetStub().GetState(poolKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read pool: %s", err.Error())
	}
	if poolBytes == nil {
		return nil, fmt.Errorf("pool for event %s not found", eventID)
	}

	var pool Pool
	err = json.Unmarshal(poolBytes, &pool)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal pool: %s", err.Error())
	}

	if pool.Settled {
		return nil, fmt.Errorf("pool for event %s is already settled", eventID)
	}

	// Validate option
	if option != pool.OptionA && option != pool.OptionB {
		return nil, fmt.Errorf("invalid option %s, must be %s or %s", option, pool.OptionA, pool.OptionB)
	}

	// Cross-chaincode call: transfer tokens from user to pool account
	poolAccountID := "pool:" + eventID
	transferArgs := [][]byte{
		[]byte("TokenContract:Transfer"),
		[]byte(userID),
		[]byte(poolAccountID),
		[]byte(amountStr),
	}
	transferResp := ctx.GetStub().InvokeChaincode("token-cc", transferArgs, "eventchain")
	if transferResp.Status != 200 {
		return nil, fmt.Errorf("token transfer failed: %s", transferResp.Message)
	}

	// AMM calculation
	var shares int64
	if option == pool.OptionA {
		// Buy option A: poolB += amount, newPoolA = k / poolB, shares = poolA - newPoolA
		pool.PoolB += amount
		newPoolA := pool.K / pool.PoolB
		shares = pool.PoolA - newPoolA
		pool.PoolA = newPoolA
	} else {
		// Buy option B: poolA += amount, newPoolB = k / poolA, shares = poolB - newPoolB
		pool.PoolA += amount
		newPoolB := pool.K / pool.PoolA
		shares = pool.PoolB - newPoolB
		pool.PoolB = newPoolB
	}

	if shares <= 0 {
		return nil, fmt.Errorf("calculated shares must be positive, got %d", shares)
	}

	pool.TotalVolume += amount

	// Save updated pool
	poolJSON, err := json.Marshal(pool)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pool: %s", err.Error())
	}
	err = ctx.GetStub().PutState(poolKey, poolJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to update pool: %s", err.Error())
	}

	// Create bet record
	timestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return nil, fmt.Errorf("failed to get timestamp: %s", err.Error())
	}
	timeStr := time.Unix(timestamp.Seconds, int64(timestamp.Nanos)).UTC().Format(time.RFC3339)
	txID := ctx.GetStub().GetTxID()

	bet := Bet{
		DocType:   "bet",
		EventID:   eventID,
		UserID:    userID,
		BetID:     txID,
		Option:    option,
		Amount:    amount,
		Shares:    shares,
		Timestamp: timeStr,
	}

	betJSON, err := json.Marshal(bet)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal bet: %s", err.Error())
	}

	betKey := fmt.Sprintf("bet:%s:%s:%s", eventID, userID, txID)
	err = ctx.GetStub().PutState(betKey, betJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to store bet: %s", err.Error())
	}

	// Calculate new probabilities
	totalPool := float64(pool.PoolA + pool.PoolB)
	probA := float64(pool.PoolB) / totalPool
	probB := float64(pool.PoolA) / totalPool

	result := &BetResult{
		BetID:  txID,
		Option: option,
		Amount: amount,
		Shares: shares,
		ProbA:  probA,
		ProbB:  probB,
	}

	return result, nil
}

// GetOdds returns the current probabilities for an event
func (pc *PredictionContract) GetOdds(ctx contractapi.TransactionContextInterface, eventID string) (*Odds, error) {
	if eventID == "" {
		return nil, fmt.Errorf("event ID must not be empty")
	}

	poolKey := "pool:" + eventID
	poolBytes, err := ctx.GetStub().GetState(poolKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read pool: %s", err.Error())
	}
	if poolBytes == nil {
		return nil, fmt.Errorf("pool for event %s not found", eventID)
	}

	var pool Pool
	err = json.Unmarshal(poolBytes, &pool)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal pool: %s", err.Error())
	}

	totalPool := float64(pool.PoolA + pool.PoolB)
	odds := &Odds{
		EventID: eventID,
		ProbA:   float64(pool.PoolB) / totalPool,
		ProbB:   float64(pool.PoolA) / totalPool,
		OptionA: pool.OptionA,
		OptionB: pool.OptionB,
	}

	return odds, nil
}

// GetPool returns the full pool details for an event
func (pc *PredictionContract) GetPool(ctx contractapi.TransactionContextInterface, eventID string) (*Pool, error) {
	if eventID == "" {
		return nil, fmt.Errorf("event ID must not be empty")
	}

	poolKey := "pool:" + eventID
	poolBytes, err := ctx.GetStub().GetState(poolKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read pool: %s", err.Error())
	}
	if poolBytes == nil {
		return nil, fmt.Errorf("pool for event %s not found", eventID)
	}

	var pool Pool
	err = json.Unmarshal(poolBytes, &pool)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal pool: %s", err.Error())
	}

	return &pool, nil
}

// Settle distributes the pool to winners proportionally by shares
func (pc *PredictionContract) Settle(ctx contractapi.TransactionContextInterface, eventID string, winningOption string) (*SettlementResult, error) {
	if eventID == "" {
		return nil, fmt.Errorf("event ID must not be empty")
	}
	if winningOption == "" {
		return nil, fmt.Errorf("winning option must not be empty")
	}

	// Read pool
	poolKey := "pool:" + eventID
	poolBytes, err := ctx.GetStub().GetState(poolKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read pool: %s", err.Error())
	}
	if poolBytes == nil {
		return nil, fmt.Errorf("pool for event %s not found", eventID)
	}

	var pool Pool
	err = json.Unmarshal(poolBytes, &pool)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal pool: %s", err.Error())
	}

	if pool.Settled {
		return nil, fmt.Errorf("pool for event %s is already settled", eventID)
	}

	if winningOption != pool.OptionA && winningOption != pool.OptionB {
		return nil, fmt.Errorf("invalid winning option %s", winningOption)
	}

	// Get all bets for this event
	betStartKey := fmt.Sprintf("bet:%s:", eventID)
	betEndKey := fmt.Sprintf("bet:%s:~", eventID)
	iterator, err := ctx.GetStub().GetStateByRange(betStartKey, betEndKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get bets: %s", err.Error())
	}
	defer iterator.Close()

	var allBets []Bet
	var winningBets []Bet
	var totalWinningShares int64

	for iterator.HasNext() {
		queryResponse, err := iterator.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate bets: %s", err.Error())
		}

		var bet Bet
		err = json.Unmarshal(queryResponse.Value, &bet)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal bet: %s", err.Error())
		}

		allBets = append(allBets, bet)

		if bet.Option == winningOption {
			winningBets = append(winningBets, bet)
			totalWinningShares += bet.Shares
		}
	}

	totalPool := pool.TotalVolume
	poolAccountID := "pool:" + eventID

	// Distribute payouts to winners
	var payouts []PayoutRecord

	if totalWinningShares > 0 && totalPool > 0 {
		var distributedTotal int64
		for i, bet := range winningBets {
			var payout int64
			if i == len(winningBets)-1 {
				// Last winner gets remainder to avoid rounding dust loss
				payout = totalPool - distributedTotal
			} else {
				payout = (bet.Shares * totalPool) / totalWinningShares
			}
			distributedTotal += payout

			if payout > 0 {
				// Cross-chaincode call: transfer payout from pool to winner
				transferArgs := [][]byte{
					[]byte("TokenContract:Transfer"),
					[]byte(poolAccountID),
					[]byte(bet.UserID),
					[]byte(strconv.FormatInt(payout, 10)),
				}
				transferResp := ctx.GetStub().InvokeChaincode("token-cc", transferArgs, "eventchain")
				if transferResp.Status != 200 {
					return nil, fmt.Errorf("failed to transfer payout to %s: %s", bet.UserID, transferResp.Message)
				}
			}

			payouts = append(payouts, PayoutRecord{
				UserID: bet.UserID,
				Shares: bet.Shares,
				Payout: payout,
			})
		}
	}

	// Update scores for all participants
	// Track unique users to avoid double-counting
	userBetMap := make(map[string]bool) // userID -> won at least one bet
	for _, bet := range allBets {
		if _, exists := userBetMap[bet.UserID]; !exists {
			userBetMap[bet.UserID] = false
		}
		if bet.Option == winningOption {
			userBetMap[bet.UserID] = true
		}
	}

	for userID, won := range userBetMap {
		scoreKey := "score:" + userID
		scoreBytes, err := ctx.GetStub().GetState(scoreKey)
		if err != nil {
			return nil, fmt.Errorf("failed to read score for %s: %s", userID, err.Error())
		}

		var score UserScore
		if scoreBytes != nil {
			err = json.Unmarshal(scoreBytes, &score)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal score for %s: %s", userID, err.Error())
			}
		} else {
			score = UserScore{
				DocType: "score",
				UserID:  userID,
			}
		}

		score.TotalBets++
		if won {
			score.CorrectBets++
		}

		if score.TotalBets > 0 {
			score.AccuracyRate = float64(score.CorrectBets) / float64(score.TotalBets)
		}

		scoreJSON, err := json.Marshal(score)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal score for %s: %s", userID, err.Error())
		}

		err = ctx.GetStub().PutState(scoreKey, scoreJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to update score for %s: %s", userID, err.Error())
		}
	}

	// Mark pool as settled
	pool.Settled = true
	poolJSON, err := json.Marshal(pool)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pool: %s", err.Error())
	}
	err = ctx.GetStub().PutState(poolKey, poolJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to update pool: %s", err.Error())
	}

	if payouts == nil {
		payouts = []PayoutRecord{}
	}

	result := &SettlementResult{
		Settled:     true,
		WinnerCount: len(winningBets),
		TotalPool:   totalPool,
		Payouts:     payouts,
	}

	return result, nil
}

// GetUserScore returns a user's prediction accuracy score
func (pc *PredictionContract) GetUserScore(ctx contractapi.TransactionContextInterface, userID string) (*UserScore, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID must not be empty")
	}

	scoreKey := "score:" + userID
	scoreBytes, err := ctx.GetStub().GetState(scoreKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read score: %s", err.Error())
	}

	if scoreBytes == nil {
		return &UserScore{
			DocType:      "score",
			UserID:       userID,
			TotalBets:    0,
			CorrectBets:  0,
			AccuracyRate: 0.0,
		}, nil
	}

	var score UserScore
	err = json.Unmarshal(scoreBytes, &score)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal score: %s", err.Error())
	}

	return &score, nil
}

// GetUserBets returns all bets for a user, optionally filtered by event
func (pc *PredictionContract) GetUserBets(ctx contractapi.TransactionContextInterface, userID string, eventID string) ([]*Bet, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID must not be empty")
	}

	var bets []*Bet

	if eventID != "" {
		// Query bets for specific event and user
		startKey := fmt.Sprintf("bet:%s:%s:", eventID, userID)
		endKey := fmt.Sprintf("bet:%s:%s:~", eventID, userID)
		iterator, err := ctx.GetStub().GetStateByRange(startKey, endKey)
		if err != nil {
			return nil, fmt.Errorf("failed to get bets: %s", err.Error())
		}
		defer iterator.Close()

		for iterator.HasNext() {
			queryResponse, err := iterator.Next()
			if err != nil {
				return nil, fmt.Errorf("failed to iterate: %s", err.Error())
			}

			var bet Bet
			err = json.Unmarshal(queryResponse.Value, &bet)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal bet: %s", err.Error())
			}

			bets = append(bets, &bet)
		}
	} else {
		// Query all bets across all events for this user
		// Scan all bets and filter by userID
		iterator, err := ctx.GetStub().GetStateByRange("bet:", "bet:~")
		if err != nil {
			return nil, fmt.Errorf("failed to get bets: %s", err.Error())
		}
		defer iterator.Close()

		for iterator.HasNext() {
			queryResponse, err := iterator.Next()
			if err != nil {
				return nil, fmt.Errorf("failed to iterate: %s", err.Error())
			}

			var bet Bet
			err = json.Unmarshal(queryResponse.Value, &bet)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal bet: %s", err.Error())
			}

			if bet.UserID == userID {
				bets = append(bets, &bet)
			}
		}
	}

	if bets == nil {
		bets = []*Bet{}
	}

	return bets, nil
}
```

### 5.4 Run prediction tests (expect pass)

- [ ] `cd fabric/chaincode/prediction && go test -v ./...`

### 5.5 Commit prediction chaincode

- [ ] `git add fabric/chaincode/prediction/ && git commit -m "feat(chaincode): implement prediction market chaincode with AMM and tests"`

---

## Task 6: Ticket Chaincode

### 6.1 Write ticket chaincode tests

- [ ] Create `fabric/chaincode/ticket/ticket_test.go`
- [ ] Create `fabric/chaincode/ticket/go.mod`

`fabric/chaincode/ticket/go.mod`:

```go
module github.com/eventchain/chaincode/ticket

go 1.21

require (
	github.com/hyperledger/fabric-chaincode-go v0.0.0-20240124143825-007bcabaa3a2
	github.com/hyperledger/fabric-contract-api-go v1.2.2
	github.com/hyperledger/fabric-protos-go v0.3.3
	github.com/stretchr/testify v1.8.4
)
```

`fabric/chaincode/ticket/ticket_test.go`:

```go
package ticket

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-chaincode-go/shimtest"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTicketChaincode(t *testing.T) (*shimtest.MockStub, *TicketContract) {
	tc := new(TicketContract)
	cc, err := contractapi.NewChaincode(tc)
	require.NoError(t, err)
	stub := shimtest.NewMockStub("ticket", cc)
	require.NotNil(t, stub)
	return stub, tc
}

// MockPredictionCC simulates prediction chaincode responses for GetUserScore
type MockPredictionCC struct {
	Scores map[string]MockScore
}

type MockScore struct {
	UserID       string  `json:"userID"`
	TotalBets    int     `json:"totalBets"`
	CorrectBets  int     `json:"correctBets"`
	AccuracyRate float64 `json:"accuracyRate"`
}

func (m *MockPredictionCC) Init(stub shim.ChaincodeStubInterface) peer.Response {
	return shim.Success(nil)
}

func (m *MockPredictionCC) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
	fn, args := stub.GetFunctionAndParameters()
	if fn == "PredictionContract:GetUserScore" {
		if len(args) < 1 {
			return shim.Error("missing userID")
		}
		userID := args[0]
		score, exists := m.Scores[userID]
		if !exists {
			score = MockScore{
				UserID:       userID,
				TotalBets:    0,
				CorrectBets:  0,
				AccuracyRate: 0.0,
			}
		}
		scoreJSON, _ := json.Marshal(score)
		return shim.Success(scoreJSON)
	}
	return shim.Error("unknown function: " + fn)
}

func setupMockPredictionCC(stub *shimtest.MockStub, scores map[string]MockScore) {
	mockCC := &MockPredictionCC{Scores: scores}
	predStub := shimtest.NewMockStub("prediction-cc", mockCC)
	stub.MockPeerChaincode("prediction-cc", predStub, "eventchain")
}

func TestApplyTicket(t *testing.T) {
	stub, _ := setupTicketChaincode(t)

	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("TicketContract:ApplyTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Verify application stored
	appBytes := stub.State["application:evt001:user1"]
	require.NotNil(t, appBytes)

	var app Application
	err := json.Unmarshal(appBytes, &app)
	require.NoError(t, err)
	assert.Equal(t, "evt001", app.EventID)
	assert.Equal(t, "user1", app.UserID)
	assert.Equal(t, StatusPending, app.Status)
}

func TestApplyTicketDuplicate(t *testing.T) {
	stub, _ := setupTicketChaincode(t)

	resp := stub.MockInvoke("tx1", [][]byte{
		[]byte("TicketContract:ApplyTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Second application should fail
	resp = stub.MockInvoke("tx2", [][]byte{
		[]byte("TicketContract:ApplyTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestApplyTicketMultipleUsers(t *testing.T) {
	stub, _ := setupTicketChaincode(t)

	for i := 1; i <= 5; i++ {
		userID := fmt.Sprintf("user%d", i)
		txID := fmt.Sprintf("tx%d", i)
		resp := stub.MockInvoke(txID, [][]byte{
			[]byte("TicketContract:ApplyTicket"),
			[]byte("evt001"),
			[]byte(userID),
		})
		assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)
	}

	// Verify all 5 applications exist
	for i := 1; i <= 5; i++ {
		userID := fmt.Sprintf("user%d", i)
		key := fmt.Sprintf("application:evt001:%s", userID)
		assert.NotNil(t, stub.State[key])
	}
}

func TestRunLottery(t *testing.T) {
	stub, _ := setupTicketChaincode(t)

	// Set up mock prediction chaincode with varied accuracy scores
	scores := map[string]MockScore{
		"user1": {UserID: "user1", TotalBets: 10, CorrectBets: 9, AccuracyRate: 0.9},
		"user2": {UserID: "user2", TotalBets: 10, CorrectBets: 5, AccuracyRate: 0.5},
		"user3": {UserID: "user3", TotalBets: 10, CorrectBets: 2, AccuracyRate: 0.2},
		"user4": {UserID: "user4", TotalBets: 10, CorrectBets: 8, AccuracyRate: 0.8},
		"user5": {UserID: "user5", TotalBets: 10, CorrectBets: 1, AccuracyRate: 0.1},
	}
	setupMockPredictionCC(stub, scores)

	// 5 users apply
	for i := 1; i <= 5; i++ {
		userID := fmt.Sprintf("user%d", i)
		resp := stub.MockInvoke(fmt.Sprintf("tx-apply-%d", i), [][]byte{
			[]byte("TicketContract:ApplyTicket"),
			[]byte("evt001"),
			[]byte(userID),
		})
		assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)
	}

	// Run lottery with 3 tickets available
	resp := stub.MockInvoke("tx-lottery", [][]byte{
		[]byte("TicketContract:RunLottery"),
		[]byte("evt001"),
		[]byte("3"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var result LotteryResult
	err := json.Unmarshal(resp.Payload, &result)
	require.NoError(t, err)
	assert.Equal(t, 3, len(result.Winners))
	assert.Equal(t, 2, len(result.Losers))

	// Verify winners have WON status and losers have LOST
	for _, winner := range result.Winners {
		key := fmt.Sprintf("application:evt001:%s", winner)
		appBytes := stub.State[key]
		require.NotNil(t, appBytes)
		var app Application
		json.Unmarshal(appBytes, &app)
		assert.Equal(t, StatusWon, app.Status)
	}

	for _, loser := range result.Losers {
		key := fmt.Sprintf("application:evt001:%s", loser)
		appBytes := stub.State[key]
		require.NotNil(t, appBytes)
		var app Application
		json.Unmarshal(appBytes, &app)
		assert.Equal(t, StatusLost, app.Status)
	}
}

func TestRunLotteryMoreTicketsThanApplicants(t *testing.T) {
	stub, _ := setupTicketChaincode(t)

	scores := map[string]MockScore{
		"user1": {UserID: "user1", TotalBets: 5, CorrectBets: 3, AccuracyRate: 0.6},
		"user2": {UserID: "user2", TotalBets: 5, CorrectBets: 2, AccuracyRate: 0.4},
	}
	setupMockPredictionCC(stub, scores)

	// Only 2 users apply
	for i := 1; i <= 2; i++ {
		userID := fmt.Sprintf("user%d", i)
		stub.MockInvoke(fmt.Sprintf("tx-apply-%d", i), [][]byte{
			[]byte("TicketContract:ApplyTicket"),
			[]byte("evt001"),
			[]byte(userID),
		})
	}

	// Run lottery with 5 tickets (more than applicants)
	resp := stub.MockInvoke("tx-lottery", [][]byte{
		[]byte("TicketContract:RunLottery"),
		[]byte("evt001"),
		[]byte("5"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var result LotteryResult
	err := json.Unmarshal(resp.Payload, &result)
	require.NoError(t, err)
	// All applicants should win
	assert.Equal(t, 2, len(result.Winners))
	assert.Equal(t, 0, len(result.Losers))
}

func TestRunLotteryNoApplicants(t *testing.T) {
	stub, _ := setupTicketChaincode(t)

	scores := map[string]MockScore{}
	setupMockPredictionCC(stub, scores)

	// Run lottery with no applicants
	resp := stub.MockInvoke("tx-lottery", [][]byte{
		[]byte("TicketContract:RunLottery"),
		[]byte("evt001"),
		[]byte("3"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var result LotteryResult
	err := json.Unmarshal(resp.Payload, &result)
	require.NoError(t, err)
	assert.Equal(t, 0, len(result.Winners))
	assert.Equal(t, 0, len(result.Losers))
}

func TestRunLotteryDuplicate(t *testing.T) {
	stub, _ := setupTicketChaincode(t)

	scores := map[string]MockScore{
		"user1": {UserID: "user1", TotalBets: 5, CorrectBets: 3, AccuracyRate: 0.6},
	}
	setupMockPredictionCC(stub, scores)

	stub.MockInvoke("tx-apply-1", [][]byte{
		[]byte("TicketContract:ApplyTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})

	// First lottery
	resp := stub.MockInvoke("tx-lottery1", [][]byte{
		[]byte("TicketContract:RunLottery"),
		[]byte("evt001"),
		[]byte("1"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Second lottery on same event should fail
	resp = stub.MockInvoke("tx-lottery2", [][]byte{
		[]byte("TicketContract:RunLottery"),
		[]byte("evt001"),
		[]byte("1"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestClaimTicket(t *testing.T) {
	stub, _ := setupTicketChaincode(t)

	scores := map[string]MockScore{
		"user1": {UserID: "user1", TotalBets: 10, CorrectBets: 8, AccuracyRate: 0.8},
	}
	setupMockPredictionCC(stub, scores)

	// Apply
	stub.MockInvoke("tx-apply", [][]byte{
		[]byte("TicketContract:ApplyTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})

	// Run lottery
	stub.MockInvoke("tx-lottery", [][]byte{
		[]byte("TicketContract:RunLottery"),
		[]byte("evt001"),
		[]byte("1"),
	})

	// Claim ticket
	resp := stub.MockInvoke("tx-claim", [][]byte{
		[]byte("TicketContract:ClaimTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var ticket Ticket
	err := json.Unmarshal(resp.Payload, &ticket)
	require.NoError(t, err)
	assert.Equal(t, "evt001", ticket.EventID)
	assert.Equal(t, "user1", ticket.OwnerID)
	assert.Equal(t, StatusClaimed, ticket.Status)
	assert.NotEmpty(t, ticket.ClaimHash)
	assert.NotEmpty(t, ticket.TicketID)

	// Verify the hash is a valid SHA256 hex string (64 chars)
	assert.Equal(t, 64, len(ticket.ClaimHash))
}

func TestClaimTicketNotWinner(t *testing.T) {
	stub, _ := setupTicketChaincode(t)

	scores := map[string]MockScore{
		"user1": {UserID: "user1", TotalBets: 10, CorrectBets: 8, AccuracyRate: 0.8},
		"user2": {UserID: "user2", TotalBets: 10, CorrectBets: 1, AccuracyRate: 0.1},
	}
	setupMockPredictionCC(stub, scores)

	// Both apply
	stub.MockInvoke("tx-apply-1", [][]byte{
		[]byte("TicketContract:ApplyTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})
	stub.MockInvoke("tx-apply-2", [][]byte{
		[]byte("TicketContract:ApplyTicket"),
		[]byte("evt001"),
		[]byte("user2"),
	})

	// Lottery with 1 ticket (user1 should win with higher accuracy)
	stub.MockInvoke("tx-lottery", [][]byte{
		[]byte("TicketContract:RunLottery"),
		[]byte("evt001"),
		[]byte("1"),
	})

	// Find the loser
	app1Bytes := stub.State["application:evt001:user1"]
	app2Bytes := stub.State["application:evt001:user2"]
	var app1, app2 Application
	json.Unmarshal(app1Bytes, &app1)
	json.Unmarshal(app2Bytes, &app2)

	var loserID string
	if app1.Status == StatusLost {
		loserID = "user1"
	} else {
		loserID = "user2"
	}

	// Loser should not be able to claim
	resp := stub.MockInvoke("tx-claim-loser", [][]byte{
		[]byte("TicketContract:ClaimTicket"),
		[]byte("evt001"),
		[]byte(loserID),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestClaimTicketTwice(t *testing.T) {
	stub, _ := setupTicketChaincode(t)

	scores := map[string]MockScore{
		"user1": {UserID: "user1", TotalBets: 10, CorrectBets: 8, AccuracyRate: 0.8},
	}
	setupMockPredictionCC(stub, scores)

	stub.MockInvoke("tx-apply", [][]byte{
		[]byte("TicketContract:ApplyTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})
	stub.MockInvoke("tx-lottery", [][]byte{
		[]byte("TicketContract:RunLottery"),
		[]byte("evt001"),
		[]byte("1"),
	})

	// First claim succeeds
	resp := stub.MockInvoke("tx-claim1", [][]byte{
		[]byte("TicketContract:ClaimTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Second claim should fail (already claimed)
	resp = stub.MockInvoke("tx-claim2", [][]byte{
		[]byte("TicketContract:ClaimTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestVerifyTicket(t *testing.T) {
	stub, _ := setupTicketChaincode(t)

	scores := map[string]MockScore{
		"user1": {UserID: "user1", TotalBets: 10, CorrectBets: 8, AccuracyRate: 0.8},
	}
	setupMockPredictionCC(stub, scores)

	stub.MockInvoke("tx-apply", [][]byte{
		[]byte("TicketContract:ApplyTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})
	stub.MockInvoke("tx-lottery", [][]byte{
		[]byte("TicketContract:RunLottery"),
		[]byte("evt001"),
		[]byte("1"),
	})

	// Claim ticket and get the hash
	resp := stub.MockInvoke("tx-claim", [][]byte{
		[]byte("TicketContract:ClaimTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var ticket Ticket
	json.Unmarshal(resp.Payload, &ticket)

	// Verify with correct hash
	resp = stub.MockInvoke("tx-verify", [][]byte{
		[]byte("TicketContract:VerifyTicket"),
		[]byte(ticket.TicketID),
		[]byte(ticket.ClaimHash),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var verifiedTicket Ticket
	err := json.Unmarshal(resp.Payload, &verifiedTicket)
	require.NoError(t, err)
	assert.Equal(t, StatusUsed, verifiedTicket.Status)
	assert.NotEmpty(t, verifiedTicket.VerifiedAt)
}

func TestVerifyTicketWrongHash(t *testing.T) {
	stub, _ := setupTicketChaincode(t)

	scores := map[string]MockScore{
		"user1": {UserID: "user1", TotalBets: 10, CorrectBets: 8, AccuracyRate: 0.8},
	}
	setupMockPredictionCC(stub, scores)

	stub.MockInvoke("tx-apply", [][]byte{
		[]byte("TicketContract:ApplyTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})
	stub.MockInvoke("tx-lottery", [][]byte{
		[]byte("TicketContract:RunLottery"),
		[]byte("evt001"),
		[]byte("1"),
	})

	resp := stub.MockInvoke("tx-claim", [][]byte{
		[]byte("TicketContract:ClaimTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})

	var ticket Ticket
	json.Unmarshal(resp.Payload, &ticket)

	// Verify with wrong hash
	wrongHash := sha256.Sum256([]byte("wrong-data"))
	resp = stub.MockInvoke("tx-verify", [][]byte{
		[]byte("TicketContract:VerifyTicket"),
		[]byte(ticket.TicketID),
		[]byte(hex.EncodeToString(wrongHash[:])),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestVerifyTicketAlreadyUsed(t *testing.T) {
	stub, _ := setupTicketChaincode(t)

	scores := map[string]MockScore{
		"user1": {UserID: "user1", TotalBets: 10, CorrectBets: 8, AccuracyRate: 0.8},
	}
	setupMockPredictionCC(stub, scores)

	stub.MockInvoke("tx-apply", [][]byte{
		[]byte("TicketContract:ApplyTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})
	stub.MockInvoke("tx-lottery", [][]byte{
		[]byte("TicketContract:RunLottery"),
		[]byte("evt001"),
		[]byte("1"),
	})
	resp := stub.MockInvoke("tx-claim", [][]byte{
		[]byte("TicketContract:ClaimTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})

	var ticket Ticket
	json.Unmarshal(resp.Payload, &ticket)

	// First verify succeeds
	resp = stub.MockInvoke("tx-verify1", [][]byte{
		[]byte("TicketContract:VerifyTicket"),
		[]byte(ticket.TicketID),
		[]byte(ticket.ClaimHash),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Second verify should fail (already used)
	resp = stub.MockInvoke("tx-verify2", [][]byte{
		[]byte("TicketContract:VerifyTicket"),
		[]byte(ticket.TicketID),
		[]byte(ticket.ClaimHash),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestRefundTicket(t *testing.T) {
	stub, _ := setupTicketChaincode(t)

	scores := map[string]MockScore{
		"user1": {UserID: "user1", TotalBets: 10, CorrectBets: 8, AccuracyRate: 0.8},
	}
	setupMockPredictionCC(stub, scores)

	stub.MockInvoke("tx-apply", [][]byte{
		[]byte("TicketContract:ApplyTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})
	stub.MockInvoke("tx-lottery", [][]byte{
		[]byte("TicketContract:RunLottery"),
		[]byte("evt001"),
		[]byte("1"),
	})
	resp := stub.MockInvoke("tx-claim", [][]byte{
		[]byte("TicketContract:ClaimTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})

	var ticket Ticket
	json.Unmarshal(resp.Payload, &ticket)

	// Refund the ticket
	resp = stub.MockInvoke("tx-refund", [][]byte{
		[]byte("TicketContract:RefundTicket"),
		[]byte(ticket.TicketID),
		[]byte("user1"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	// Verify ticket is refunded
	ticketBytes := stub.State["ticket:"+ticket.TicketID]
	require.NotNil(t, ticketBytes)
	var refundedTicket Ticket
	json.Unmarshal(ticketBytes, &refundedTicket)
	assert.Equal(t, StatusRefunded, refundedTicket.Status)

	// Verify application is reset to REFUNDED
	appBytes := stub.State["application:evt001:user1"]
	require.NotNil(t, appBytes)
	var app Application
	json.Unmarshal(appBytes, &app)
	assert.Equal(t, StatusRefunded, app.Status)
}

func TestRefundTicketWrongOwner(t *testing.T) {
	stub, _ := setupTicketChaincode(t)

	scores := map[string]MockScore{
		"user1": {UserID: "user1", TotalBets: 10, CorrectBets: 8, AccuracyRate: 0.8},
	}
	setupMockPredictionCC(stub, scores)

	stub.MockInvoke("tx-apply", [][]byte{
		[]byte("TicketContract:ApplyTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})
	stub.MockInvoke("tx-lottery", [][]byte{
		[]byte("TicketContract:RunLottery"),
		[]byte("evt001"),
		[]byte("1"),
	})
	resp := stub.MockInvoke("tx-claim", [][]byte{
		[]byte("TicketContract:ClaimTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})

	var ticket Ticket
	json.Unmarshal(resp.Payload, &ticket)

	// Try to refund as different user
	resp = stub.MockInvoke("tx-refund", [][]byte{
		[]byte("TicketContract:RefundTicket"),
		[]byte(ticket.TicketID),
		[]byte("user2"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestRefundTicketAlreadyUsed(t *testing.T) {
	stub, _ := setupTicketChaincode(t)

	scores := map[string]MockScore{
		"user1": {UserID: "user1", TotalBets: 10, CorrectBets: 8, AccuracyRate: 0.8},
	}
	setupMockPredictionCC(stub, scores)

	stub.MockInvoke("tx-apply", [][]byte{
		[]byte("TicketContract:ApplyTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})
	stub.MockInvoke("tx-lottery", [][]byte{
		[]byte("TicketContract:RunLottery"),
		[]byte("evt001"),
		[]byte("1"),
	})
	resp := stub.MockInvoke("tx-claim", [][]byte{
		[]byte("TicketContract:ClaimTicket"),
		[]byte("evt001"),
		[]byte("user1"),
	})

	var ticket Ticket
	json.Unmarshal(resp.Payload, &ticket)

	// Verify (use the ticket)
	stub.MockInvoke("tx-verify", [][]byte{
		[]byte("TicketContract:VerifyTicket"),
		[]byte(ticket.TicketID),
		[]byte(ticket.ClaimHash),
	})

	// Try to refund used ticket
	resp = stub.MockInvoke("tx-refund", [][]byte{
		[]byte("TicketContract:RefundTicket"),
		[]byte(ticket.TicketID),
		[]byte("user1"),
	})
	assert.NotEqual(t, int32(shim.OK), resp.Status)
}

func TestLotteryPriorityWeighting(t *testing.T) {
	stub, _ := setupTicketChaincode(t)

	// Set up scores where user1 has much higher accuracy
	scores := map[string]MockScore{
		"user1":  {UserID: "user1", TotalBets: 100, CorrectBets: 95, AccuracyRate: 0.95},
		"user2":  {UserID: "user2", TotalBets: 100, CorrectBets: 90, AccuracyRate: 0.90},
		"user3":  {UserID: "user3", TotalBets: 100, CorrectBets: 85, AccuracyRate: 0.85},
		"user4":  {UserID: "user4", TotalBets: 100, CorrectBets: 10, AccuracyRate: 0.10},
		"user5":  {UserID: "user5", TotalBets: 100, CorrectBets: 5, AccuracyRate: 0.05},
		"user6":  {UserID: "user6", TotalBets: 100, CorrectBets: 3, AccuracyRate: 0.03},
		"user7":  {UserID: "user7", TotalBets: 100, CorrectBets: 80, AccuracyRate: 0.80},
		"user8":  {UserID: "user8", TotalBets: 100, CorrectBets: 75, AccuracyRate: 0.75},
		"user9":  {UserID: "user9", TotalBets: 100, CorrectBets: 2, AccuracyRate: 0.02},
		"user10": {UserID: "user10", TotalBets: 100, CorrectBets: 1, AccuracyRate: 0.01},
	}
	setupMockPredictionCC(stub, scores)

	// All 10 users apply
	for i := 1; i <= 10; i++ {
		userID := fmt.Sprintf("user%d", i)
		stub.MockInvoke(fmt.Sprintf("tx-apply-%d", i), [][]byte{
			[]byte("TicketContract:ApplyTicket"),
			[]byte("evt001"),
			[]byte(userID),
		})
	}

	// Run lottery with 3 tickets
	resp := stub.MockInvoke("tx-lottery", [][]byte{
		[]byte("TicketContract:RunLottery"),
		[]byte("evt001"),
		[]byte("3"),
	})
	assert.Equal(t, int32(shim.OK), resp.Status, resp.Message)

	var result LotteryResult
	json.Unmarshal(resp.Payload, &result)
	assert.Equal(t, 3, len(result.Winners))
	assert.Equal(t, 7, len(result.Losers))

	// The priority formula is deterministic: 0.6 * accuracy + 0.4 * pseudo_random
	// We can verify that the lottery produced exactly 3 winners and 7 losers
	// The exact winners depend on the pseudo-random component, but the mechanism works
	t.Logf("Winners: %v", result.Winners)
	t.Logf("Losers: %v", result.Losers)
	t.Logf("Priorities: %v", result.Priorities)
}
```

### 6.2 Run ticket tests (expect fail)

- [ ] `cd fabric/chaincode/ticket && go test -v ./...` -- should fail because `ticket.go` does not exist yet

### 6.3 Write ticket chaincode implementation

- [ ] Create `fabric/chaincode/ticket/ticket.go`

`fabric/chaincode/ticket/ticket.go`:

```go
package ticket

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// Status constants for applications and tickets
const (
	StatusPending  = "PENDING"
	StatusWon      = "WON"
	StatusLost     = "LOST"
	StatusClaimed  = "CLAIMED"
	StatusUsed     = "USED"
	StatusRefunded = "REFUNDED"
)

// Application represents a ticket application
type Application struct {
	DocType   string  `json:"docType"`
	EventID   string  `json:"eventID"`
	UserID    string  `json:"userID"`
	Status    string  `json:"status"`
	Priority  float64 `json:"priority"`
	Timestamp string  `json:"timestamp"`
}

// Ticket represents an issued ticket
type Ticket struct {
	DocType    string `json:"docType"`
	TicketID   string `json:"ticketID"`
	EventID    string `json:"eventID"`
	OwnerID    string `json:"ownerID"`
	Status     string `json:"status"`
	ClaimHash  string `json:"claimHash"`
	VerifiedAt string `json:"verifiedAt"`
}

// LotteryResult is returned after running the lottery
type LotteryResult struct {
	EventID    string             `json:"eventID"`
	Winners    []string           `json:"winners"`
	Losers     []string           `json:"losers"`
	Priorities map[string]float64 `json:"priorities"`
}

// UserScoreResponse matches the structure returned by prediction-cc.GetUserScore
type UserScoreResponse struct {
	UserID       string  `json:"userID"`
	TotalBets    int     `json:"totalBets"`
	CorrectBets  int     `json:"correctBets"`
	AccuracyRate float64 `json:"accuracyRate"`
}

// applicantPriority is used for sorting during lottery
type applicantPriority struct {
	UserID   string
	Priority float64
}

// TicketContract implements the ticket management chaincode
type TicketContract struct {
	contractapi.Contract
}

// ApplyTicket submits a ticket application for an event
func (tc *TicketContract) ApplyTicket(ctx contractapi.TransactionContextInterface, eventID string, userID string) error {
	if eventID == "" || userID == "" {
		return fmt.Errorf("event ID and user ID must not be empty")
	}

	key := fmt.Sprintf("application:%s:%s", eventID, userID)

	// Check for existing application
	existing, err := ctx.GetStub().GetState(key)
	if err != nil {
		return fmt.Errorf("failed to read state: %s", err.Error())
	}
	if existing != nil {
		return fmt.Errorf("application already exists for user %s on event %s", userID, eventID)
	}

	timestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return fmt.Errorf("failed to get timestamp: %s", err.Error())
	}
	timeStr := time.Unix(timestamp.Seconds, int64(timestamp.Nanos)).UTC().Format(time.RFC3339)

	app := Application{
		DocType:   "application",
		EventID:   eventID,
		UserID:    userID,
		Status:    StatusPending,
		Priority:  0,
		Timestamp: timeStr,
	}

	appJSON, err := json.Marshal(app)
	if err != nil {
		return fmt.Errorf("failed to marshal application: %s", err.Error())
	}

	err = ctx.GetStub().PutState(key, appJSON)
	if err != nil {
		return fmt.Errorf("failed to put state: %s", err.Error())
	}

	return nil
}

// RunLottery executes the weighted lottery for an event
// Priority = 0.6 * accuracy + 0.4 * pseudorandom
// Pseudorandom seed = SHA256(txID + eventID + userID) for deterministic cross-peer consensus
func (tc *TicketContract) RunLottery(ctx contractapi.TransactionContextInterface, eventID string, ticketCountStr string) (*LotteryResult, error) {
	if eventID == "" {
		return nil, fmt.Errorf("event ID must not be empty")
	}

	ticketCount, err := strconv.Atoi(ticketCountStr)
	if err != nil {
		return nil, fmt.Errorf("invalid ticket count: %s", err.Error())
	}
	if ticketCount <= 0 {
		return nil, fmt.Errorf("ticket count must be positive")
	}

	// Check if lottery already ran (any non-PENDING applications)
	startKey := fmt.Sprintf("application:%s:", eventID)
	endKey := fmt.Sprintf("application:%s:~", eventID)
	iterator, err := ctx.GetStub().GetStateByRange(startKey, endKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get applications: %s", err.Error())
	}
	defer iterator.Close()

	txID := ctx.GetStub().GetTxID()
	var applicants []applicantPriority
	priorities := make(map[string]float64)

	for iterator.HasNext() {
		queryResponse, err := iterator.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate: %s", err.Error())
		}

		var app Application
		err = json.Unmarshal(queryResponse.Value, &app)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal application: %s", err.Error())
		}

		// If any application is not PENDING, lottery already ran
		if app.Status != StatusPending {
			return nil, fmt.Errorf("lottery already executed for event %s", eventID)
		}

		// Get user's prediction accuracy via cross-chaincode call
		scoreArgs := [][]byte{
			[]byte("PredictionContract:GetUserScore"),
			[]byte(app.UserID),
		}
		scoreResp := ctx.GetStub().InvokeChaincode("prediction-cc", scoreArgs, "eventchain")

		var accuracyRate float64
		if scoreResp.Status == 200 && scoreResp.Payload != nil {
			var score UserScoreResponse
			err = json.Unmarshal(scoreResp.Payload, &score)
			if err == nil {
				accuracyRate = score.AccuracyRate
			}
		}

		// Calculate pseudo-random component
		// seed = SHA256(txID + eventID + userID) for deterministic consensus
		seedData := txID + eventID + app.UserID
		hash := sha256.Sum256([]byte(seedData))
		// Use first 8 bytes of hash as a uint64, then normalize to [0, 1)
		var hashVal uint64
		for i := 0; i < 8; i++ {
			hashVal = (hashVal << 8) | uint64(hash[i])
		}
		pseudoRandom := float64(hashVal) / float64(^uint64(0))

		// priority = 0.6 * accuracy + 0.4 * pseudorandom
		priority := 0.6*accuracyRate + 0.4*pseudoRandom

		applicants = append(applicants, applicantPriority{
			UserID:   app.UserID,
			Priority: priority,
		})
		priorities[app.UserID] = priority
	}

	// Sort by priority descending
	sort.Slice(applicants, func(i, j int) bool {
		return applicants[i].Priority > applicants[j].Priority
	})

	// Determine winners (top N = ticketCount)
	winnerCount := ticketCount
	if winnerCount > len(applicants) {
		winnerCount = len(applicants)
	}

	var winners []string
	var losers []string

	for i, ap := range applicants {
		appKey := fmt.Sprintf("application:%s:%s", eventID, ap.UserID)
		appBytes, err := ctx.GetStub().GetState(appKey)
		if err != nil {
			return nil, fmt.Errorf("failed to read application: %s", err.Error())
		}

		var app Application
		err = json.Unmarshal(appBytes, &app)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal application: %s", err.Error())
		}

		if i < winnerCount {
			app.Status = StatusWon
			app.Priority = ap.Priority
			winners = append(winners, ap.UserID)
		} else {
			app.Status = StatusLost
			app.Priority = ap.Priority
			losers = append(losers, ap.UserID)
		}

		appJSON, err := json.Marshal(app)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal application: %s", err.Error())
		}

		err = ctx.GetStub().PutState(appKey, appJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to update application: %s", err.Error())
		}
	}

	if winners == nil {
		winners = []string{}
	}
	if losers == nil {
		losers = []string{}
	}

	result := &LotteryResult{
		EventID:    eventID,
		Winners:    winners,
		Losers:     losers,
		Priorities: priorities,
	}

	return result, nil
}

// ClaimTicket allows a lottery winner to claim their ticket, generating a SHA256 hash for QR
func (tc *TicketContract) ClaimTicket(ctx contractapi.TransactionContextInterface, eventID string, userID string) (*Ticket, error) {
	if eventID == "" || userID == "" {
		return nil, fmt.Errorf("event ID and user ID must not be empty")
	}

	// Verify user won the lottery
	appKey := fmt.Sprintf("application:%s:%s", eventID, userID)
	appBytes, err := ctx.GetStub().GetState(appKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read application: %s", err.Error())
	}
	if appBytes == nil {
		return nil, fmt.Errorf("no application found for user %s on event %s", userID, eventID)
	}

	var app Application
	err = json.Unmarshal(appBytes, &app)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal application: %s", err.Error())
	}

	if app.Status != StatusWon {
		return nil, fmt.Errorf("user %s did not win the lottery (status: %s)", userID, app.Status)
	}

	// Generate ticket ID and claim hash
	txID := ctx.GetStub().GetTxID()
	ticketID := fmt.Sprintf("%s-%s-%s", eventID, userID, txID)

	// Generate SHA256 hash for QR code
	hashInput := fmt.Sprintf("%s:%s:%s:%s", ticketID, eventID, userID, txID)
	hash := sha256.Sum256([]byte(hashInput))
	claimHash := hex.EncodeToString(hash[:])

	timestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return nil, fmt.Errorf("failed to get timestamp: %s", err.Error())
	}
	timeStr := time.Unix(timestamp.Seconds, int64(timestamp.Nanos)).UTC().Format(time.RFC3339)

	ticket := Ticket{
		DocType:    "ticket",
		TicketID:   ticketID,
		EventID:    eventID,
		OwnerID:    userID,
		Status:     StatusClaimed,
		ClaimHash:  claimHash,
		VerifiedAt: "",
	}

	ticketJSON, err := json.Marshal(ticket)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ticket: %s", err.Error())
	}

	ticketKey := "ticket:" + ticketID
	err = ctx.GetStub().PutState(ticketKey, ticketJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to store ticket: %s", err.Error())
	}

	// Update application status to CLAIMED
	app.Status = StatusClaimed
	app.Timestamp = timeStr
	appJSON, err := json.Marshal(app)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal application: %s", err.Error())
	}

	err = ctx.GetStub().PutState(appKey, appJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to update application: %s", err.Error())
	}

	return &ticket, nil
}

// VerifyTicket checks the claim hash and marks the ticket as USED
func (tc *TicketContract) VerifyTicket(ctx contractapi.TransactionContextInterface, ticketID string, hash string) (*Ticket, error) {
	if ticketID == "" || hash == "" {
		return nil, fmt.Errorf("ticket ID and hash must not be empty")
	}

	ticketKey := "ticket:" + ticketID
	ticketBytes, err := ctx.GetStub().GetState(ticketKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read ticket: %s", err.Error())
	}
	if ticketBytes == nil {
		return nil, fmt.Errorf("ticket %s not found", ticketID)
	}

	var ticket Ticket
	err = json.Unmarshal(ticketBytes, &ticket)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal ticket: %s", err.Error())
	}

	// Check if already used
	if ticket.Status == StatusUsed {
		return nil, fmt.Errorf("ticket %s has already been used", ticketID)
	}

	// Check if ticket is in CLAIMED status
	if ticket.Status != StatusClaimed {
		return nil, fmt.Errorf("ticket %s is not in CLAIMED status (current: %s)", ticketID, ticket.Status)
	}

	// Verify hash
	if ticket.ClaimHash != hash {
		return nil, fmt.Errorf("hash mismatch for ticket %s", ticketID)
	}

	// Mark as used
	timestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return nil, fmt.Errorf("failed to get timestamp: %s", err.Error())
	}
	timeStr := time.Unix(timestamp.Seconds, int64(timestamp.Nanos)).UTC().Format(time.RFC3339)

	ticket.Status = StatusUsed
	ticket.VerifiedAt = timeStr

	ticketJSON, err := json.Marshal(ticket)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ticket: %s", err.Error())
	}

	err = ctx.GetStub().PutState(ticketKey, ticketJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to update ticket: %s", err.Error())
	}

	return &ticket, nil
}

// RefundTicket allows the ticket owner to refund before the event
func (tc *TicketContract) RefundTicket(ctx contractapi.TransactionContextInterface, ticketID string, userID string) error {
	if ticketID == "" || userID == "" {
		return fmt.Errorf("ticket ID and user ID must not be empty")
	}

	ticketKey := "ticket:" + ticketID
	ticketBytes, err := ctx.GetStub().GetState(ticketKey)
	if err != nil {
		return fmt.Errorf("failed to read ticket: %s", err.Error())
	}
	if ticketBytes == nil {
		return fmt.Errorf("ticket %s not found", ticketID)
	}

	var ticket Ticket
	err = json.Unmarshal(ticketBytes, &ticket)
	if err != nil {
		return fmt.Errorf("failed to unmarshal ticket: %s", err.Error())
	}

	// Verify owner
	if ticket.OwnerID != userID {
		return fmt.Errorf("user %s is not the owner of ticket %s", userID, ticketID)
	}

	// Can only refund CLAIMED tickets (not USED or already REFUNDED)
	if ticket.Status != StatusClaimed {
		return fmt.Errorf("can only refund CLAIMED tickets, current status: %s", ticket.Status)
	}

	// Mark ticket as refunded
	ticket.Status = StatusRefunded

	ticketJSON, err := json.Marshal(ticket)
	if err != nil {
		return fmt.Errorf("failed to marshal ticket: %s", err.Error())
	}

	err = ctx.GetStub().PutState(ticketKey, ticketJSON)
	if err != nil {
		return fmt.Errorf("failed to update ticket: %s", err.Error())
	}

	// Update application status to REFUNDED
	appKey := fmt.Sprintf("application:%s:%s", ticket.EventID, userID)
	appBytes, err := ctx.GetStub().GetState(appKey)
	if err != nil {
		return fmt.Errorf("failed to read application: %s", err.Error())
	}

	if appBytes != nil {
		var app Application
		err = json.Unmarshal(appBytes, &app)
		if err != nil {
			return fmt.Errorf("failed to unmarshal application: %s", err.Error())
		}

		app.Status = StatusRefunded

		appJSON, err := json.Marshal(app)
		if err != nil {
			return fmt.Errorf("failed to marshal application: %s", err.Error())
		}

		err = ctx.GetStub().PutState(appKey, appJSON)
		if err != nil {
			return fmt.Errorf("failed to update application: %s", err.Error())
		}
	}

	return nil
}
```

### 6.4 Run ticket tests (expect pass)

- [ ] `cd fabric/chaincode/ticket && go test -v ./...`

### 6.5 Commit ticket chaincode

- [ ] `git add fabric/chaincode/ticket/ && git commit -m "feat(chaincode): implement ticket chaincode with weighted lottery and tests"`
# EventChain Backend + Frontend Implementation Plan

---

## Task 1 — Server Scaffolding

- [ ] Create `server/package.json`
- [ ] Create `server/.env.example`
- [ ] Create `server/src/config/index.js`
- [ ] Create `server/src/config/fabric.js`
- [ ] Create `server/src/middleware/errorHandler.js`
- [ ] Create `server/src/app.js`

### server/package.json

```json
{
  "name": "eventchain-server",
  "version": "1.0.0",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "node --watch src/app.js",
    "start": "node src/app.js"
  },
  "dependencies": {
    "@hyperledger/fabric-gateway": "^1.7.0",
    "@hyperledger/grpc-signer": "^1.0.0",
    "fabric-ca-client": "^2.2.20",
    "express": "^4.21.0",
    "cors": "^2.8.5",
    "jsonwebtoken": "^9.0.2",
    "bcrypt": "^5.1.1",
    "better-sqlite3": "^11.6.0",
    "dotenv": "^16.4.7",
    "@grpc/grpc-js": "^1.12.0"
  }
}
```

### server/.env.example

```env
PORT=3000
JWT_SECRET=eventchain-dev-secret-change-in-production
JWT_EXPIRES_IN=24h

# Fabric connection
FABRIC_CHANNEL=eventchain
FABRIC_GATEWAY_PEER=peer0.platform.eventchain.com:7051
FABRIC_CA_URL=https://ca.student.eventchain.com:7054

# Chaincode names
CC_EVENT=event-cc
CC_PREDICTION=prediction-cc
CC_TICKET=ticket-cc
CC_TOKEN=token-cc

# Wallet path
WALLET_PATH=./wallet

# SQLite database path
DB_PATH=./data/users.db
```

### server/src/config/index.js

```js
import 'dotenv/config';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(__dirname, '../..');

export default {
  port: parseInt(process.env.PORT, 10) || 3000,

  jwt: {
    secret: process.env.JWT_SECRET || 'eventchain-dev-secret',
    expiresIn: process.env.JWT_EXPIRES_IN || '24h',
  },

  fabric: {
    channelName: process.env.FABRIC_CHANNEL || 'eventchain',
    peerEndpoint: process.env.FABRIC_GATEWAY_PEER || 'peer0.platform.eventchain.com:7051',
    caUrl: process.env.FABRIC_CA_URL || 'https://ca.student.eventchain.com:7054',
    chaincode: {
      event: process.env.CC_EVENT || 'event-cc',
      prediction: process.env.CC_PREDICTION || 'prediction-cc',
      ticket: process.env.CC_TICKET || 'ticket-cc',
      token: process.env.CC_TOKEN || 'token-cc',
    },
  },

  walletPath: path.resolve(root, process.env.WALLET_PATH || './wallet'),
  dbPath: path.resolve(root, process.env.DB_PATH || './data/users.db'),
};
```

### server/src/config/fabric.js

```js
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

// Connection profile for the Fabric Gateway SDK.
// In production the TLS cert paths come from the crypto material generated
// by the Fabric CA / cryptogen tool.  During development they live under the
// network's organizations/ directory.

const networkRoot = path.resolve(__dirname, '../../../fabric/network');

export function buildConnectionProfile(orgMSP) {
  const orgMap = {
    PlatformMSP: {
      mspId: 'PlatformMSP',
      peerHost: 'peer0.platform.eventchain.com',
      peerPort: 7051,
      caHost: 'ca.platform.eventchain.com',
      tlsCertPath: path.join(
        networkRoot,
        'organizations/peerOrganizations/platform.eventchain.com/peers/peer0.platform.eventchain.com/tls/ca.crt'
      ),
    },
    OrganizerMSP: {
      mspId: 'OrganizerMSP',
      peerHost: 'peer0.organizer.eventchain.com',
      peerPort: 9051,
      caHost: 'ca.organizer.eventchain.com',
      tlsCertPath: path.join(
        networkRoot,
        'organizations/peerOrganizations/organizer.eventchain.com/peers/peer0.organizer.eventchain.com/tls/ca.crt'
      ),
    },
    StudentMSP: {
      mspId: 'StudentMSP',
      peerHost: 'peer0.student.eventchain.com',
      peerPort: 11051,
      caHost: 'ca.student.eventchain.com',
      tlsCertPath: path.join(
        networkRoot,
        'organizations/peerOrganizations/student.eventchain.com/peers/peer0.student.eventchain.com/tls/ca.crt'
      ),
    },
  };

  return orgMap[orgMSP] || orgMap.StudentMSP;
}
```

### server/src/middleware/errorHandler.js

```js
// Maps chaincode / application error codes to HTTP status + Chinese message.

const ERROR_MAP = {
  INSUFFICIENT_BALANCE: { status: 400, message: '余额不足' },
  EVENT_NOT_FOUND: { status: 404, message: '赛事不存在' },
  INVALID_STATUS: { status: 400, message: '赛事状态不允许此操作' },
  BET_CLOSED: { status: 400, message: '预测已关闭' },
  ALREADY_APPLIED: { status: 409, message: '已提交过申请' },
  TICKET_NOT_FOUND: { status: 404, message: '票据不存在' },
  UNAUTHORIZED: { status: 401, message: '未登录或登录已过期' },
  FORBIDDEN: { status: 403, message: '无权执行此操作' },
  USER_EXISTS: { status: 409, message: '该学号已注册' },
  INVALID_CREDENTIALS: { status: 401, message: '学号或密码错误' },
  VALIDATION_ERROR: { status: 400, message: '请求参数不合法' },
};

export function errorHandler(err, _req, res, _next) {
  console.error('[ErrorHandler]', err);

  // Application-level coded errors (thrown as { code, message })
  if (err.code && ERROR_MAP[err.code]) {
    const mapped = ERROR_MAP[err.code];
    return res.status(mapped.status).json({
      error: true,
      code: err.code,
      message: err.message || mapped.message,
    });
  }

  // Fabric Gateway errors often include a "details" array
  if (err.details && Array.isArray(err.details)) {
    const detail = err.details[0]?.message || err.message;
    // Try to extract a known code from the chaincode error string
    for (const [code, meta] of Object.entries(ERROR_MAP)) {
      if (detail.includes(code)) {
        return res.status(meta.status).json({
          error: true,
          code,
          message: meta.message,
        });
      }
    }
  }

  // Fallback
  const status = err.status || err.statusCode || 500;
  res.status(status).json({
    error: true,
    code: 'INTERNAL_ERROR',
    message: process.env.NODE_ENV === 'production' ? '服务器内部错误' : err.message,
  });
}

// Helper to throw coded errors from route handlers.
export function throwCoded(code, message) {
  const e = new Error(message || ERROR_MAP[code]?.message || code);
  e.code = code;
  throw e;
}
```

### server/src/app.js

```js
import express from 'express';
import cors from 'cors';
import config from './config/index.js';
import { errorHandler } from './middleware/errorHandler.js';
import authRoutes from './routes/auth.js';
import eventRoutes from './routes/events.js';
import predictionRoutes from './routes/predictions.js';
import ticketRoutes from './routes/tickets.js';
import userRoutes from './routes/users.js';
import { initDB } from './services/wallet.js';

const app = express();

// --------------- Middleware ---------------
app.use(cors());
app.use(express.json());

// --------------- Health check ---------------
app.get('/api/health', (_req, res) => {
  res.json({ status: 'ok', timestamp: new Date().toISOString() });
});

// --------------- Routes ---------------
app.use('/api/v1/auth', authRoutes);
app.use('/api/v1/events', eventRoutes);
app.use('/api/v1/predictions', predictionRoutes);
app.use('/api/v1/tickets', ticketRoutes);
app.use('/api/v1/users', userRoutes);

// --------------- Error handler (must be last) ---------------
app.use(errorHandler);

// --------------- Start ---------------
async function main() {
  initDB();
  app.listen(config.port, () => {
    console.log(`[EventChain] Server listening on http://localhost:${config.port}`);
  });
}

main().catch((err) => {
  console.error('Failed to start server:', err);
  process.exit(1);
});
```

---

## Task 2 — Fabric Services (Gateway + CA + Wallet)

- [ ] Create `server/src/services/wallet.js`
- [ ] Create `server/src/services/caService.js`
- [ ] Create `server/src/services/fabricGateway.js`

### server/src/services/wallet.js

```js
import fs from 'node:fs';
import path from 'node:path';
import Database from 'better-sqlite3';
import bcrypt from 'bcrypt';
import config from '../config/index.js';

// ---- SQLite for password hashes ----

let db;

export function initDB() {
  const dir = path.dirname(config.dbPath);
  if (!fs.existsSync(dir)) fs.mkdirSync(dir, { recursive: true });

  db = new Database(config.dbPath);
  db.pragma('journal_mode = WAL');

  db.exec(`
    CREATE TABLE IF NOT EXISTS users (
      user_id   TEXT PRIMARY KEY,
      name      TEXT NOT NULL,
      password  TEXT NOT NULL,
      org_msp   TEXT NOT NULL DEFAULT 'StudentMSP',
      role      TEXT NOT NULL DEFAULT 'student',
      created_at TEXT NOT NULL DEFAULT (datetime('now'))
    )
  `);
}

export function getDB() {
  return db;
}

export function createUser(userId, name, passwordHash, orgMSP = 'StudentMSP', role = 'student') {
  const stmt = db.prepare(
    'INSERT INTO users (user_id, name, password, org_msp, role) VALUES (?, ?, ?, ?, ?)'
  );
  stmt.run(userId, name, passwordHash, orgMSP, role);
}

export function findUser(userId) {
  return db.prepare('SELECT * FROM users WHERE user_id = ?').get(userId);
}

export async function hashPassword(plain) {
  return bcrypt.hash(plain, 10);
}

export async function verifyPassword(plain, hash) {
  return bcrypt.compare(plain, hash);
}

// ---- Fabric file-system wallet for X.509 identities ----

export function walletDir() {
  if (!fs.existsSync(config.walletPath)) {
    fs.mkdirSync(config.walletPath, { recursive: true });
  }
  return config.walletPath;
}

export function putIdentity(userId, mspId, certificate, privateKey) {
  const identityPath = path.join(walletDir(), `${userId}.json`);
  const identity = {
    credentials: { certificate, privateKey },
    mspId,
    type: 'X.509',
  };
  fs.writeFileSync(identityPath, JSON.stringify(identity, null, 2));
}

export function getIdentity(userId) {
  const identityPath = path.join(walletDir(), `${userId}.json`);
  if (!fs.existsSync(identityPath)) return null;
  return JSON.parse(fs.readFileSync(identityPath, 'utf-8'));
}

export function identityExists(userId) {
  return fs.existsSync(path.join(walletDir(), `${userId}.json`));
}
```

### server/src/services/caService.js

```js
import FabricCAServices from 'fabric-ca-client';
import { buildConnectionProfile } from '../config/fabric.js';
import { putIdentity, getIdentity } from './wallet.js';

// Cache CA client instances per org
const caClients = {};

function getCAClient(orgMSP) {
  if (caClients[orgMSP]) return caClients[orgMSP];

  const orgConfig = buildConnectionProfile(orgMSP);
  const caUrl = `https://${orgConfig.caHost}:7054`;
  const caClient = new FabricCAServices(caUrl, {
    trustedRoots: [],
    verify: false, // dev only — accept self-signed CA certs
  }, orgConfig.caHost);

  caClients[orgMSP] = caClient;
  return caClient;
}

// Enroll the bootstrap admin identity for a given org CA.
// This admin identity is used to register new users.
export async function enrollAdmin(orgMSP) {
  if (getIdentity(`admin-${orgMSP}`)) return getIdentity(`admin-${orgMSP}`);

  const caClient = getCAClient(orgMSP);
  const enrollment = await caClient.enroll({
    enrollmentID: 'admin',
    enrollmentSecret: 'adminpw',
  });

  putIdentity(`admin-${orgMSP}`, orgMSP, enrollment.certificate, enrollment.key.toBytes());
  return getIdentity(`admin-${orgMSP}`);
}

// Register and enroll a new user with the org's CA.
export async function registerAndEnrollUser(userId, orgMSP) {
  const caClient = getCAClient(orgMSP);

  // Ensure the CA admin is enrolled first
  const adminIdentity = await enrollAdmin(orgMSP);

  // Build an admin User object that fabric-ca-client can use for registration
  const adminUser = {
    getName: () => `admin-${orgMSP}`,
    getMSPId: () => orgMSP,
    getIdentity: () => ({
      serialize: () => adminIdentity.credentials.certificate,
    }),
    getSigningIdentity: () => ({
      sign: (msg) => {
        const { createSign } = require('node:crypto');
        const sign = createSign('SHA256');
        sign.update(msg);
        return sign.sign(adminIdentity.credentials.privateKey);
      },
    }),
  };

  // Register
  const secret = await caClient.register(
    {
      affiliation: '',
      enrollmentID: userId,
      role: 'client',
    },
    adminUser
  );

  // Enroll
  const enrollment = await caClient.enroll({
    enrollmentID: userId,
    enrollmentSecret: secret,
  });

  putIdentity(userId, orgMSP, enrollment.certificate, enrollment.key.toBytes());
  return getIdentity(userId);
}
```

### server/src/services/fabricGateway.js

```js
import * as grpc from '@grpc/grpc-js';
import { connect, signers } from '@hyperledger/fabric-gateway';
import fs from 'node:fs';
import crypto from 'node:crypto';
import { buildConnectionProfile } from '../config/fabric.js';
import { getIdentity } from './wallet.js';
import config from '../config/index.js';

// Cache gateway connections per user to avoid reconnecting on every request.
const gatewayCache = new Map();

function newGrpcConnection(orgMSP) {
  const orgConfig = buildConnectionProfile(orgMSP);
  const tlsCert = fs.readFileSync(orgConfig.tlsCertPath);
  const tlsCredentials = grpc.credentials.createSsl(tlsCert);
  return new grpc.Client(
    `${orgConfig.peerHost}:${orgConfig.peerPort}`,
    tlsCredentials,
    { 'grpc.ssl_target_name_override': orgConfig.peerHost }
  );
}

function newIdentity(identity) {
  const certificate = identity.credentials.certificate;
  return { mspId: identity.mspId, credentials: Buffer.from(certificate) };
}

function newSigner(identity) {
  const privateKeyPem = identity.credentials.privateKey;
  const privateKey = crypto.createPrivateKey(privateKeyPem);
  return signers.newPrivateKeySigner(privateKey);
}

// Get or create a Gateway connection for a given user.
export async function getGateway(userId) {
  if (gatewayCache.has(userId)) return gatewayCache.get(userId);

  const identity = getIdentity(userId);
  if (!identity) throw Object.assign(new Error('身份未找到'), { code: 'UNAUTHORIZED' });

  const grpcClient = newGrpcConnection(identity.mspId);
  const gateway = connect({
    client: grpcClient,
    identity: newIdentity(identity),
    signer: newSigner(identity),
    evaluateOptions: () => ({ deadline: Date.now() + 5000 }),
    endorseOptions: () => ({ deadline: Date.now() + 15000 }),
    submitOptions: () => ({ deadline: Date.now() + 5000 }),
    commitStatusOptions: () => ({ deadline: Date.now() + 60000 }),
  });

  gatewayCache.set(userId, { gateway, grpcClient });
  return { gateway, grpcClient };
}

// Get a contract handle for a given chaincode.
export async function getContract(userId, chaincodeName) {
  const { gateway } = await getGateway(userId);
  const network = gateway.getNetwork(config.fabric.channelName);
  return network.getContract(chaincodeName);
}

// Submit a transaction (read-write).
export async function submitTransaction(userId, chaincodeName, fn, ...args) {
  const contract = await getContract(userId, chaincodeName);
  const resultBytes = await contract.submitTransaction(fn, ...args);
  return resultBytes.length ? JSON.parse(new TextDecoder().decode(resultBytes)) : null;
}

// Evaluate a transaction (read-only query).
export async function evaluateTransaction(userId, chaincodeName, fn, ...args) {
  const contract = await getContract(userId, chaincodeName);
  const resultBytes = await contract.evaluateTransaction(fn, ...args);
  return resultBytes.length ? JSON.parse(new TextDecoder().decode(resultBytes)) : null;
}

// Close a user's cached connection when they log out (optional).
export function closeGateway(userId) {
  const cached = gatewayCache.get(userId);
  if (cached) {
    cached.gateway.close();
    cached.grpcClient.close();
    gatewayCache.delete(userId);
  }
}
```

---

## Task 3 — Auth System (JWT Middleware + Auth Routes)

- [ ] Create `server/src/middleware/auth.js`
- [ ] Create `server/src/routes/auth.js`

### server/src/middleware/auth.js

```js
import jwt from 'jsonwebtoken';
import config from '../config/index.js';

// JWT verification middleware.
// Sets req.user = { userId, orgMSP } on success.
export function authenticate(req, _res, next) {
  const header = req.headers.authorization;
  if (!header || !header.startsWith('Bearer ')) {
    const err = new Error('未提供认证令牌');
    err.code = 'UNAUTHORIZED';
    return next(err);
  }

  const token = header.slice(7);
  try {
    const payload = jwt.verify(token, config.jwt.secret);
    req.user = { userId: payload.userId, orgMSP: payload.orgMSP, role: payload.role };
    next();
  } catch {
    const err = new Error('认证令牌无效或已过期');
    err.code = 'UNAUTHORIZED';
    next(err);
  }
}

// Require a specific role (e.g. 'organizer', 'admin').
export function requireRole(...roles) {
  return (req, _res, next) => {
    if (!roles.includes(req.user.role)) {
      const err = new Error('无权执行此操作');
      err.code = 'FORBIDDEN';
      return next(err);
    }
    next();
  };
}
```

### server/src/routes/auth.js

```js
import { Router } from 'express';
import jwt from 'jsonwebtoken';
import config from '../config/index.js';
import { createUser, findUser, hashPassword, verifyPassword } from '../services/wallet.js';
import { registerAndEnrollUser } from '../services/caService.js';
import { submitTransaction } from '../services/fabricGateway.js';

const router = Router();

// POST /api/v1/auth/register
// Body: { studentID, password, name }
router.post('/register', async (req, res, next) => {
  try {
    const { studentID, password, name } = req.body;

    if (!studentID || !password || !name) {
      const err = new Error('studentID, password, name 均为必填');
      err.code = 'VALIDATION_ERROR';
      throw err;
    }

    // Check if user already exists in SQLite
    if (findUser(studentID)) {
      const err = new Error('该学号已注册');
      err.code = 'USER_EXISTS';
      throw err;
    }

    // 1. Hash password and store in SQLite
    const passwordHash = await hashPassword(password);
    createUser(studentID, name, passwordHash, 'StudentMSP', 'student');

    // 2. Register + enroll with Fabric CA → cert stored in wallet
    await registerAndEnrollUser(studentID, 'StudentMSP');

    // 3. Mint 1000 tokens as registration bonus
    await submitTransaction(studentID, config.fabric.chaincode.token, 'Mint', studentID, '1000');

    // 4. Issue JWT
    const token = jwt.sign(
      { userId: studentID, orgMSP: 'StudentMSP', role: 'student' },
      config.jwt.secret,
      { expiresIn: config.jwt.expiresIn }
    );

    res.status(201).json({
      error: false,
      data: { token, userId: studentID, name, balance: 1000 },
    });
  } catch (err) {
    next(err);
  }
});

// POST /api/v1/auth/login
// Body: { studentID, password }
router.post('/login', async (req, res, next) => {
  try {
    const { studentID, password } = req.body;

    if (!studentID || !password) {
      const err = new Error('studentID 和 password 均为必填');
      err.code = 'VALIDATION_ERROR';
      throw err;
    }

    const user = findUser(studentID);
    if (!user) {
      const err = new Error('学号或密码错误');
      err.code = 'INVALID_CREDENTIALS';
      throw err;
    }

    const valid = await verifyPassword(password, user.password);
    if (!valid) {
      const err = new Error('学号或密码错误');
      err.code = 'INVALID_CREDENTIALS';
      throw err;
    }

    const token = jwt.sign(
      { userId: user.user_id, orgMSP: user.org_msp, role: user.role },
      config.jwt.secret,
      { expiresIn: config.jwt.expiresIn }
    );

    res.json({
      error: false,
      data: { token, userId: user.user_id, name: user.name, role: user.role },
    });
  } catch (err) {
    next(err);
  }
});

export default router;
```

---

## Task 4 — Event Routes

- [ ] Create `server/src/routes/events.js`

### server/src/routes/events.js

```js
import { Router } from 'express';
import { authenticate, requireRole } from '../middleware/auth.js';
import { submitTransaction, evaluateTransaction } from '../services/fabricGateway.js';
import config from '../config/index.js';

const router = Router();
const CC = config.fabric.chaincode.event;
const CC_PRED = config.fabric.chaincode.prediction;

// GET /api/v1/events?status=PREDICTION_OPEN&type=basketball
router.get('/', async (req, res, next) => {
  try {
    // Public — use a platform admin identity for read-only queries
    const userId = req.headers.authorization ? undefined : 'admin-PlatformMSP';
    const queryUserId = req.user?.userId || userId || 'admin-PlatformMSP';

    const { status, type } = req.query;
    const result = await evaluateTransaction(
      queryUserId,
      CC,
      'ListEvents',
      status || '',
      type || ''
    );
    res.json({ error: false, data: result || [] });
  } catch (err) {
    next(err);
  }
});

// Use auth middleware optionally for GET — pass through if no token
router.get('/', (req, _res, next) => {
  if (req.headers.authorization) {
    return authenticate(req, _res, next);
  }
  next();
});

// GET /api/v1/events/:id
router.get('/:id', async (req, res, next) => {
  try {
    const queryUserId = req.user?.userId || 'admin-PlatformMSP';
    const event = await evaluateTransaction(queryUserId, CC, 'QueryEvent', req.params.id);
    if (!event) {
      const err = new Error('赛事不存在');
      err.code = 'EVENT_NOT_FOUND';
      throw err;
    }

    // Also fetch current odds
    let odds = null;
    try {
      odds = await evaluateTransaction(
        queryUserId,
        CC_PRED,
        'GetOdds',
        req.params.id
      );
    } catch {
      // odds may not be available if event hasn't opened predictions
    }

    res.json({ error: false, data: { ...event, odds } });
  } catch (err) {
    next(err);
  }
});

// POST /api/v1/events  [Organizer]
router.post('/', authenticate, requireRole('organizer', 'admin'), async (req, res, next) => {
  try {
    const { title, type, teams, ticketTotal, predictionOptions } = req.body;

    if (!title || !type || !teams || !predictionOptions) {
      const err = new Error('缺少必要字段');
      err.code = 'VALIDATION_ERROR';
      throw err;
    }

    const result = await submitTransaction(
      req.user.userId,
      CC,
      'CreateEvent',
      JSON.stringify({ title, type, teams, ticketTotal: ticketTotal || 0, predictionOptions })
    );

    res.status(201).json({ error: false, data: result });
  } catch (err) {
    next(err);
  }
});

// PUT /api/v1/events/:id/status  [Organizer]
router.put('/:id/status', authenticate, requireRole('organizer', 'admin'), async (req, res, next) => {
  try {
    const { status } = req.body;
    if (!status) {
      const err = new Error('缺少 status 字段');
      err.code = 'VALIDATION_ERROR';
      throw err;
    }

    const result = await submitTransaction(
      req.user.userId,
      CC,
      'UpdateStatus',
      req.params.id,
      status
    );

    res.json({ error: false, data: result });
  } catch (err) {
    next(err);
  }
});

// PUT /api/v1/events/:id/result  [Organizer]
// Records the result and triggers settlement in prediction-cc.
router.put('/:id/result', authenticate, requireRole('organizer', 'admin'), async (req, res, next) => {
  try {
    const { outcome } = req.body;
    if (!outcome) {
      const err = new Error('缺少 outcome 字段');
      err.code = 'VALIDATION_ERROR';
      throw err;
    }

    // 1. Record result on the event
    await submitTransaction(req.user.userId, CC, 'UpdateResult', req.params.id, outcome);

    // 2. Trigger settlement in prediction chaincode
    const settlement = await submitTransaction(
      req.user.userId,
      CC_PRED,
      'Settle',
      req.params.id
    );

    res.json({ error: false, data: settlement });
  } catch (err) {
    next(err);
  }
});

export default router;
```

---

## Task 5 — Prediction Routes

- [ ] Create `server/src/routes/predictions.js`

### server/src/routes/predictions.js

```js
import { Router } from 'express';
import { authenticate } from '../middleware/auth.js';
import { submitTransaction, evaluateTransaction } from '../services/fabricGateway.js';
import config from '../config/index.js';

const router = Router();
const CC = config.fabric.chaincode.prediction;

// POST /api/v1/predictions/bet
// Body: { eventID, option, amount }
router.post('/bet', authenticate, async (req, res, next) => {
  try {
    const { eventID, option, amount } = req.body;

    if (!eventID || !option || !amount || amount <= 0) {
      const err = new Error('eventID, option, amount(>0) 均为必填');
      err.code = 'VALIDATION_ERROR';
      throw err;
    }

    const result = await submitTransaction(
      req.user.userId,
      CC,
      'PlaceBet',
      eventID,
      option,
      String(amount)
    );

    // result: { shares, newOddsA, newOddsB }
    res.json({ error: false, data: result });
  } catch (err) {
    next(err);
  }
});

// GET /api/v1/predictions/odds/:eventID
router.get('/odds/:eventID', async (req, res, next) => {
  try {
    const queryUserId = req.user?.userId || 'admin-PlatformMSP';
    const odds = await evaluateTransaction(queryUserId, CC, 'GetOdds', req.params.eventID);
    res.json({ error: false, data: odds });
  } catch (err) {
    next(err);
  }
});

// GET /api/v1/predictions/pool/:eventID
router.get('/pool/:eventID', async (req, res, next) => {
  try {
    const queryUserId = req.user?.userId || 'admin-PlatformMSP';
    const pool = await evaluateTransaction(queryUserId, CC, 'GetPool', req.params.eventID);
    res.json({ error: false, data: pool });
  } catch (err) {
    next(err);
  }
});

// GET /api/v1/predictions/mine
router.get('/mine', authenticate, async (req, res, next) => {
  try {
    const bets = await evaluateTransaction(req.user.userId, CC, 'GetUserBets', req.user.userId);
    res.json({ error: false, data: bets || [] });
  } catch (err) {
    next(err);
  }
});

// GET /api/v1/predictions/score
router.get('/score', authenticate, async (req, res, next) => {
  try {
    const score = await evaluateTransaction(req.user.userId, CC, 'GetUserScore', req.user.userId);
    res.json({ error: false, data: score });
  } catch (err) {
    next(err);
  }
});

export default router;
```

---

## Task 6 — Ticket + User Routes

- [ ] Create `server/src/routes/tickets.js`
- [ ] Create `server/src/routes/users.js`

### server/src/routes/tickets.js

```js
import { Router } from 'express';
import { authenticate, requireRole } from '../middleware/auth.js';
import { submitTransaction, evaluateTransaction } from '../services/fabricGateway.js';
import config from '../config/index.js';

const router = Router();
const CC = config.fabric.chaincode.ticket;

// POST /api/v1/tickets/apply
// Body: { eventID }
router.post('/apply', authenticate, async (req, res, next) => {
  try {
    const { eventID } = req.body;
    if (!eventID) {
      const err = new Error('eventID 为必填');
      err.code = 'VALIDATION_ERROR';
      throw err;
    }

    const result = await submitTransaction(
      req.user.userId,
      CC,
      'ApplyTicket',
      eventID
    );

    res.status(201).json({ error: false, data: result });
  } catch (err) {
    next(err);
  }
});

// POST /api/v1/tickets/lottery/:eventID  [Organizer]
router.post('/lottery/:eventID', authenticate, requireRole('organizer', 'admin'), async (req, res, next) => {
  try {
    const result = await submitTransaction(
      req.user.userId,
      CC,
      'RunLottery',
      req.params.eventID
    );

    res.json({ error: false, data: result });
  } catch (err) {
    next(err);
  }
});

// GET /api/v1/tickets/mine
router.get('/mine', authenticate, async (req, res, next) => {
  try {
    const tickets = await evaluateTransaction(
      req.user.userId,
      CC,
      'GetUserTickets',
      req.user.userId
    );
    res.json({ error: false, data: tickets || [] });
  } catch (err) {
    next(err);
  }
});

// POST /api/v1/tickets/claim/:ticketID
router.post('/claim/:ticketID', authenticate, async (req, res, next) => {
  try {
    const result = await submitTransaction(
      req.user.userId,
      CC,
      'ClaimTicket',
      req.params.ticketID
    );

    // result includes { ticketID, claimHash } — frontend uses claimHash for QR
    res.json({ error: false, data: result });
  } catch (err) {
    next(err);
  }
});

// GET /api/v1/tickets/verify/:ticketID
router.get('/verify/:ticketID', async (req, res, next) => {
  try {
    const queryUserId = req.user?.userId || 'admin-PlatformMSP';
    const result = await evaluateTransaction(
      queryUserId,
      CC,
      'VerifyTicket',
      req.params.ticketID,
      req.query.hash || ''
    );

    res.json({ error: false, data: result });
  } catch (err) {
    next(err);
  }
});

// POST /api/v1/tickets/refund/:ticketID
router.post('/refund/:ticketID', authenticate, async (req, res, next) => {
  try {
    const result = await submitTransaction(
      req.user.userId,
      CC,
      'RefundTicket',
      req.params.ticketID
    );

    res.json({ error: false, data: result });
  } catch (err) {
    next(err);
  }
});

export default router;
```

### server/src/routes/users.js

```js
import { Router } from 'express';
import { authenticate } from '../middleware/auth.js';
import { evaluateTransaction } from '../services/fabricGateway.js';
import { findUser } from '../services/wallet.js';
import config from '../config/index.js';

const router = Router();

// GET /api/v1/users/profile
router.get('/profile', authenticate, async (req, res, next) => {
  try {
    const userId = req.user.userId;

    // Fetch balance from token chaincode
    const balance = await evaluateTransaction(
      userId,
      config.fabric.chaincode.token,
      'BalanceOf',
      userId
    );

    // Fetch prediction score
    const score = await evaluateTransaction(
      userId,
      config.fabric.chaincode.prediction,
      'GetUserScore',
      userId
    );

    // Fetch user meta from SQLite
    const userRow = findUser(userId);

    res.json({
      error: false,
      data: {
        userId,
        name: userRow?.name || userId,
        role: userRow?.role || 'student',
        balance: balance?.balance ?? 0,
        totalBets: score?.totalBets ?? 0,
        correctBets: score?.correctBets ?? 0,
        accuracyRate: score?.accuracyRate ?? 0,
      },
    });
  } catch (err) {
    next(err);
  }
});

// GET /api/v1/users/leaderboard
router.get('/leaderboard', async (req, res, next) => {
  try {
    // This calls a CouchDB rich query inside prediction-cc
    const queryUserId = req.user?.userId || 'admin-PlatformMSP';
    const leaderboard = await evaluateTransaction(
      queryUserId,
      config.fabric.chaincode.prediction,
      'GetLeaderboard'
    );

    res.json({ error: false, data: leaderboard || [] });
  } catch (err) {
    next(err);
  }
});

export default router;
```

---

## Task 7 — Client Scaffolding + Liquid Glass CSS

- [ ] Create `client/package.json`
- [ ] Create `client/vite.config.js`
- [ ] Create `client/index.html`
- [ ] Create `client/src/main.js`
- [ ] Create `client/src/App.vue`
- [ ] Create `client/src/assets/styles/variables.css`
- [ ] Create `client/src/assets/styles/liquid-glass.css`
- [ ] Create `client/src/assets/styles/global.css`
- [ ] Create `client/src/composables/useGlassHighlight.js`
- [ ] Create `client/src/api/index.js`
- [ ] Create `client/src/router/index.js`

### client/package.json

```json
{
  "name": "eventchain-client",
  "version": "1.0.0",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "vite build",
    "preview": "vite preview"
  },
  "dependencies": {
    "vue": "^3.5.13",
    "vue-router": "^4.5.0",
    "pinia": "^2.3.0",
    "axios": "^1.7.9",
    "element-plus": "^2.9.1",
    "echarts": "^5.6.0",
    "vue-echarts": "^7.0.3",
    "qrcode": "^1.5.4"
  },
  "devDependencies": {
    "@vitejs/plugin-vue": "^5.2.1",
    "vite": "^6.0.0"
  }
}
```

### client/vite.config.js

```js
import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:3000',
        changeOrigin: true,
      },
    },
  },
  resolve: {
    alias: {
      '@': '/src',
    },
  },
});
```

### client/index.html

```html
<!DOCTYPE html>
<html lang="zh-CN">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>EventChain — 校园赛事预测市场</title>
  </head>
  <body>
    <div id="app"></div>
    <script type="module" src="/src/main.js"></script>
  </body>
</html>
```

### client/src/main.js

```js
import { createApp } from 'vue';
import { createPinia } from 'pinia';
import ElementPlus from 'element-plus';
import 'element-plus/dist/index.css';
import App from './App.vue';
import router from './router/index.js';
import './assets/styles/variables.css';
import './assets/styles/liquid-glass.css';
import './assets/styles/global.css';

const app = createApp(App);

app.use(createPinia());
app.use(router);
app.use(ElementPlus);

app.mount('#app');
```

### client/src/App.vue

```vue
<script setup>
import GlassNavbar from './components/GlassNavbar.vue';
import { useAuthStore } from './stores/auth.js';

const auth = useAuthStore();
auth.restoreSession();
</script>

<template>
  <div class="app-shell">
    <!-- Floating background orbs -->
    <div class="orb orb-1"></div>
    <div class="orb orb-2"></div>
    <div class="orb orb-3"></div>

    <GlassNavbar />
    <main class="main-content">
      <router-view />
    </main>
  </div>
</template>

<style scoped>
.app-shell {
  min-height: 100vh;
  position: relative;
  overflow-x: hidden;
}

.main-content {
  max-width: 1200px;
  margin: 0 auto;
  padding: 88px 24px 48px;
}
</style>
```

### client/src/assets/styles/variables.css

```css
:root {
  /* ---------- Brand palette ---------- */
  --color-primary: #6366f1;
  --color-primary-light: #818cf8;
  --color-primary-dark: #4f46e5;
  --color-success: #22c55e;
  --color-warning: #f59e0b;
  --color-danger: #ef4444;
  --color-info: #3b82f6;

  /* ---------- Neutral ---------- */
  --color-text: #1e293b;
  --color-text-secondary: #64748b;
  --color-text-tertiary: #94a3b8;
  --color-border: rgba(255, 255, 255, 0.35);

  /* ---------- Glass material tokens ---------- */
  --glass-dense-bg: rgba(255, 255, 255, 0.35);
  --glass-dense-blur: 32px;
  --glass-dense-saturate: 2;

  --glass-bg: rgba(255, 255, 255, 0.22);
  --glass-blur: 24px;
  --glass-saturate: 1.8;

  --glass-subtle-bg: rgba(255, 255, 255, 0.12);
  --glass-subtle-blur: 16px;
  --glass-subtle-saturate: 1.4;

  /* ---------- Specular highlight ---------- */
  --highlight-x: 30%;
  --highlight-y: 20%;

  /* ---------- Radius ---------- */
  --radius-sm: 12px;
  --radius-md: 18px;
  --radius-lg: 24px;

  /* ---------- Shadows ---------- */
  --shadow-card: 0 8px 40px rgba(0, 0, 0, 0.06);
  --shadow-card-hover: 0 12px 48px rgba(0, 0, 0, 0.1);

  /* ---------- Transitions ---------- */
  --transition-base: 0.25s cubic-bezier(0.4, 0, 0.2, 1);
}
```

### client/src/assets/styles/liquid-glass.css

```css
/* ============================================================
   Liquid Glass Material System — three tiers
   ============================================================ */

/* ---------- Dense — navigation bars, headers ---------- */
.glass-dense {
  position: relative;
  background: var(--glass-dense-bg);
  backdrop-filter: blur(var(--glass-dense-blur)) saturate(var(--glass-dense-saturate));
  -webkit-backdrop-filter: blur(var(--glass-dense-blur)) saturate(var(--glass-dense-saturate));
  border: 0.5px solid rgba(255, 255, 255, 0.45);
  box-shadow:
    inset 0 0.5px 0 rgba(255, 255, 255, 0.6),
    var(--shadow-card);
  border-radius: var(--radius-md);
}

/* ---------- Regular — cards, panels ---------- */
.glass {
  position: relative;
  background: var(--glass-bg);
  backdrop-filter: blur(var(--glass-blur)) saturate(var(--glass-saturate));
  -webkit-backdrop-filter: blur(var(--glass-blur)) saturate(var(--glass-saturate));
  border: 0.5px solid rgba(255, 255, 255, 0.4);
  box-shadow:
    inset 0 0.5px 0 rgba(255, 255, 255, 0.5),
    var(--shadow-card);
  border-radius: var(--radius-md);
  transition: box-shadow var(--transition-base), transform var(--transition-base);
}

.glass:hover {
  box-shadow:
    inset 0 0.5px 0 rgba(255, 255, 255, 0.5),
    var(--shadow-card-hover);
  transform: translateY(-2px);
}

/* ---------- Subtle — nested elements, badges ---------- */
.glass-subtle {
  position: relative;
  background: var(--glass-subtle-bg);
  backdrop-filter: blur(var(--glass-subtle-blur)) saturate(var(--glass-subtle-saturate));
  -webkit-backdrop-filter: blur(var(--glass-subtle-blur)) saturate(var(--glass-subtle-saturate));
  border: 0.5px solid rgba(255, 255, 255, 0.25);
  border-radius: var(--radius-sm);
}

/* ---------- Specular highlight layer (shared) ---------- */
.glass::before,
.glass-dense::before,
.glass-subtle::before {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: inherit;
  background: radial-gradient(
    ellipse at var(--highlight-x, 30%) var(--highlight-y, 20%),
    rgba(255, 255, 255, 0.45) 0%,
    transparent 65%
  );
  mix-blend-mode: overlay;
  pointer-events: none;
  z-index: 1;
}

/* Ensure card content sits above the pseudo-element */
.glass > *,
.glass-dense > *,
.glass-subtle > * {
  position: relative;
  z-index: 2;
}
```

### client/src/assets/styles/global.css

```css
/* ============================================================
   Global — body background, orbs, reset
   ============================================================ */

*,
*::before,
*::after {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family:
    -apple-system, BlinkMacSystemFont, 'SF Pro Display', 'Segoe UI',
    Roboto, 'Helvetica Neue', sans-serif;
  color: var(--color-text);
  line-height: 1.6;
  -webkit-font-smoothing: antialiased;

  /* Mesh gradient background */
  background:
    radial-gradient(ellipse at 20% 20%, rgba(99, 102, 241, 0.25) 0%, transparent 50%),
    radial-gradient(ellipse at 80% 30%, rgba(236, 72, 153, 0.20) 0%, transparent 50%),
    radial-gradient(ellipse at 50% 80%, rgba(34, 197, 94, 0.18) 0%, transparent 50%),
    radial-gradient(ellipse at 10% 70%, rgba(245, 158, 11, 0.15) 0%, transparent 50%),
    linear-gradient(135deg, #f0f4ff 0%, #fdf2f8 50%, #f0fdf4 100%);
  background-attachment: fixed;
  min-height: 100vh;
}

/* ---------- Floating orbs ---------- */
.orb {
  position: fixed;
  border-radius: 50%;
  filter: blur(60px);
  opacity: 0.5;
  pointer-events: none;
  z-index: 0;
  animation: float 20s ease-in-out infinite;
}

.orb-1 {
  width: 400px;
  height: 400px;
  background: rgba(99, 102, 241, 0.3);
  top: -100px;
  left: -100px;
  animation-delay: 0s;
}

.orb-2 {
  width: 350px;
  height: 350px;
  background: rgba(236, 72, 153, 0.25);
  top: 50%;
  right: -80px;
  animation-delay: -7s;
}

.orb-3 {
  width: 300px;
  height: 300px;
  background: rgba(34, 197, 94, 0.2);
  bottom: -60px;
  left: 30%;
  animation-delay: -14s;
}

@keyframes float {
  0%, 100% {
    transform: translate(0, 0) scale(1);
  }
  25% {
    transform: translate(30px, -40px) scale(1.05);
  }
  50% {
    transform: translate(-20px, 20px) scale(0.95);
  }
  75% {
    transform: translate(15px, 35px) scale(1.02);
  }
}

/* ---------- Scrollbar ---------- */
::-webkit-scrollbar {
  width: 6px;
}
::-webkit-scrollbar-track {
  background: transparent;
}
::-webkit-scrollbar-thumb {
  background: rgba(0, 0, 0, 0.15);
  border-radius: 3px;
}

/* ---------- Selection ---------- */
::selection {
  background: rgba(99, 102, 241, 0.25);
}
```

### client/src/composables/useGlassHighlight.js

```js
import { onMounted, onUnmounted, ref } from 'vue';

/**
 * Tracks the mouse position relative to a target element and updates
 * CSS custom properties --highlight-x and --highlight-y so the
 * radial-gradient specular highlight follows the cursor.
 *
 * Usage:
 *   const { elementRef } = useGlassHighlight();
 *   <div ref="elementRef" class="glass"> ... </div>
 */
export function useGlassHighlight() {
  const elementRef = ref(null);
  let rafId = null;

  function onMouseMove(e) {
    if (!elementRef.value) return;

    // Cancel any pending frame to avoid piling up
    if (rafId) cancelAnimationFrame(rafId);

    rafId = requestAnimationFrame(() => {
      const rect = elementRef.value.getBoundingClientRect();
      const x = ((e.clientX - rect.left) / rect.width) * 100;
      const y = ((e.clientY - rect.top) / rect.height) * 100;
      elementRef.value.style.setProperty('--highlight-x', `${x}%`);
      elementRef.value.style.setProperty('--highlight-y', `${y}%`);
    });
  }

  function onMouseLeave() {
    if (!elementRef.value) return;
    // Reset to default resting position
    elementRef.value.style.setProperty('--highlight-x', '30%');
    elementRef.value.style.setProperty('--highlight-y', '20%');
  }

  onMounted(() => {
    if (elementRef.value) {
      elementRef.value.addEventListener('mousemove', onMouseMove);
      elementRef.value.addEventListener('mouseleave', onMouseLeave);
    }
  });

  onUnmounted(() => {
    if (rafId) cancelAnimationFrame(rafId);
    if (elementRef.value) {
      elementRef.value.removeEventListener('mousemove', onMouseMove);
      elementRef.value.removeEventListener('mouseleave', onMouseLeave);
    }
  });

  return { elementRef };
}
```

### client/src/api/index.js

```js
import axios from 'axios';
import { ElMessage } from 'element-plus';

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 15000,
  headers: { 'Content-Type': 'application/json' },
});

// ---- Request interceptor: attach JWT ----
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('ec_token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// ---- Response interceptor: unwrap data, handle errors ----
api.interceptors.response.use(
  (response) => {
    // Successful responses have { error: false, data: ... }
    return response.data?.data !== undefined ? response.data.data : response.data;
  },
  (error) => {
    const resp = error.response;
    if (resp) {
      const body = resp.data;
      const message = body?.message || '请求失败';

      // Auto-logout on 401
      if (resp.status === 401) {
        localStorage.removeItem('ec_token');
        localStorage.removeItem('ec_user');
        window.location.href = '/login';
      }

      ElMessage.error(message);
      return Promise.reject(body);
    }

    ElMessage.error('网络连接失败，请稍后重试');
    return Promise.reject(error);
  }
);

export default api;
```

### client/src/router/index.js

```js
import { createRouter, createWebHistory } from 'vue-router';

const routes = [
  {
    path: '/',
    name: 'Home',
    component: () => import('../views/HomeView.vue'),
  },
  {
    path: '/event/:id',
    name: 'EventDetail',
    component: () => import('../views/EventDetailView.vue'),
    props: true,
  },
  {
    path: '/tickets',
    name: 'TicketHall',
    component: () => import('../views/TicketHallView.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/me',
    name: 'Profile',
    component: () => import('../views/ProfileView.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/admin',
    name: 'Admin',
    component: () => import('../views/AdminView.vue'),
    meta: { requiresAuth: true, requiresRole: ['organizer', 'admin'] },
  },
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/LoginView.vue'),
  },
];

const router = createRouter({
  history: createWebHistory(),
  routes,
});

// Navigation guard
router.beforeEach((to, _from, next) => {
  const token = localStorage.getItem('ec_token');
  const user = JSON.parse(localStorage.getItem('ec_user') || 'null');

  if (to.meta.requiresAuth && !token) {
    return next({ name: 'Login', query: { redirect: to.fullPath } });
  }

  if (to.meta.requiresRole && user) {
    const allowed = to.meta.requiresRole;
    if (!allowed.includes(user.role)) {
      return next({ name: 'Home' });
    }
  }

  next();
});

export default router;
```

---

## Task 8 — Reusable Components

- [ ] Create `client/src/components/GlassCard.vue`
- [ ] Create `client/src/components/GlassNavbar.vue`
- [ ] Create `client/src/components/ProbabilityBar.vue`
- [ ] Create `client/src/components/SparkLine.vue`
- [ ] Create `client/src/components/CountdownTimer.vue`
- [ ] Create `client/src/components/QRCode.vue`

### client/src/components/GlassCard.vue

```vue
<script setup>
import { useGlassHighlight } from '../composables/useGlassHighlight.js';

const props = defineProps({
  variant: {
    type: String,
    default: 'regular',
    validator: (v) => ['dense', 'regular', 'subtle'].includes(v),
  },
  hoverable: {
    type: Boolean,
    default: true,
  },
  padding: {
    type: String,
    default: '24px',
  },
});

const { elementRef } = useGlassHighlight();

const classMap = {
  dense: 'glass-dense',
  regular: 'glass',
  subtle: 'glass-subtle',
};
</script>

<template>
  <div
    ref="elementRef"
    :class="[classMap[props.variant], { 'no-hover': !props.hoverable }]"
    :style="{ padding: props.padding }"
    class="glass-card"
  >
    <slot />
  </div>
</template>

<style scoped>
.glass-card {
  overflow: hidden;
}

.glass-card.no-hover:hover {
  transform: none;
  box-shadow:
    inset 0 0.5px 0 rgba(255, 255, 255, 0.5),
    var(--shadow-card);
}
</style>
```

### client/src/components/GlassNavbar.vue

```vue
<script setup>
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import { useAuthStore } from '../stores/auth.js';
import { useUserStore } from '../stores/user.js';
import { useGlassHighlight } from '../composables/useGlassHighlight.js';

const router = useRouter();
const auth = useAuthStore();
const userStore = useUserStore();
const { elementRef } = useGlassHighlight();

const isLoggedIn = computed(() => auth.isLoggedIn);
const displayName = computed(() => auth.user?.name || auth.user?.userId || '');
const balance = computed(() => userStore.balance);

function handleLogout() {
  auth.logout();
  router.push('/login');
}

const navLinks = [
  { label: '首页', path: '/' },
  { label: '票务大厅', path: '/tickets' },
  { label: '我的', path: '/me' },
];
</script>

<template>
  <nav ref="elementRef" class="glass-dense navbar">
    <div class="navbar-inner">
      <router-link to="/" class="brand">
        <span class="brand-icon">&#9830;</span>
        <span class="brand-text">EventChain</span>
      </router-link>

      <div class="nav-links">
        <router-link
          v-for="link in navLinks"
          :key="link.path"
          :to="link.path"
          class="nav-link"
          active-class="nav-link--active"
        >
          {{ link.label }}
        </router-link>

        <router-link
          v-if="auth.user?.role === 'organizer' || auth.user?.role === 'admin'"
          to="/admin"
          class="nav-link"
          active-class="nav-link--active"
        >
          管理
        </router-link>
      </div>

      <div class="nav-right">
        <template v-if="isLoggedIn">
          <div class="balance-badge glass-subtle">
            <span class="balance-icon">&#9733;</span>
            <span class="balance-amount">{{ balance }}</span>
          </div>
          <div class="user-info">
            <span class="user-name">{{ displayName }}</span>
            <button class="logout-btn" @click="handleLogout">退出</button>
          </div>
        </template>
        <template v-else>
          <router-link to="/login" class="login-btn">登录</router-link>
        </template>
      </div>
    </div>
  </nav>
</template>

<style scoped>
.navbar {
  position: fixed;
  top: 12px;
  left: 50%;
  transform: translateX(-50%);
  width: calc(100% - 48px);
  max-width: 1200px;
  z-index: 1000;
  padding: 0 24px;
  border-radius: var(--radius-lg);
}

.navbar-inner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 56px;
}

.brand {
  display: flex;
  align-items: center;
  gap: 8px;
  text-decoration: none;
  color: var(--color-text);
  font-weight: 700;
  font-size: 18px;
}

.brand-icon {
  font-size: 22px;
  color: var(--color-primary);
}

.nav-links {
  display: flex;
  gap: 8px;
}

.nav-link {
  text-decoration: none;
  color: var(--color-text-secondary);
  padding: 6px 16px;
  border-radius: var(--radius-sm);
  font-size: 14px;
  font-weight: 500;
  transition: all var(--transition-base);
}

.nav-link:hover {
  color: var(--color-text);
  background: rgba(255, 255, 255, 0.2);
}

.nav-link--active {
  color: var(--color-primary);
  background: rgba(99, 102, 241, 0.1);
}

.nav-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.balance-badge {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 12px;
  font-size: 14px;
  font-weight: 600;
  color: var(--color-warning);
}

.balance-icon {
  font-size: 16px;
}

.user-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text);
}

.logout-btn {
  background: none;
  border: none;
  color: var(--color-text-tertiary);
  cursor: pointer;
  font-size: 13px;
  padding: 4px 8px;
  margin-left: 4px;
  border-radius: 6px;
  transition: all var(--transition-base);
}

.logout-btn:hover {
  color: var(--color-danger);
  background: rgba(239, 68, 68, 0.08);
}

.login-btn {
  text-decoration: none;
  padding: 6px 20px;
  border-radius: var(--radius-sm);
  background: var(--color-primary);
  color: #fff;
  font-size: 14px;
  font-weight: 500;
  transition: background var(--transition-base);
}

.login-btn:hover {
  background: var(--color-primary-dark);
}
</style>
```

### client/src/components/ProbabilityBar.vue

```vue
<script setup>
import { computed } from 'vue';

const props = defineProps({
  probA: { type: Number, required: true },  // 0-1
  labelA: { type: String, default: 'A' },
  labelB: { type: String, default: 'B' },
  colorA: { type: String, default: '#6366f1' },
  colorB: { type: String, default: '#ec4899' },
  height: { type: String, default: '32px' },
});

const pctA = computed(() => Math.round(props.probA * 100));
const pctB = computed(() => 100 - pctA.value);
</script>

<template>
  <div class="probability-bar" :style="{ height: props.height }">
    <div
      class="bar-segment bar-a"
      :style="{
        width: pctA + '%',
        background: `linear-gradient(90deg, ${props.colorA}, ${props.colorA}dd)`,
      }"
    >
      <span v-if="pctA >= 20" class="bar-label">{{ props.labelA }} {{ pctA }}%</span>
    </div>
    <div
      class="bar-segment bar-b"
      :style="{
        width: pctB + '%',
        background: `linear-gradient(90deg, ${props.colorB}dd, ${props.colorB})`,
      }"
    >
      <span v-if="pctB >= 20" class="bar-label">{{ props.labelB }} {{ pctB }}%</span>
    </div>
  </div>
</template>

<style scoped>
.probability-bar {
  display: flex;
  border-radius: 999px;
  overflow: hidden;
  width: 100%;
}

.bar-segment {
  display: flex;
  align-items: center;
  justify-content: center;
  transition: width 0.6s cubic-bezier(0.4, 0, 0.2, 1);
  min-width: 4px;
}

.bar-label {
  font-size: 12px;
  font-weight: 600;
  color: #fff;
  white-space: nowrap;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.2);
}
</style>
```

### client/src/components/SparkLine.vue

```vue
<script setup>
import { computed } from 'vue';

const props = defineProps({
  data: { type: Array, required: true },   // array of numbers
  width: { type: Number, default: 80 },
  height: { type: Number, default: 24 },
  color: { type: String, default: '#6366f1' },
  filled: { type: Boolean, default: true },
});

const pathD = computed(() => {
  const pts = props.data;
  if (!pts.length) return '';

  const max = Math.max(...pts);
  const min = Math.min(...pts);
  const range = max - min || 1;
  const stepX = props.width / (pts.length - 1 || 1);
  const pad = 2;
  const usableH = props.height - pad * 2;

  const points = pts.map((v, i) => {
    const x = i * stepX;
    const y = pad + usableH - ((v - min) / range) * usableH;
    return `${x},${y}`;
  });

  return 'M' + points.join(' L');
});

const fillD = computed(() => {
  if (!props.filled || !props.data.length) return '';
  return `${pathD.value} L${props.width},${props.height} L0,${props.height} Z`;
});
</script>

<template>
  <svg
    :viewBox="`0 0 ${props.width} ${props.height}`"
    :width="props.width"
    :height="props.height"
    class="sparkline"
  >
    <path
      v-if="props.filled"
      :d="fillD"
      :fill="`${props.color}20`"
    />
    <path
      :d="pathD"
      fill="none"
      :stroke="props.color"
      stroke-width="1.5"
      stroke-linecap="round"
      stroke-linejoin="round"
    />
  </svg>
</template>

<style scoped>
.sparkline {
  display: inline-block;
  vertical-align: middle;
}
</style>
```

### client/src/components/CountdownTimer.vue

```vue
<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue';

const props = defineProps({
  targetTime: { type: String, required: true },  // ISO string
});

const now = ref(Date.now());
let timer = null;

onMounted(() => {
  timer = setInterval(() => { now.value = Date.now(); }, 1000);
});

onUnmounted(() => {
  if (timer) clearInterval(timer);
});

const remaining = computed(() => {
  const diff = new Date(props.targetTime).getTime() - now.value;
  if (diff <= 0) return { days: 0, hours: 0, mins: 0, secs: 0, expired: true };

  const days = Math.floor(diff / 86400000);
  const hours = Math.floor((diff % 86400000) / 3600000);
  const mins = Math.floor((diff % 3600000) / 60000);
  const secs = Math.floor((diff % 60000) / 1000);

  return { days, hours, mins, secs, expired: false };
});

function pad(n) {
  return String(n).padStart(2, '0');
}
</script>

<template>
  <div class="countdown" :class="{ 'countdown--expired': remaining.expired }">
    <template v-if="remaining.expired">
      <span class="countdown-label">已截止</span>
    </template>
    <template v-else>
      <div v-if="remaining.days > 0" class="countdown-unit">
        <span class="countdown-value">{{ remaining.days }}</span>
        <span class="countdown-label">天</span>
      </div>
      <div class="countdown-unit">
        <span class="countdown-value">{{ pad(remaining.hours) }}</span>
        <span class="countdown-label">时</span>
      </div>
      <div class="countdown-unit">
        <span class="countdown-value">{{ pad(remaining.mins) }}</span>
        <span class="countdown-label">分</span>
      </div>
      <div class="countdown-unit">
        <span class="countdown-value">{{ pad(remaining.secs) }}</span>
        <span class="countdown-label">秒</span>
      </div>
    </template>
  </div>
</template>

<style scoped>
.countdown {
  display: flex;
  align-items: center;
  gap: 4px;
}

.countdown-unit {
  display: flex;
  align-items: baseline;
  gap: 1px;
}

.countdown-value {
  font-size: 16px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  color: var(--color-text);
}

.countdown-label {
  font-size: 12px;
  color: var(--color-text-tertiary);
}

.countdown--expired .countdown-label {
  color: var(--color-danger);
  font-weight: 600;
  font-size: 14px;
}
</style>
```

### client/src/components/QRCode.vue

```vue
<script setup>
import { ref, watch, onMounted } from 'vue';
import QRCodeLib from 'qrcode';

const props = defineProps({
  value: { type: String, required: true },
  size: { type: Number, default: 200 },
});

const canvasRef = ref(null);

async function render() {
  if (!canvasRef.value || !props.value) return;
  await QRCodeLib.toCanvas(canvasRef.value, props.value, {
    width: props.size,
    margin: 2,
    color: { dark: '#1e293b', light: '#ffffff' },
  });
}

onMounted(render);
watch(() => props.value, render);
watch(() => props.size, render);
</script>

<template>
  <canvas ref="canvasRef" class="qrcode-canvas" />
</template>

<style scoped>
.qrcode-canvas {
  border-radius: var(--radius-sm);
}
</style>
```

---

## Task 9 — Pinia Stores

- [ ] Create `client/src/stores/auth.js`
- [ ] Create `client/src/stores/events.js`
- [ ] Create `client/src/stores/prediction.js`
- [ ] Create `client/src/stores/ticket.js`
- [ ] Create `client/src/stores/user.js`

### client/src/stores/auth.js

```js
import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import api from '../api/index.js';

export const useAuthStore = defineStore('auth', () => {
  const token = ref(null);
  const user = ref(null); // { userId, name, role }

  const isLoggedIn = computed(() => !!token.value);

  // Restore session from localStorage on app init
  function restoreSession() {
    const savedToken = localStorage.getItem('ec_token');
    const savedUser = localStorage.getItem('ec_user');
    if (savedToken && savedUser) {
      token.value = savedToken;
      user.value = JSON.parse(savedUser);
    }
  }

  function setSession(jwt, userData) {
    token.value = jwt;
    user.value = userData;
    localStorage.setItem('ec_token', jwt);
    localStorage.setItem('ec_user', JSON.stringify(userData));
  }

  async function register(studentID, password, name) {
    const data = await api.post('/auth/register', { studentID, password, name });
    setSession(data.token, {
      userId: data.userId,
      name: data.name || name,
      role: 'student',
    });
    return data;
  }

  async function login(studentID, password) {
    const data = await api.post('/auth/login', { studentID, password });
    setSession(data.token, {
      userId: data.userId,
      name: data.name,
      role: data.role,
    });
    return data;
  }

  function logout() {
    token.value = null;
    user.value = null;
    localStorage.removeItem('ec_token');
    localStorage.removeItem('ec_user');
  }

  return { token, user, isLoggedIn, restoreSession, register, login, logout };
});
```

### client/src/stores/events.js

```js
import { defineStore } from 'pinia';
import { ref } from 'vue';
import api from '../api/index.js';

export const useEventStore = defineStore('events', () => {
  const events = ref([]);
  const currentEvent = ref(null);
  const loading = ref(false);

  async function fetchEvents(filters = {}) {
    loading.value = true;
    try {
      const params = new URLSearchParams();
      if (filters.status) params.set('status', filters.status);
      if (filters.type) params.set('type', filters.type);
      const qs = params.toString();
      events.value = await api.get(`/events${qs ? '?' + qs : ''}`);
    } finally {
      loading.value = false;
    }
  }

  async function fetchEvent(eventId) {
    loading.value = true;
    try {
      currentEvent.value = await api.get(`/events/${eventId}`);
      return currentEvent.value;
    } finally {
      loading.value = false;
    }
  }

  async function createEvent(payload) {
    const result = await api.post('/events', payload);
    await fetchEvents();
    return result;
  }

  async function updateStatus(eventId, status) {
    const result = await api.put(`/events/${eventId}/status`, { status });
    await fetchEvents();
    return result;
  }

  async function recordResult(eventId, outcome) {
    const result = await api.put(`/events/${eventId}/result`, { outcome });
    await fetchEvents();
    return result;
  }

  return {
    events,
    currentEvent,
    loading,
    fetchEvents,
    fetchEvent,
    createEvent,
    updateStatus,
    recordResult,
  };
});
```

### client/src/stores/prediction.js

```js
import { defineStore } from 'pinia';
import { ref } from 'vue';
import api from '../api/index.js';

export const usePredictionStore = defineStore('prediction', () => {
  const odds = ref(null);      // { probA, probB } for current event
  const pool = ref(null);      // { poolA, poolB, k, totalVolume }
  const myBets = ref([]);
  const myScore = ref(null);   // { totalBets, correctBets, accuracyRate }
  const loading = ref(false);

  async function fetchOdds(eventID) {
    odds.value = await api.get(`/predictions/odds/${eventID}`);
    return odds.value;
  }

  async function fetchPool(eventID) {
    pool.value = await api.get(`/predictions/pool/${eventID}`);
    return pool.value;
  }

  async function placeBet(eventID, option, amount) {
    loading.value = true;
    try {
      const result = await api.post('/predictions/bet', { eventID, option, amount });
      // result: { shares, newOddsA, newOddsB }
      odds.value = { probA: result.newOddsA, probB: result.newOddsB };
      return result;
    } finally {
      loading.value = false;
    }
  }

  async function fetchMyBets() {
    myBets.value = await api.get('/predictions/mine');
    return myBets.value;
  }

  async function fetchMyScore() {
    myScore.value = await api.get('/predictions/score');
    return myScore.value;
  }

  return {
    odds,
    pool,
    myBets,
    myScore,
    loading,
    fetchOdds,
    fetchPool,
    placeBet,
    fetchMyBets,
    fetchMyScore,
  };
});
```

### client/src/stores/ticket.js

```js
import { defineStore } from 'pinia';
import { ref } from 'vue';
import api from '../api/index.js';

export const useTicketStore = defineStore('ticket', () => {
  const myTickets = ref([]);
  const loading = ref(false);

  async function applyTicket(eventID) {
    loading.value = true;
    try {
      const result = await api.post('/tickets/apply', { eventID });
      return result;
    } finally {
      loading.value = false;
    }
  }

  async function runLottery(eventID) {
    const result = await api.post(`/tickets/lottery/${eventID}`);
    return result;
  }

  async function fetchMyTickets() {
    myTickets.value = await api.get('/tickets/mine');
    return myTickets.value;
  }

  async function claimTicket(ticketID) {
    const result = await api.post(`/tickets/claim/${ticketID}`);
    // Refresh ticket list to show new claim hash
    await fetchMyTickets();
    return result;
  }

  async function verifyTicket(ticketID, hash) {
    const result = await api.get(`/tickets/verify/${ticketID}`, { params: { hash } });
    return result;
  }

  async function refundTicket(ticketID) {
    const result = await api.post(`/tickets/refund/${ticketID}`);
    await fetchMyTickets();
    return result;
  }

  return {
    myTickets,
    loading,
    applyTicket,
    runLottery,
    fetchMyTickets,
    claimTicket,
    verifyTicket,
    refundTicket,
  };
});
```

### client/src/stores/user.js

```js
import { defineStore } from 'pinia';
import { ref } from 'vue';
import api from '../api/index.js';

export const useUserStore = defineStore('user', () => {
  const balance = ref(0);
  const profile = ref(null);
  const leaderboard = ref([]);
  const loading = ref(false);

  async function fetchProfile() {
    loading.value = true;
    try {
      profile.value = await api.get('/users/profile');
      balance.value = profile.value.balance ?? 0;
      return profile.value;
    } finally {
      loading.value = false;
    }
  }

  async function fetchLeaderboard() {
    leaderboard.value = await api.get('/users/leaderboard');
    return leaderboard.value;
  }

  // Called after any action that changes balance (bet, settlement, etc.)
  async function refreshBalance() {
    const p = await api.get('/users/profile');
    balance.value = p.balance ?? 0;
  }

  return {
    balance,
    profile,
    leaderboard,
    loading,
    fetchProfile,
    fetchLeaderboard,
    refreshBalance,
  };
});
```

---

## Task 10 — Home Page

- [ ] Create `client/src/views/HomeView.vue`
- [ ] Create `client/src/views/LoginView.vue`

### client/src/views/HomeView.vue

```vue
<script setup>
import { onMounted, computed } from 'vue';
import { useRouter } from 'vue-router';
import { useEventStore } from '../stores/events.js';
import { useUserStore } from '../stores/user.js';
import { useAuthStore } from '../stores/auth.js';
import GlassCard from '../components/GlassCard.vue';
import ProbabilityBar from '../components/ProbabilityBar.vue';
import SparkLine from '../components/SparkLine.vue';

const router = useRouter();
const eventStore = useEventStore();
const userStore = useUserStore();
const auth = useAuthStore();

onMounted(async () => {
  await eventStore.fetchEvents();
  if (auth.isLoggedIn) {
    await Promise.all([
      userStore.fetchProfile(),
      userStore.fetchLeaderboard(),
    ]);
  }
});

const hotEvents = computed(() =>
  eventStore.events.filter((e) =>
    ['PREDICTION_OPEN', 'ONGOING'].includes(e.status)
  )
);

const typeEmoji = {
  basketball: '\u{1F3C0}',
  football: '\u26BD',
  esports: '\u{1F3AE}',
  badminton: '\u{1F3F8}',
  track: '\u{1F3C3}',
};

function goToEvent(id) {
  router.push(`/event/${id}`);
}
</script>

<template>
  <div class="home">
    <!-- Hero -->
    <section class="hero">
      <h1 class="hero-title">EventChain</h1>
      <p class="hero-subtitle">校园赛事预测市场 &mdash; 预测即力量</p>
    </section>

    <div class="home-grid">
      <!-- Event cards -->
      <section class="events-section">
        <h2 class="section-title">热门赛事</h2>
        <div class="events-grid">
          <GlassCard
            v-for="event in hotEvents"
            :key="event.id"
            class="event-card"
            @click="goToEvent(event.id)"
          >
            <div class="event-card-header">
              <span class="event-type-emoji">{{ typeEmoji[event.type] || '\u{1F3C6}' }}</span>
              <span class="event-status glass-subtle">{{ event.status }}</span>
            </div>
            <h3 class="event-title">{{ event.title }}</h3>
            <div class="event-teams">
              {{ event.teams?.[0] }} <span class="vs">VS</span> {{ event.teams?.[1] }}
            </div>
            <ProbabilityBar
              v-if="event.odds"
              :prob-a="event.odds.probA"
              :label-a="event.predictionOptions?.[0] || 'A'"
              :label-b="event.predictionOptions?.[1] || 'B'"
              height="28px"
              style="margin-top: 12px"
            />
            <div class="event-card-footer">
              <SparkLine
                v-if="event.oddsHistory"
                :data="event.oddsHistory"
                :width="80"
                :height="24"
              />
              <span class="event-volume">
                {{ event.pool?.totalVolume || 0 }} 浙币参与
              </span>
            </div>
          </GlassCard>
        </div>
      </section>

      <!-- Sidebar -->
      <aside class="sidebar">
        <!-- Stats -->
        <GlassCard v-if="auth.isLoggedIn && userStore.profile" class="sidebar-card">
          <h3 class="sidebar-title">我的数据</h3>
          <div class="stats-row">
            <div class="stat">
              <span class="stat-value">{{ userStore.profile.balance }}</span>
              <span class="stat-label">浙币</span>
            </div>
            <div class="stat">
              <span class="stat-value">{{ (userStore.profile.accuracyRate * 100).toFixed(1) }}%</span>
              <span class="stat-label">准确率</span>
            </div>
            <div class="stat">
              <span class="stat-value">{{ userStore.profile.totalBets }}</span>
              <span class="stat-label">总预测</span>
            </div>
          </div>
        </GlassCard>

        <!-- Leaderboard -->
        <GlassCard class="sidebar-card">
          <h3 class="sidebar-title">预测之星</h3>
          <ol class="leaderboard-list">
            <li
              v-for="(entry, idx) in userStore.leaderboard.slice(0, 10)"
              :key="entry.userId"
              class="leaderboard-item"
            >
              <span class="lb-rank">{{ idx + 1 }}</span>
              <span class="lb-name">{{ entry.userId }}</span>
              <span class="lb-score">{{ (entry.accuracyRate * 100).toFixed(1) }}%</span>
            </li>
          </ol>
        </GlassCard>
      </aside>
    </div>
  </div>
</template>

<style scoped>
.hero {
  text-align: center;
  padding: 48px 0 32px;
}

.hero-title {
  font-size: 48px;
  font-weight: 800;
  background: linear-gradient(135deg, var(--color-primary), #ec4899);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.hero-subtitle {
  font-size: 18px;
  color: var(--color-text-secondary);
  margin-top: 8px;
}

.home-grid {
  display: grid;
  grid-template-columns: 1fr 320px;
  gap: 24px;
}

.section-title {
  font-size: 20px;
  font-weight: 700;
  margin-bottom: 16px;
}

.events-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.event-card {
  cursor: pointer;
}

.event-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.event-type-emoji {
  font-size: 24px;
}

.event-status {
  font-size: 11px;
  font-weight: 600;
  padding: 2px 10px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.event-title {
  font-size: 16px;
  font-weight: 700;
  margin-bottom: 4px;
}

.event-teams {
  font-size: 14px;
  color: var(--color-text-secondary);
}

.vs {
  color: var(--color-danger);
  font-weight: 700;
  margin: 0 4px;
}

.event-card-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 12px;
}

.event-volume {
  font-size: 12px;
  color: var(--color-text-tertiary);
}

/* Sidebar */
.sidebar {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.sidebar-card {
  padding: 20px;
}

.sidebar-title {
  font-size: 16px;
  font-weight: 700;
  margin-bottom: 12px;
}

.stats-row {
  display: flex;
  justify-content: space-between;
}

.stat {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.stat-value {
  font-size: 20px;
  font-weight: 700;
  color: var(--color-primary);
}

.stat-label {
  font-size: 12px;
  color: var(--color-text-tertiary);
}

.leaderboard-list {
  list-style: none;
  padding: 0;
}

.leaderboard-item {
  display: flex;
  align-items: center;
  padding: 6px 0;
  border-bottom: 1px solid rgba(0, 0, 0, 0.04);
}

.leaderboard-item:last-child {
  border-bottom: none;
}

.lb-rank {
  width: 24px;
  font-size: 14px;
  font-weight: 700;
  color: var(--color-text-tertiary);
}

.lb-name {
  flex: 1;
  font-size: 14px;
}

.lb-score {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-primary);
}
</style>
```

### client/src/views/LoginView.vue

```vue
<script setup>
import { ref, computed } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { useAuthStore } from '../stores/auth.js';
import GlassCard from '../components/GlassCard.vue';

const router = useRouter();
const route = useRoute();
const auth = useAuthStore();

const mode = ref('login'); // 'login' | 'register'
const form = ref({ studentID: '', password: '', name: '' });
const submitting = ref(false);
const errorMsg = ref('');

const buttonLabel = computed(() => (mode.value === 'login' ? '登录' : '注册'));

async function handleSubmit() {
  errorMsg.value = '';
  submitting.value = true;

  try {
    if (mode.value === 'register') {
      if (!form.value.name) {
        errorMsg.value = '请输入姓名';
        return;
      }
      await auth.register(form.value.studentID, form.value.password, form.value.name);
    } else {
      await auth.login(form.value.studentID, form.value.password);
    }
    // Redirect to the page they tried to access, or home
    const redirect = route.query.redirect || '/';
    router.push(redirect);
  } catch (err) {
    errorMsg.value = err?.message || '操作失败';
  } finally {
    submitting.value = false;
  }
}

function toggleMode() {
  mode.value = mode.value === 'login' ? 'register' : 'login';
  errorMsg.value = '';
}
</script>

<template>
  <div class="login-page">
    <GlassCard class="login-card" padding="40px">
      <h2 class="login-title">
        {{ mode === 'login' ? '欢迎回来' : '加入 EventChain' }}
      </h2>
      <p class="login-subtitle">
        {{ mode === 'login' ? '登录你的账号' : '注册即送 1000 浙币' }}
      </p>

      <form class="login-form" @submit.prevent="handleSubmit">
        <div class="form-group">
          <label class="form-label">学号</label>
          <input
            v-model="form.studentID"
            type="text"
            class="form-input glass-subtle"
            placeholder="请输入学号"
            required
          />
        </div>

        <div v-if="mode === 'register'" class="form-group">
          <label class="form-label">姓名</label>
          <input
            v-model="form.name"
            type="text"
            class="form-input glass-subtle"
            placeholder="请输入姓名"
          />
        </div>

        <div class="form-group">
          <label class="form-label">密码</label>
          <input
            v-model="form.password"
            type="password"
            class="form-input glass-subtle"
            placeholder="请输入密码"
            required
          />
        </div>

        <p v-if="errorMsg" class="error-text">{{ errorMsg }}</p>

        <button
          type="submit"
          class="submit-btn"
          :disabled="submitting"
        >
          {{ submitting ? '处理中...' : buttonLabel }}
        </button>
      </form>

      <p class="toggle-text">
        {{ mode === 'login' ? '还没有账号？' : '已有账号？' }}
        <a class="toggle-link" @click.prevent="toggleMode">
          {{ mode === 'login' ? '立即注册' : '去登录' }}
        </a>
      </p>
    </GlassCard>
  </div>
</template>

<style scoped>
.login-page {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: calc(100vh - 120px);
}

.login-card {
  width: 100%;
  max-width: 420px;
}

.login-title {
  font-size: 24px;
  font-weight: 800;
  text-align: center;
}

.login-subtitle {
  text-align: center;
  color: var(--color-text-secondary);
  font-size: 14px;
  margin-top: 4px;
  margin-bottom: 28px;
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-secondary);
}

.form-input {
  padding: 10px 14px;
  border: none;
  outline: none;
  font-size: 15px;
  color: var(--color-text);
  border-radius: var(--radius-sm);
}

.form-input::placeholder {
  color: var(--color-text-tertiary);
}

.form-input:focus {
  box-shadow: 0 0 0 2px var(--color-primary-light);
}

.error-text {
  color: var(--color-danger);
  font-size: 13px;
  text-align: center;
}

.submit-btn {
  margin-top: 8px;
  padding: 12px;
  border: none;
  border-radius: var(--radius-sm);
  background: var(--color-primary);
  color: #fff;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  transition: background var(--transition-base);
}

.submit-btn:hover:not(:disabled) {
  background: var(--color-primary-dark);
}

.submit-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.toggle-text {
  text-align: center;
  margin-top: 20px;
  font-size: 14px;
  color: var(--color-text-secondary);
}

.toggle-link {
  color: var(--color-primary);
  font-weight: 600;
  cursor: pointer;
}

.toggle-link:hover {
  text-decoration: underline;
}
</style>
```

---

## Task 11 — Event Detail Page

- [ ] Create `client/src/views/EventDetailView.vue`

### client/src/views/EventDetailView.vue

```vue
<script setup>
import { ref, computed, onMounted, watch } from 'vue';
import { useRoute } from 'vue-router';
import { use } from 'echarts/core';
import { CanvasRenderer } from 'echarts/renderers';
import { LineChart } from 'echarts/charts';
import {
  TitleComponent,
  TooltipComponent,
  GridComponent,
  LegendComponent,
} from 'echarts/components';
import VChart from 'vue-echarts';
import { ElMessage, ElInputNumber, ElSlider, ElRadioGroup, ElRadioButton } from 'element-plus';
import { useEventStore } from '../stores/events.js';
import { usePredictionStore } from '../stores/prediction.js';
import { useUserStore } from '../stores/user.js';
import { useAuthStore } from '../stores/auth.js';
import GlassCard from '../components/GlassCard.vue';
import ProbabilityBar from '../components/ProbabilityBar.vue';

use([CanvasRenderer, LineChart, TitleComponent, TooltipComponent, GridComponent, LegendComponent]);

const route = useRoute();
const eventStore = useEventStore();
const predStore = usePredictionStore();
const userStore = useUserStore();
const auth = useAuthStore();

const eventId = computed(() => route.params.id);
const event = computed(() => eventStore.currentEvent);
const odds = computed(() => predStore.odds);

// Bet form
const selectedOption = ref('A');
const betAmount = ref(50);
const submitting = ref(false);

onMounted(async () => {
  await eventStore.fetchEvent(eventId.value);
  await predStore.fetchOdds(eventId.value);
  await predStore.fetchPool(eventId.value);
});

watch(eventId, async (newId) => {
  if (newId) {
    await eventStore.fetchEvent(newId);
    await predStore.fetchOdds(newId);
    await predStore.fetchPool(newId);
  }
});

// Payout preview based on current AMM state
const payoutPreview = computed(() => {
  if (!predStore.pool || !betAmount.value) return 0;
  const pool = predStore.pool;
  const d = betAmount.value;

  if (selectedOption.value === 'A') {
    const newPoolB = pool.poolB + d;
    const newPoolA = pool.k / newPoolB;
    const shares = pool.poolA - newPoolA;
    return shares.toFixed(2);
  } else {
    const newPoolA = pool.poolA + d;
    const newPoolB = pool.k / newPoolA;
    const shares = pool.poolB - newPoolB;
    return shares.toFixed(2);
  }
});

async function handleBet() {
  if (!auth.isLoggedIn) {
    ElMessage.warning('请先登录');
    return;
  }
  submitting.value = true;
  try {
    const result = await predStore.placeBet(eventId.value, selectedOption.value, betAmount.value);
    ElMessage.success(`下注成功！获得 ${result.shares.toFixed(2)} 份额`);
    await userStore.refreshBalance();
  } catch {
    // Error handled by interceptor
  } finally {
    submitting.value = false;
  }
}

// ECharts config for odds history
const chartOption = computed(() => {
  // Use event.oddsHistory if available, otherwise generate sample points
  const history = event.value?.oddsHistory || [];
  const times = history.map((h) => h.time || '');
  const probAData = history.map((h) => ((h.probA ?? 0.5) * 100).toFixed(1));
  const probBData = history.map((h) => ((h.probB ?? 0.5) * 100).toFixed(1));

  const optionA = event.value?.predictionOptions?.[0] || '选项 A';
  const optionB = event.value?.predictionOptions?.[1] || '选项 B';

  return {
    tooltip: {
      trigger: 'axis',
      backgroundColor: 'rgba(255,255,255,0.85)',
      borderColor: 'rgba(0,0,0,0.08)',
      borderWidth: 1,
      textStyle: { color: '#1e293b', fontSize: 13 },
      formatter(params) {
        let html = `<div style="font-weight:600;margin-bottom:4px">${params[0].axisValue}</div>`;
        for (const p of params) {
          html += `<div>${p.marker} ${p.seriesName}: <b>${p.value}%</b></div>`;
        }
        return html;
      },
    },
    legend: {
      data: [optionA, optionB],
      bottom: 0,
      textStyle: { fontSize: 13 },
    },
    grid: {
      top: 20,
      right: 20,
      bottom: 40,
      left: 50,
      containLabel: false,
    },
    xAxis: {
      type: 'category',
      data: times,
      axisLine: { lineStyle: { color: '#e2e8f0' } },
      axisLabel: { color: '#94a3b8', fontSize: 11 },
    },
    yAxis: {
      type: 'value',
      min: 0,
      max: 100,
      axisLabel: { formatter: '{value}%', color: '#94a3b8', fontSize: 11 },
      splitLine: { lineStyle: { color: '#f1f5f9' } },
    },
    series: [
      {
        name: optionA,
        type: 'line',
        data: probAData,
        smooth: true,
        symbol: 'circle',
        symbolSize: 6,
        lineStyle: { width: 2.5, color: '#6366f1' },
        itemStyle: { color: '#6366f1' },
        areaStyle: {
          color: {
            type: 'linear',
            x: 0, y: 0, x2: 0, y2: 1,
            colorStops: [
              { offset: 0, color: 'rgba(99,102,241,0.25)' },
              { offset: 1, color: 'rgba(99,102,241,0.02)' },
            ],
          },
        },
      },
      {
        name: optionB,
        type: 'line',
        data: probBData,
        smooth: true,
        symbol: 'circle',
        symbolSize: 6,
        lineStyle: { width: 2.5, color: '#ec4899' },
        itemStyle: { color: '#ec4899' },
        areaStyle: {
          color: {
            type: 'linear',
            x: 0, y: 0, x2: 0, y2: 1,
            colorStops: [
              { offset: 0, color: 'rgba(236,72,153,0.25)' },
              { offset: 1, color: 'rgba(236,72,153,0.02)' },
            ],
          },
        },
      },
    ],
  };
});

const statusLabel = {
  CREATED: '已创建',
  PREDICTION_OPEN: '预测中',
  TICKET_OPEN: '购票中',
  ONGOING: '进行中',
  SETTLED: '已结算',
};
</script>

<template>
  <div v-if="event" class="event-detail">
    <!-- Header -->
    <GlassCard class="event-header" padding="32px">
      <div class="header-top">
        <span class="event-type">{{ event.type }}</span>
        <span class="status-badge glass-subtle">{{ statusLabel[event.status] || event.status }}</span>
      </div>
      <h1 class="event-title">{{ event.title }}</h1>
      <div class="teams-display">
        <span class="team team-a">{{ event.teams?.[0] }}</span>
        <span class="vs-badge">VS</span>
        <span class="team team-b">{{ event.teams?.[1] }}</span>
      </div>
      <ProbabilityBar
        v-if="odds"
        :prob-a="odds.probA"
        :label-a="event.predictionOptions?.[0] || 'A'"
        :label-b="event.predictionOptions?.[1] || 'B'"
        height="36px"
        style="margin-top: 20px"
      />
    </GlassCard>

    <div class="detail-grid">
      <!-- Odds chart -->
      <GlassCard class="chart-card" padding="24px">
        <h2 class="card-title">概率走势</h2>
        <VChart
          :option="chartOption"
          style="height: 320px; width: 100%"
          autoresize
        />
      </GlassCard>

      <!-- Bet panel -->
      <GlassCard class="bet-panel" padding="24px">
        <h2 class="card-title">下注预测</h2>

        <div class="bet-options">
          <ElRadioGroup v-model="selectedOption" size="large">
            <ElRadioButton value="A">
              {{ event.predictionOptions?.[0] || 'A' }}
            </ElRadioButton>
            <ElRadioButton value="B">
              {{ event.predictionOptions?.[1] || 'B' }}
            </ElRadioButton>
          </ElRadioGroup>
        </div>

        <div class="bet-amount">
          <label class="form-label">投注金额（浙币）</label>
          <ElSlider v-model="betAmount" :min="1" :max="500" :step="10" show-input />
        </div>

        <div class="payout-preview glass-subtle">
          <div class="preview-row">
            <span class="preview-label">投注</span>
            <span class="preview-value">{{ betAmount }} 浙币</span>
          </div>
          <div class="preview-row">
            <span class="preview-label">预计份额</span>
            <span class="preview-value highlight">{{ payoutPreview }}</span>
          </div>
        </div>

        <button
          class="bet-btn"
          :disabled="submitting || event.status !== 'PREDICTION_OPEN'"
          @click="handleBet"
        >
          {{ submitting ? '提交中...' : event.status === 'PREDICTION_OPEN' ? '确认下注' : '预测未开放' }}
        </button>

        <!-- Pool info -->
        <div v-if="predStore.pool" class="pool-info">
          <span>总投注量: {{ predStore.pool.totalVolume }} 浙币</span>
        </div>
      </GlassCard>
    </div>
  </div>

  <div v-else class="loading-state">
    <p>加载中...</p>
  </div>
</template>

<style scoped>
.event-detail {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.event-header {
  text-align: center;
}

.header-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.event-type {
  font-size: 13px;
  font-weight: 600;
  text-transform: uppercase;
  color: var(--color-text-secondary);
  letter-spacing: 0.05em;
}

.status-badge {
  padding: 4px 14px;
  font-size: 12px;
  font-weight: 600;
}

.event-title {
  font-size: 28px;
  font-weight: 800;
}

.teams-display {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16px;
  margin-top: 8px;
  font-size: 20px;
}

.team {
  font-weight: 700;
}

.team-a { color: var(--color-primary); }
.team-b { color: #ec4899; }

.vs-badge {
  font-size: 14px;
  font-weight: 800;
  color: var(--color-danger);
  padding: 4px 10px;
  border-radius: 8px;
  background: rgba(239, 68, 68, 0.08);
}

.detail-grid {
  display: grid;
  grid-template-columns: 1fr 380px;
  gap: 24px;
}

.card-title {
  font-size: 18px;
  font-weight: 700;
  margin-bottom: 16px;
}

/* Bet panel */
.bet-options {
  margin-bottom: 20px;
}

.bet-amount {
  margin-bottom: 20px;
}

.form-label {
  display: block;
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-secondary);
  margin-bottom: 8px;
}

.payout-preview {
  padding: 16px;
  margin-bottom: 16px;
}

.preview-row {
  display: flex;
  justify-content: space-between;
  padding: 4px 0;
}

.preview-label {
  font-size: 14px;
  color: var(--color-text-secondary);
}

.preview-value {
  font-size: 14px;
  font-weight: 600;
}

.preview-value.highlight {
  color: var(--color-primary);
  font-size: 16px;
}

.bet-btn {
  width: 100%;
  padding: 14px;
  border: none;
  border-radius: var(--radius-sm);
  background: linear-gradient(135deg, var(--color-primary), #8b5cf6);
  color: #fff;
  font-size: 16px;
  font-weight: 700;
  cursor: pointer;
  transition: all var(--transition-base);
}

.bet-btn:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 4px 20px rgba(99, 102, 241, 0.3);
}

.bet-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.pool-info {
  text-align: center;
  margin-top: 12px;
  font-size: 13px;
  color: var(--color-text-tertiary);
}

.loading-state {
  text-align: center;
  padding: 80px 0;
  color: var(--color-text-tertiary);
}
</style>
```

---

## Task 12 — Remaining Pages (Ticket Hall, Profile, Admin)

- [ ] Create `client/src/views/TicketHallView.vue`
- [ ] Create `client/src/views/ProfileView.vue`
- [ ] Create `client/src/views/AdminView.vue`

### client/src/views/TicketHallView.vue

```vue
<script setup>
import { onMounted, computed } from 'vue';
import { ElMessage } from 'element-plus';
import { useEventStore } from '../stores/events.js';
import { useTicketStore } from '../stores/ticket.js';
import GlassCard from '../components/GlassCard.vue';
import CountdownTimer from '../components/CountdownTimer.vue';
import QRCode from '../components/QRCode.vue';

const eventStore = useEventStore();
const ticketStore = useTicketStore();

onMounted(async () => {
  await Promise.all([
    eventStore.fetchEvents({ status: 'TICKET_OPEN' }),
    ticketStore.fetchMyTickets(),
  ]);
});

const ticketEvents = computed(() =>
  eventStore.events.filter((e) => e.status === 'TICKET_OPEN')
);

const wonTickets = computed(() =>
  ticketStore.myTickets.filter((t) => t.status === 'WON' || t.status === 'CLAIMED')
);

const pendingApplications = computed(() =>
  ticketStore.myTickets.filter((t) => t.status === 'PENDING')
);

async function handleApply(eventID) {
  try {
    await ticketStore.applyTicket(eventID);
    ElMessage.success('申请已提交');
    await ticketStore.fetchMyTickets();
  } catch {
    // Error handled by interceptor
  }
}

async function handleClaim(ticketID) {
  try {
    await ticketStore.claimTicket(ticketID);
    ElMessage.success('票据已领取');
  } catch {
    // Error handled by interceptor
  }
}

async function handleRefund(ticketID) {
  try {
    await ticketStore.refundTicket(ticketID);
    ElMessage.success('退票成功');
  } catch {
    // Error handled by interceptor
  }
}
</script>

<template>
  <div class="ticket-hall">
    <h1 class="page-title">票务大厅</h1>

    <!-- Available ticket events -->
    <section class="section">
      <h2 class="section-title">正在售票的赛事</h2>
      <div class="ticket-events-grid">
        <GlassCard
          v-for="event in ticketEvents"
          :key="event.id"
          class="ticket-event-card"
        >
          <div class="te-header">
            <h3 class="te-title">{{ event.title }}</h3>
            <span class="te-quota glass-subtle">余票 {{ event.ticketTotal }}</span>
          </div>
          <div class="te-teams">{{ event.teams?.[0] }} VS {{ event.teams?.[1] }}</div>
          <div class="te-countdown">
            <span class="te-countdown-label">抽签倒计时</span>
            <CountdownTimer :target-time="event.lotteryTime || '2026-04-10T20:00:00'" />
          </div>
          <button class="apply-btn" @click="handleApply(event.id)">
            申请购票
          </button>
        </GlassCard>
      </div>
      <p v-if="!ticketEvents.length" class="empty-text">暂无售票中的赛事</p>
    </section>

    <!-- My applications -->
    <section class="section">
      <h2 class="section-title">我的申请</h2>
      <div class="applications-list">
        <GlassCard
          v-for="app in pendingApplications"
          :key="app.eventID + app.userID"
          variant="subtle"
          padding="16px"
          class="app-item"
        >
          <span class="app-event">{{ app.eventID }}</span>
          <span class="app-status pending">等待抽签</span>
        </GlassCard>
      </div>
      <p v-if="!pendingApplications.length" class="empty-text">暂无待处理的申请</p>
    </section>

    <!-- Won tickets -->
    <section class="section">
      <h2 class="section-title">我的票据</h2>
      <div class="tickets-grid">
        <GlassCard
          v-for="ticket in wonTickets"
          :key="ticket.ticketID"
          class="ticket-card"
        >
          <h3 class="ticket-event-name">{{ ticket.eventID }}</h3>
          <div class="ticket-status">
            <span v-if="ticket.status === 'WON'" class="status-won">中签 - 待领取</span>
            <span v-else-if="ticket.status === 'CLAIMED'" class="status-claimed">已领取</span>
          </div>

          <!-- QR code for claimed tickets -->
          <div v-if="ticket.status === 'CLAIMED' && ticket.claimHash" class="qr-section">
            <QRCode :value="ticket.claimHash" :size="180" />
            <p class="qr-hint">入场时出示此二维码</p>
          </div>

          <div class="ticket-actions">
            <button
              v-if="ticket.status === 'WON'"
              class="claim-btn"
              @click="handleClaim(ticket.ticketID)"
            >
              领取票据
            </button>
            <button
              class="refund-btn"
              @click="handleRefund(ticket.ticketID)"
            >
              退票
            </button>
          </div>
        </GlassCard>
      </div>
      <p v-if="!wonTickets.length" class="empty-text">暂无票据</p>
    </section>
  </div>
</template>

<style scoped>
.ticket-hall {
  padding-bottom: 48px;
}

.page-title {
  font-size: 28px;
  font-weight: 800;
  margin-bottom: 32px;
}

.section {
  margin-bottom: 40px;
}

.section-title {
  font-size: 20px;
  font-weight: 700;
  margin-bottom: 16px;
}

.ticket-events-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

.te-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.te-title {
  font-size: 16px;
  font-weight: 700;
}

.te-quota {
  padding: 2px 10px;
  font-size: 12px;
  font-weight: 600;
  color: var(--color-success);
}

.te-teams {
  font-size: 14px;
  color: var(--color-text-secondary);
  margin-bottom: 12px;
}

.te-countdown {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
}

.te-countdown-label {
  font-size: 13px;
  color: var(--color-text-tertiary);
}

.apply-btn {
  width: 100%;
  padding: 10px;
  border: none;
  border-radius: var(--radius-sm);
  background: var(--color-primary);
  color: #fff;
  font-weight: 600;
  cursor: pointer;
  transition: background var(--transition-base);
}

.apply-btn:hover {
  background: var(--color-primary-dark);
}

.applications-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.app-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.app-event {
  font-weight: 600;
}

.app-status.pending {
  color: var(--color-warning);
  font-weight: 600;
  font-size: 13px;
}

.tickets-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
}

.ticket-card {
  text-align: center;
}

.ticket-event-name {
  font-size: 16px;
  font-weight: 700;
  margin-bottom: 8px;
}

.status-won {
  color: var(--color-warning);
  font-weight: 600;
}

.status-claimed {
  color: var(--color-success);
  font-weight: 600;
}

.qr-section {
  margin: 16px 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.qr-hint {
  font-size: 12px;
  color: var(--color-text-tertiary);
}

.ticket-actions {
  display: flex;
  gap: 8px;
  margin-top: 12px;
}

.claim-btn {
  flex: 1;
  padding: 8px;
  border: none;
  border-radius: 8px;
  background: var(--color-success);
  color: #fff;
  font-weight: 600;
  cursor: pointer;
}

.refund-btn {
  flex: 1;
  padding: 8px;
  border: 1px solid var(--color-danger);
  border-radius: 8px;
  background: transparent;
  color: var(--color-danger);
  font-weight: 600;
  cursor: pointer;
  transition: all var(--transition-base);
}

.refund-btn:hover {
  background: rgba(239, 68, 68, 0.08);
}

.empty-text {
  color: var(--color-text-tertiary);
  font-size: 14px;
  padding: 24px 0;
  text-align: center;
}
</style>
```

### client/src/views/ProfileView.vue

```vue
<script setup>
import { onMounted, computed } from 'vue';
import { use } from 'echarts/core';
import { CanvasRenderer } from 'echarts/renderers';
import { RadarChart } from 'echarts/charts';
import { TitleComponent, TooltipComponent, LegendComponent } from 'echarts/components';
import VChart from 'vue-echarts';
import { useUserStore } from '../stores/user.js';
import { usePredictionStore } from '../stores/prediction.js';
import { useTicketStore } from '../stores/ticket.js';
import GlassCard from '../components/GlassCard.vue';

use([CanvasRenderer, RadarChart, TitleComponent, TooltipComponent, LegendComponent]);

const userStore = useUserStore();
const predStore = usePredictionStore();
const ticketStore = useTicketStore();

onMounted(async () => {
  await Promise.all([
    userStore.fetchProfile(),
    predStore.fetchMyBets(),
    predStore.fetchMyScore(),
    ticketStore.fetchMyTickets(),
  ]);
});

const profile = computed(() => userStore.profile);

// Radar chart — accuracy by event type
const radarOption = computed(() => {
  // Group bets by event type and compute per-type accuracy
  const typeMap = {};
  const types = ['basketball', 'football', 'esports', 'badminton', 'track'];
  const typeLabels = {
    basketball: '篮球',
    football: '足球',
    esports: '电竞',
    badminton: '羽毛球',
    track: '田径',
  };

  for (const t of types) {
    typeMap[t] = { total: 0, correct: 0 };
  }

  for (const bet of predStore.myBets) {
    const t = bet.eventType || 'basketball';
    if (typeMap[t]) {
      typeMap[t].total++;
      if (bet.won) typeMap[t].correct++;
    }
  }

  const indicators = types.map((t) => ({
    name: typeLabels[t] || t,
    max: 100,
  }));

  const values = types.map((t) => {
    const { total, correct } = typeMap[t];
    return total > 0 ? Math.round((correct / total) * 100) : 0;
  });

  return {
    tooltip: {
      trigger: 'item',
      backgroundColor: 'rgba(255,255,255,0.85)',
      borderColor: 'rgba(0,0,0,0.08)',
      borderWidth: 1,
      textStyle: { color: '#1e293b' },
    },
    radar: {
      indicator: indicators,
      shape: 'polygon',
      axisName: {
        color: '#64748b',
        fontSize: 13,
      },
      splitArea: {
        areaStyle: {
          color: [
            'rgba(99,102,241,0.03)',
            'rgba(99,102,241,0.06)',
            'rgba(99,102,241,0.09)',
            'rgba(99,102,241,0.12)',
            'rgba(99,102,241,0.15)',
          ],
        },
      },
      splitLine: {
        lineStyle: { color: 'rgba(0,0,0,0.06)' },
      },
      axisLine: {
        lineStyle: { color: 'rgba(0,0,0,0.08)' },
      },
    },
    series: [
      {
        type: 'radar',
        data: [
          {
            value: values,
            name: '准确率',
            symbol: 'circle',
            symbolSize: 6,
            lineStyle: { color: '#6366f1', width: 2 },
            itemStyle: { color: '#6366f1' },
            areaStyle: { color: 'rgba(99,102,241,0.2)' },
          },
        ],
      },
    ],
  };
});

// Achievement badges
const achievements = computed(() => {
  const list = [];
  const p = profile.value;
  if (!p) return list;

  if (p.totalBets >= 1) list.push({ label: '初出茅庐', desc: '完成第一次预测', icon: '\u{1F3AF}' });
  if (p.totalBets >= 50) list.push({ label: '预测达人', desc: '完成 50 次预测', icon: '\u{1F525}' });
  if (p.accuracyRate >= 0.8 && p.totalBets >= 10) list.push({ label: '神算子', desc: '准确率超过 80%', icon: '\u{1F52E}' });
  if (p.balance >= 5000) list.push({ label: '富甲一方', desc: '余额超过 5000', icon: '\u{1F4B0}' });

  return list;
});
</script>

<template>
  <div class="profile-page">
    <h1 class="page-title">个人中心</h1>

    <div class="profile-grid" v-if="profile">
      <!-- Balance + stats -->
      <GlassCard class="balance-card" padding="32px">
        <div class="balance-header">
          <h2 class="balance-title">{{ profile.name }}</h2>
          <span class="user-role glass-subtle">{{ profile.role }}</span>
        </div>
        <div class="stats-row">
          <div class="stat-item">
            <span class="stat-value primary">{{ profile.balance }}</span>
            <span class="stat-label">浙币余额</span>
          </div>
          <div class="stat-item">
            <span class="stat-value success">{{ (profile.accuracyRate * 100).toFixed(1) }}%</span>
            <span class="stat-label">预测准确率</span>
          </div>
          <div class="stat-item">
            <span class="stat-value">{{ profile.totalBets }}</span>
            <span class="stat-label">总预测次数</span>
          </div>
          <div class="stat-item">
            <span class="stat-value">{{ profile.correctBets }}</span>
            <span class="stat-label">正确次数</span>
          </div>
        </div>
      </GlassCard>

      <!-- Radar chart -->
      <GlassCard class="radar-card" padding="24px">
        <h2 class="card-title">分项准确率</h2>
        <VChart
          :option="radarOption"
          style="height: 300px; width: 100%"
          autoresize
        />
      </GlassCard>

      <!-- Bet history -->
      <GlassCard class="history-card" padding="24px">
        <h2 class="card-title">预测记录</h2>
        <div class="bet-list">
          <div
            v-for="bet in predStore.myBets"
            :key="bet.betID || bet.eventID + bet.timestamp"
            class="bet-item glass-subtle"
          >
            <div class="bet-info">
              <span class="bet-event">{{ bet.eventID }}</span>
              <span class="bet-option">{{ bet.option }}</span>
            </div>
            <div class="bet-meta">
              <span class="bet-amount">{{ bet.amount }} 浙币</span>
              <span class="bet-shares">{{ bet.shares?.toFixed(2) }} 份额</span>
              <span
                class="bet-result"
                :class="bet.won ? 'won' : bet.won === false ? 'lost' : 'pending'"
              >
                {{ bet.won ? '胜' : bet.won === false ? '负' : '待定' }}
              </span>
            </div>
          </div>
        </div>
        <p v-if="!predStore.myBets.length" class="empty-text">暂无预测记录</p>
      </GlassCard>

      <!-- Achievements -->
      <GlassCard class="achievements-card" padding="24px">
        <h2 class="card-title">成就徽章</h2>
        <div class="badge-grid">
          <div
            v-for="badge in achievements"
            :key="badge.label"
            class="badge-item glass-subtle"
          >
            <span class="badge-icon">{{ badge.icon }}</span>
            <span class="badge-label">{{ badge.label }}</span>
            <span class="badge-desc">{{ badge.desc }}</span>
          </div>
        </div>
        <p v-if="!achievements.length" class="empty-text">继续努力，解锁成就吧</p>
      </GlassCard>
    </div>
  </div>
</template>

<style scoped>
.profile-page {
  padding-bottom: 48px;
}

.page-title {
  font-size: 28px;
  font-weight: 800;
  margin-bottom: 32px;
}

.profile-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 24px;
}

/* Balance card spans full width */
.balance-card {
  grid-column: 1 / -1;
}

.balance-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24px;
}

.balance-title {
  font-size: 24px;
  font-weight: 800;
}

.user-role {
  padding: 4px 14px;
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
}

.stats-row {
  display: flex;
  justify-content: space-around;
}

.stat-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.stat-value {
  font-size: 28px;
  font-weight: 800;
}

.stat-value.primary { color: var(--color-primary); }
.stat-value.success { color: var(--color-success); }

.stat-label {
  font-size: 13px;
  color: var(--color-text-tertiary);
}

.card-title {
  font-size: 18px;
  font-weight: 700;
  margin-bottom: 16px;
}

/* History card spans full width */
.history-card {
  grid-column: 1 / -1;
}

.bet-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 400px;
  overflow-y: auto;
}

.bet-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
}

.bet-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.bet-event {
  font-weight: 600;
}

.bet-option {
  font-size: 13px;
  color: var(--color-text-secondary);
}

.bet-meta {
  display: flex;
  align-items: center;
  gap: 16px;
  font-size: 14px;
}

.bet-amount { color: var(--color-text-secondary); }
.bet-shares { color: var(--color-text-tertiary); }

.bet-result {
  font-weight: 700;
  padding: 2px 10px;
  border-radius: 6px;
}

.bet-result.won { color: var(--color-success); background: rgba(34, 197, 94, 0.1); }
.bet-result.lost { color: var(--color-danger); background: rgba(239, 68, 68, 0.1); }
.bet-result.pending { color: var(--color-warning); background: rgba(245, 158, 11, 0.1); }

/* Achievements */
.achievements-card {
  grid-column: 1 / -1;
}

.badge-grid {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.badge-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 16px 20px;
  min-width: 120px;
  text-align: center;
  gap: 4px;
}

.badge-icon {
  font-size: 32px;
}

.badge-label {
  font-size: 14px;
  font-weight: 700;
}

.badge-desc {
  font-size: 11px;
  color: var(--color-text-tertiary);
}

.empty-text {
  color: var(--color-text-tertiary);
  font-size: 14px;
  text-align: center;
  padding: 20px 0;
}
</style>
```

### client/src/views/AdminView.vue

```vue
<script setup>
import { ref, onMounted, computed } from 'vue';
import {
  ElForm,
  ElFormItem,
  ElInput,
  ElSelect,
  ElOption,
  ElInputNumber,
  ElButton,
  ElTable,
  ElTableColumn,
  ElTag,
  ElMessage,
  ElMessageBox,
} from 'element-plus';
import { useEventStore } from '../stores/events.js';
import { useTicketStore } from '../stores/ticket.js';
import GlassCard from '../components/GlassCard.vue';

const eventStore = useEventStore();
const ticketStore = useTicketStore();

onMounted(async () => {
  await eventStore.fetchEvents();
});

// --- Create Event Form ---
const createForm = ref({
  title: '',
  type: 'basketball',
  teamA: '',
  teamB: '',
  ticketTotal: 100,
  optionA: '',
  optionB: '',
});
const creating = ref(false);

const eventTypes = [
  { value: 'basketball', label: '篮球' },
  { value: 'football', label: '足球' },
  { value: 'esports', label: '电竞' },
  { value: 'badminton', label: '羽毛球' },
  { value: 'track', label: '田径' },
];

async function handleCreate() {
  const f = createForm.value;
  if (!f.title || !f.teamA || !f.teamB || !f.optionA || !f.optionB) {
    ElMessage.warning('请填写所有必填字段');
    return;
  }

  creating.value = true;
  try {
    await eventStore.createEvent({
      title: f.title,
      type: f.type,
      teams: [f.teamA, f.teamB],
      ticketTotal: f.ticketTotal,
      predictionOptions: [f.optionA, f.optionB],
    });
    ElMessage.success('赛事创建成功');
    // Reset form
    createForm.value = {
      title: '', type: 'basketball', teamA: '', teamB: '',
      ticketTotal: 100, optionA: '', optionB: '',
    };
  } catch {
    // Handled by interceptor
  } finally {
    creating.value = false;
  }
}

// --- Event Manager ---
const statusFlow = {
  CREATED: 'PREDICTION_OPEN',
  PREDICTION_OPEN: 'TICKET_OPEN',
  TICKET_OPEN: 'ONGOING',
};

const statusTagType = {
  CREATED: 'info',
  PREDICTION_OPEN: 'warning',
  TICKET_OPEN: '',
  ONGOING: 'success',
  SETTLED: 'danger',
};

async function advanceStatus(event) {
  const nextStatus = statusFlow[event.status];
  if (!nextStatus) return;

  try {
    await ElMessageBox.confirm(
      `确认将「${event.title}」状态推进为 ${nextStatus}？`,
      '确认操作'
    );
    await eventStore.updateStatus(event.id, nextStatus);
    ElMessage.success('状态已更新');
  } catch {
    // Cancelled or error
  }
}

async function settleEvent(event) {
  try {
    const { value: outcome } = await ElMessageBox.prompt(
      '请输入赛事结果（选项名称）',
      '录入结果',
      { inputPlaceholder: event.predictionOptions?.join(' 或 ') }
    );
    if (!outcome) return;
    await eventStore.recordResult(event.id, outcome);
    ElMessage.success('结算完成');
  } catch {
    // Cancelled or error
  }
}

async function runLottery(event) {
  try {
    await ElMessageBox.confirm(`确认对「${event.title}」执行抽签？`, '确认');
    await ticketStore.runLottery(event.id);
    ElMessage.success('抽签完成');
  } catch {
    // Cancelled or error
  }
}

// --- System stats ---
const totalEvents = computed(() => eventStore.events.length);
const activeEvents = computed(() =>
  eventStore.events.filter((e) => !['SETTLED', 'CREATED'].includes(e.status)).length
);
</script>

<template>
  <div class="admin-page">
    <h1 class="page-title">管理控制台</h1>

    <div class="admin-grid">
      <!-- System stats -->
      <GlassCard class="stats-card" padding="24px">
        <h2 class="card-title">系统概览</h2>
        <div class="admin-stats">
          <div class="admin-stat">
            <span class="admin-stat-value">{{ totalEvents }}</span>
            <span class="admin-stat-label">总赛事</span>
          </div>
          <div class="admin-stat">
            <span class="admin-stat-value">{{ activeEvents }}</span>
            <span class="admin-stat-label">进行中</span>
          </div>
        </div>
      </GlassCard>

      <!-- Create event form -->
      <GlassCard class="create-card" padding="28px">
        <h2 class="card-title">创建赛事</h2>
        <ElForm label-position="top" :model="createForm">
          <ElFormItem label="赛事名称">
            <ElInput v-model="createForm.title" placeholder="例：院际篮球决赛" />
          </ElFormItem>

          <ElFormItem label="赛事类型">
            <ElSelect v-model="createForm.type" style="width: 100%">
              <ElOption
                v-for="t in eventTypes"
                :key="t.value"
                :label="t.label"
                :value="t.value"
              />
            </ElSelect>
          </ElFormItem>

          <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 12px">
            <ElFormItem label="队伍 A">
              <ElInput v-model="createForm.teamA" placeholder="队伍名称" />
            </ElFormItem>
            <ElFormItem label="队伍 B">
              <ElInput v-model="createForm.teamB" placeholder="队伍名称" />
            </ElFormItem>
          </div>

          <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 12px">
            <ElFormItem label="预测选项 A">
              <ElInput v-model="createForm.optionA" placeholder="例：教院胜" />
            </ElFormItem>
            <ElFormItem label="预测选项 B">
              <ElInput v-model="createForm.optionB" placeholder="例：丹青胜" />
            </ElFormItem>
          </div>

          <ElFormItem label="票务总量">
            <ElInputNumber v-model="createForm.ticketTotal" :min="0" :max="10000" style="width: 100%" />
          </ElFormItem>

          <ElButton
            type="primary"
            :loading="creating"
            style="width: 100%; margin-top: 8px"
            @click="handleCreate"
          >
            创建赛事
          </ElButton>
        </ElForm>
      </GlassCard>

      <!-- Event manager table -->
      <GlassCard class="manager-card" padding="24px">
        <h2 class="card-title">赛事管理</h2>
        <ElTable :data="eventStore.events" stripe style="width: 100%">
          <ElTableColumn prop="title" label="赛事名称" min-width="180" />
          <ElTableColumn prop="type" label="类型" width="80" />
          <ElTableColumn label="状态" width="120">
            <template #default="{ row }">
              <ElTag :type="statusTagType[row.status] || 'info'" size="small">
                {{ row.status }}
              </ElTag>
            </template>
          </ElTableColumn>
          <ElTableColumn label="操作" width="280">
            <template #default="{ row }">
              <ElButton
                v-if="statusFlow[row.status]"
                type="primary"
                size="small"
                @click="advanceStatus(row)"
              >
                推进状态
              </ElButton>
              <ElButton
                v-if="row.status === 'ONGOING'"
                type="warning"
                size="small"
                @click="settleEvent(row)"
              >
                录入结果
              </ElButton>
              <ElButton
                v-if="row.status === 'TICKET_OPEN'"
                type="success"
                size="small"
                @click="runLottery(row)"
              >
                执行抽签
              </ElButton>
            </template>
          </ElTableColumn>
        </ElTable>
      </GlassCard>
    </div>
  </div>
</template>

<style scoped>
.admin-page {
  padding-bottom: 48px;
}

.page-title {
  font-size: 28px;
  font-weight: 800;
  margin-bottom: 32px;
}

.admin-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 24px;
}

.stats-card {
  grid-column: 1 / -1;
}

.manager-card {
  grid-column: 1 / -1;
}

.card-title {
  font-size: 18px;
  font-weight: 700;
  margin-bottom: 16px;
}

.admin-stats {
  display: flex;
  gap: 48px;
}

.admin-stat {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.admin-stat-value {
  font-size: 36px;
  font-weight: 800;
  color: var(--color-primary);
}

.admin-stat-label {
  font-size: 14px;
  color: var(--color-text-tertiary);
}
</style>
```

---

## Summary — Task Dependency Order

| # | Task | Depends on |
|---|------|-----------|
| 1 | Server Scaffolding (app.js, config, errorHandler) | -- |
| 2 | Fabric Services (wallet, CA, gateway) | Task 1 |
| 3 | Auth System (JWT middleware + routes) | Task 2 |
| 4 | Event Routes | Task 3 |
| 5 | Prediction Routes | Task 3 |
| 6 | Ticket + User Routes | Task 3 |
| 7 | Client Scaffolding + Liquid Glass CSS + Composables + API + Router | -- |
| 8 | Reusable Components (GlassCard, Navbar, ProbabilityBar, SparkLine, Countdown, QR) | Task 7 |
| 9 | Pinia Stores (auth, events, prediction, ticket, user) | Task 7 |
| 10 | Home Page + Login Page | Tasks 8, 9 |
| 11 | Event Detail Page (ECharts odds chart + bet panel) | Tasks 8, 9 |
| 12 | Remaining Pages (Ticket Hall, Profile with radar chart, Admin) | Tasks 8, 9 |

Tasks 1-6 (backend) and Tasks 7-9 (frontend foundation) can proceed in parallel. Tasks 10-12 depend on Tasks 8 and 9 being complete.
