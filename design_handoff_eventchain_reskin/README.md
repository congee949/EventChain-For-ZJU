# Handoff: EventChain Frontend Reskin → Editorial "On-Chain Broadsheet"

## Overview
Reskin the existing **EventChain (赛事链)** frontend — a closed-loop campus sports
prediction market, ticketing lottery, and service-exchange app — from its current
indigo **"Liquid Glass"** theme to the **EventChain Design System**: a warm,
ink-and-cream "on-chain broadsheet" look (condensed uppercase display type, a
monospace "ledger" voice, heavy 2px ink rules, one orange/red spine, flat warm
surfaces, no glass/blur/gradients).

Scope: **all 8 screens**, same layouts and flows as today — this is a visual
reskin, not a re-architecture. Behavior, routes, and data model are unchanged.

## About the Design Files
The file in this bundle (`EventchainScreen.dc.html`) is a **design reference
created in HTML** — an interactive prototype showing the intended look and
behavior. It is **not production code to copy directly**. Your task is to
**recreate this design inside the existing codebase** — a **Vue 3 + Vite + Pinia +
Vue Router + Element Plus** app — using its established patterns (SFCs, stores,
router, scoped styles / CSS variables). Reproduce the *look and behavior*; do not
port the prototype's structure verbatim.

The prototype uses React components from a design-system bundle for convenience.
In the Vue codebase you will **re-implement those same primitives as Vue SFCs**
(they are simple presentational components — see "Components to build").

## Fidelity
**High-fidelity (hifi).** Final colors, typography, spacing, radii, and
interactions are all specified below and in the prototype. Recreate the UI
pixel-perfectly using the codebase's Vue components and scoped styles. Where the
prototype and this README disagree, **this README wins**.

## Target codebase (what exists today)
```
EventChain/client/
├── src/
│   ├── main.js                 # Vue app bootstrap (Element Plus, Pinia, router)
│   ├── App.vue                 # shell: <GlassNavbar> + <router-view>
│   ├── router/index.js         # routes for all 8 views
│   ├── stores/                 # Pinia stores: auth.js, wallet, market, ...
│   ├── api/index.js            # backend calls (mock/real)
│   ├── assets/styles/
│   │   ├── variables.css       # ← current indigo "Liquid Glass" tokens (REPLACE)
│   │   └── global.css          # base resets + glass utility classes (REPLACE)
│   ├── components/
│   │   ├── GlassNavbar.vue     # top nav (RESKIN → masthead)
│   │   ├── GlassCard.vue       # frosted card wrapper (RESKIN → flat editorial card)
│   │   └── ...
│   └── views/
│       ├── LoginView.vue
│       ├── HomeView.vue            # 赛事市场 market home
│       ├── EventDetailView.vue     # 赛事详情
│       ├── TicketHallView.vue      # 票务大厅
│       ├── ServicesV2View.vue      # 服务兑换
│       ├── WalletV2View.vue        # A/B 钱包
│       ├── ProfileView.vue         # 我的
│       └── AdminView.vue           # 运营控制台
```
**Strategy:** the reskin is mostly (1) swap the design tokens in
`variables.css`, (2) replace the glass utility classes/`GlassCard`/`GlassNavbar`
with editorial equivalents, (3) build the small set of new presentational
components below, (4) update each view's markup to compose them. No route,
store, or API change is required.

---

## Design Tokens
Replace the contents of `src/assets/styles/variables.css` with these. **There is
no cool color anywhere — no blue/indigo/purple, no gradients, no glass/blur.**

