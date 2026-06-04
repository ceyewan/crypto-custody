#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RPC_URL="${ETH_RPC_URL:-http://127.0.0.1:8545}"
ADDRESSES="${GANACHE_ADDRESSES:-}"
JSON_OUTPUT="false"

usage() {
  cat <<'USAGE'
用法:
  scripts/ganache-inspect.sh [--rpc URL] [--addresses addr1,addr2] [--json]

示例:
  scripts/ganache-inspect.sh
  ETH_RPC_URL=http://127.0.0.1:8545 scripts/ganache-inspect.sh --json
  scripts/ganache-inspect.sh --addresses 0xabc...,0xdef...
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --rpc)
      RPC_URL="${2:?缺少 --rpc 参数}"
      shift 2
      ;;
    --addresses)
      ADDRESSES="${2:?缺少 --addresses 参数}"
      shift 2
      ;;
    --json)
      JSON_OUTPUT="true"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "未知参数: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

args=(--inspect --rpc "$RPC_URL")
if [[ -n "$ADDRESSES" ]]; then
  args+=(--addresses "$ADDRESSES")
fi
if [[ "$JSON_OUTPUT" == "true" ]]; then
  args+=(--json)
fi

cd "$ROOT_DIR/online-server"
go run ./cmd/local-eth-tool "${args[@]}"
