#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

API_BASE="http://localhost:3000/api/v1"

echo "=========================================="
echo "  EventChain — 灌入演示数据"
echo "=========================================="

echo "等待后端启动..."
for i in $(seq 1 30); do
  if curl -s "$API_BASE/events" > /dev/null 2>&1; then
    echo "后端已就绪。"
    break
  fi
  sleep 1
done

echo ""
echo "[1/4] 注册用户..."

curl -s -X POST "$API_BASE/auth/register" \
  -H "Content-Type: application/json" \
  -d '{"studentID":"admin01","password":"admin123","name":"系统管理员","role":"admin"}' | jq .

curl -s -X POST "$API_BASE/auth/register" \
  -H "Content-Type: application/json" \
  -d '{"studentID":"organizer01","password":"org123","name":"体育社团","role":"organizer"}' | jq .

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

echo ""
echo "[2/4] 创建赛事..."

ORG_TOKEN=$(curl -s -X POST "$API_BASE/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"studentID":"organizer01","password":"org123"}' | jq -r '.data.token')

curl -s -X POST "$API_BASE/events" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ORG_TOKEN" \
  -d '{"eventID":"evt001","title":"院际篮球决赛","type":"basketball","teams":["教院","丹青"],"ticketTotal":200,"predictionOptions":["教院赢","丹青赢"]}' | jq .

curl -s -X POST "$API_BASE/events" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ORG_TOKEN" \
  -d '{"eventID":"evt002","title":"校运会足球半决赛","type":"football","teams":["竺院","蓝田"],"ticketTotal":300,"predictionOptions":["竺院赢","蓝田赢"]}' | jq .

curl -s -X POST "$API_BASE/events" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ORG_TOKEN" \
  -d '{"eventID":"evt003","title":"英雄联盟校赛决赛","type":"esports","teams":["CS战队","EE战队"],"ticketTotal":150,"predictionOptions":["CS战队赢","EE战队赢"]}' | jq .

curl -s -X POST "$API_BASE/events" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ORG_TOKEN" \
  -d '{"eventID":"evt004","title":"羽毛球团体赛","type":"badminton","teams":["求是","云峰"],"ticketTotal":100,"predictionOptions":["求是赢","云峰赢"]}' | jq .

echo ""
echo "开放预测市场..."
for EVT_ID in evt001 evt002 evt003; do
  curl -s -X PUT "$API_BASE/events/$EVT_ID/status" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $ORG_TOKEN" \
    -d '{"status":"PREDICTION_OPEN"}' | jq .
done

echo ""
echo "[3/4] 模拟下注..."

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
    -d "{\"studentID\":\"$SID\",\"password\":\"$PWD\"}" | jq -r '.data.token')

  curl -s -X POST "$API_BASE/predictions/bet" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d "{\"eventID\":\"$EVT\",\"option\":\"$OPT\",\"amount\":$AMT}" | jq .

  sleep 0.5
done

echo ""
echo "[4/4] 结算一场赛事（篮球赛，教院赢）..."

curl -s -X PUT "$API_BASE/events/evt001/status" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ORG_TOKEN" \
  -d '{"status":"TICKET_OPEN"}' | jq .

for SID in 3220100001 3220100002 3220100003 3220100004 3220100005; do
  TOKEN=$(curl -s -X POST "$API_BASE/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"studentID\":\"$SID\",\"password\":\"student123\"}" | jq -r '.data.token')

  curl -s -X POST "$API_BASE/tickets/apply" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d '{"eventID":"evt001"}' | jq .
done

curl -s -X POST "$API_BASE/tickets/lottery/evt001" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ORG_TOKEN" \
  -d '{"ticketCount":3}' | jq .

curl -s -X PUT "$API_BASE/events/evt001/status" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ORG_TOKEN" \
  -d '{"status":"ONGOING"}' | jq .

curl -s -X PUT "$API_BASE/events/evt001/result" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ORG_TOKEN" \
  -d '{"outcome":"教院赢"}' | jq .

echo ""
echo "=========================================="
echo "  演示数据灌入完成！"
echo "  - 8 名学生 + 1 主办方 + 1 管理员"
echo "  - 4 场赛事（1 场已结算）"
echo "  - 11 笔预测下注"
echo "  - 篮球赛已结算，教院赢"
echo "=========================================="
