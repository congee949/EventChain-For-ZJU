# EventChain Design Spec

> ZJU 区块链课程大作业 — 校园赛事预测市场 + 公平票务系统
> 
> 一句话概述：校园版 Polymarket + 公平抢票，用虚拟代币预测赛事结果，预测准确度影响购票优先权。

---

## 1. System Architecture

```
┌─────────────────────────────────────────────────┐
│                   用户浏览器                      │
│            Vue 3 + Liquid Glass UI               │
└────────────────────┬────────────────────────────┘
                     │ REST API (JSON)
┌────────────────────▼────────────────────────────┐
│              Node.js + Express 后端               │
│  ┌──────────┐ ┌──────────┐ ┌──────────────────┐ │
│  │ 路由层    │ │ 认证中间件│ │ Fabric Gateway   │ │
│  │ /api/v1  │ │ JWT + CA │ │ SDK 连接池       │ │
│  └──────────┘ └──────────┘ └──────────────────┘ │
└────────────────────┬────────────────────────────┘
                     │ gRPC
┌────────────────────▼────────────────────────────┐
│           Hyperledger Fabric 2.5 网络             │
│                                                  │
│  PlatformOrg(peer0) + OrganizerOrg(peer0)       │
│  + StudentOrg(peer0)                             │
│                                                  │
│  Channel: eventchain                             │
│  Chaincodes: event-cc, prediction-cc,            │
│              ticket-cc, token-cc                 │
│                                                  │
│  Orderer (Raft) │ CA ×3 │ CouchDB ×3           │
└──────────────────────────────────────────────────┘
```

### Key Decisions

- Fabric network based on `fabric-samples/test-network`, extended to 3 orgs
- Single channel `eventchain`, 4 chaincodes
- Backend uses Fabric Gateway SDK (not legacy SDK) with connection pooling
- REST between frontend and backend; JWT embeds Fabric identity info
- CouchDB as state database for rich queries (filter by event type, status, time range)

---

## 2. Fabric Network

### Organizations

| Org | MSP ID | Role | Peers |
|-----|--------|------|-------|
| PlatformOrg | PlatformMSP | 平台管理方（学校/课程组） | peer0 |
| OrganizerOrg | OrganizerMSP | 赛事主办方（社团、体院） | peer0 |
| StudentOrg | StudentMSP | 学生用户 | peer0 |

### Network Components (Docker Compose)

```
orderer.eventchain.com        — Raft orderer
peer0.platform.eventchain.com — PlatformOrg peer + CouchDB
peer0.organizer.eventchain.com — OrganizerOrg peer + CouchDB
peer0.student.eventchain.com  — StudentOrg peer + CouchDB
ca.platform.eventchain.com    — Platform CA
ca.organizer.eventchain.com   — Organizer CA
ca.student.eventchain.com     — Student CA
```

### Endorsement Policy

All 4 chaincodes: `OR('PlatformMSP.peer', 'OrganizerMSP.peer', 'StudentMSP.peer')` — any single org can endorse. Keeps it simple for a course project; production would use stricter policies.

---

## 3. Chaincode Design (Go)

### 3.1 token-chaincode — 浙币管理

World State key: `token:{userID}` → `{ owner, balance, lastUpdated }`

| Method | Access | Description |
|--------|--------|-------------|
| `Initialize(totalSupply)` | PlatformOrg | Init token pool |
| `Mint(userID, amount)` | PlatformOrg | Award tokens (registration bonus: 1000) |
| `Transfer(from, to, amount)` | Owner | Transfer tokens (used by betting/settlement) |
| `BalanceOf(userID)` | Any | Query balance |
| `History(userID)` | Owner | Transaction history via CouchDB rich query |

### 3.2 event-chaincode — 赛事管理

World State key: `event:{eventID}` → `{ id, title, type, teams, ticketTotal, status, predictionOptions, result, timestamps }`

State machine: `CREATED → PREDICTION_OPEN → TICKET_OPEN → ONGOING → SETTLED`

| Method | Access | Description |
|--------|--------|-------------|
| `CreateEvent(...)` | OrganizerOrg | Create event + prediction options + ticket quota |
| `QueryEvent(eventID)` | Any | Event detail |
| `ListEvents(status, type)` | Any | Filter via CouchDB selector |
| `UpdateStatus(eventID, status)` | OrganizerOrg | Advance state machine |
| `UpdateResult(eventID, outcome)` | OrganizerOrg | Record result, triggers settlement |

### 3.3 prediction-chaincode — 预测市场 (AMM)

