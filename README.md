# EventChain — 校园赛事预测市场

ZJU 区块链课程大作业。校园版 Polymarket + 公平抢票系统，基于 Hyperledger Fabric 2.5。

## 功能

- **预测市场**：AMM 做市，虚拟代币「浙币」预测赛事结果
- **公平购票**：预测准确度加权抽签，奖励真正关注赛事的人
- **链上透明**：所有交易记录链上可查，Fabric CA 学号实名

## 技术栈

| 层 | 技术 |
|----|------|
| 前端 | Vue 3 + Element Plus + ECharts (Liquid Glass UI 风格) |
| 后端 | Node.js + Express + Fabric Gateway SDK |
| 区块链 | Hyperledger Fabric 2.5 (3 Orgs, 4 Chaincodes) |
| 链码 | Go (token / event / prediction / ticket) |
| 数据库 | CouchDB (Fabric state DB, 带 rich query 索引) + SQLite (user auth) |

### 三个组织 + 四个链码

```
PlatformMSP   (admin)        OrganizerMSP   (赛事主办)        StudentMSP   (学生)
    ↓                              ↓                              ↓
peer0.platform.eventchain.com  peer0.organizer.eventchain.com  peer0.student.eventchain.com
    ↑                              ↑                              ↑
              orderer.eventchain.com (Raft 共识)
                          ↑
              channel: eventchain
                          ↑
        ┌─────────┬──────────────┬────────┐
        │         │              │        │
      token     event     prediction    ticket
   (浙币代币)  (赛事元数据)  (AMM 池 + 下注)  (申请 + 抽奖)
```

## 前置要求

- Docker & Docker Compose（建议 OrbStack）
- Go 1.21+
- Node.js 18+
- Fabric 2.5 binaries (`peer`, `orderer`, `configtxgen`, `fabric-ca-client`)

## 快速启动

```bash
# 1. 一次性安装 Fabric binaries + Docker images（项目根目录运行）
curl -sSLO https://raw.githubusercontent.com/hyperledger/fabric/main/scripts/install-fabric.sh
chmod +x install-fabric.sh
./install-fabric.sh --fabric-version 2.5.10 --ca-version 1.5.12 docker binary

# 2. 一键启动（fabric 网络 + 4 个链码 + 后端 + 前端）
./scripts/start-all.sh

# 3. 灌入演示数据（10 个用户、4 个赛事、11 笔下注、1 场已结算）
./scripts/seed-data.sh

# 4. 打开浏览器
open http://localhost:5173
```

> **二进制文件位置说明**：`install-fabric.sh` 把 binaries 装到了项目根的 `bin/`，但 `network.sh` 期待 `fabric/network/bin/`。本仓库已在 `fabric/network/bin → ../../bin` 建好软链解决。

## 演示账号

| 角色 | 学号/账号 | 密码 | 余额 |
|------|---------|------|------|
| 学生 | `3220100001`（张同学） | `student123` | 1000 浙币 |
| 学生 | `3220100002`–`3220100008` | `student123` | 1000 浙币 |
| 主办方 | `organizer01` | `org123` | 0 |
| 管理员 | `admin01` | `admin123` | 0 |

## 主要服务

| 服务 | 端口 | 说明 |
|------|------|------|
| 前端 (vite) | 5173 | Vue 3 SPA |
| 后端 (express) | 3000 | REST API + Fabric Gateway |
| Orderer admin | 7053 | osnadmin channel join |
| Orderer raft | 7050 | gRPC consensus |
| Peer0 platform | 7051 | gRPC |
| Peer0 organizer | 9051 | gRPC |
| Peer0 student | 11051 | gRPC |
| CA platform | 7054 | HTTPS |
| CA organizer | 8054 | HTTPS |
| CA student | 9054 | HTTPS |
| CA orderer | 10054 | HTTPS |
| CouchDB 0/1/2 | 5984/5985/5986 | Peer state DB |

## 项目结构

