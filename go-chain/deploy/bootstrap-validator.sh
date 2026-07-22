#!/usr/bin/env bash
set -euo pipefail

export DEBIAN_FRONTEND=noninteractive
CHAIN_ID="${CHAIN_ID:-tdr-mainnet-1}"
GENESIS_FILE="${GENESIS_FILE:-genesis_mainnet.json}"
DATA_DIR="${DATA_DIR:-/root/data}"
P2P_PORT="${P2P_PORT:-3030}"
API_PORT="${API_PORT:-8080}"
METRICS_PORT="${METRICS_PORT:-9090}"
CONSENSUS="${CONSENSUS:-pos}"
STRICT_P2P="${STRICT_P2P:-true}"
ENABLE_AUTH="${ENABLE_AUTH:-true}"
BOOTSTRAP_PEERS="${BOOTSTRAP_PEERS:-}"

if [[ $EUID -ne 0 ]]; then
   echo "Run as root" 
   exit 1
fi

systemctl stop tender-node || true
useradd -r -s /bin/false tender || true
mkdir -p "${DATA_DIR}" /var/log/tender
chown -R tender:tender "${DATA_DIR}" /var/log/tender

if ! command -v docker &> /dev/null; then
   apt-get update -y && apt-get install -y docker.io docker-compose git curl jq
fi

if [ ! -f "/root/${GENESIS_FILE}" ]; then
   echo "Genesis file not found at /root/${GENESIS_FILE}"
   exit 1
fi

cp "/root/${GENESIS_FILE}" "${DATA_DIR}/genesis.json"

mkdir -p /etc/tender
cat > /etc/tender/env <<EOF
TENDER_CHAIN_ID=${CHAIN_ID}
TENDER_DATA_DIR=${DATA_DIR}
TENDER_P2P_PORT=${P2P_PORT}
TENDER_API_PORT=${API_PORT}
TENDER_METRICS_PORT=${METRICS_PORT}
TENDER_CONSENSUS=${CONSENSUS}
TENDER_STRICT_P2P=${STRICT_P2P}
TENDER_ENABLE_AUTH=${ENABLE_AUTH}
TENDER_BOOTSTRAP_PEERS=${BOOTSTRAP_PEERS}
EOF

docker rm -f tender-node || true
docker run -d \
  --name tender-node \
  --restart unless-stopped \
  -p "${API_PORT}:8080" \
  -p "${P2P_PORT}:3030" \
  -p "${METRICS_PORT}:9090" \
  -v "${DATA_DIR}:/root/data" \
  -v "/etc/tender/env:/root/.env" \
  --log-driver json-file \
  --log-opt max-size=100m \
  tender-node:latest \
  --chain-id "${CHAIN_ID}" \
  --data-dir "/root/data" \
  --consensus "${CONSENSUS}" \
  --api-port 8080 \
  --p2p-port 3030 \
  $([ "${STRICT_P2P}" = "true" ] && echo "--strict-p2p") \
  $([ "${ENABLE_AUTH}" = "true" ] && echo "--enable-auth")

echo "Validator bootstrap complete"
echo "Node running on API:${API_PORT} P2P:${P2P_PORT}"
echo "Check logs: docker logs -f tender-node"