```css
:root {
  /* ---------- Spine (brand) ---------- */
  --ec-orange:      #FF6A2B;  /* primary action, dark-theme accent */
  --ec-orange-hi:   #FF854E;  /* primary hover (lightens) */
  --ec-red:         #E8482C;  /* light-theme accent, links, kickers */
  --ec-amber:       #FFB224;  /* gold/focus, 2nd outcome */
  --ec-clay:        #C67A45;  /* 3rd outcome, B_bonus */

  /* ---------- Light "broadsheet" surfaces (the app shell) ---------- */
  --ec-canvas:      #E7E1D6;  /* page behind the shell */
  --ec-paper:       #F7F1E6;  /* shell / panel paper */
  --ec-card:        #FBF7EF;  /* card fill */
  --ec-card-white:  #FFFFFF;  /* the one "trading" card */
  --ec-inset:       #F1EADA;  /* inset tiles / rows */
  --ec-ink:         #211913;  /* primary text + 2px rules */
  --ec-ink-2:       #4A4034;  /* body text */
  --ec-muted:       #6E6152;  /* secondary text */
  --ec-faint:       #8A7D6A;  /* tertiary / fine print */
  --ec-faint-2:     #A99C87;  /* placeholder */
  --ec-line:        #DAD0BE;  /* hairline border (solid tan) */

  /* ---------- Dark "broadcast/terminal" surfaces ---------- */
  --ec-dark:        #211913;  /* dark panel */
  --ec-dark-2:      #2C2118;  /* dark inset tile */
  --ec-dark-3:      #17110B;  /* deeper */
  --ec-dark-4:      #0C0805;  /* deepest (code blocks) */
  --ec-cream:       #F3ECDD;  /* text on dark */
  --ec-cream-2:     #C7A98A;  /* mono label on dark */
  --ec-cream-3:     #9C8E7A;  /* muted on dark */
  --ec-cream-4:     #B7A992;  /* body on dark */

  /* ---------- Semantics ---------- */
  --ec-open:        #48843F;  /* live/open/won (light) */
  --ec-open-dark:   #48D17A;  /* live/open/won (dark) */
  --ec-pending:     #9A6700;  /* waiting/pending (amber-ink) */
  --ec-danger:      #B42318;  /* challenge/lost */

  /* ---------- Radius ladder ---------- */
  --ec-r-chip:   8px;
  --ec-r-field: 10px;
  --ec-r-card:  14px;
  --ec-r-panel: 16px;
  --ec-r-shell: 20px;
  --ec-r-pill:  999px;

  /* ---------- Shadow (essentially the only one) ---------- */
  --ec-shadow-shell: 0 40px 90px -30px rgba(30,20,10,.4);

  /* ---------- Type families ---------- */
  --ec-font-display: 'Saira Condensed', sans-serif;   /* 500–900 */
  --ec-font-body:    'Archivo', sans-serif;           /* 400–800 */
  --ec-font-mono:    'JetBrains Mono', monospace;      /* 400–700 */
}
```
Load the three families from Google Fonts (weights: Saira Condensed 500–900,
Archivo 400–800, JetBrains Mono 400–700) in `index.html` or via `@import`.

### Type roles
- **Saira Condensed** — display. UPPERCASE, condensed, `line-height:.9`,
  `letter-spacing:-.01em`. Heroes 54–76px, stat numbers, section titles, button
  labels.
- **Archivo** — body & UI. ~14px / `line-height:1.7`. Prose, secondary labels.
- **JetBrains Mono** — the "ledger" voice. Micro-labels 10–11px,
  `letter-spacing:.14–.22em`, UPPERCASE; block/hash IDs, data cells, numbers.

### The bilingual rule (core to the brand)
Chinese for everything a student reads (titles, labels, body, buttons). **English,
monospace, ALL-CAPS** for the "system/ledger" layer: kickers
(`PUBLIC SNAPSHOTS`, `COMMIT–REVEAL LOTTERY`, `PARI-MUTUEL POOL`) and raw
identifiers (`mkt_cs_ee_0722`, `BLOCK #148,392`, `finance-cc`). A Chinese title
almost always sits **under** a red/orange mono kicker. Middle dot ` · ` separates
metadata tokens. **No emoji, ever.**

---

## Components to build (Vue SFCs, presentational)
Re-implement these as small scoped-style SFCs (replace the glass equivalents).
Exact contracts mirror the design-system primitives used in the prototype.

### `Logo.vue`  (replaces brand lockup in GlassNavbar)
- A 45°-rotated square chip (~13px, `border-radius:2px`) in `--ec-red` (light) /
  `--ec-orange` (dark), immediately followed by **EVENTCHAIN** in Saira Condensed
  800–900, UPPERCASE, `letter-spacing:.04em`. Props: `theme` (light/dark),
  `size` (px, controls wordmark size). This wordmark lockup **is** the logo — no
  other mark exists.

### `EcButton.vue`
- Variants: `primary` (fill `--ec-orange`, ink text, hover → `--ec-orange-hi`),
  `ink` (fill `--ec-ink`, cream text), `outline` (1px border, transparent fill),
  `ghost` (no border, faint fill on hover). Label = Saira Condensed, UPPERCASE.
  Sizes `sm` (~40px h) / `md` (~44px h). Radius `--ec-r-field`. Transition
  ~120–160ms ease-out. **No scale/bounce**; press = subtle darken/opacity.

