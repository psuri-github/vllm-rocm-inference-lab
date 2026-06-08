# Runbook: AMD GPU Droplet vLLM Benchmark Session

## Goal

Run vLLM on an AMD ROCm GPU Droplet and collect one benchmark result.

## Before creating Droplet

- Confirm GPU Droplet availability
- Confirm credit/payment safety
- Confirm repo is pushed
- Confirm benchmark plan is current

## Droplet setup

```bash
git clone https://github.com/psuri-github/vllm-rocm-inference-lab.git
cd vllm-rocm-inference-lab
chmod +x scripts/*.sh



## Verify environment
```bash
./scripts/01-remote-gpu-checks.sh

## Start vLLM
```bash
HOST_PORT=8002 GPU_MEMORY_UTILIZATION=0.60 ./scripts/02-run-vllm-rocm.sh

## Check status
```bash
HOST_PORT=8002 ./scripts/05-vllm-status.sh

## Run benchmark
```bash
BASE_URL=http://localhost:8002 ./scripts/06-single-request-benchmark.sh

## Inspect result
```bash
cat benchmark-results/single-request-results.jsonl | jq .

## Save result
```bash
cp benchmark-results/single-request-results.jsonl docs/