```
EventChain/
├── bin/                  # Fabric 二进制（install-fabric.sh 装的）
├── config/               # core.yaml (peer 配置)
├── fabric/
│   ├── chaincode/        # 4 个 Go 链码
│   │   ├── event/
│   │   ├── prediction/   # META-INF/statedb/couchdb/indexes/ 含 leaderboard 索引
│   │   ├── ticket/
│   │   └── token/
│   └── network/          # Fabric 网络
│       ├── bin → ../../bin (软链)
│       ├── network.sh    # up / createChannel / deployCC / down 主控
│       ├── docker/       # docker-compose-net.yaml + docker-compose-ca.yaml + .env
│       ├── configtx/     # configtx.yaml (channel + org 配置, 含 anchor peer)
│       ├── organizations/
│       │   ├── fabric-ca/      # CA server 配置（启动后会有 db + msp 子目录）
│       │   └── peerOrganizations/  # 启动时由 registerEnroll.sh 生成
│       ├── channel-artifacts/  # 启动时生成的 channel block
│       └── scripts/      # createChannel.sh, deployCC.sh, envVar.sh, registerEnroll.sh
├── server/               # Node.js 后端
│   ├── src/
│   │   ├── routes/       # auth, events, predictions, tickets, users
│   │   ├── services/     # fabricGateway, caService, wallet
│   │   ├── middleware/   # auth (JWT)
│   │   └── config/       # index, fabric (连接 profile)
│   ├── data/users.db     # SQLite (启动时生成)
│   └── wallet/           # CA 颁发的用户证书 (启动时生成)
├── client/               # Vue 3 前端
│   ├── src/
│   │   ├── views/        # HomeView, EventDetailView, ProfileView, ...
│   │   ├── stores/       # Pinia: auth, user, events, prediction
│   │   ├── components/   # GlassCard, GlassNavbar, ProbabilityBar, SparkLine
│   │   └── api/          # axios 封装 + JWT 拦截器
│   └── vite.config.js
└── scripts/              # start-all.sh, seed-data.sh, cleanup.sh
```

## 常用命令

```bash
# 完整启动（fabric + chaincodes + server + client）
./scripts/start-all.sh

# 灌入演示数据
./scripts/seed-data.sh

# 清理一切（容器 + 卷 + 本地状态）
./scripts/cleanup.sh

# 只重启 fabric 网络
cd fabric/network
./network.sh down
./network.sh up -ca -s couchdb
./network.sh createChannel
./network.sh deployCC -ccn token      -ccp ../chaincode/token      -ccl go
./network.sh deployCC -ccn event      -ccp ../chaincode/event      -ccl go
./network.sh deployCC -ccn prediction -ccp ../chaincode/prediction -ccl go
./network.sh deployCC -ccn ticket     -ccp ../chaincode/ticket     -ccl go

# 只重启后端（保留 fabric 状态）
cd server && npm start

# 只重启前端
cd client && npm run dev
```

## 状态机

### Event 生命周期
```
CREATED → PREDICTION_OPEN → TICKET_OPEN → ONGOING → SETTLED
                  ↑                                       ↓
            自动初始化 prediction pool          自动 settle 派彩
```
状态机是 forward-only，没有回退。`PUT /events/:id/status` 切到 `PREDICTION_OPEN` 时后端会自动调 `prediction.InitializePool` 创建 pool。

### Prediction Pool（AMM）
- 初始 PoolA = PoolB = 10000 浙币（虚拟）
- k = PoolA × PoolB = 1e8（不变）
- 下注 X 浙币 on A → PoolB += X, newPoolA = k / newPoolB, shares = oldPoolA - newPoolA
- 概率 probA = PoolB / (PoolA + PoolB)
- 结算时：赢家按 shares 占比瓜分 totalPool

## 已知限制（Demo Only）

> ⚠️ 这是一个**校园课程项目 Demo**，不适合生产部署。下列限制是设计折中。

1. **`POST /api/v1/auth/register` 是公开 endpoint 且接受 `role` 字段** — 任何人发请求加 `"role":"admin"` 都能拿到 admin 权限的 JWT，并且 Fabric CA 会真的为他签发 PlatformMSP 身份。生产部署前必须改成只允许 `role=student` 自由注册，特权角色走 admin 后台。
2. **`PUT /events/:id/status` → `PREDICTION_OPEN` 有 consistency window** — 后端先调 `event.UpdateStatus` 后调 `prediction.InitializePool`，两步之间没有原子事务（Fabric 不支持多链码原子）。如果第二步失败，event 卡在 `PREDICTION_OPEN` 但 pool 不存在，state machine forward-only 没法回退。
3. **CA `enrollmentSecret` 是 deterministic 的 `${userId}-pw`** — 是为了让 wallet 重建后能 re-enroll。生产环境应改为随机 secret 存在 KMS。
4. **`server/wallet/` 是文件系统钱包** — 适合单机 demo，多节点部署需要换成 HSM 或共享存储。
5. **没有 Rate Limiting / CSRF / 真实 HTTPS** — Express 全裸跑，前端直连后端 3000。
6. **JWT secret hardcode 在 `.env`** — 实际是 `eventchain-dev-secret-change-in-production`。

