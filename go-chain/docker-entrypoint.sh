#!/bin/sh
set -e

API_PORT="${PORT:-${TENDER_API_PORT:-10000}}"
P2P_PORT="${TENDER_P2P_PORT:-3030}"
DATA_DIR="${TENDER_DATA_DIR:-/data}"
CONSENSUS="${TENDER_CONSENSUS:-pos}"

exec /root/tender-node \
  -api-port "$API_PORT" \
  -p2p-port "$P2P_PORT" \
  -data-dir "$DATA_DIR" \
  -consensus "$CONSENSUS" \
  "$@"
