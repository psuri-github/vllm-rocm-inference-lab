#!/usr/bin/env bash
set -u

HOST_PORT="${HOST_PORT:-8000}"
CONTAINER_NAME="${CONTAINER_NAME:-vllm-rocm}"
BASE_URL="${BASE_URL:-http://localhost:$HOST_PORT}"

echo "Checking vLLM status..."
echo "Container name: $CONTAINER_NAME"
echo "Base URL:       $BASE_URL"

echo
echo "Checking container: vllm-rocm"

if docker ps -a --format '{{.Names}}' | grep -qx "$CONTAINER_NAME"; then
  echo "OK: container $CONTAINER_NAME exists"
  docker ps -a --filter "name=^/${CONTAINER_NAME}$"
  echo
  echo "Recent logs:"
  docker logs --tail 40 "$CONTAINER_NAME"
else
  echo "WARNING: container $CONTAINER_NAME does not exist."
fi

echo
echo "Checking health endpoint: $BASE_URL/health"

if curl -i "$BASE_URL/health"; then
  echo
  echo "OK: health endpoint reachable."
else
  echo
  echo "WARNING: health endpoint is not reachable."
  echo "Possible reasons:"
  echo "- vLLM is not running."
  echo "- Wrong HOST_PORT."
  echo "- SSH tunnel is not open."
fi

echo
echo "Checking models endpoint: $BASE_URL/v1/models"

if curl -i "$BASE_URL/v1/models"; then
  echo
  echo "OK: models endpoint reachable."
else
  echo
  echo "WARNING: models endpoint is not reachable."
  echo "Possible reasons:"
  echo "- vLLM is not running."
  echo "- Wrong HOST_PORT."
  echo "- SSH tunnel is not open."
fi
echo
echo "Checking GPU status..."

if command -v rocm-smi >/dev/null 2>&1; then
  rocm-smi
else
  echo "WARNING: rocm-smi is not installed. Skipping GPU status."
fi

echo
echo "vLLM status check complete."
