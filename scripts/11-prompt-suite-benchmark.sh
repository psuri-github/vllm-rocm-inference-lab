#!/usr/bin/env bash
set -u

PROMPTS_FILE="${PROMPTS_FILE:-benchmarks/prompts.jsonl}"
BASE_URL="${BASE_URL:-http://localhost:8000}"
MODEL="${MODEL:-Qwen/Qwen2.5-0.5B-Instruct}"
MAX_TOKENS="${MAX_TOKENS:-100}"
RESULTS_DIR="${RESULTS_DIR:-benchmark-results}"
CONTINUE_ON_ERROR="${CONTINUE_ON_ERROR:-false}"

RUN_ID="$(date +%Y%m%dT%H%M%S)"
RESULTS_FILE="${RESULTS_FILE:-$RESULTS_DIR/prompt-suite-$RUN_ID.jsonl}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SINGLE_REQUEST_SCRIPT="$SCRIPT_DIR/06-single-request-benchmark.sh"

echo "Running prompt-suite benchmark..."
echo "Prompts file:      $PROMPTS_FILE"
echo "Base URL:          $BASE_URL"
echo "Model:             $MODEL"
echo "Max tokens:        $MAX_TOKENS"
echo "Results file:      $RESULTS_FILE"
echo "Continue on error: $CONTINUE_ON_ERROR"
echo "Run ID:            $RUN_ID"

echo
echo "Checking required commands..."

required_commands=("date" "jq")

for command_name in "${required_commands[@]}"; do
  if command -v "$command_name" >/dev/null 2>&1; then
    echo "OK: $command_name found"
  else
    echo "ERROR: required command not found: $command_name"
    exit 1
  fi
done

echo
echo "Checking benchmark script..."

if [ ! -x "$SINGLE_REQUEST_SCRIPT" ]; then
  echo "ERROR: single-request benchmark script is missing or not executable:"
  echo "$SINGLE_REQUEST_SCRIPT"
  exit 1
fi

echo "OK: found $SINGLE_REQUEST_SCRIPT"

echo
echo "Checking prompts file..."

if [ ! -f "$PROMPTS_FILE" ]; then
  if [ -f "$REPO_ROOT/$PROMPTS_FILE" ]; then
    PROMPTS_FILE="$REPO_ROOT/$PROMPTS_FILE"
  else
    echo "ERROR: prompts file not found:"
    echo "$PROMPTS_FILE"
    exit 1
  fi
fi

if [ ! -s "$PROMPTS_FILE" ]; then
  echo "ERROR: prompts file is empty:"
  echo "$PROMPTS_FILE"
  exit 1
fi

echo "OK: found $PROMPTS_FILE"

mkdir -p "$RESULTS_DIR"

echo
echo "Validating prompts file..."

line_number=0

while IFS= read -r line || [ -n "$line" ]; do
  line_number=$((line_number + 1))

  if [ -z "$line" ]; then
    continue
  fi

  if ! echo "$line" | jq -e '.prompt_id and .prompt' >/dev/null 2>&1; then
    echo "ERROR: invalid prompt JSON at line $line_number"
    echo "$line"
    echo
    echo "Expected format:"
    echo '{"prompt_id":"short_explain","prompt":"Explain GPU inference in one short paragraph."}'
    exit 1
  fi
done < "$PROMPTS_FILE"

echo "OK: prompts file is valid"

echo
echo "Starting prompt-suite benchmark..."

successful_requests=0
failed_requests=0
prompt_count=0

while IFS= read -r line || [ -n "$line" ]; do
  if [ -z "$line" ]; then
    continue
  fi

  prompt_count=$((prompt_count + 1))

  prompt_id="$(echo "$line" | jq -r '.prompt_id')"
  prompt="$(echo "$line" | jq -r '.prompt')"

  echo
  echo "===== Prompt $prompt_count: $prompt_id ====="
  echo "Prompt: $prompt"

  if PROMPT_ID="$prompt_id" \
    PROMPT="$prompt" \
    BASE_URL="$BASE_URL" \
    MODEL="$MODEL" \
    MAX_TOKENS="$MAX_TOKENS" \
    RESULTS_DIR="$RESULTS_DIR" \
    RESULTS_FILE="$RESULTS_FILE" \
    "$SINGLE_REQUEST_SCRIPT"; then

    echo "OK: prompt $prompt_id completed successfully."
    successful_requests=$((successful_requests + 1))

  else
    echo "ERROR: prompt $prompt_id failed."
    failed_requests=$((failed_requests + 1))

    if [ "$CONTINUE_ON_ERROR" = "true" ]; then
      echo "CONTINUE_ON_ERROR=true, continuing to next prompt."
    else
      echo "CONTINUE_ON_ERROR=false, stopping prompt-suite benchmark."
      exit 1
    fi
  fi

done < "$PROMPTS_FILE"

echo
echo "Prompt-suite benchmark complete."
echo "Total prompts:        $prompt_count"
echo "Successful requests:  $successful_requests"
echo "Failed requests:      $failed_requests"
echo "Results file:         $RESULTS_FILE"

echo
echo "Summary from this run:"

if [ "$successful_requests" -gt 0 ]; then
  jq -s '{
    runs: length,
    prompts: map({
      prompt_id: .prompt_id,
      elapsed_ms: .elapsed_ms,
      prompt_tokens: .prompt_tokens,
      completion_tokens: .completion_tokens,
      total_tokens: .total_tokens,
      finish_reason: .finish_reason,
      completion_tokens_per_sec: .completion_tokens_per_sec
    }),
    avg_elapsed_ms: (map(.elapsed_ms) | add / length),
    min_elapsed_ms: (map(.elapsed_ms) | min),
    max_elapsed_ms: (map(.elapsed_ms) | max),
    avg_prompt_tokens: (map(.prompt_tokens) | add / length),
    avg_completion_tokens_per_sec: (map(.completion_tokens_per_sec) | add / length)
  }' "$RESULTS_FILE"
else
  echo "No successful requests. Skipping summary."
fi

echo
echo "Done."
