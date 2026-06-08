#!/usr/bin/env bash
set -u

HOST_PORT="${HOST_PORT:-8000}"
BASE_URL="${BASE_URL:-http://localhost:$HOST_PORT}"
MODEL="${MODEL:-Qwen/Qwen2.5-0.5B-Instruct}"
MAX_TOKENS="${MAX_TOKENS:-100}"
PROMPT="${PROMPT:-Explain GPU inference in one short paragraph.}"
RESULTS_DIR="${RESULTS_DIR:-benchmark-results}"
RESULTS_FILE="${RESULTS_FILE:-$RESULTS_DIR/single-request-results.jsonl}"

echo "Running single-request benchmark..."
echo "Host Port:   $HOST_PORT"
echo "Base URL:   $BASE_URL"
echo "Model:      $MODEL"
echo "Max tokens: $MAX_TOKENS"
echo "Prompt:     $PROMPT"
echo "Result Dir: " $RESULTS_DIR

echo
echo "Checking required commands..."

required_commands=("curl" "jq" "date" "awk")

for cmd in "${required_commands[@]}"; do
  if command -v "$cmd" >/dev/null 2>&1; then
    echo "OK: $cmd found"
  else
    echo "ERROR: $cmd is missing."
    exit 1
  fi
done

request_body="$(jq -n \
  --arg model "$MODEL" \
  --arg prompt "$PROMPT" \
  --argjson max_tokens "$MAX_TOKENS" \
  '{
    model: $model,
    messages: [
      {
        role: "user",
        content: $prompt
      }
    ],
    max_tokens: $max_tokens,
    temperature: 0
  }')"

echo
echo "Sending request..."

start_ns="$(date +%s%N)"

if ! response="$(curl -sS "$BASE_URL/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d "$request_body")"; then
  end_ns="$(date +%s%N)"
  elapsed_ms="$(( (end_ns - start_ns) / 1000000 ))"

  echo
  echo "ERROR: request failed."
  echo "Elapsed time before failure: ${elapsed_ms} ms"
  echo "Possible reasons:"
  echo "- vLLM is not running."
  echo "- Wrong BASE_URL or HOST_PORT."
  echo "- SSH tunnel is not open if testing from your laptop."
  exit 1
fi

if ! echo "$response" | jq . >/dev/null 2>&1; then
  echo "ERROR: response was not valid JSON."
  echo "$response"
  exit 1
fi

if echo "$response" | jq -e '.error' >/dev/null 2>&1; then
  echo "ERROR: API returned an error:"
  echo "$response" | jq .
  exit 1
fi

timestamp="$(date -Is)"
prompt_tokens="$(echo "$response" | jq -r '.usage.prompt_tokens // 0')"
completion_tokens="$(echo "$response" | jq -r '.usage.completion_tokens // 0')"
total_tokens="$(echo "$response" | jq -r '.usage.total_tokens // 0')"
finish_reason="$(echo "$response" | jq -r '.choices[0].finish_reason // "unknown"')"
response_model="$(echo "$response" | jq -r '.model // "unknown"')"

if [ "$elapsed_ms" -gt 0 ]; then
  tokens_per_sec="$(awk "BEGIN { printf \"%.2f\", $completion_tokens / ($elapsed_ms / 1000) }")"
else
  tokens_per_sec="unknown"
fi

result_row="$(jq -n \
  --arg timestamp "$timestamp" \
  --arg base_url "$BASE_URL" \
  --arg requested_model "$MODEL" \
  --arg response_model "$response_model" \
  --arg prompt "$PROMPT" \
  --arg max_tokens "$MAX_TOKENS" \
  --arg elapsed_ms "$elapsed_ms" \
  --arg prompt_tokens "$prompt_tokens" \
  --arg completion_tokens "$completion_tokens" \
  --arg total_tokens "$total_tokens" \
  --arg finish_reason "$finish_reason" \
  --arg tokens_per_sec "$tokens_per_sec" \
  '{
    timestamp: $timestamp,
    base_url: $base_url,
    requested_model: $requested_model,
    response_model: $response_model,
    prompt: $prompt,
    max_tokens: ($max_tokens | tonumber),
    elapsed_ms: ($elapsed_ms | tonumber),
    prompt_tokens: ($prompt_tokens | tonumber),
    completion_tokens: ($completion_tokens | tonumber),
    total_tokens: ($total_tokens | tonumber),
    finish_reason: $finish_reason,
    completion_tokens_per_sec: (
      if $tokens_per_sec == "unknown"
      then null
      else ($tokens_per_sec | tonumber)
      end
    )
  }')"

echo
echo "Creating Results Dir"
mkdir -p "$RESULTS_DIR"

echo "$result_row" >> "$RESULTS_FILE"
echo "Result written to: $RESULTS_FILE"
echo
echo "Benchmark result:"
echo "Response model:        $response_model"
echo "Elapsed time:          ${elapsed_ms} ms"
echo "Prompt tokens:         $prompt_tokens"
echo "Completion tokens:     $completion_tokens"
echo "Total tokens:          $total_tokens"
echo "Finish reason:         $finish_reason"
echo "Completion tokens/sec: $tokens_per_sec"
