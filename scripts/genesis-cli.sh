#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

usage() {
  echo "Usage: $0 [generate|sign|verify] [options]"
  echo ""
  echo "generate: create unsigned genesis"
  echo "  --output PATH       default: genesis_mainnet.json"
  echo "  --chain-id ID       default: tdr-mainnet-1"
  echo "  --validator-count N default: 7"
  echo ""
  echo "sign: add your signature to genesis"
  echo "  --genesis PATH      default: genesis_mainnet.json"
  echo "  --private-key HEX   required"
  echo "  --address ADDR      required"
  echo "  --output PATH       default: same as --genesis"
  echo ""
  echo "verify: verify signatures and show genesis hash"
  echo "  --genesis PATH      default: genesis_mainnet.json"
  exit 1
}

action="${1:-}"
shift || true

case "$action" in
  generate)
    cd "$REPO_ROOT/go-chain/tools/genesis-tool"
    go run main.go --action generate "$@"
    ;;
  sign)
    cd "$REPO_ROOT/go-chain/tools/genesis-tool"
    go run main.go --action sign "$@"
    ;;
  verify)
    cd "$REPO_ROOT/go-chain/tools/genesis-tool"
    go run main.go --action verify "$@"
    ;;
  *)
    usage
    ;;
esac
