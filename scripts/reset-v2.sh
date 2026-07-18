#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

if [ "${1:-}" != "--yes" ]; then
  echo "This removes the local EventChain ledger, CA state, generated identities, wallet, and SQLite demo users."
  echo "Re-run with: $0 --yes"
  exit 2
fi

echo "Resetting the local V1 demo state..."
cd "$PROJECT_DIR/fabric/network"
./network.sh down

# These exact application-owned directories are regenerated during bootstrap.
for target in "$PROJECT_DIR/server/wallet" "$PROJECT_DIR/server/data"; do
  if [ -e "$target" ]; then
    rm -rf -- "$target"
  fi
done

echo "Starting a clean V2 network..."
./network.sh up
./network.sh createChannel
./network.sh deployCCs
echo "V2 network reset complete. Start the API, bootstrap privileged identities, initialize finance, then seed demo data."
