#!/bin/bash
set -euo pipefail

API_BASE="${API_BASE:-http://localhost:3000/api/v2}"
BOOTSTRAP_KEY="${DEMO_BOOTSTRAP_KEY:-eventchain-demo-bootstrap}"

post_json() {
  local url="$1" body="$2" token="${3:-}" idem="${4:-}"
  local args=(-sS -f -X POST "$API_BASE$url" -H "Content-Type: application/json")
  [ -n "$token" ] && args+=(-H "Authorization: Bearer $token")
  [ -n "$idem" ] && args+=(-H "Idempotency-Key: $idem")
  curl "${args[@]}" -d "$body"
}

bootstrap() {
  local login="$1" password="$2" name="$3" role="$4"
  local existing
  if existing=$(post_json /auth/login "{\"studentID\":\"$login\",\"password\":\"$password\"}" 2>/dev/null); then
    echo "$existing"
    return
  fi
  curl -sS -f -X POST "$API_BASE/auth/bootstrap" \
    -H "Content-Type: application/json" -H "X-Demo-Bootstrap-Key: $BOOTSTRAP_KEY" \
    -d "{\"studentID\":\"$login\",\"password\":\"$password\",\"name\":\"$name\",\"role\":\"$role\"}"
}

login() {
  post_json "/auth/login" "{\"studentID\":\"$1\",\"password\":\"$2\"}" | jq -r '.data.token'
}

echo "Waiting for EventChain V2 API..."
for _ in $(seq 1 60); do
  curl -fsS "${API_BASE%/v2}/health" >/dev/null 2>&1 && break
  sleep 1
done

echo "Bootstrapping role-bound opaque identities..."
ADMIN_JSON=$(bootstrap admin01 eventchain-admin-2026 系统管理员 admin)
OPERATOR_JSON=$(bootstrap operator01 eventchain-operator-2026 结算运营员 operator)
VERIFIER_JSON=$(bootstrap verifier01 eventchain-verifier-2026 独立验证员 verifier)
ORGANIZER_JSON=$(bootstrap organizer01 eventchain-organizer-2026 体育社团 organizer)
ARB1_JSON=$(bootstrap arb01 eventchain-arb01-2026 仲裁员一 arbitrator)
ARB2_JSON=$(bootstrap arb02 eventchain-arb02-2026 仲裁员二 arbitrator)
ARB3_JSON=$(bootstrap arb03 eventchain-arb03-2026 仲裁员三 arbitrator)

ADMIN_TOKEN=$(jq -r '.data.token' <<<"$ADMIN_JSON")
OPERATOR_TOKEN=$(jq -r '.data.token' <<<"$OPERATOR_JSON")
VERIFIER_TOKEN=$(jq -r '.data.token' <<<"$VERIFIER_JSON")
ORGANIZER_TOKEN=$(jq -r '.data.token' <<<"$ORGANIZER_JSON")
ORGANIZER_ID=$(jq -r '.data.userId' <<<"$ORGANIZER_JSON")
OPERATOR_ID=$(jq -r '.data.userId' <<<"$OPERATOR_JSON")
VERIFIER_ID=$(jq -r '.data.userId' <<<"$VERIFIER_JSON")
ARB1_ID=$(jq -r '.data.userId' <<<"$ARB1_JSON")
ARB2_ID=$(jq -r '.data.userId' <<<"$ARB2_JSON")
ARB3_ID=$(jq -r '.data.userId' <<<"$ARB3_JSON")

echo "Initializing finance V2 and governance rosters..."
post_json /finance/admin/initialize '{}' "$ADMIN_TOKEN" >/dev/null
for category in 'basketball:篮球' 'football:足球' 'badminton:羽毛球' 'venue:场馆'; do
  IFS=: read -r id name <<<"$category"
  post_json /finance/admin/categories "{\"id\":\"$id\",\"name\":\"$name\"}" "$ADMIN_TOKEN" >/dev/null 2>&1 || true
done
for account in "$ARB1_ID" "$ARB2_ID" "$ARB3_ID"; do
  post_json "/finance/admin/arbitrators/$account" '{}' "$ADMIN_TOKEN" >/dev/null
done
for account in "$(jq -r '.data.userId' <<<"$ADMIN_JSON")" "$OPERATOR_ID" "$VERIFIER_ID"; do
  post_json "/finance/admin/emergency-approvers/$account" '{}' "$ADMIN_TOKEN" >/dev/null
done

