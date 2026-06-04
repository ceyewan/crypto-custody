#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

CONTAINER_NAME="${OFFLINE_CONTAINER_NAME:-crypto-custody-offline-server}"
DB_PATH="${OFFLINE_DB_PATH:-$REPO_ROOT/offline-server-handoff/data/crypto-custody.db}"
BACKUP_DIR="${OFFLINE_DB_BACKUP_DIR:-$SCRIPT_DIR/runs/db-backups}"
STAMP="$(date +%Y%m%d-%H%M%S)"

mkdir -p "$BACKUP_DIR"

if [[ -f "$DB_PATH" ]]; then
  cp "$DB_PATH" "$BACKUP_DIR/crypto-custody.$STAMP.db"
  echo "[OK] backed up offline DB to $BACKUP_DIR/crypto-custody.$STAMP.db"
else
  echo "[INFO] offline DB does not exist yet: $DB_PATH"
fi

if docker ps --format '{{.Names}}' | grep -qx "$CONTAINER_NAME"; then
  docker stop "$CONTAINER_NAME" >/dev/null
  echo "[OK] stopped $CONTAINER_NAME"
fi

rm -f "$DB_PATH" "$DB_PATH-shm" "$DB_PATH-wal"
echo "[OK] removed offline DB files under $DB_PATH"

docker start "$CONTAINER_NAME" >/dev/null
echo "[OK] started $CONTAINER_NAME"

for _ in {1..30}; do
  if curl -fsS \
    -H 'Content-Type: application/json' \
    -d '{"identifier":"admin","username":"admin","password":"admin123"}' \
    "http://127.0.0.1:${OFFLINE_WEB_HOST_PORT:-8080}/user/login" >/dev/null 2>&1; then
    echo "[OK] offline server is healthy"
    exit 0
  fi
  sleep 1
done

echo "[WARN] offline server did not accept login within 30s; check docker logs $CONTAINER_NAME" >&2