完整 TODO 见 `TODO.md`。

## 调试技巧

### 容器全挂了，或者 CA 启动失败
```bash
./scripts/cleanup.sh   # 全清
# 注意：cleanup 不会清 CA identity registry。如果遇到 "Identity already registered"，
# 多删一步：
for org in platform organizer student ordererOrg; do
  rm -f fabric/network/organizations/fabric-ca/${org}/fabric-ca-server.db
done
```

### 看链码错误（最有用的命令）
```bash
docker logs peer0.platform.eventchain.com --since 5m 2>&1 | grep -iE "ERRO|chaincode response 500|WARN.*gateway"
```

### 看后端日志
```bash
tail -f /tmp/eventchain-server.log    # 如果用 start-all.sh 启的
```

### 查 CouchDB 索引
```bash
curl -s http://admin:adminpw@localhost:5984/eventchain_prediction/_index | jq
```

### 查链码部署版本
```bash
docker exec peer0.platform.eventchain.com peer lifecycle chaincode querycommitted -C eventchain
```

### 排查 "unable to verify the first certificate"
通常是 stale 后端 server 还在跑（grpc client cache 持有旧 TLS cert）。
```bash
pkill -9 -f "node src/app.js"
lsof -iTCP:3000 -sTCP:LISTEN  # 必须返回空
```

### 干净重启 + 灌数据（最稳的 incantation）
```bash
./scripts/cleanup.sh
for org in platform organizer student ordererOrg; do
  rm -f fabric/network/organizations/fabric-ca/${org}/fabric-ca-server.db
done
pkill -9 -f "node src/app.js"
pkill -9 -f vite
# 先单独把 CAs 启起来（绕开 startCAs sleep 时序问题）
cd fabric/network && \
  docker compose --env-file docker/.env -f docker/docker-compose-ca.yaml up -d && \
  sleep 6 && \
  cd ../..
# 然后跑全套
./scripts/start-all.sh > /tmp/eventchain-start.log 2>&1 &
# 等 ~5 分钟（4 个链码部署 + 后端前端起来）
sleep 200
./scripts/seed-data.sh
```

## API 速查

### 认证
- `POST /api/v1/auth/register` `{studentID, password, name, role?}`
- `POST /api/v1/auth/login` `{studentID, password}`
- 响应：`{error, data: {token, userId, name, role, ...}}`
- 后续请求：`Authorization: Bearer <token>`

### 事件
- `GET /api/v1/events`（公开） — 列出所有
- `GET /api/v1/events/:id`（公开） — 单个事件 + odds
- `POST /api/v1/events`（organizer）`{eventID, title, type, teams[], ticketTotal, predictionOptions[]}`
- `PUT /api/v1/events/:id/status`（organizer）`{status}`
- `PUT /api/v1/events/:id/result`（organizer）`{outcome}` — 触发自动结算

### 预测
- `POST /api/v1/predictions/bet`（已登录）`{eventID, option, amount}`
- `GET /api/v1/predictions/odds/:eventID`（公开）
- `GET /api/v1/predictions/pool/:eventID`（公开）
- `GET /api/v1/predictions/mine`（已登录）
- `GET /api/v1/predictions/score`（已登录）

### 票务
- `POST /api/v1/tickets/apply`（已登录）`{eventID}`
- `POST /api/v1/tickets/lottery/:eventID`（organizer）`{ticketCount}`
- `POST /api/v1/tickets/claim/:eventID`（已登录）
- `GET /api/v1/tickets/mine`（已登录）
- `GET /api/v1/tickets/verify/:ticketID?hash=xxx`（公开）
- `POST /api/v1/tickets/refund/:ticketID`（已登录）

### 用户
- `GET /api/v1/users/profile`（已登录） — 余额、placedBets、totalBets、accuracyRate
- `GET /api/v1/users/leaderboard`（公开） — 按 accuracyRate 排序前 20

---

**最后更新**：2026-04-10（debug + bug-fix session）
**维护**：吴高哲 3230101837
