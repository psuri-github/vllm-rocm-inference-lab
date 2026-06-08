#!/usr/bin/env bash
set -u

CONTAINER_NAME="${CONTAINER_NAME:-vllm-rocm}"

echo "Cleaning up vLLM container..."
echo "Container name: $CONTAINER_NAME"

echo
echo "Checking Docker..."

if ! command -v docker >/dev/null 2>&1; then
  echo "ERROR: docker is not installed or not in PATH."
  exit 1
fi

if ! docker ps >/dev/null 2>&1; then
  echo "ERROR: docker is installed, but docker ps failed."
  echo "Docker may not be running, or this user may not have permission."
  exit 1
fi

echo "OK: Docker is available and usable."

echo
echo "Checking whether container exists..."

if docker ps -a --format '{{.Names}}' | grep -qx "$CONTAINER_NAME"; then
  echo "Found container: $CONTAINER_NAME"

  echo
  echo "Stopping and removing $CONTAINER_NAME..."
  docker rm -f "$CONTAINER_NAME"

  echo
  echo "OK: container $CONTAINER_NAME removed."
else
  echo "Container $CONTAINER_NAME does not exist. Nothing to clean up."
fi

echo
echo "Remaining containers:"
docker ps -a

echo
echo "Cleanup complete."
