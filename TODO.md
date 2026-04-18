# EventChain TODO

按优先级排列。`[x]` = 完成；`[ ]` = 待办；`[~]` = 部分完成 / 已记录但未实现。

最后更新：2026-04-10

---

## P0 — 生产部署前必修（demo 安全洞）

- [ ] **`auth.js` register 端点不应允许公开 role 升级**
  - 当前：任何人 POST `/api/v1/auth/register` 加 `"role":"admin"` 就能拿到 admin JWT，且 Fabric CA 会真发 PlatformMSP 身份证书（damage 到链上）
  - 已加 `SECURITY:` 注释提醒（`server/src/routes/auth.js:13-21`）
  - 修复方案：
    - (a) 删除 role 参数，register 永远 = student；admin/organizer 通过 server 启动时 `.env` 读 `ADMIN_BOOTSTRAP_ID/PASSWORD` 一次性创建，后续 organizer 通过 `POST /admin/users` 走 admin auth
    - (b) 或：保留 role 参数但要求 caller 持有 admin token 才能创建特权角色

- [ ] **`PUT /events/:id/status` → `PREDICTION_OPEN` 的 consistency window**
  - 当前：`event.UpdateStatus` 和 `prediction.InitializePool` 两个 chaincode submit 之间没有原子性。如果第二步失败，event 卡在 `PREDICTION_OPEN` 但 pool 不存在，state machine forward-only 没法回退
  - 已加 `CONSISTENCY WINDOW:` 注释（`server/src/routes/events.js:118-130`）
  - 修复方案：
    - (a)（推荐）：把 InitializePool 移进 `event` 链码的 UpdateStatus 函数里，作为 cross-chaincode invoke 在同一个 fabric tx 内执行；这样要么都成功要么都失败
    - (b)：暴露 `POST /events/:id/init-pool` 作为人工恢复 endpoint（idempotent）；后端 try/catch InitializePool 失败时返回明确的错误指引人工调用

- [ ] **JWT secret 写死在 `.env`**
  - 当前：`JWT_SECRET=eventchain-dev-secret-change-in-production`
  - 修复：从 KMS / Vault 读，部署时按环境注入

- [ ] **Fabric CA `enrollmentSecret` 是 deterministic 的 `${userId}-pw`**
  - 当前：`server/src/services/caService.js:48` 用 `${userId}-pw` 让 wallet 重建后能 re-enroll
  - 已加注释解释 demo-only
  - 修复：生成随机 secret 存到 KMS，wallet 丢失后从 KMS 读回 secret 重新 enroll

---

## P1 — 重要可用性 / 健壮性

- [ ] **`scripts/cleanup.sh` 应该顺手清 fabric-ca-server.db**
  - 当前：cleanup 不清 CA identity 注册库，导致重 seed 时 "Identity already registered" 报错
  - 我已经在 `caService.js` 加了 idempotent register 兜底，但 cleanup 顺手清更干净
  - 修复：在 cleanup.sh 里加：
    ```bash
    for org in platform organizer student ordererOrg; do
      rm -f fabric/network/organizations/fabric-ca/${org}/fabric-ca-server.db
    done
    ```

- [ ] **`network.sh` `startCAs` `sleep 3` 不够**
  - 当前：`fabric/network/network.sh:123` sleep 3，但 CAs 首次启动时生成 TLS cert 需要 5-8 秒。第一次 fresh start 会偶发 `ERROR: ca_platform is not running`
  - 修复：换成 wait-for-health 循环，curl `https://localhost:7054/cainfo` 直到 200，超时 30 秒
  - workaround：先单独 `docker compose up -d` 4 个 CA + sleep 6 + 再跑 `network.sh up`

- [ ] **没有 Rate Limiting / CSRF 防护**
  - 后端 Express 全裸跑
  - 修复：加 `express-rate-limit` + `csurf`（cookie token）

- [ ] **没有 HTTPS**
  - 前端直连后端 3000 (HTTP)
  - 修复：要么前端走 vite dev proxy + Caddy/nginx termination，要么后端启 HTTPS

- [ ] **前端缺少全局错误边界**
  - 当前：网络/后端错误时只在 console 报，UI 显示空白或部分数据
  - 修复：加 Vue ErrorBoundary 组件 + 全局 axios 拦截器统一弹 toast

---

## P2 — UX 细节

