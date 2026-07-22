#!/usr/bin/env bash
set -euo pipefail

NODE_ENV="${NODE_ENV:-production}"
SECRETS_MGR="${SECRETS_MGR:-vault}"

if [ "$SECRETS_MGR" = "vault" ]; then
  export TENDER_API_KEY="$(vault kv get -field=api_key secret/tender/node 2>/dev/null || echo '')"
  export TENDER_VALIDATOR_KEY="$(vault kv get -field=validator_key secret/tender/validator 2>/dev/null || echo '')"
  export TENDER_CONSENSUS_KEY="$(vault kv get -field=consensus_key secret/tender/consensus 2>/dev/null || echo '')"
elif [ "$SECRETS_MGR" = "aws" ]; then
  export TENDER_API_KEY="$(aws secretsmanager get-secret-value --secret-id tender/node/api-key --query SecretString --output text 2>/dev/null)"
  export TENDER_VALIDATOR_KEY="$(aws secretsmanager get-secret-value --secret-id tender/validator/key --query SecretString --output text 2>/dev/null)"
  export TENDER_CONSENSUS_KEY="$(aws secretsmanager get-secret-value --secret-id tender/consensus/key --query SecretString --output text 2>/dev/null)"
fi

if [ -z "${TENDER_API_KEY:-}" ] || [ -z "${TENDER_VALIDATOR_KEY:-}" ] || [ -z "${TENDER_CONSENSUS_KEY:-}" ]; then
  echo "Missing required secrets" >&2
  exit 1
fi

exec /usr/local/bin/tender-node \
  --api-port "${TENDER_API_PORT:-8080}" \
  --p2p-port "${TENDER_P2P_PORT:-3030}" \
  --data-dir "${TENDER_DATA_DIR:-/var/lib/tender}" \
  --consensus "${TENDER_CONSENSUS:-pos}" \
  --api-key "$TENDER_API_KEY"
