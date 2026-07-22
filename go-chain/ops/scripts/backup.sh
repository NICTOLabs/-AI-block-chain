#!/usr/bin/env bash
set -euo pipefail
NODE_DATA_DIR="${TENDER_DATA_DIR:-/opt/tender/data}"
BACKUP_DIR="${TENDER_BACKUP_DIR:-/opt/tender/backups}"
TS="$(date +%Y%m%d-%H%M%S)"
mkdir -p "$BACKUP_DIR"
mkdir -p "$NODE_DATA_DIR"

tar -czf "$BACKUP_DIR/tender-$TS.tgz" -C "$(dirname "$NODE_DATA_DIR")" "$(basename "$NODE_DATA_DIR")"
find "$BACKUP_DIR" -name 'tender-*.tgz' -type f -mtime +7 -delete

echo "Backup completed: $BACKUP_DIR/tender-$TS.tgz"
