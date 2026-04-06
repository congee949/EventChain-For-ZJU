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
