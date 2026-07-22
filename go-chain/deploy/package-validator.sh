#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CHAIN_ID="${CHAIN_ID:-tdr-mainnet-1}"
OUTPUT_DIR="${OUTPUT_DIR:-/root}"
NODE_INDEX="${NODE_INDEX:-1}"
COUNTRY="${COUNTRY:-KE}"
CITY="${CITY:-Nairobi}"
CONTACT="${CONTACT:-validator@tender.network}"
P2P_PORT="${P2P_PORT:-3030}"
API_PORT="${API_PORT:-8080}"

mkdir -p "${OUTPUT_DIR}/validator-${NODE_INDEX}"

echo "Generating validator identity ${NODE_INDEX}..."
go run "${SCRIPT_DIR}/../tools/genesis-tool/main.go" \
  --output "${OUTPUT_DIR}/validator-${NODE_INDEX}/identity.json" \
  --chain-id "${CHAIN_ID}" \
  --validator \
  --country "${COUNTRY}" \
  --city "${CITY}" \
  --contact "${CONTACT}" \
  --ports "p2p=${P2P_PORT},api=${API_PORT}"

echo "Generating systemd unit..."
cat > "${OUTPUT_DIR}/validator-${NODE_INDEX}/tender-validator.service" <<EOF
[Unit]
Description=TDR Validator Node ${NODE_INDEX}
After=network.target

[Service]
Type=simple
User=tender
WorkingDirectory=/opt/tender
ExecStart=/opt/tender/tender-node --chain-id ${CHAIN_ID} --data-dir /opt/tender/data --consensus ${CONSENSUS:-pos} --api-port ${API_PORT} --p2p-port ${P2P_PORT}
Restart=always
RestartSec=5
Environment="TENDER_ENABLE_AUTH=true"
Environment="TENDER_STRICT_P2P=true"
Environment="TENDER_API_KEY=${TENDER_API_KEY:-change-me-in-production}"

[Install]
WantedBy=multi-user.target
EOF

echo "Generating bootstrap script..."
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

echo "Validator package written to ${OUTPUT_DIR}/validator-${NODE_INDEX}/"
ls -la "${OUTPUT_DIR}/validator-${NODE_INDEX}/"
