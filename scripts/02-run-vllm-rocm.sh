#!/usr/bin/env bash
set -u

MODEL="${MODEL:-Qwen/Qwen2.5-0.5B-Instruct}"
IMAGE="${IMAGE:-vllm/vllm-openai-rocm:latest}"
CONTAINER_NAME="${CONTAINER_NAME:-vllm-rocm}"
HOST_PORT="${HOST_PORT:-8000}"
CONTAINER_PORT="${CONTAINER_PORT:-8000}"
GPU_MEMORY_UTILIZATION="${GPU_MEMORY_UTILIZATION:-0.60}"
TENSOR_PARALLEL_SIZE="${TENSOR_PARALLEL_SIZE:-1}"
PIPELINE_PARALLEL_SIZE="${PIPELINE_PARALLEL_SIZE:-1}"
DATA_PARALLEL_SIZE="${DATA_PARALLEL_SIZE:-1}"

print_gpu_stats() {
  local label="${1:-GPU status:}"
  echo 
  echo "$label"
  rocm-smi
}

echo "Starting vLLM ROCm container..."

echo "Image:          $IMAGE"
echo "Model:          $MODEL"
echo "Container name: $CONTAINER_NAME"
echo "Host port:      $HOST_PORT"
echo "Container port: $CONTAINER_PORT"
echo "GPU memory util: $GPU_MEMORY_UTILIZATION"
echo "Tensor Parallel Size : $TENSOR_PARALLEL_SIZE"
echo "Pipeline Parallel Size : $PIPELINE_PARALLEL_SIZE"
echo "Data Parallel Size : $DATA_PARALLEL_SIZE"

echo
echo "Checking whether 'ss' is available for host port inspection..."
if ! command -v ss >/dev/null 2>&1; then
  echo "ERROR: ss command is missing. Cannot check host port availability."
  exit 1
fi

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
echo "Checking AMD ROCm/GPU prerequisites before starting container..."
if ! command -v rocm-smi >/dev/null 2>&1; then
  echo "ERROR: rocm-smi is missing."
  echo "This script should be run on the AMD GPU Droplet with ROCm installed."
  exit 1
fi

device_files=("/dev/kfd" "/dev/dri")

for device_file in "${device_files[@]}"; do
  if [ ! -e "$device_file" ]; then
    echo "ERROR: $device_file is missing."
    echo "This script should be run on the AMD GPU Droplet."
    exit 1
  fi
done

echo "OK: AMD ROCm/GPU prerequisites are present."

echo
echo "Removing any existing container named $CONTAINER_NAME..."
docker rm -f "$CONTAINER_NAME" >/dev/null 2>&1 || true

echo
echo "Checking whether host port $HOST_PORT is available..."

if ss -ltn | grep -q ":$HOST_PORT "; then
  echo "ERROR: host port $HOST_PORT is still in use after removing $CONTAINER_NAME."
  echo "Another process or container is using this port."
  echo "To inspect:"
  echo "  sudo ss -ltnp | grep ':$HOST_PORT'"
  echo "  docker ps -a"
  echo " Workaround : Try running with a different port, for example:"
  echo "  HOST_PORT=8002 ./02-run-vllm-rocm.sh"
  exit 1
fi

echo "OK: host port $HOST_PORT appears available."

print_gpu_stats "GPU status before starting vLLM:"

echo
echo "Starting vLLM container..."

docker run -d \
  --name "$CONTAINER_NAME" \
  --device=/dev/kfd \
  --device=/dev/dri \
  --group-add video \
  --ipc=host \
  -p "$HOST_PORT:$CONTAINER_PORT" \
  "$IMAGE" \
  --model "$MODEL" \
  --host 0.0.0.0 \
  --port "$CONTAINER_PORT" \
  --gpu-memory-utilization "$GPU_MEMORY_UTILIZATION"

echo
echo "Container started: $CONTAINER_NAME"
echo
echo "To follow logs:"
echo "  docker logs -f $CONTAINER_NAME"
echo
echo "To test health from the Droplet:"
echo "  curl http://localhost:$HOST_PORT/health"

print_gpu_stats "GPU status immediately after starting vLLM:"
