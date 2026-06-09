#!/usr/bin/env bash
set -u


RUNS="${RUNS:-10}"
BASE_URL="${BASE_URL:-http://localhost:8000}"
MODEL="${MODEL:-Qwen/Qwen2.5-0.5B-Instruct}"
MAX_TOKENS="${MAX_TOKENS:-100}"
PROMPT="${PROMPT:-Explain GPU inference in one short paragraph.}"
RESULTS_DIR="${RESULTS_DIR:-benchmark-results}"
RESULTS_FILE="${RESULTS_FILE:-$RESULTS_DIR/single-request-results.jsonl}"
CONTINUE_ON_ERROR="${CONTINUE_ON_ERROR:-false}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SINGLE_REQUEST_SCRIPT="$SCRIPT_DIR/06-single-request-benchmark.sh"

echo "Running repeated-request benchmark..."
echo "Runs:         $RUNS"
echo "Base URL:     $BASE_URL"
echo "Model:        $MODEL"
echo "Max tokens:   $MAX_TOKENS"
echo "Prompt:       $PROMPT"
echo "Results file: $RESULTS_FILE"
echo "Continue on error : $CONTINUE_ON_ERROR"

echo
echo "Checking benchmark script..."

if [ ! -x "$SINGLE_REQUEST_SCRIPT" ]; then
  echo "ERROR: single-request benchmark script is missing or not executable:"
  echo "$SINGLE_REQUEST_SCRIPT"
  exit 1
fi

echo "OK: found $SINGLE_REQUEST_SCRIPT"

echo
echo "Starting repeated benchmark runs..."

for run_id in $(seq 1 "$RUNS"); do
  echo
  echo "===== Run $run_id of $RUNS ====="

  if BASE_URL="$BASE_URL" \
    MODEL="$MODEL" \
    MAX_TOKENS="$MAX_TOKENS" \
    PROMPT="$PROMPT" \
    RESULTS_DIR="$RESULTS_DIR" \
    RESULTS_FILE="$RESULTS_FILE" \
    "$SINGLE_REQUEST_SCRIPT"; then
    echo "OK: run $run_id completed successfully."

  else
    echo "ERROR: run $run_id failed."

    if [ "$CONTINUE_ON_ERROR" = "true" ]; then
      echo "CONTINUE_ON_ERROR=true, continuing to next run."
    else
      echo "CONTINUE_ON_ERROR=false, stopping benchmark."
      exit 1
    fi
  fi
done

echo
echo "Repeated-request benchmark complete."
echo "Results written to: $RESULTS_FILE"
