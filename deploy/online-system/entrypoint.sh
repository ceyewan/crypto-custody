#!/usr/bin/env sh

set -eu

export WEB_PORT="${WEB_PORT:-8088}"
export BACKEND_BIND_ADDR="${BACKEND_BIND_ADDR:-127.0.0.1}"
export BACKEND_PORT="${BACKEND_PORT:-22221}"
export DATABASE_PATH="${DATABASE_PATH:-/app/database/crypto-custody.db}"

if [ -z "${JWT_SECRET_KEY:-}" ]; then
  echo "Error: JWT_SECRET_KEY is required." >&2
  exit 1
fi

mkdir -p /app/database /app/logs /app/backups /run/nginx

envsubst '${WEB_PORT} ${BACKEND_PORT}' \
  < /etc/nginx/http.d/default.conf.template \
  > /etc/nginx/http.d/default.conf

/app/online-server &
backend_pid="$!"

for _ in $(seq 1 30); do
  if ! kill -0 "$backend_pid" 2>/dev/null; then
    wait "$backend_pid"
    exit $?
  fi
  if curl -s -o /dev/null "http://127.0.0.1:${BACKEND_PORT}/api/check-auth"; then
    break
  fi
  sleep 1
done

term_handler() {
  kill "$backend_pid" 2>/dev/null || true
  nginx -s quit 2>/dev/null || true
  wait "$backend_pid" 2>/dev/null || true
}
trap term_handler INT TERM

nginx -g 'daemon off;' &
nginx_pid="$!"

wait "$nginx_pid"
term_handler
