#!/usr/bin/env bash
set -u

BASE_URL="${BASE_URL:-http://localhost:8000}"
MODEL="${MODEL:-Qwen/Qwen2.5-0.5B-Instruct}"

echo "Testing vLLM chat completion endpoint..."
echo "Base URL: $BASE_URL"
echo "Model:    $MODEL"

if curl "$BASE_URL/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d "{
    \"model\": \"$MODEL\",
    \"messages\": [
      {
        \"role\": \"user\",
        \"content\": \"Explain GPU inference in one short paragraph.\"
      }
    ],
    \"max_tokens\": 100,
    \"temperature\": 0
  }"; then
  echo
  echo "OK: chat completion request completed."
else
  echo
  echo "ERROR: chat completion request failed."
  echo "Possible reasons:"
  echo "- vLLM is not running."
  echo "- Model is still loading."
  echo "- The /v1/chat/completions endpoint is not reachable."
  echo "- SSH tunnel is not open if testing from your laptop."
  exit 1
fi
