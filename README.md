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
