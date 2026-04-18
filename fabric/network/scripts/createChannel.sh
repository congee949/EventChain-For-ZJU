#!/bin/bash
# fabric/network/scripts/createChannel.sh
# Create the 'eventchain' channel and join all peers
#
# Uses osnadmin CLI for channel participation API (Fabric 2.5 pattern — no system channel)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NETWORK_DIR="$(dirname "$SCRIPT_DIR")"

CHANNEL_NAME="${1:-eventchain}"
DELAY="${2:-3}"
MAX_RETRY="${3:-5}"

# Source environment variables
. "${SCRIPT_DIR}/envVar.sh"

export FABRIC_CFG_PATH="${NETWORK_DIR}/configtx"

# ============================================================
# Create channel genesis block using configtxgen
# ============================================================
createChannelGenesisBlock() {
  echo ""
  echo "Generating channel genesis block '${CHANNEL_NAME}.block'..."

  configtxgen \
    -profile EventChainGenesis \
    -outputBlock "${NETWORK_DIR}/channel-artifacts/${CHANNEL_NAME}.block" \
    -channelID "$CHANNEL_NAME"

  echo "Genesis block created: ${NETWORK_DIR}/channel-artifacts/${CHANNEL_NAME}.block"
}

# ============================================================
# Join orderer to channel via osnadmin
# ============================================================
joinOrdererToChannel() {
  echo ""
  echo "Joining orderer to channel '${CHANNEL_NAME}'..."

  # Retry — orderer admin endpoint may take a few seconds after container start
  local attempts=0
  while true; do
    if osnadmin channel join \
      --channelID "$CHANNEL_NAME" \
      --config-block "${NETWORK_DIR}/channel-artifacts/${CHANNEL_NAME}.block" \
      -o localhost:7053 \
      --ca-file "$ORDERER_CA" \
      --client-cert "$ORDERER_ADMIN_TLS_SIGN_CERT" \
      --client-key "$ORDERER_ADMIN_TLS_PRIVATE_KEY"; then
      break
    fi
    attempts=$((attempts + 1))
    if [ "$attempts" -ge 10 ]; then
      echo "ERROR: orderer channel join failed after 10 attempts"
      exit 1
    fi
    echo "Orderer admin endpoint not ready, retrying in ${DELAY}s... (${attempts}/10)"
    sleep "$DELAY"
  done

  echo "Orderer joined channel '${CHANNEL_NAME}'"

  # Verify
  osnadmin channel list \
    -o localhost:7053 \
    --ca-file "$ORDERER_CA" \
    --client-cert "$ORDERER_ADMIN_TLS_SIGN_CERT" \
    --client-key "$ORDERER_ADMIN_TLS_PRIVATE_KEY"
}

# ============================================================
# Join a peer to the channel
# ============================================================
joinPeerToChannel() {
  local org="$1"
  export FABRIC_CFG_PATH="$(cd "${NETWORK_DIR}/../../config" && pwd)"
  setGlobals "$org"

  local counter=0
  while true; do
    peer channel join \
      -b "${NETWORK_DIR}/channel-artifacts/${CHANNEL_NAME}.block" && break

    counter=$((counter + 1))
    if [ "$counter" -ge "$MAX_RETRY" ]; then
      echo "ERROR: peer channel join failed for ${org} after ${MAX_RETRY} attempts"
      exit 1
    fi
    echo "Retrying in ${DELAY}s... (attempt ${counter}/${MAX_RETRY})"
    sleep "$DELAY"
  done

  echo "Peer for ${org} joined channel '${CHANNEL_NAME}'"
}

