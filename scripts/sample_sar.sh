#!/usr/bin/env bash
set -euo pipefail

OUT_FILE="${1:-}"
SYSSTAT_DIR="${SYSSTAT_DIR:-/var/log/sysstat}"
SAR_INTERVAL="${SAR_INTERVAL:-5}"
SAR_COUNT="${SAR_COUNT:-12}"

if [[ -z "$OUT_FILE" ]]; then
	OUT_FILE="${SYSSTAT_DIR}/sa$(date +%d)"
fi

mkdir -p "$(dirname "$OUT_FILE")"

sudo sar -o "$OUT_FILE" "$SAR_INTERVAL" "$SAR_COUNT"