### `Badge.vue`
- Mono category tag (sport / service type). Small (radius ~5px), UPPERCASE mono
  ~10px. Tones: `clay`, `red` (tint background + matching text).

### `StatusPill.vue`
- Pill (`--ec-r-pill`) encapsulating the status-glyph vocabulary. States →
  glyph+color: `open`=● live green (pulsing), `waiting`=◷ amber, `pending`=◷
  amber, `won`=★ green, `lost`=◌ danger, `closed`=◌ muted. Props: `status`,
  `label`, `theme`.

### `LedgerStrip.vue`
- The signature on-chain ledger bar under the masthead. Mono, UPPERCASE. Variants
  `dateline` (tag + items + date, static) / `ticker` (marquee via `ecTick`
  translateX). Example items: `BLOCK #148,392 · 0x9f3a…c2e1 · finance-cc ·
  committed 24s ago`. Leading `◆` marker + tag `ON-CHAIN LEDGER`.

### `SectionHeader.vue`
- Mono red/orange kicker (UPPERCASE, `letter-spacing:.18em`) over a Saira title;
  optional `meta` on the right; optional 2px `--ec-ink` `rule` underline. Props:
  `kicker`, `title`, `meta`, `size`, `rule`, `theme`.

### `StatTile.vue`
- Big Saira pull-number over a mono caption. Variants: `card` / `inset` (fill
  `--ec-inset`) / `band` (part of a dark 4-col number band with hairline
  dividers). Optional `accent` (orange/clay) colors the number.

### `Field.vue`
- Labelled display input: mono micro-label above; input/select/stepper with fill
  `#FFFDF8`, 1px `--ec-line`, radius `--ec-r-field`; focus border `--ec-red`.
  Stepper uses `−` / `+` glyph buttons flanking a Saira number.

### `ProbabilityBar.vue`
- 2–3-way pari-mutuel split. Variants `split` (segments side by side, each
  labelled + %) / `stacked` (thin stacked bar + legend). Outcome colors in order:
  `--ec-orange`, `--ec-amber`, `--ec-clay`, on tinted fills (e.g.
  `rgba(255,106,43,.16)`). Props: `outcomes:[{label,pct}]`, `variant`, `theme`.

### `CategoryWallet.vue`
- Per-sport card showing B_paid (available/locked) and B_bonus (available/locked)
  for one category. Props: `sport`, `code`, `paid`, `paidLocked`, `bonus`,
  `bonusLocked`, `theme`. Numbers in Saira, captions in mono.

### `MarketCard.vue` (optional — HomeView uses inline rows)
- Composes Badge + StatusPill + ProbabilityBar into a prediction-market tile.

### Iconography
No icon set. Use JetBrains-Mono geometric Unicode glyphs colored by meaning:
`●`(live) `◷`(pending) `★`(won/focus) `◌`(lost) `◆`(ledger) `◈ ⚖ ↻ ✓ − + ▾ ←`.
Never emoji, icon font, or SVG icon library.

---

## Global shell & cards (replace the "glass" utilities)
- **Page**: `background:var(--ec-canvas)`; app centered.
- **Shell** (`App.vue` wrapper): width **1180px**, `background:var(--ec-paper)`,
  `border-radius:var(--ec-r-shell)`, `box-shadow:var(--ec-shadow-shell)`,
  `overflow:hidden`. Replaces the frosted container.
- **Card** (replaces `GlassCard`): `background:var(--ec-card)`, `1px solid
  var(--ec-line)`, `border-radius:var(--ec-r-card)`, pad 18–24px, **flat — no
  blur, no drop shadow** (elevation is reserved for the shell). The one "trading"
  card (position panel, live tickets) goes `--ec-card-white` with a
  `1px var(--ec-ink)` border.
- **No** colored left-border accent cards; **no** per-card shadows.

---

## Screens / Views

### 1. LoginView (登录)
- **Layout**: centered card, width ~452px. Top 6px bar in `--ec-red`. Content pad
  40px. Logo centered; mono kicker `CLOSED-LOOP ACCOUNT`; Saira title **欢迎回来**
  (34px, 800, UPPERCASE); Archivo subcopy (*"登录你的封闭积分账户；课程名单由管理员
  预置。"*). Two `Field`s: **学号 · STUDENT ID** (mono input, placeholder
  `3220100001`) and **密码 · PASSWORD** (password). Full-width `EcButton primary`
  **登录**. Fine print: *"没有账号？请联系课程管理员加入名单。"* Inline error text in
  `--ec-danger` on empty submit.