# ============================================================
# Set anchor peer for an org
#
# NOTE: This function is currently UNUSED. The anchor peer step was removed
# from the Main section below because configtx.yaml already declares anchor
# peers in each org block, and the genesis profile pulls them in. The
# function is preserved here so you can re-enable it if you ever need to
# add anchor peers AFTER channel creation (e.g., adding a new org). To
# re-enable, uncomment the `setAnchorPeer …` calls in the Main section.
# ============================================================
setAnchorPeer() {
  local org="$1"
  export FABRIC_CFG_PATH="$(cd "${NETWORK_DIR}/../../config" && pwd)"
  setGlobals "$org"

  echo "Setting anchor peer for ${org}..."

  # Fetch current channel config (with retry — orderer may still be committing)
  local fetch_attempts=0
  while true; do
    if peer channel fetch config "${NETWORK_DIR}/channel-artifacts/config_block.pb" \
      -o localhost:7050 \
      --ordererTLSHostnameOverride orderer.eventchain.com \
      -c "$CHANNEL_NAME" \
      --tls \
      --cafile "$ORDERER_CA"; then
      break
    fi
    fetch_attempts=$((fetch_attempts + 1))
    if [ "$fetch_attempts" -ge 10 ]; then
      echo "ERROR: failed to fetch channel config after 10 attempts"
      exit 1
    fi
    echo "Channel config fetch failed, retrying in 3s... (${fetch_attempts}/10)"
    sleep 3
  done

  cd "${NETWORK_DIR}/channel-artifacts"

  # Decode to JSON
  configtxlator proto_decode \
    --input config_block.pb \
    --type common.Block \
    --output config_block.json

  # Extract config
  jq '.data.data[0].payload.data.config' config_block.json > config.json

  # Determine anchor peer host and port
  local anchor_host anchor_port
  case "$org" in
    platform)
      anchor_host="peer0.platform.eventchain.com"
      anchor_port=7051
      ;;
    organizer)
      anchor_host="peer0.organizer.eventchain.com"
      anchor_port=9051
      ;;
    student)
      anchor_host="peer0.student.eventchain.com"
      anchor_port=11051
      ;;
  esac

  local msp_id
  msp_id="${CORE_PEER_LOCALMSPID}"

  # Modify config to add anchor peer
  jq --arg MSP "$msp_id" --arg HOST "$anchor_host" --argjson PORT "$anchor_port" \
    '.channel_group.groups.Application.groups[$MSP].values += {
      "AnchorPeers": {
        "mod_policy": "Admins",
        "value": {
          "anchor_peers": [{"host": $HOST, "port": $PORT}]
        },
        "version": "0"
      }
    }' config.json > modified_config.json

  # Encode original and modified configs
  configtxlator proto_encode \
    --input config.json \
    --type common.Config \
    --output config.pb

  configtxlator proto_encode \
    --input modified_config.json \
    --type common.Config \
    --output modified_config.pb

  # Compute update delta
  configtxlator compute_update \
    --channel_id "$CHANNEL_NAME" \
    --original config.pb \
    --updated modified_config.pb \
    --output config_update.pb

  configtxlator proto_decode \
    --input config_update.pb \
    --type common.ConfigUpdate \
    --output config_update.json

  # Wrap in envelope
  echo '{"payload":{"header":{"channel_header":{"channel_id":"'"$CHANNEL_NAME"'","type":2}},"data":{"config_update":'$(cat config_update.json)'}}}' | \
    jq . > config_update_in_envelope.json

  configtxlator proto_encode \
    --input config_update_in_envelope.json \
    --type common.Envelope \
    --output config_update_in_envelope.pb

  # Submit config update
  peer channel update \
    -f config_update_in_envelope.pb \
    -c "$CHANNEL_NAME" \
    -o localhost:7050 \
    --ordererTLSHostnameOverride orderer.eventchain.com \
    --tls \
    --cafile "$ORDERER_CA"

  echo "Anchor peer set for ${org}"

  # Clean up temp files
  rm -f config_block.pb config_block.json config.json modified_config.json \
        config.pb modified_config.pb config_update.pb config_update.json \
        config_update_in_envelope.json config_update_in_envelope.pb

  cd "$NETWORK_DIR"
}

# ============================================================
# Main
# ============================================================
echo ""
echo "============================================"
echo " EventChain — Channel Setup"
echo " Channel: ${CHANNEL_NAME}"
echo "============================================"

mkdir -p "${NETWORK_DIR}/channel-artifacts"

# Step 1: Generate genesis block
createChannelGenesisBlock

# Step 2: Join orderer
joinOrdererToChannel

# Step 3: Join all peers
echo ""
echo "Joining all peers to channel..."
joinPeerToChannel platform
joinPeerToChannel organizer
joinPeerToChannel student

# Step 4: Set anchor peers for each org
# NOTE: Skipped — AnchorPeers are already baked into the genesis block via
# configtx.yaml (each org has an `AnchorPeers:` list and the Genesis profile
# includes all orgs under Application). In Fabric 2.x, the separate
# "peer channel update" step for anchor peers is only needed when anchor
# peers are added AFTER channel creation. For a fresh network boot we can
# skip it entirely and the config update BAD_REQUEST bug goes away.
echo ""
echo "Anchor peers already in genesis block (configtx.yaml) — skipping update"

echo ""
echo "============================================"
echo " Channel '${CHANNEL_NAME}' ready"
echo " All 3 peers joined"
echo "============================================"
