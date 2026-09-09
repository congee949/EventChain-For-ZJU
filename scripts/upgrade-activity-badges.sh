#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
NETWORK_DIR="$PROJECT_DIR/fabric/network"

cd "$NETWORK_DIR"
. scripts/envVar.sh
setGlobals platform

CURRENT_JSON=$(peer lifecycle chaincode querycommitted --channelID eventchain --name activity --output json)
CURRENT_SEQUENCE=$(jq -r '.sequence' <<<"$CURRENT_JSON")
CURRENT_VERSION=$(jq -r '.version' <<<"$CURRENT_JSON")

TARGET_SEQUENCE=$((CURRENT_SEQUENCE + 1))
TARGET_VERSION="review-$TARGET_SEQUENCE"
bash scripts/deployCC.sh \
  -ccn activity -ccp ../chaincode/activity -ccl go \
  -ccv "$TARGET_VERSION" -ccs "$TARGET_SEQUENCE" -c eventchain \
  -cce "AND('PlatformMSP.peer','StudentMSP.peer')" \
  -ccco ../chaincode/activity/collections_config.json

setGlobals platform
FINAL_JSON=$(peer lifecycle chaincode querycommitted --channelID eventchain --name activity --output json)
jq -e --arg version "$TARGET_VERSION" --argjson sequence "$TARGET_SEQUENCE" '.version == $version and .sequence == $sequence' <<<"$FINAL_JSON" >/dev/null
jq -e '.validation_parameter == "CjMSDBIKCAISAggAEgIIARoREg8KC1BsYXRmb3JtTVNQEAMaEBIOCgpTdHVkZW50TVNQEAM="' <<<"$FINAL_JSON" >/dev/null
jq -e '[.collections.config[].Payload.StaticCollectionConfig.name] | sort == ["activity_private","badge_private"]' <<<"$FINAL_JSON" >/dev/null
jq -e '.collections.config[] | .Payload.StaticCollectionConfig | select(.name == "badge_private") | .member_only_read == true and .member_only_write == true and .required_peer_count == 1 and .maximum_peer_count == 2' <<<"$FINAL_JSON" >/dev/null
jq -e '.collections.config[] | .Payload.StaticCollectionConfig | select(.name == "badge_private") | .member_orgs_policy.Payload.SignaturePolicy as $member | .endorsement_policy.Type.SignaturePolicy as $endorse | ($member.rule.Type.NOutOf.n == 1) and ($endorse.rule.Type.NOutOf.n == 2) and ([$member.identities[].principal] | sort == ["CgpTdHVkZW50TVNQ","CgtQbGF0Zm9ybU1TUA=="]) and ([$endorse.identities[].principal] | sort == ["CgpTdHVkZW50TVNQEAM=","CgtQbGF0Zm9ybU1TUBAD"])' <<<"$FINAL_JSON" >/dev/null
echo "Verified activity $TARGET_VERSION sequence $TARGET_SEQUENCE, AND Platform+Student endorsement, both private collections, and badge member-only policy."
