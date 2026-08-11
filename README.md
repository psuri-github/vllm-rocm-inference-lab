# vLLM ROCm Inference Lab

Hands-on lab for deploying and operating **vLLM on AMD ROCm GPU infrastructure**.

This repository is designed for AI infrastructure engineers, MLOps/platform operators, and ROCm/AMD GPU teams who want to validate an OpenAI-compatible inference stack on AMD GPU cloud resources.

## What this repo does

This project helps you:

* validate AMD ROCm GPU readiness on a remote Droplet
* launch a ROCm-compatible vLLM Docker container
* verify OpenAI-compatible inference endpoints (`/health`, `/v1/models`, `/v1/chat/completions`)
* run a simple benchmark and collect result artifacts
* follow a cost-safe workflow for ephemeral GPU cloud resources

## Audience

This repo is most useful for:

* infrastructure engineers evaluating ROCm-based inference
* platform teams validating GPU access and container runtime behavior
* operators testing vendor-specific GPU enablement
* engineers exploring Docker-based vLLM serving before moving to orchestration

## Current scope

The current scope focuses on direct Docker-based serving on an AMD GPU Droplet. It is intentionally not a full production Kubernetes deployment, but it covers practical operational validation, endpoint testing, and basic benchmark capture.

## Why AMD ROCm?

Most GPU inference tutorials assume NVIDIA CUDA. This lab intentionally explores the AMD ROCm path using AMD MI300X infrastructure and Docker.

Key platform topics include:

* AMD ROCm runtime verification
* AMD device files such as `/dev/kfd` and `/dev/dri`
* ROCm-compatible vLLM Docker images
* GPU access from inside containers
* operational checks on blank versus preconfigured GPU images

## Prerequisites

* AMD GPU Droplet or VM with ROCm-compatible OS/image
* Docker installed and usable by the current user
* SSH access to the GPU host
* local tools: `ssh`, `scp`, `curl`, `git`
* host port availability for the vLLM service (default `8002`)
* optional: `jq` for inspecting JSON benchmark output

## Operational workflow

1. clone the repo on the GPU host
2. verify ROCm and GPU access
3. launch the vLLM ROCm container
4. verify health and inference endpoints
5. run benchmark tests
6. inspect results and clean up
7. destroy the GPU Droplet

## Deployment paths

This lab now has two deployment paths:

### Path 1: Direct Docker-based serving

The original workflow runs vLLM directly as a ROCm-compatible Docker container on an AMD GPU Droplet.

This path is useful for validating:

* ROCm runtime readiness
* AMD GPU device access
* vLLM container startup
* OpenAI-compatible endpoint behavior
* benchmark and metrics collection

### Path 2: Kubernetes/K3s-based serving

The Kubernetes path deploys vLLM into a K3s cluster using standard Kubernetes resources.

The manifests are stored under `kubernetes/` and include:

* Namespace
* PersistentVolumeClaim for Hugging Face model cache
* Deployment for the vLLM server
* ClusterIP Service for in-cluster access
* GPU test pod
* K3s lab cheatsheet

This path is useful for understanding how Kubernetes handles GPU-backed inference workloads using generic primitives such as Deployments, Services, PVCs, probes, and device-plugin-managed resources.

It is not a custom Kubernetes operator yet. The goal is to establish the foundation for understanding where generic Kubernetes reconciliation ends and inference-aware operator logic could begin.

## Quickstart

```bash
git clone https://github.com/psuri-github/vllm-rocm-inference-lab.git
cd vllm-rocm-inference-lab
chmod +x scripts/*.sh
./scripts/01-remote-gpu-checks.sh
HOST_PORT=8002 GPU_MEMORY_UTILIZATION=0.60 ./scripts/02-run-vllm-rocm.sh
BASE_URL=http://localhost:8002 ./scripts/06-single-request-benchmark.sh
./scripts/07-cleanup-vllm.sh
```

For the full step-by-step workflow, see `docs/runbook.md`.

## Scripts

### `00-local-preflight.sh`

Run locally to confirm the workstation has the tools needed to use the lab.

Checks:

* `ssh`
* `scp`
* `curl`
* `git`

### `01-remote-gpu-checks.sh`

Run on the AMD GPU Droplet.

Validates:

* Bash version
* `rocm-smi`
* Docker installation and usability
* GPU device files `/dev/kfd` and `/dev/dri`
* optional `rocminfo` if present

### `02-run-vllm-rocm.sh`

Run on the Droplet to start the ROCm-compatible vLLM container.

Defaults:

* Docker image: `vllm/vllm-openai-rocm:latest`
* model: `Qwen/Qwen2.5-0.5B-Instruct`

Example:

```bash
HOST_PORT=8002 GPU_MEMORY_UTILIZATION=0.60 ./scripts/02-run-vllm-rocm.sh
```

### `03-test-health.sh`

Checks the vLLM `/health` endpoint.

Example:

```bash
BASE_URL=http://localhost:8002 ./scripts/03-test-health.sh
```

This can run on the Droplet or locally with SSH port forwarding.

### `04-test-chat-completion.sh`

Sends a sample request to `/v1/chat/completions` to verify inference.

### `05-vllm-status.sh`

Collects service and GPU status information.

Reports:

* container existence
* recent container logs
* `/health` status
* `/v1/models` status
* GPU status via `rocm-smi`

Example:

```bash
HOST_PORT=8002 ./scripts/05-vllm-status.sh
```

### `06-single-request-benchmark.sh`

Runs a single OpenAI-compatible chat benchmark request and captures timing and token metrics.

Outputs:

* `benchmark-results/single-request-results.jsonl`

Example:

```bash
BASE_URL=http://localhost:8002 ./scripts/06-single-request-benchmark.sh
```

### `07-cleanup-vllm.sh`

Stops and removes only the `vllm-rocm` container created by this project.

This avoids deleting other preconfigured containers on ROCm-ready images.

## Cost safety

GPU Droplets are billed while they exist, even if powered off.

Recommended workflow:

* create GPU Droplet
* run verification, serving, and benchmark tests
* save results and notes
* clean up project-created containers
* destroy the Droplet

Do not rely on a powered-off Droplet to stop billing.

## Benchmarking and results

This lab captures basic inference performance and operational validation rather than exhaustive benchmarking.

Benchmark results are summarized in `docs/benchmark-results.md`.

Benchmark prompts and test configuration are stored in `benchmarks/prompts.jsonl`.

## Repository structure

```text
.
├── README.md
├── benchmarks
│   └── prompts.jsonl
├── docs
│   ├── benchmark-plan.md
│   ├── benchmark-results.md
│   ├── runbook.md
│   └── session-log.md
└── scripts
    ├── 00-local-preflight.sh
    ├── 01-remote-gpu-checks.sh
    ├── 02-run-vllm-rocm.sh
    ├── 03-test-health.sh
    ├── 04-test-chat-completion.sh
    ├── 05-vllm-status.sh
    ├── 06-single-request-benchmark.sh
    └── 07-cleanup-vllm.sh
```

## Next steps

Future work may include:

* broader observability and metrics collection
* multi-node inference validation
* expanded benchmark suites
* more extensive ROCm image compatibility testing


```text
Create GPU Droplet
Run verification , serving, and benchmark tests
Save notes and summarized results
Clean up project-created containers
Destroy GPU Droplet
```

Do not rely on powering off the Droplet to stop billing.

## Recommended Workflow
For the most current step-by-step instructions, see:

docs/runbook.md

A typical session looks like:

1. Clone the repository on the Droplet
```text
git clone https://github.com/psuri-github/vllm-rocm-inference-lab.git
cd vllm-rocm-inference-lab
chmod +x scripts/*.sh
```

2. Verify GPU and ROCm environment
```text
./scripts/01-remote-gpu-checks.sh
```

3. Start vLLM
```text
HOST_PORT=8002 GPU_MEMORY_UTILIZATION=0.60 ./scripts/02-run-vllm-rocm.sh
```

4. Follow container logs
```text
docker logs -f vllm-rocm
```

Look for server readiness messages such as:

```text
Starting vLLM server on http://0.0.0.0:8000
Application startup complete.
```