World State keys:
- `pool:{eventID}` → `{ poolA, poolB, k, totalVolume }`
- `bet:{eventID}:{userID}:{betID}` → `{ option, amount, shares, timestamp }`
- `score:{userID}` → `{ totalBets, correctBets, accuracyRate }`

AMM (Constant Product):
```
x * y = k

Buy option A with d tokens:
  poolB += d
  newPoolA = k / poolB
  shares = poolA - newPoolA  (user receives)
  poolA = newPoolA

Probability:
  P(A) = poolB / (poolA + poolB)
  P(B) = poolA / (poolA + poolB)
```

| Method | Access | Description |
|--------|--------|-------------|
| `PlaceBet(eventID, option, amount)` | Any | Bet — calls token-cc to deduct, AMM calculates shares |
| `GetOdds(eventID)` | Any | Current probability from pool state |
| `GetPool(eventID)` | Any | Full pool details (for charts) |
| `Settle(eventID)` | OrganizerOrg | Post-match settlement — distribute pool to winners proportionally by shares, calls token-cc |
| `GetUserScore(userID)` | Any | Historical accuracy rate |
| `GetUserBets(userID, eventID?)` | Owner | Bet history |

Initial pool: each option starts with 10000 shares, k = 10000 * 10000 = 100,000,000.

Pool accounting: when a user bets, tokens are transferred to a virtual pool account `pool:{eventID}`. During settlement, tokens are transferred out of this pool account to winners. The pool account is managed by prediction-cc internally — the cross-chaincode `Transfer` call runs under the transaction submitter's identity, but the `from` field specifies the pool account ID.

Settlement logic:
```
totalPool = pool.totalVolume (sum of all bets)
For each winning bet:
  payout = (bet.shares / totalWinningShares) * totalPool
  token-cc.Transfer("pool:{eventID}", bet.userID, payout)
Update score: correctBets++ for winners, totalBets++ for all
Remaining dust (rounding) stays in pool (negligible)
```

### 3.4 ticket-chaincode — 票务管理

World State keys:
- `application:{eventID}:{userID}` → `{ status, priority, timestamp }`
- `ticket:{ticketID}` → `{ eventID, ownerID, status, claimHash, verifiedAt }`

| Method | Access | Description |
|--------|--------|-------------|
| `ApplyTicket(eventID)` | Any | Submit ticket application |
| `RunLottery(eventID)` | OrganizerOrg | Weighted lottery: `priority = 0.6 * accuracy + 0.4 * pseudorandom`. Random seed = SHA256(txID + eventID + userID) for deterministic cross-peer consensus. |
| `ClaimTicket(ticketID)` | Winner | Claim ticket, generates unique hash for QR |
| `VerifyTicket(ticketID, hash)` | OrganizerOrg | Scan QR to verify, mark as USED |
| `RefundTicket(ticketID)` | Owner | Refund before event, release slot |

### Cross-chaincode Calls

- `prediction-cc` → `token-cc.Transfer()` on bet placement and settlement
- `ticket-cc` → `prediction-cc.GetUserScore()` for lottery weighting
- All via Fabric's `InvokeChaincode()` within the same channel

---

## 4. Backend API (Express)

### Authentication

```
POST /api/v1/auth/register   { studentID, password, name }
  → Backend: hash password (bcrypt) → store in local SQLite/JSON
  → Fabric CA: register(studentID) → enroll → store cert in file wallet
  → token-cc.Mint(studentID, 1000)
  → Return JWT

POST /api/v1/auth/login      { studentID, password }
  → Backend: verify bcrypt hash from local store
  → Return JWT { userID, orgMSP, exp }
```

Password storage: backend maintains a lightweight local store (SQLite or JSON file) for bcrypt-hashed passwords. Fabric CA handles X.509 certificate lifecycle only — it does not store or verify user passwords. The file-based Fabric identity wallet (`server/wallet/`) holds enrolled certificates keyed by studentID.

JWT middleware: extracts identity from JWT, loads corresponding Fabric certificate from wallet for chaincode calls.

### Events

```
GET    /api/v1/events                    — List (query: ?status=&type=)
GET    /api/v1/events/:id                — Detail + current odds
POST   /api/v1/events                    — [Organizer] Create event
PUT    /api/v1/events/:id/status         — [Organizer] Advance status
PUT    /api/v1/events/:id/result         — [Organizer] Record result + trigger settle
```

### Predictions

```
POST   /api/v1/predictions/bet           — { eventID, option, amount }
GET    /api/v1/predictions/odds/:eventID — Current odds
GET    /api/v1/predictions/pool/:eventID — Pool details (for chart)
GET    /api/v1/predictions/mine          — My bet history
GET    /api/v1/predictions/score         — My accuracy score
```

