#!/usr/bin/env bash
set -u

TOTAL_REQUESTS="${TOTAL_REQUESTS:-4}"
BASE_URL="${BASE_URL:-http://localhost:8000}"
MODEL="${MODEL:-Qwen/Qwen2.5-0.5B-Instruct}"
MAX_TOKENS="${MAX_TOKENS:-100}"
PROMPT="${PROMPT:-Explain GPU inference in one short paragraph.}"
RESULTS_DIR="${RESULTS_DIR:-benchmark-results}"
RESULTS_FILE="${RESULTS_FILE:-$RESULTS_DIR/concurrent-request-results.jsonl}"
CONTINUE_ON_ERROR="${CONTINUE_ON_ERROR:-false}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONCURRENT_LAUNCHER_SCRIPT="$SCRIPT_DIR/09-concurrent-request-launcher.sh"

RUN_ID="$(date +%Y%m%dT%H%M%S)"
TEMP_DIR="$(mktemp -d)"
REQUEST_RESULTS_DIR="$TEMP_DIR/results"
REQUEST_LOGS_DIR="$TEMP_DIR/logs"
START_GATE_FILE="$TEMP_DIR/start-gate"

# Ensure workers do not pass the gate before the parent releases them.
rm -f "$START_GATE_FILE"

cleanup() {
  if [ -n "${START_GATE_FILE:-}" ] && [ -f "$START_GATE_FILE" ]; then
    echo
    echo "Cleaning up start gate file..."
    rm -f "$START_GATE_FILE"
    echo "OK: removed $START_GATE_FILE"
  fi
}

trap cleanup EXIT

mkdir -p "$RESULTS_DIR"
mkdir -p "$REQUEST_RESULTS_DIR"
mkdir -p "$REQUEST_LOGS_DIR"

echo "Running concurrent-request benchmark..."
echo "Total requests:    $TOTAL_REQUESTS"
echo "Base URL:          $BASE_URL"
echo "Model:             $MODEL"
echo "Max tokens:        $MAX_TOKENS"
echo "Prompt:            $PROMPT"
echo "Results file:      $RESULTS_FILE"
echo "Continue on error: $CONTINUE_ON_ERROR"
echo "Run ID:            $RUN_ID"
echo "Temp dir:          $TEMP_DIR"
echo "Start gate file:   $START_GATE_FILE"

echo
echo "Checking required commands..."

required_commands=("date" "jq" "mktemp")

for command_name in "${required_commands[@]}"; do
  if command -v "$command_name" >/dev/null 2>&1; then
    echo "OK: $command_name found"
  else
    echo "ERROR: required command not found: $command_name"
    exit 1
  fi
done

echo
echo "Checking concurrent launcher script..."

if [ ! -x "$CONCURRENT_LAUNCHER_SCRIPT" ]; then
  echo "ERROR: concurrent launcher script is missing or not executable:"
  echo "$CONCURRENT_LAUNCHER_SCRIPT"
  exit 1
fi

echo "OK: found $CONCURRENT_LAUNCHER_SCRIPT"

echo
echo "Launching $TOTAL_REQUESTS worker processes..."

pids=()
request_ids=()

for request_id in $(seq 1 "$TOTAL_REQUESTS"); do
  request_result_file="$REQUEST_RESULTS_DIR/request-$request_id.jsonl"
  request_log_file="$REQUEST_LOGS_DIR/request-$request_id.log"

  echo "Preparing request $request_id..."

  START_GATE_FILE="$START_GATE_FILE" \
  REQUEST_ID="$request_id" \
  BASE_URL="$BASE_URL" \
  MODEL="$MODEL" \
  MAX_TOKENS="$MAX_TOKENS" \
  PROMPT="$PROMPT" \
  RESULTS_DIR="$REQUEST_RESULTS_DIR" \
  RESULTS_FILE="$request_result_file" \
  "$CONCURRENT_LAUNCHER_SCRIPT" > "$request_log_file" 2>&1 &

  pids+=("$!")
  request_ids+=("$request_id")
done

echo
echo "All workers launched."
echo "Releasing start gate..."

benchmark_start_ns="$(date +%s%N)"
touch "$START_GATE_FILE"

successful_requests=0
failed_requests=0

echo
echo "Waiting for requests to finish..."

for index in "${!pids[@]}"; do
  pid="${pids[$index]}"
  current_request_id="${request_ids[$index]}"
  request_log_file="$REQUEST_LOGS_DIR/request-$current_request_id.log"

  if wait "$pid"; then
    echo "OK: request $current_request_id completed successfully."
    successful_requests=$((successful_requests + 1))
  else
    echo "ERROR: request $current_request_id failed."
    failed_requests=$((failed_requests + 1))

    echo
    echo "Log for failed request $current_request_id:"
    cat "$request_log_file"

    if [ "$CONTINUE_ON_ERROR" = "true" ]; then
      echo "CONTINUE_ON_ERROR=true, continuing."
    else
      echo "CONTINUE_ON_ERROR=false, stopping benchmark."
      echo "Temporary files are in: $TEMP_DIR"
      exit 1
    fi
  fi
done

benchmark_end_ns="$(date +%s%N)"
benchmark_elapsed_ms="$(( (benchmark_end_ns - benchmark_start_ns) / 1000000 ))"

echo
echo "Combining successful request results..."

combined_count=0

for result_file in "$REQUEST_RESULTS_DIR"/request-*.jsonl; do
  if [ -e "$result_file" ]; then
    cat "$result_file" >> "$RESULTS_FILE"
    combined_count=$((combined_count + 1))
  fi
done

echo
echo "Concurrent benchmark complete."
echo "Successful requests:              $successful_requests"
echo "Failed requests:                  $failed_requests"
echo "Combined result files:            $combined_count"
echo "Benchmark wall-clock elapsed ms:  $benchmark_elapsed_ms"
echo "Results file:                     $RESULTS_FILE"
echo "Temporary logs:                   $REQUEST_LOGS_DIR"

echo
echo "Summary from this run:"

if [ "$combined_count" -gt 0 ]; then
  jq -s --argjson benchmark_elapsed_ms "$benchmark_elapsed_ms" '{
    runs: length,
    benchmark_wall_clock_elapsed_ms: $benchmark_elapsed_ms,
    total_completion_tokens: (map(.completion_tokens) | add),
    aggregate_completion_tokens_per_sec: ((map(.completion_tokens) | add) / ($benchmark_elapsed_ms / 1000)),
    avg_elapsed_ms: (map(.elapsed_ms) | add / length),
    min_elapsed_ms: (map(.elapsed_ms) | min),
    max_elapsed_ms: (map(.elapsed_ms) | max),
    avg_completion_tokens_per_sec_per_request: (map(.completion_tokens_per_sec) | add / length),
    min_completion_tokens_per_sec_per_request: (map(.completion_tokens_per_sec) | min),
    max_completion_tokens_per_sec_per_request: (map(.completion_tokens_per_sec) | max)
  }' "$REQUEST_RESULTS_DIR"/request-*.jsonl
else
  echo "No successful result files found. Skipping summary."
fi

echo
echo "Done."

