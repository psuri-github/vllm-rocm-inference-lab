#!/usr/bin/env bash
set -u

START_GATE_FILE="${START_GATE_FILE:?START_GATE_FILE is required}"
REQUEST_ID="${REQUEST_ID:?REQUEST_ID is required}"

BASE_URL="${BASE_URL:-http://localhost:8000}"
MODEL="${MODEL:-Qwen/Qwen2.5-0.5B-Instruct}"
MAX_TOKENS="${MAX_TOKENS:-100}"
PROMPT_ID="${PROMPT_ID:-manual}"
PROMPT="${PROMPT:-Explain GPU inference in one short paragraph.}"
RESULTS_DIR="${RESULTS_DIR:-benchmark-results}"
RESULTS_FILE="${RESULTS_FILE:?RESULTS_FILE is required}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SINGLE_REQUEST_SCRIPT="$SCRIPT_DIR/06-single-request-benchmark.sh"

echo "Launcher for request $REQUEST_ID started."
echo "Waiting for start gate: $START_GATE_FILE"

if [ ! -x "$SINGLE_REQUEST_SCRIPT" ]; then
  echo "ERROR: single-request benchmark script is missing or not executable:"
  echo "$SINGLE_REQUEST_SCRIPT"
  exit 1
fi

while [ ! -f "$START_GATE_FILE" ]; do
  sleep 0.001
done

echo "Start gate detected. Running request $REQUEST_ID..."

BASE_URL="$BASE_URL" \
MODEL="$MODEL" \
MAX_TOKENS="$MAX_TOKENS" \
PROMPT_ID="$PROMPT_ID" \
PROMPT="$PROMPT" \
RESULTS_DIR="$RESULTS_DIR" \
RESULTS_FILE="$RESULTS_FILE" \
"$SINGLE_REQUEST_SCRIPT"

echo "Launcher for request $REQUEST_ID complete."