### Tickets

```
POST   /api/v1/tickets/apply             — { eventID }
POST   /api/v1/tickets/lottery/:eventID  — [Organizer] Run lottery
GET    /api/v1/tickets/mine              — My tickets
POST   /api/v1/tickets/claim/:ticketID   — Claim ticket
GET    /api/v1/tickets/verify/:ticketID  — Verify ticket (QR scan)
POST   /api/v1/tickets/refund/:ticketID  — Refund ticket
```

### Users

```
GET    /api/v1/users/profile             — My profile + balance + accuracy
GET    /api/v1/users/leaderboard         — Top predictors
```

### Error Response Format

```json
{ "error": true, "code": "INSUFFICIENT_BALANCE", "message": "余额不足" }
```

---

## 5. Frontend Design — Liquid Glass

### Tech Stack

- Vue 3 (Composition API + `<script setup>`)
- Vue Router 4
- Pinia (state management)
- Element Plus (component structure, custom CSS theme override)
- ECharts 5 (odds charts, accuracy radar)
- Liquid Glass CSS system (custom)

### Liquid Glass CSS System

Three material tiers:

```css
/* Dense — navigation bars, headers */
.glass-dense {
  background: rgba(255, 255, 255, 0.35);
  backdrop-filter: blur(32px) saturate(2);
  border: 0.5px solid rgba(255, 255, 255, 0.45);
  box-shadow: inset 0 0.5px 0 rgba(255,255,255,0.6),
              0 8px 40px rgba(0,0,0,0.06);
}

/* Regular — cards, panels */
.glass {
  background: rgba(255, 255, 255, 0.22);
  backdrop-filter: blur(24px) saturate(1.8);
  border: 0.5px solid rgba(255, 255, 255, 0.4);
}

/* Subtle — nested elements, badges */
.glass-subtle {
  background: rgba(255, 255, 255, 0.12);
  backdrop-filter: blur(16px) saturate(1.4);
}
```

Dynamic specular highlight (all `.glass` elements):
```css
.glass::before {
  content: '';
  position: absolute; inset: 0;
  background: radial-gradient(
    ellipse at var(--highlight-x, 30%) var(--highlight-y, 20%),
    rgba(255,255,255,0.45) 0%, transparent 65%
  );
  mix-blend-mode: overlay;
  pointer-events: none;
}
```

JS mouse tracking updates `--highlight-x` / `--highlight-y` per element.

Background: mesh gradient with 4 radial color stops (blue, pink, green, amber) + 3 floating blurred orbs with `animation: float 20s ease-in-out infinite`.

### Pages

#### 5.1 Home (`/`)
- Hero: gradient text title + subtitle
- Hot events grid (2 columns): EventCard with live odds, sparkline, bet buttons
- Sidebar: My Stats card, Leaderboard card, Ticket Status card

#### 5.2 Event Detail (`/event/:id`)
- Event header: teams, status badge, participant count
- **ECharts odds history chart**: line chart showing probability changes over time
  - X axis: time, Y axis: probability %
  - Two lines (option A / option B), area fill
  - Tooltip shows exact odds + volume at each point
- Bet panel: slider to choose amount, real-time payout preview based on current AMM state
- Recent bets feed: live updates of who bet what
- Related events

#### 5.3 Ticket Hall (`/tickets`)
- Upcoming events with ticket availability
- My applications + status (pending / won / lost)
- Countdown timers for lottery execution
- Won tickets with QR code (generated client-side from ticket hash)

#### 5.4 My Profile (`/me`)
- Balance + accuracy rate + total profit/loss
- **ECharts radar chart**: accuracy across different event types (basketball, football, esports, etc.)
- Bet history list with outcomes
- My tickets list
- Achievement badges (first bet, 10-win streak, top 10 accuracy, etc.)

#### 5.5 Admin (`/admin`)
- Create event form
- Active events management (advance status, record results)
- System stats dashboard (total users, total volume, active events)
- Manual token mint (for testing)

### Vue Component Tree

