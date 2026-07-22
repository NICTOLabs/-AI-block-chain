#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CERT_DIR="${SCRIPT_DIR}/../certs"
DAYS=365

mkdir -p "${CERT_DIR}"

echo "Generating self-signed TLS certificates for development..."
openssl req -x509 -nodes -newkey rsa:4096 -keyout "${CERT_DIR}/privkey.pem" -out "${CERT_DIR}/fullchain.pem" -days "${DAYS}" -subj "/C=KE/ST=Nairobi/L=Nairobi/O=Tender Africa/OU=Infrastructure/CN=tdr-mainnet-1.tender.network"

chmod 600 "${CERT_DIR}/privkey.pem"
chmod 644 "${CERT_DIR}/fullchain.pem"

echo "Certificates generated:"
echo "  Private key: ${CERT_DIR}/privkey.pem"
echo "  Certificate: ${CERT_DIR}/fullchain.pem"
echo "Valid for ${DAYS} days"
