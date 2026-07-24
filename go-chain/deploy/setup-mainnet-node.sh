#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
NODE_NAME="${NODE_NAME:-tender-node-01}"
CHAIN_ID="${CHAIN_ID:-tdr-mainnet-1}"
DATA_DIR="${DATA_DIR:-/opt/tender/data}"
API_PORT="${API_PORT:-8080}"
P2P_PORT="${P2P_PORT:-3030}"
METRICS_PORT="${METRICS_PORT:-9090}"
CONSENSUS="${CONSENSUS:-pos}"
BOOTSTRAP_PEERS="${BOOTSTRAP_PEERS:-}"

if [[ $EUID -ne 0 ]]; then
   echo "This script must be run as root" 
   exit 1
fi

echo "=== TDR Mainnet Node Setup ==="
echo "Node: ${NODE_NAME}"
echo "Chain: ${CHAIN_ID}"
echo "Data: ${DATA_DIR}"

useradd -r -s /bin/false tender || true
mkdir -p "${DATA_DIR}" /var/log/tender /opt/tender
chown -R tender:tender "${DATA_DIR}" /var/log/tender /opt/tender

if ! command -v docker &> /dev/null; then
   echo "Installing Docker..."
   apt-get update -y && apt-get install -y docker.io docker-compose git
fi

if [ -f "${REPO_DIR}/genesis_mainnet.json" ]; then
   cp "${REPO_DIR}/genesis_mainnet.json" "${DATA_DIR}/genesis.json"
   chown tender:tender "${DATA_DIR}/genesis.json"
fi

cat > /etc/tender/node.env <<EOF
TENDER_API_KEY=${TENDER_API_KEY:-}
TENDER_ENABLE_AUTH=true
TENDER_STRICT_P2P=true
TENDER_DATA_DIR=${DATA_DIR}
TENDER_CONSENSUS=${CONSENSUS}
TENDER_CHAIN_ID=${CHAIN_ID}
TENDER_API_PORT=${API_PORT}
TENDER_P2P_PORT=${P2P_PORT}
TENDER_METRICS_PORT=${METRICS_PORT}
TENDER_BOOTSTRAP_PEERS=${BOOTSTRAP_PEERS}
EOF

cp "${REPO_DIR}/go-chain/deploy/tender-node.service" /etc/systemd/system/
systemctl daemon-reload
systemctl enable tender-node
systemctl start tender-node

echo "Setup complete. Check status: systemctl status tender-node"
echo "Logs: journalctl -u tender-node -f"