```
App.vue
├── GlassNavbar.vue              — sticky nav, balance display, avatar
├── views/
│   ├── HomeView.vue
│   │   ├── HeroSection.vue
│   │   ├── EventCard.vue        — glass card with odds, sparkline, bet buttons
│   │   ├── StatsCard.vue        — personal stats sidebar
│   │   ├── LeaderboardCard.vue  — top predictors
│   │   └── TicketStatusCard.vue — upcoming ticket events
│   ├── EventDetailView.vue
│   │   ├── EventHeader.vue      — teams, status, participants
│   │   ├── OddsChart.vue        — ECharts line chart
│   │   ├── BetPanel.vue         — amount slider + payout preview
│   │   └── RecentBets.vue       — live bet feed
│   ├── TicketHallView.vue
│   │   ├── TicketEventCard.vue  — event + apply button + countdown
│   │   └── MyTicketCard.vue     — QR code display
│   ├── ProfileView.vue
│   │   ├── BalanceCard.vue
│   │   ├── AccuracyRadar.vue    — ECharts radar
│   │   ├── BetHistoryList.vue
│   │   └── AchievementBadges.vue
│   └── AdminView.vue
│       ├── CreateEventForm.vue
│       ├── EventManager.vue
│       └── SystemStats.vue
└── components/
    ├── GlassCard.vue            — base glass container (dense/regular/subtle)
    ├── ProbabilityBar.vue       — colored progress bar for odds
    ├── SparkLine.vue            — mini SVG chart
    ├── CountdownTimer.vue
    └── QRCode.vue               — client-side QR from ticket hash
```

### Pinia Stores

```
stores/
├── auth.ts       — user session, JWT, login/register
├── events.ts     — event list, current event, polling
├── prediction.ts — odds, bets, user score
├── ticket.ts     — applications, my tickets
└── user.ts       — balance, profile, leaderboard
```

---

## 6. Data Flow

### Registration
```
Student → POST /auth/register { studentID, pwd, name }
  → Backend: Fabric CA register(studentID) → enroll → get certificate
  → Backend: token-cc.Mint(studentID, 1000)
  → Backend: sign JWT { userID, org: StudentMSP }
  → Frontend: store JWT, redirect to home
```

### Place Bet
```
Student → POST /predictions/bet { eventID: "evt001", option: "A", amount: 100 }
  → Backend: JWT → load Fabric identity
  → prediction-cc.PlaceBet("evt001", "A", 100)
    → token-cc.Transfer(student, pool, 100)     [cross-chaincode]
    → AMM: poolB += 100, newPoolA = k/poolB
    → shares = poolA - newPoolA
    → store bet record
  → Response: { shares: 91, newOddsA: 0.55, newOddsB: 0.45 }
  → Frontend: update odds chart, balance, bet list
```

### Settlement
```
Organizer → PUT /events/evt001/result { outcome: "A" }
  → event-cc.UpdateResult("evt001", "A")
  → prediction-cc.Settle("evt001")
    → for each bet on option A:
        payout = (shares / totalWinShares) * totalPool
        token-cc.Transfer(pool, winner, payout)    [cross-chaincode]
    → update score: winners.correctBets++, all.totalBets++
  → Response: { settled: true, winners: 45, totalPayout: 8500 }
```

### Ticket Lottery
```
Organizer → POST /tickets/lottery/evt001
  → ticket-cc.RunLottery("evt001")
    → for each applicant:
        score = prediction-cc.GetUserScore(userID)  [cross-chaincode]
        priority = 0.6 * score.accuracyRate + 0.4 * random()
    → sort by priority desc, top N = ticketTotal
    → mark winners as WON, others as LOST
  → Winners → POST /tickets/claim/:ticketID
    → generate unique hash → QR code
```

---

## 7. Project Structure