- **Behavior**: submit → set auth store `loggedIn`, route to `/` (market home).

### 2. HomeView (赛事市场 / market home)
- **Masthead** (60px, `border-bottom:2px solid --ec-ink`): Logo left; center nav
  (Saira UPPERCASE, ~15px): 赛事市场 · 票务大厅 · 服务兑换 · A/B 钱包 · 我的 ·
  管理 (active = `--ec-red`, else `--ec-muted`); right cluster (mono): `A 1,240`
  in `--ec-red`, user name, 退出.
- **LedgerStrip** dateline below masthead.
- **Hero** (pad 36/30, `border-bottom:2px solid --ec-ink`): two-col
  `1fr / 320px`. Left: mono kicker `CAMPUS SPORTS · CLOSED-LOOP POINTS`; Saira
  hero **赛事、积分与活动 在同一条可信链路上** (64px, 900, `line-height:.9`); body;
  two buttons (`ink` 查看活动票务, `outline` 管理 A/B). Right: **dark panel**
  (`--ec-dark`, radius 16px, cream text): mono `我的封闭积分 · A`; Saira 56px number
  in `--ec-orange`; two dark inset tiles (`--ec-dark-2`) B_PAID / B_BONUS.
- **Trust strip**: mono row, `border-bottom:1px solid --ec-line`:
  `◆ 组织级私有数据 · ◷ 7 天 PENDING CLAIM · ⚖ 挑战与仲裁 · ↻ 到期与销毁规则`.
- **Body**: two-col `1fr / 320px`, divider `1px --ec-line`.
  - Left: `SectionHeader` kicker `PUBLIC SNAPSHOTS` title **正在进行的预测市场**.
    Market **rows** (each `border-bottom:1px --ec-line`, hover fill `--ec-inset`):
    big Saira index number in `#C9BCA9`; mono sport kicker in `--ec-red`; Saira
    title; mono meta (`currency · 池 12,480 · 距锁盘 02:14:33`); a 190px-wide
    `ProbabilityBar variant="stacked"`. Click → EventDetail.
  - Right sidebar: `SectionHeader` **近期活动** + upcoming rows (title / time /
    count, click → Tickets); then a **dark callout** card (`--ec-dark`): `◈`,
    Saira **不是"聪明钱"排行榜**, cream body about rounded/delayed snapshots.
- **Countdowns tick live** (1s interval).

### 3. EventDetailView (赛事详情)
- Back link **← 返回赛事市场** (Saira, `--ec-red`).
- **Header card** (`--ec-card`, centered): row of category mono label + right
  `StatusPill` (预测开放/等待锁盘); mono `marketId` in `--ec-red`; Saira title
  48px; subcopy about "五分钟取整快照"; optional countdown line; centered
  `ProbabilityBar variant="split"` (max-width 520px).
- **Two-col** `1fr / 340px`:
  - Left stack of cards: **奖金池状态** (kicker `PARI-MUTUEL POOL`, currency tag;
    three `StatTile variant="inset"`: 公开池规模 / 市场上限 / 参与人数; note about
    private positions); **关联活动** (`CONNECTED ACTIVITY` + activity meta +
    `outline` button 前往票务大厅); **结算与纠错路径** (`SETTLEMENT SAFETY`, rule,
    4 numbered steps: 组织者提交结果+证据 → 验证者二次确认+挑战期 → 争议仲裁投票 →
    Claim 7 天成熟).
  - Right **position panel** (white card, `1px --ec-ink`): kicker
    `PRIVATE POSITION`, Saira **建立预测仓位**; available bucket line; outcome
    select buttons (selected = `--ec-red` border + `#FCEEE9` fill); **投入数量**
    stepper (`−` / Saira number input / `+`); full-width `primary` **确认建立仓位**;
    fine print about non-transferable positions + fee only on a winner.
    - **Non-student roles** see a read-only panel instead (kicker `ROLE VIEW`,
      Saira **只读市场视图**, `ink` button → 运营控制台).
- **Behavior**: select outcome → highlight; stepper ±5; 确认 → decrement wallet
  bucket available / increment locked + pool + participants; toast **仓位已写入
  私有账本**. Guard: amount>0 and ≤ available.

### 4. TicketHallView (票务大厅)
- **Header** (`border-bottom:2px --ec-ink`): kicker `COMMIT–REVEAL LOTTERY`, Saira
  **活动票务大厅** 56px, body about lottery not first-come; right: big Saira active
  count + mono `进行中活动`.
