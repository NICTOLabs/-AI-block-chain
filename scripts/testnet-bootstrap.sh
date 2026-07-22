#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CHAIN_ID="${CHAIN_ID:-tdr-testnet-1}"
SEALOAD_DIR="${SCRIPT_DIR}/seaload"
GENESIS_TIME="${GENESIS_TIME:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
BOOTSTRAP_PEERS="${BOOTSTRAP_PEERS:-}"

mkdir -p "${SEALOAD_DIR}"

cat > "${SEALOAD_DIR}/chain.json" <<EOF
{
  "chain_id": "${CHAIN_ID}",
  "genesis_time": "${GENESIS_TIME}",
  "consensus": "hybrid-pos-poa",
  "initial_supply": 1000000000,
  "max_supply": 10000000000,
  "allocations": [],
  "validators": [],
  "economics": {
    "base_gas_fee": 5,
    "burn_rate_percent": 1,
    "reward_rate_percent": 4
  }
}
EOF

if [ -n "${BOOTSTRAP_PEERS}" ]; then
  echo "Bootstrapping with peers: ${BOOTSTRAP_PEERS}"
fi

echo "Testnet seaload written to ${SEALOAD_DIR}/chain.json"