- [ ] **概率走势图（EventDetailView）目前是空的**
  - 当前：ECharts 图渲染但没数据点（`event.oddsHistory` 一直是空）
  - 修复：链码 `prediction.PlaceBet` 里追加 odds 历史到 pool；server 暴露 `GET /predictions/odds-history/:eventID`；前端 fetch 后填给 ECharts

- [ ] **下注后 selectedOption 不重置**
  - 用户下了一注后再看其他事件，可能保留上一个事件的 option 名（但被覆盖了，影响小）

- [ ] **预测之星 leaderboard 显示用户 ID 而不是 name**
  - 当前：HomeView leaderboard 显示 `3220100001`
  - 修复：服务端 join SQLite users 拿 name，或前端二次 fetch profile

- [ ] **`总下注` 和 `已结算预测` 的展示分裂**
  - 已修：HomeView 用 placedBets, ProfileView 显示「正确/已结算」双指标
  - 还可以更清楚：加 tooltip 解释什么是「已结算」

- [ ] **首页 PREDICTION_OPEN 之外的事件没地方看**
  - 当前：HomeView 只 filter `PREDICTION_OPEN/ONGOING`
  - 修复：加 tab 切换查看 SETTLED 历史 / CREATED 草稿

- [ ] **事件创建表单只在 seed 脚本里**
  - 没有 organizer 的 web UI 创建事件
  - 修复：加 `/admin/events/new` 页面

---

## P3 — 性能 / 代码质量

- [ ] **`buildConnectionProfile` 硬编码 `localhost:7051` 等**
  - 当前：`server/src/config/fabric.js` 三个 org 的 host/port 硬编码
  - 修复：从 `.env` 读

- [ ] **`fabricGateway.js` 的 gatewayCache 没有过期机制**
  - 当前：`gatewayCache.set(userId, ...)` 一直存
  - 风险：用户多时内存泄漏 + cert/identity rotation 后 cache 还是旧的
  - 修复：LRU + TTL，或者 close 老连接

- [ ] **CouchDB 索引只覆盖了 `accuracyRate desc`**
  - 当前：`fabric/chaincode/prediction/META-INF/statedb/couchdb/indexes/indexLeaderboard.json` 单字段
  - 优化：可以加 partial filter 索引限定 `docType=score AND totalBets > 0`，用 mango index 避免全表 scan

- [ ] **`enroll admin` 在 server 启动时跑了 4 次**
  - 一次 enroll 5-8 秒，启动慢
  - 修复：并行 + 缓存到 wallet（已经有 cache，但可以再快）

- [ ] **链码 cross-call 用部署名字符串**
  - 当前：`prediction.go:177` 写 `"token"`，`ticket.go:170` 写 `"prediction"`
  - 风险：如果重命名链码部署名，源码不会跟着改
  - 修复：从环境变量或 init args 读

---

## P4 — 增强功能

- [ ] **退款（refund）流程未走通**
  - `POST /api/v1/tickets/refund/:ticketID` 已修参数，但没测过
  - 需要：写 e2e 测试

- [ ] **链码单元测试**
  - `event_test.go`, `prediction_test.go`, `ticket_test.go`, `token_test.go` 已存在
  - 需要：`cd fabric/chaincode/<name> && go test ./...` 跑过一遍，可能因为依赖更新需要修

- [ ] **后端单元测试**
  - 当前：0 测试
  - 加：`vitest` 覆盖 routes/services 关键路径

- [ ] **前端 e2e 测试**
  - 加：`@playwright/test` 覆盖登录 → 下注 → 抽奖 → 结算流程

- [ ] **CI/CD**
  - GitHub Actions：lint + 单元测试 + chaincode `go vet` + chaincode `go test`

- [ ] **Docker compose 一键 dev**
  - 把后端和前端也容器化，单 `docker compose up` 全起

- [ ] **添加更多链码测试场景**
  - 同一 user 在同一 event 多次下注的 score 计算
  - 退款后的余额恢复
  - 抽奖时无人申请的边界

---

## 文档 TODO

- [ ] **API 文档：用 OpenAPI / Swagger 替代 README 里的 "API 速查"**
  - 现在的速查是手维护的，会漂移

- [ ] **链码函数签名表**
  - 列出每个链码的所有 public 函数 + 参数类型 + 返回类型，对照后端调用方便查

