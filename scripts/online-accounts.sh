#!/usr/bin/env bash
set -euo pipefail

API_URL="${API_URL:-http://127.0.0.1:22221}"
LOGIN_USER="${LOGIN_USER:-${API_USERNAME:-admin}}"
PASSWORD="${PASSWORD:-${DEFAULT_ADMIN_PASSWORD:-admin123}}"

usage() {
  cat <<'USAGE'
用法:
  scripts/online-accounts.sh list
  scripts/online-accounts.sh import accounts.json
  scripts/online-accounts.sh import - < accounts.json
  scripts/online-accounts.sh sync

环境变量:
  API_URL=http://127.0.0.1:22221
  LOGIN_USER=admin
  PASSWORD=admin123

accounts.json 支持两种格式:
  [{"address":"0x...","coinType":"ETH"}]
  {"accounts":[{"address":"0x...","coinType":"ETH"}]}
USAGE
}

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "缺少命令: $1" >&2
    exit 1
  fi
}

pretty_json() {
  python3 -m json.tool
}

http_json() {
  local tmp_file
  tmp_file="$(mktemp)"
  local status
  if ! status="$(curl -sS -w '%{http_code}' -o "$tmp_file" "$@")"; then
    rm -f "$tmp_file"
    return 1
  fi
  cat "$tmp_file"
  rm -f "$tmp_file"
  if (( status >= 400 )); then
    return 1
  fi
}

login() {
  local payload
  payload="$(
    python3 - "$LOGIN_USER" "$PASSWORD" <<'PY'
import json
import sys

username, password = sys.argv[1], sys.argv[2]
print(json.dumps({"identifier": username, "username": username, "password": password}))
PY
  )"
  local response
  if ! response="$(
    http_json \
      -H 'Content-Type: application/json' \
      -d "$payload" \
      "$API_URL/api/login"
  )"; then
    echo "$response" | pretty_json >&2 || echo "$response" >&2
    exit 1
  fi
  python3 -c 'import json,sys; print(json.load(sys.stdin)["data"]["token"])' <<<"$response"
}

wrap_accounts_payload() {
  local input_file="$1"
  local output_file="$2"
  python3 - "$input_file" "$output_file" <<'PY'
import json
import sys

input_file, output_file = sys.argv[1], sys.argv[2]
with open(input_file, "r", encoding="utf-8") as f:
    data = json.load(f)
if isinstance(data, list):
    payload = {"accounts": data}
elif isinstance(data, dict) and isinstance(data.get("accounts"), list):
    payload = data
else:
    raise SystemExit("JSON 需要是数组，或包含 accounts 数组")
with open(output_file, "w", encoding="utf-8") as f:
    json.dump(payload, f, ensure_ascii=False)
PY
}

main() {
  need curl
  need python3

  local command="${1:-}"
  if [[ -z "$command" || "$command" == "-h" || "$command" == "--help" ]]; then
    usage
    exit 0
  fi
  shift || true

  local token
  token="$(login)"

  case "$command" in
    list)
      http_json -H "Authorization: $token" "$API_URL/api/accounts?page=1&pageSize=100" | pretty_json
      ;;
    import)
      local input="${1:-}"
      if [[ -z "$input" ]]; then
        echo "缺少 JSON 文件路径，或用 - 从 stdin 读取" >&2
        exit 2
      fi
      local input_file="$input"
      local stdin_tmp=""
      if [[ "$input" == "-" ]]; then
        stdin_tmp="$(mktemp)"
        cat >"$stdin_tmp"
        input_file="$stdin_tmp"
      fi
      local payload_tmp
      payload_tmp="$(mktemp)"
      trap "rm -f '$payload_tmp' '$stdin_tmp'" EXIT
      wrap_accounts_payload "$input_file" "$payload_tmp"
      http_json \
        -H "Authorization: $token" \
        -H 'Content-Type: application/json' \
        --data-binary "@$payload_tmp" \
        "$API_URL/api/accounts/import" | pretty_json
      ;;
    sync)
      http_json \
        -H "Authorization: $token" \
        -X POST \
        "$API_URL/api/accounts/sync-balances" | pretty_json
      ;;
    *)
      echo "未知命令: $command" >&2
      usage >&2
      exit 2
      ;;
  esac
}

main "$@"
