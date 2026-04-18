#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo "=========================================="
echo "  EventChain — 清理所有资源"
echo "=========================================="

echo "[1/3] 停止后端/前端进程..."
pkill -f "node.*server/src/app.js" 2>/dev/null || true
pkill -f "vite" 2>/dev/null || true

echo "[2/3] 关闭 Fabric 网络..."
cd "$PROJECT_DIR/fabric/network"
./network.sh down 2>/dev/null || true

echo "[3/3] 清理本地数据..."
rm -rf "$PROJECT_DIR/server/wallet"
rm -f "$PROJECT_DIR/server/data/users.db"

echo ""
echo "清理完成。"