```
/Users/Apple/EventChain/
├── fabric/
│   ├── network/                    # Based on test-network
│   │   ├── docker/
│   │   │   ├── docker-compose-net.yaml
│   │   │   ├── docker-compose-ca.yaml
│   │   │   └── docker-compose-couch.yaml
│   │   ├── organizations/
│   │   │   ├── cryptogen/          # Crypto config for 3 orgs
│   │   │   └── fabric-ca/          # CA server configs
│   │   ├── configtx/
│   │   │   └── configtx.yaml       # Channel config with 3 orgs
│   │   ├── scripts/
│   │   │   ├── createChannel.sh
│   │   │   ├── deployCC.sh
│   │   │   └── envVar.sh
│   │   └── network.sh              # Main network management script
│   └── chaincode/
│       ├── event/
│       │   ├── go.mod
│       │   └── event.go
│       ├── prediction/
│       │   ├── go.mod
│       │   └── prediction.go
│       ├── ticket/
│       │   ├── go.mod
│       │   └── ticket.go
│       └── token/
│           ├── go.mod
│           └── token.go
├── server/
│   ├── src/
│   │   ├── app.js                  # Express app setup
│   │   ├── config/
│   │   │   ├── fabric.js           # Fabric connection profile
│   │   │   └── index.js            # App config
│   │   ├── middleware/
│   │   │   ├── auth.js             # JWT verification
│   │   │   └── errorHandler.js
│   │   ├── routes/
│   │   │   ├── auth.js
│   │   │   ├── events.js
│   │   │   ├── predictions.js
│   │   │   ├── tickets.js
│   │   │   └── users.js
│   │   └── services/
│   │       ├── fabricGateway.js     # Gateway SDK wrapper
│   │       ├── caService.js         # Fabric CA operations
│   │       └── wallet.js            # Identity wallet management
│   ├── package.json
│   └── .env.example
├── client/
│   ├── src/
│   │   ├── App.vue
│   │   ├── main.js
│   │   ├── assets/
│   │   │   └── styles/
│   │   │       ├── liquid-glass.css # Glass material system
│   │   │       ├── variables.css    # CSS custom properties
│   │   │       └── global.css
│   │   ├── components/
│   │   │   ├── GlassCard.vue
│   │   │   ├── GlassNavbar.vue
│   │   │   ├── ProbabilityBar.vue
│   │   │   ├── SparkLine.vue
│   │   │   ├── CountdownTimer.vue
│   │   │   └── QRCode.vue
│   │   ├── views/
│   │   │   ├── HomeView.vue
│   │   │   ├── EventDetailView.vue
│   │   │   ├── TicketHallView.vue
│   │   │   ├── ProfileView.vue
│   │   │   └── AdminView.vue
│   │   ├── stores/
│   │   │   ├── auth.js
│   │   │   ├── events.js
│   │   │   ├── prediction.js
│   │   │   ├── ticket.js
│   │   │   └── user.js
│   │   ├── router/
│   │   │   └── index.js
│   │   └── api/
│   │       └── index.js             # Axios instance + interceptors
│   ├── index.html
│   ├── vite.config.js
│   └── package.json
├── scripts/
│   ├── start-all.sh                 # Start network + server + client
│   ├── seed-data.sh                 # Seed demo data
│   └── cleanup.sh                   # Tear down everything
├── docs/
│   └── superpowers/
│       └── specs/
│           └── 2026-04-05-eventchain-design.md
├── .gitignore
└── README.md
```

---

## 8. Demo & Seed Data Strategy

For presentation, pre-seed the system with realistic data:

### Seed Events
1. 🏀 院际篮球决赛：教院 vs 丹青 (ONGOING, prediction open)
2. ⚽ 校运会足球半决赛：竺院 vs 蓝田 (PREDICTION_OPEN)
3. 🎮 英雄联盟校赛决赛：CS战队 vs EE战队 (PREDICTION_OPEN)
4. 🏸 羽毛球团体赛：求是 vs 云峰 (CREATED)
5. 🏃 校运会百米决赛 (SETTLED — demo completed flow)

### Seed Users
- 5-10 student accounts with varying prediction histories
- 1 organizer account
- 1 platform admin account

### Seed Predictions
- Spread bets across events to show AMM in action
- Some accounts with high accuracy (80%+), some low — demonstrates leaderboard differentiation

### Live Demo Script
1. Login as student → show home page with live odds
2. Place a bet → watch odds update in real-time on the chart
3. Switch to organizer → record match result → trigger settlement
4. Switch back to student → show balance increased, accuracy updated
5. Show ticket lottery → weighted by prediction accuracy
6. Show QR ticket verification flow

---

## 9. Testing Strategy

### Chaincode Unit Tests (Go)
- Test each chaincode method with mock stub
- Test AMM math: verify `x * y = k` invariant holds after every bet
- Test settlement: verify total payout = total pool (no funds leak)
- Test lottery: verify priority formula and winner count = ticket quota

### API Integration Tests
- Test auth flow: register → login → JWT validation
- Test bet flow: place bet → verify balance deducted → verify odds changed
- Test settlement: record result → verify winners paid
- Test ticket flow: apply → lottery → claim → verify

### Frontend
- Manual testing across 5 pages
- Verify ECharts renders correctly with real data
- Verify Liquid Glass effects work in Chrome/Safari/Firefox

---

## 10. Non-Goals (Explicit Exclusions)

- No WebSocket real-time push (polling is fine for a course project)
- No mobile responsive design (desktop-first for presentation)
- No i18n (Chinese only)
- No CI/CD pipeline
- No production deployment (local Docker only)
- No multi-channel Fabric setup (single channel suffices)
- No complex access control per chaincode method (simplified for demo)
