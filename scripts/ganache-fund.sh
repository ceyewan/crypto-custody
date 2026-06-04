#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RPC_URL="${ETH_RPC_URL:-http://127.0.0.1:8545}"
TO_ADDRESS=""
AMOUNT="${GANACHE_FUND_AMOUNT:-100}"
JSON_OUTPUT="false"

usage() {
  cat <<'USAGE'
用法:
  scripts/ganache-fund.sh 0x收款地址
  scripts/ganache-fund.sh --to 0x收款地址 [--amount 100] [--rpc URL] [--json]

示例:
  scripts/ganache-fund.sh 0xabc...
  scripts/ganache-fund.sh --to 0xabc... --amount 50
  ETH_RPC_URL=http://127.0.0.1:8545 scripts/ganache-fund.sh 0xabc...

说明:
  默认从 Ganache/RPC 已解锁账户中自动选择一个余额足够的账户，给目标地址转 100 ETH。
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --rpc)
      RPC_URL="${2:?缺少 --rpc 参数}"
      shift 2
      ;;
    --to)
      TO_ADDRESS="${2:?缺少 --to 参数}"
      shift 2
      ;;
    --amount)
      AMOUNT="${2:?缺少 --amount 参数}"
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
    --*)
      echo "未知参数: $1" >&2
      usage >&2
      exit 2
      ;;
    *)
      if [[ -n "$TO_ADDRESS" ]]; then
        echo "只能提供一个收款地址" >&2
        usage >&2
        exit 2
      fi
      TO_ADDRESS="$1"
      shift
      ;;
  esac
done

if [[ -z "$TO_ADDRESS" ]]; then
  echo "缺少收款地址" >&2
  usage >&2
  exit 2
fi

args=(--fund --rpc "$RPC_URL" --to "$TO_ADDRESS" --amount "$AMOUNT")
if [[ "$JSON_OUTPUT" == "true" ]]; then
  args+=(--json)
fi

cd "$ROOT_DIR/online-server"
go run ./cmd/local-eth-tool "${args[@]}"