- **Available activities**: `SectionHeader` `AVAILABLE` title **可以报名的活动**;
  3-col grid of cards. Each: Badge(category) + StatusPill(报名中); Saira title;
  two inset tiles 已申请 / 容量; mono `报名截止` countdown + `活动开始` time; then
  either `primary` **提交抽签申请** (if applyable) or a status row showing my
  application state (PENDING amber / WON green / LOST muted).
- **Two-col** below: **我的申请** (rows: glyph + title + status; WON rows get a
  green **领取票据** action) and **我的动态票据** (white cards: StatusPill 有效, Saira
  title, mono ticketId, a 128px **QR placeholder** =
  `repeating-conic-gradient(#211913 0 25%,#FFFFFF 0 50%)` `background-size:12px`,
  mono note *"二维码每 30 秒变化，包含本机秘密，请勿转发截图。"*). Empty states for both.
- **Behavior**: 申请 → set application PENDING + increment applied, toast; WON +
  领取 → create ticket, toast; ticket proof/id derive from a 30s rotation.

### 5. ServicesV2View (服务兑换)
- **Header** (`border-bottom:2px --ec-ink`): kicker `B_PAID UTILITY`, Saira
  **器材、场馆与活动兑换** 56px, body about deposits.
- 3-col grid of **offer cards** (`--ec-card`): Badge(type, tone `red`) + mono
  `余量 N`; Saira title; price row = Saira 38px number + mono `B_paid`; mono line
  `取消赔付 1.50× · 保证金余 500 A`; a disabled "预约时间 ◷" field; `ink` button
  **兑换**.
- **Behavior**: 兑换 → decrement inventory, toast **订单已创建，取消保证金已预先锁定**.

### 6. WalletV2View (A/B 钱包)
- **Header** (`border-bottom:2px --ec-ink`): kicker `PRIVATE WALLET`, Saira
  **A / B 钱包** 56px, body about A / B_paid / B_bonus semantics.
- **Dark A banner** (`--ec-dark`, radius 16px): mono `A 可用`, Saira 52px number
  in `--ec-orange`, mono `冻结 120`.
- 3-col grid of `CategoryWallet` cards (basketball / football / badminton).
- **兑换与转让** card: `SectionHeader` rule; a category display select; a stepper;
  three buttons — `primary` **A → B_paid**, `outline` **B_paid → A**, `ghost`
  **转让 B_paid**; mono fine print with the limits (*每日 500 B_paid、最多 5 个
  收款方、转入后冷却 24 小时；B_paid 365 天到期自动退 A，B_bonus 90 天到期销毁*).
- **Behavior**: convert (A→B_paid, guard A balance), redeem (B_paid→A, −0.5% fee),
  transfer (locks/removes B_paid, 24h cooldown) — each mutates wallet + toasts.

### 7. ProfileView (我的)
- **Header** (`border-bottom:2px --ec-ink`): `陈` monogram tile (`--ec-dark`,
  `--ec-orange` initial, radius 18px); kicker `PRIVATE ACCOUNT`; Saira **陈同学**
  36px; mono `role · acct_7f3ac2e1…9b04`; right pill `● 学号未写入链上` (green).
- Grid: full-width **资产总览** card (`ASSET OVERVIEW` + 管理钱包→) with four
  `StatTile variant="band"` (A 可用 orange / B_paid / B_bonus clay / 有效票据);
  **分类余额** card (rows per category: paid + bonus); **哪些信息不会公开** dark card
  (`PRIVACY BOUNDARY`, four `◆` rows: 真实身份 / 精确仓位 / 资金追踪 / 现实边界);
  full-width **票务申请** card (rows, → 票务大厅); full-width **积分生命周期** card
  (`LIFECYCLE`, rule, four tiles: 365 天 / 90 天 / 24 小时 / 7 天).

### 8. AdminView (运营控制台)
- **Header** (`border-bottom:2px --ec-ink`): kicker `ROLE-BOUND OPERATIONS`, Saira
  **运营控制台** 54px, body naming current role; right: pulsing `● V2 NETWORK`
  green + mono `finance · activity`.
- **Role switcher**: mono label + pills for `student / organizer / verifier /
  operator / arbitrator / admin` (active = `--ec-ink` fill + `--ec-orange` text).
- **Overview**: four `StatTile variant="card"` (预测市场 / 活动 / 服务商品 / 证书角色).
- **创建业务对象** (only when role=organizer): 3-col cards (预测市场 / 活动与抽签 /
  服务兑换商品) of display fields + a `primary` create button each.
