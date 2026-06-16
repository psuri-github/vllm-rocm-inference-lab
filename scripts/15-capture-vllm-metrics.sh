#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8000}"
RESULTS_DIR="${RESULTS_DIR:-benchmark-results}"
RUN_ID="${RUN_ID:-$(date -u +%Y%m%dT%H%M%S)}"

mkdir -p "$RESULTS_DIR"

echo "Capturing vLLM metrics..."
curl -s "$BASE_URL/metrics" > "$RESULTS_DIR/vllm-metrics-$RUN_ID.prom"

echo "Capturing ROCm SMI..."
if command -v rocm-smi >/dev/null 2>&1; then
  rocm-smi > "$RESULTS_DIR/rocm-smi-$RUN_ID.txt"
  rocm-smi --showuse --showmemuse --showtemp > "$RESULTS_DIR/rocm-smi-summary-$RUN_ID.txt"
else
  echo "rocm-smi not found" > "$RESULTS_DIR/rocm-smi-$RUN_ID.txt"
  echo "rocm-smi not found" > "$RESULTS_DIR/rocm-smi-summary-$RUN_ID.txt"
fi

echo "Saved:"
echo "  $RESULTS_DIR/vllm-metrics-$RUN_ID.prom"
echo "  $RESULTS_DIR/rocm-smi-$RUN_ID.txt"
