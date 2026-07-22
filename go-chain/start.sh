#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
export GO111MODULE=on
exec go run . --api-port 8080 --p2p-port 3030 --api-key "${TENDER_API_KEY:-change-me-in-production}" "$@"