- **CONTROL PLANE**: 3 cards — 市场动作 / 抽签与活动状态 (buttons **filtered by
  role**: 开放/锁盘/提交结果/二次确认/临时结算/仲裁投票 etc.) and, for
  operator/admin, a **dark** 分批 Claim Worker card with a `primary` button and a
  mono JSON report `<pre>` in `--ec-dark-4`.
- **LIVE OBJECTS**: mono 4-col table of current markets + activities
  (id / category / status / extra).
- **Behavior**: switching role re-computes which action buttons render; create
  buttons append mock objects; process-claims sets the report + toast. Button
  *visibility is a hint only* — real permission is enforced server-side by Fabric
  cert attributes + chaincode (keep that note in the UI copy).

---

## Interactions & Behavior (global)
- **Nav**: masthead links switch views via Vue Router (existing routes). Active
  link = `--ec-red`.
- **Toasts**: bottom-center dark pill (`--ec-ink`, cream text, `--ec-r-pill`),
  mono ~12.5px, leading `✓`, auto-dismiss ~2.6s. (Codebase already uses Element
  Plus `ElMessage` — either restyle it to this or use a small custom toast.)
- **Live timers**: 1s interval driving all countdowns (market lock, application
  deadlines) and the 30s ticket-QR rotation.
- **Hover/press**: primary orange **lightens** (`#FF6A2B→#FF854E`); ink/outline
  shift opacity; ghost gets faint fill; links warm red→orange; rows fill
  `--ec-inset`. **No scale/bounce.** Transitions ~120–200ms ease-out.
- **Animations**: only functional — `ecPulse` (opacity+slight scale, 1.6s) on
  live dots; `ecTick` translateX marquee for the ledger ticker.

## State Management (maps to existing Pinia stores)
- **auth**: `loggedIn`, `userName` ('陈同学'), `role` (student default; the admin
  switcher sets this). Login sets loggedIn + routes home; 退出 clears.
- **wallet**: `aBalance` (1240), per-category `{ paid:{available,locked},
  bonus:{available,locked} }` for cat_basketball / cat_football / cat_badminton;
  `aReserved` (120). Mutated by place-position, convert, redeem, transfer.
- **market**: `markets[]` (id, categoryId, sport, title, currency, stakeBucket
  PAID/BONUS, pool, cap, status, participants, eventId, closeOffset, outcomes),
  `selectedOutcome{}`, `amount`.
- **activity/tickets**: `activities[]` (id, categoryId, title, capacity, applied,
  status, appCloseOffset, startsIn), `applications{id→PENDING|WON|LOST}`,
  `tickets{id→true}`.
- **exchange**: `offers[]` (offerId, categoryId, offerType, typeLabel, title,
  price, inventory, cancellationBps, guarantee).
- **admin**: `claimReport` object; role drives button visibility.
The prototype's `EventchainScreen.dc.html` logic class contains ready-to-copy
**mock seed data** and every handler's exact mutation — mine it directly.

## Assets
**None.** No images, icons, or logo files — by design. The EVENTCHAIN wordmark
lockup is the only mark; all glyphs are Unicode. Fonts come from Google Fonts
(Saira Condensed, Archivo, JetBrains Mono). The only "textures" are the QR
`repeating-conic-gradient` checker and probability-bar tint fills.

## Screenshots
Full-shell reference captures live in `screenshots/` (2× retina):
- `1-login.png` — 登录
- `2-market.png` — 赛事市场 (market home)
- `3-event-detail.png` — 赛事详情
- `4-tickets.png` — 票务大厅
- `5-services.png` — 服务兑换
- `6-wallet.png` — A/B 钱包
- `7-profile.png` — 我的
- `8-admin.png` — 运营控制台 (organizer role, full create + control plane)

## Files
- `EventchainScreen.dc.html` — the interactive hifi prototype (all 8 screens +
  nav + mock data + handlers). Open in a browser to click through; read its
  `<script>` logic class for exact seed data, formatters, and state transitions.
- `screenshots/` — per-screen reference images (see above).

## Notes for implementation
- Recreate in the **existing Vue codebase** using SFCs + scoped styles / the token
  CSS above — do **not** ship the HTML prototype or introduce React.
- Keep routes, stores, and API contracts as they are; this is a visual reskin.
- Where prototype ≠ README, **README wins**. Match hex values and type roles
  exactly — the design is intentionally warm, flat, and glass-free.
