#!/usr/bin/env bash
set -u

BASE_URL="${BASE_URL:-http://localhost:8000}"

echo "Checking vLLM health endpoint..."
echo "Base URL: $BASE_URL"

if curl -i "$BASE_URL/health"; then
  echo
  echo "OK: vLLM health endpoint is reachable."
else
  echo
  echo "ERROR: Could not reach vLLM health endpoint."
  echo "Possible reasons:"
  echo "- vLLM is not running."
  echo "- Docker container is still starting or failed."
  echo "- Port 8000 is not mapped."
  echo "- SSH tunnel is not open if testing from your laptop."
  exit 1
fi