- [ ] **架构图升级**
  - 现在 README 里是 ASCII，可以画一张正经的 Mermaid sequence diagram
  - 关键流程：register、bet、settle、apply ticket、lottery

- [ ] **添加 ARCHITECTURE.md**
  - 解释为什么 3 orgs / 4 chaincodes 的拆分
  - state machine 设计的 trade-off

---

## 已修复（参考用，不要再改）

- [x] `fabric/network/bin` 缺失 → symlink → `../../bin`
- [x] `start-all.sh` 把 `up` 和 `createChannel` 塞一条命令 → 拆开
- [x] `createChannel.sh` `setAnchorPeer` 步骤生成的 config update 缺 mod_policy → 跳过（Fabric 2.x genesis 已含）
- [x] `createChannel.sh` `peer channel fetch config` 没 retry → 加 10 次 retry
- [x] `createChannel.sh` `joinOrdererToChannel` 没 retry → 加 10 次 retry
- [x] `createChannel.sh` `FABRIC_CFG_PATH` 硬编码绝对路径 → 改成 `${NETWORK_DIR}/../../config`
- [x] `envVar.sh` 同上
- [x] `auth.js` register 硬编码 role=student → 读 body.role + ROLE_TO_MSP 映射
- [x] `auth.js` register 不接受 admin/organizer，导致 seed 创建 organizer01 时角色错 → 同上
- [x] `seed-data.sh` `jq -r '.token'` 取错字段 → `.data.token`（响应是 `{error, data: {token}}`）
- [x] `seed-data.sh` 创建事件不传 eventID 但后续步骤硬编码 evt001 → 显式传 eventID
- [x] `seed-data.sh` `lottery` 调用没传 ticketCount → 加 body
- [x] `events.js` POST `/events` 把整个对象 JSON 化为单 arg → 按链码签名拆 6 个 string arg
- [x] `events.js` POST `/events` `ticketTotal || 0` 是谎言（链码拒绝 <=0）→ 必填 + 校验正整数
- [x] `events.js` POST `/events` 没 eventID 格式校验 → 加 `^[A-Za-z0-9_-]{1,64}$`
- [x] `events.js` POST `/events` 响应是 stub，缺 status/createdAt → evaluateTransaction 读回
- [x] `events.js` PUT `/:id/status` → PREDICTION_OPEN 时没初始化 pool → 自动调 InitializePool
- [x] `events.js` PUT `/:id/result` 调 Settle 缺 winningOption 参数 → 加 outcome
- [x] `predictions.js` POST `/bet` 缺 userID 参数（链码签名 4 个 args）→ 加 `req.user.userId`
- [x] `tickets.js` POST `/apply` 缺 userID → 加
- [x] `tickets.js` POST `/lottery/:eventID` 缺 ticketCountStr → 读 body.ticketCount
- [x] `tickets.js` POST `/claim/:ticketID` 路由参数名错（应该是 eventID）+ 缺 userID → 改名 + 加 userID
- [x] `tickets.js` POST `/refund/:ticketID` 缺 userID → 加
- [x] `prediction.go` `InvokeChaincode("token-cc", ...)` 应是 `"token"` → 改（2 处）
- [x] `ticket.go` `InvokeChaincode("prediction-cc", ...)` 应是 `"prediction"` → 改
- [x] `prediction.go` `UserScore` 缺 placedBets 字段（totalBets 实际是已结算事件参与数，不是下注数）→ 加 placedBets，PlaceBet 中 ++
- [x] `prediction` 链码 `GetLeaderboard` 用 CouchDB rich query 但没索引 → 加 `META-INF/statedb/couchdb/indexes/indexLeaderboard.json`
- [x] `caService.js` `registerAndEnrollUser` 不幂等 → deterministic secret + catch code 74 fall-through
- [x] `EventDetailView.vue` radio button `value="A"` 应是真实选项名 → `:value="event.predictionOptions[0]"`
- [x] `HomeView.vue` / `ProfileView.vue` 用 totalBets 显示「总预测」语义错 → 改用 placedBets

---

## Memory

- **17 个原始 bug + 7 个 reviewer 找的 + 3 个深挖发现的 = 27 修复**
- 全 reset + seed 第一次跑通，**0 errors**
- 完整 demo 流程跑通（注册 → 创建 → 开盘 → 下注 → 票务 → 抽奖 → 结算）

详细 bug fix 历史见 git log。
