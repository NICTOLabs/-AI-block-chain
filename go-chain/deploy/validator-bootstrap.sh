#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
CHAIN_ID="${CHAIN_ID:-tdr-mainnet-1}"
OUTPUT_DIR="${OUTPUT_DIR:-/tmp/validator-package}"
NODE_INDEX="${NODE_INDEX:-1}"
COUNTRY="${COUNTRY:-KE}"
CITY="${CITY:-Nairobi}"
API_PORT="${API_PORT:-8080}"
P2P_PORT="${P2P_PORT:-3030}"

mkdir -p "${OUTPUT_DIR}/validator-${NODE_INDEX}"

echo "=== Validator Bootstrap Package ==="
echo "Node: ${NODE_INDEX}"
echo "Country: ${COUNTRY}"
echo "Output: ${OUTPUT_DIR}/validator-${NODE_INDEX}"

go run "${REPO_ROOT}/go-chain/tools/genesis-tool/main.go" \
  --output "${OUTPUT_DIR}/validator-${NODE_INDEX}/identity.json" \
  --chain-id "${CHAIN_ID}" \
  --validator-count 1

cat > "${OUTPUT_DIR}/validator-${NODE_INDEX}/tender-validator.service" <<EOF
[Unit]
Description=TDR Validator Node ${NODE_INDEX}
After=network.target

[Service]
Type=simple
User=tender
WorkingDirectory=/opt/tender
ExecStart=/opt/tender/tender-node --chain-id ${CHAIN_ID} --data-dir /opt/tender/data --consensus pos --api-port ${API_PORT} --p2p-port ${P2P_PORT} --enable-auth --strict-p2p
Restart=always
RestartSec=5
Environment="TENDER_API_KEY=${TENDER_API_KEY:-}"
Environment="TENDER_ENABLE_AUTH=true"
Environment="TENDER_STRICT_P2P=true"

[Install]
WantedBy=multi-user.target
EOF

cat > "${OUTPUT_DIR}/validator-${NODE_INDEX}/setup.sh" <<EOF
#!/bin/bash
set -euo pipefail
mkdir -p /opt/tender/data
cp identity.json /opt/tender/data/
systemctl daemon-reload
systemctl enable tender-validator
systemctl start tender-validator
echo "Validator ${NODE_INDEX} started"
EOF
chmod +x "${OUTPUT_DIR}/validator-${NODE_INDEX}/setup.sh"

echo "Validator package ready at: ${OUTPUT_DIR}/validator-${NODE_INDEX}/"
ls -la "${OUTPUT_DIR}/validator-${NODE_INDEX}/"