5. Check service status
```text
HOST_PORT=8002 ./scripts/05-vllm-status.sh
```

6. Test the health endpoint
```text
BASE_URL=http://localhost:8002 ./scripts/03-test-health.sh
```

7. Test chat completion
```text
BASE_URL=http://localhost:8002 ./scripts/04-test-chat-completion.sh
```

8. Run a single-request benchmark
```text
BASE_URL=http://localhost:8002 ./scripts/06-single-request-benchmark.sh
```

9. Inspect benchmark results
```text
cat benchmark-results/single-request-results.jsonl | jq .
```

10. Clean up the vLLM container
```text
./scripts/07-cleanup-vllm.sh
```

11. Destroy the Droplet

Destroy the GPU Droplet after the session to avoid continued billing.

### Benchmark Results

The first benchmark milestone used repeated non-streaming /v1/chat/completions requests against:

```text
Qwen/Qwen2.5-0.5B-Instruct
```

Summarized results are documented in:

```text
docs/benchmark-results.md
```

The initial benchmark captures approximate end-to-end completion tokens/sec, not streaming time-to-first-token or sustained concurrent throughput.

### Operational Lessons

This project has already surfaced several real-world operational details:

Preconfigured ROCm images may already have containers running.
Host ports such as 8000 may already be occupied.
GPU memory may already be consumed by preloaded services.
rocm-smi, docker ps, docker logs, and ss are useful first-line debugging tools.
vLLM readiness should be confirmed through logs and endpoint checks, not only container startup.
Benchmark results should distinguish between simple end-to-end request timing and deeper serving metrics such as streaming throughput or time to first token.

### Current Milestones

Milestone 1:

* [x] Create local preflight script
* [x] Create remote GPU verification script
* [x] Create vLLM ROCm container startup script
* [x] Create health endpoint test script
* [x] Create chat completion test script
* [x] Run scripts on AMD GPU Droplet
* [x] Verify vLLM starts successfully
* [x] Verify chat completion response
* [x] Record first session findings

Milestone 2:

* [x] Improve Docker-based serving operations
* [x] Add host port conflict detection
* [x] Make host port configurable
* [x] Make GPU memory utilization configurable
* [x] Add vLLM status script
* [x] Add cleanup script
* [x] Document operational failure modes

Milestone 3: 
* [x] Add basic benchmarking
* [x] Add benchmark plan
* [x] Add single-request benchmark script
* [x] Write benchmark results to JSONL
* [x] Summarize first benchmark result

Milestone 4:
* [x] Install and validate K3s cluster on the AMD GPU Droplet
* [x] Install the AMD GPU Plugin
* [x] Verify that Kubernetes exposes the GPU as amd.com/gpu
* [x] Add a GPU test pod to confirm /dev/kfd and /dev/dri access inside Kubernetes
* [x] Create a dedicated vllm namespace
* [x] Add a PVC for Hugging Face Model cache storage
* [x] Add a vLLM deployment using the ROCm-compatible vLLM image
* [x] Request one AMD GPU from Kubernetes using amd.com/gpu: "1"
* [x] Add startup, readiness, and liveness probes against the vLLM /health endpoint
* [x] Add a ClusterIP Service for in-cluster access to the vLLM server
* [x] Verify that the vLLM service could serve OpenAI-compatible inference requests from inside the cluster
* [x] Document the end-to-end K3s workflow in a Kubernetes lab cheatsheet 

Possible Future Milestones
* Prompt-suite benchmarking
* Run a small set of repeatable prompts
* Capture per-prompt latency and token counts
* Compare behavior across prompt types
* Model comparison
* Compare small open models served through vLLM ROCm
* Capture latency, output token counts, and memory behavior
* Treat public model leaderboards as context, not as directly comparable serving benchmarks
* Observability
* Explore vLLM /metrics
* Identify useful inference-serving metrics
* Document GPU and service-level signals


## Notes
This is a learning project. The goal is to understand each step rather than hide everything behind a prebuilt deployment.

The project intentionally starts simple with Docker before moving into more advanced serving, benchmarking, and infrastructure topics.
