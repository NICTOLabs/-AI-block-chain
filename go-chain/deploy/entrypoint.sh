#!/bin/bash
set -euo pipefail

: "${TENDER_DATA_DIR:=/root/data}"
: "${TENDER_API_PORT:=8080}"
: "${TENDER_P2P_PORT:=3030}"
: "${TENDER_CONSENSUS:=pos}"
: "${TENDER_CHAIN_ID:=tdr-mainnet-1}"
: "${TENDER_ENABLE_AUTH:=true}"
: "${TENDER_STRICT_P2P:=true}"

mkdir -p "${TENDER_DATA_DIR}" /root/logs

if [ ! -f "${TENDER_DATA_DIR}/chain.json" ]; then
  echo "Initializing fresh chain state..."
fi

exec /root/tender-node \
  --api-port "${TENDER_API_PORT}" \
  --p2p-port "${TENDER_P2P_PORT}" \
  --data-dir "${TENDER_DATA_DIR}" \
  --consensus "${TENDER_CONSENSUS}" \
  --chain-id "${TENDER_CHAIN_ID}" \
  --enable-auth \
  --strict-p2p