echo "Registering demo students and seeding A/B balances..."
STUDENT_LOGINS=(3220100001 3220100002 3220100003 3220100004 3220100005)
STUDENT_IDS=()
STUDENT_TOKENS=()
for index in "${!STUDENT_LOGINS[@]}"; do
  login_id="${STUDENT_LOGINS[$index]}"
  response=$(bootstrap "$login_id" eventchain-student-2026 "学生$((index+1))" student)
  account_id=$(jq -r '.data.userId' <<<"$response")
  token=$(jq -r '.data.token' <<<"$response")
  STUDENT_IDS+=("$account_id")
  STUDENT_TOKENS+=("$token")
  post_json /finance/admin/demo/a-credit "{\"accountId\":\"$account_id\",\"amount\":\"1000\",\"reason\":\"V2 course demo seed\"}" "$ADMIN_TOKEN" "seed-a-$index" >/dev/null
  post_json /finance/admin/demo/bonus-credit "{\"accountId\":\"$account_id\",\"categoryId\":\"basketball\",\"amount\":\"300\"}" "$ADMIN_TOKEN" "seed-bonus-$index" >/dev/null
done
post_json /finance/admin/demo/a-credit "{\"accountId\":\"$ORGANIZER_ID\",\"amount\":\"5000\",\"reason\":\"listing and result bonds\"}" "$ADMIN_TOKEN" seed-a-organizer >/dev/null
post_json /finance/admin/demo/bonus-credit "{\"accountId\":\"$ORGANIZER_ID\",\"categoryId\":\"basketball\",\"amount\":\"1000\"}" "$ADMIN_TOKEN" seed-bonus-organizer >/dev/null

CLOSE_AT=$(node -e 'console.log(new Date(Date.now()+2*3600e3).toISOString())')
APP_CLOSE=$(node -e 'console.log(new Date(Date.now()+4*3600e3).toISOString())')
START_AT=$(node -e 'console.log(new Date(Date.now()+5*3600e3).toISOString())')
END_AT=$(node -e 'console.log(new Date(Date.now()+7*3600e3).toISOString())')

echo "Creating the default B_bonus market and activity..."
post_json /finance/markets "{\"marketId\":\"basketball-final\",\"eventId\":\"basketball-final\",\"categoryId\":\"basketball\",\"outcomes\":[{\"id\":\"home\",\"label\":\"主队胜\"},{\"id\":\"draw\",\"label\":\"平局\"},{\"id\":\"away\",\"label\":\"客队胜\"}],\"closeAt\":\"$CLOSE_AT\",\"stakeBucket\":\"BONUS\",\"marketCap\":\"4000\"}" "$ORGANIZER_TOKEN" >/dev/null
post_json /finance/markets/basketball-final/open '{}' "$ORGANIZER_TOKEN" >/dev/null

post_json /activities "{\"id\":\"basketball-final\",\"categoryId\":\"basketball\",\"title\":\"院际篮球决赛\",\"capacity\":3,\"applicationCloseAt\":\"$APP_CLOSE\",\"startsAt\":\"$START_AT\",\"endsAt\":\"$END_AT\"}" "$ORGANIZER_TOKEN" >/dev/null
DRAW_SEED="independent-verifier-seed-eventchain-2026"
post_json /activities/basketball-final/draw-commitment "{\"seed\":\"$DRAW_SEED\"}" "$VERIFIER_TOKEN" >/dev/null

echo "Creating a prefunded venue offer and sample positions/applications..."
post_json /finance/offers "{\"offerId\":\"court-01\",\"categoryId\":\"basketball\",\"offerType\":\"VENUE\",\"title\":\"篮球馆 1 小时\",\"price\":\"50\",\"cancellationBps\":\"15000\",\"guaranteeBudget\":\"500\",\"bonusBudget\":\"300\",\"inventory\":20}" "$ORGANIZER_TOKEN" >/dev/null
OUTCOMES=(home away home draw home)
for index in "${!STUDENT_TOKENS[@]}"; do
  token="${STUDENT_TOKENS[$index]}"
  post_json /finance/markets/basketball-final/positions "{\"outcomeId\":\"${OUTCOMES[$index]}\",\"amount\":\"$((20+index*5))\"}" "$token" "seed-position-$index" >/dev/null
  post_json /activities/basketball-final/applications '{}' "$token" >/dev/null
done

echo "V2 demo seeded."
echo "Accounts: admin01 / organizer01 / operator01 / verifier01 / arb01..03"
echo "Student password: eventchain-student-2026; privileged passwords use eventchain-<role>-2026."
echo "Keep this draw seed for the post-deadline reveal: $DRAW_SEED"
