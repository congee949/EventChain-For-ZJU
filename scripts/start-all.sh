#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

if [ -x /opt/homebrew/opt/node@22/bin/node ]; then
  export PATH="/opt/homebrew/opt/node@22/bin:$PATH"
fi

echo "=========================================="
echo "  EventChain — 一键启动"
echo "=========================================="

# Step 1: Start Fabric network
echo ""
echo "[1/5] 启动 Fabric 网络..."
cd "$PROJECT_DIR/fabric/network"
./network.sh up
./network.sh createChannel

# Step 2: Deploy chaincodes
echo ""
echo "[2/5] 部署链码..."
./network.sh deployCCs

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
echo "等待后端就绪..."
for i in $(seq 1 30); do
  curl -s http://localhost:3000/api/health > /dev/null 2>&1 && break
  sleep 1
done

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

trap "kill $SERVER_PID $CLIENT_PID 2>/dev/null; exit 0" INT TERM
wait
