# vLLM ROCm Inference Lab

This project is a hands-on learning lab for deploying and operating **vLLM on AMD ROCm GPU infrastructure**.

The first milestone uses a **DigitalOcean AMD GPU Droplet** to learn how to verify GPU access, run a ROCm-compatible vLLM container, and test the OpenAI-compatible API exposed by vLLM.

The project intentionally starts with Docker-based serving before moving into deeper topics such as benchmarking, observability, and possible Kubernetes-based deployment

## Project Goals

This project is intended to build practical understanding of:

* AMD GPU infrastructure for LLM inference
* ROCm runtime verification
* Docker-based vLLM serving
* OpenAI-compatible inference APIs
* Safe workflows for ephemeral GPU cloud resources
* Operational debugging of model-serving environments
* Basic inference benchmarking
* Future observability and serving infrastructure experiments

## Current Scope

The current scope focuses on running and measuring vLLM directly on an AMD GPU Droplet using Docker.

The project currently includes scripts to:

* Check required local tools
* Verify ROCm and AMD GPU access on a remote Droplet
* Start a vLLM ROCm Docker container
* Test the vLLM health endpoint
* Send a chat completion request to the vLLM OpenAI-compatible API
* Check vLLM container, endpoint, and GPU status
* Run a single-request benchmark and save results
* Clean up the vLLM container created by this project

## Why AMD ROCm?

Most GPU inference tutorials assume NVIDIA CUDA. This project intentionally explores the AMD ROCm path using AMD MI300X infrastructure.

This helps build a better understanding of vendor-specific GPU enablement, including:

* AMD ROCm runtime
* AMD GPU device files such as `/dev/kfd` and `/dev/dri`
* ROCm-compatible vLLM Docker images
* GPU access from inside containers
* Operational differences between blank machines and preconfigured GPU images

## Repository Structure

```text
.
├── README.md
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

## Scripts

### `00-local-preflight.sh`

Runs on the local development machine.

Checks whether basic tools are available:

* `ssh`
* `scp`
* `curl`
* `git`

These tools are needed to connect to the GPU Droplet, copy files, test endpoints, and manage the project through GitHub.

### `01-remote-gpu-checks.sh`

Runs on the AMD GPU Droplet.

Checks whether the Droplet has the expected GPU and runtime environment:

* Bash version
* rocm-smi
* Docker
* Docker usability
* /dev/kfd
* /dev/dri

The script also checks for rocminfo when available. Some ROCm images may not include rocminfo, but the project has successfully run vLLM with rocm-smi, Docker, AMD device files, and the ROCm-compatible vLLM container.

### `02-run-vllm-rocm.sh`

Runs on the AMD GPU Droplet.

Starts the vLLM ROCm Docker container after verifying that Docker and ROCm/GPU prerequisites are present.

The script supports configurable runtime values:

```bash
HOST_PORT=8002 GPU_MEMORY_UTILIZATION=0.60 ./scripts/02-run-vllm-rocm.sh
```

The default model is:

```text
Qwen/Qwen2.5-0.5B-Instruct
```

The default Docker image is:

```text
vllm/vllm-openai-rocm:latest
```

The script includes operational preflight checks for:

* Docker availability
* Docker usability
* AMD GPU device files
* host port conflicts
* GPU status before and immediately after launch

### `03-test-health.sh`

Tests the vLLM health endpoint:

```text
/health
```
Example:

```bash
BASE_URL=http://localhost:8002 ./scripts/03-test-health.sh
```

This script can run either on the Droplet or on the local machine when using SSH port forwarding.

### `04-test-chat-completion.sh`

Sends a test request to the vLLM OpenAI-compatible chat completions endpoint:

```text
/v1/chat/completions
```

This verifies that the model server is reachable and able to generate a response.

### `05-vllm-status.sh`

Checks the current operational status of the vLLM service.

It reports:

* whether the vllm-rocm container exists
* recent container logs
* /health endpoint status
* /v1/models endpoint status
* GPU status using rocm-smi, when available

Example:
```bash
HOST_PORT=8002 ./scripts/05-vllm-status.sh
```

### `06-single-request-benchmark.sh`

Runs a single non-streaming /v1/chat/completions benchmark request.

It captures:

* elapsed request time
* prompt token count
* completion token count
* total token count
* finish reason
* approximate completion tokens/sec

Example:

```bash
BASE_URL=http://localhost:8002 ./scripts/06-single-request-benchmark.sh
```
Successful results are written to:

```text
benchmark-results/single-request-results.jsonl
```

Raw benchmark result files are ignored by Git. Summarized results are captured in:

```text
docs/benchmark-results.md
```

### `07-cleanup-vllm.sh`

Stops and removes only the vLLM container created by this project.

By default, it targets:

```text
vllm-rocm
```

It does not remove preconfigured containers that may be present on DigitalOcean ROCm images, such as rocm, open-webui, or other provider-provided containers.

## Cost Safety

GPU Droplets are billed while they exist, even if they are powered off.

The intended workflow is:

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
git clone https://github.com/psuri-github/vllm-rocm-inference-lab.git
cd vllm-rocm-inference-lab
chmod +x scripts/*.sh
2. Verify GPU and ROCm environment
./scripts/01-remote-gpu-checks.sh
3. Start vLLM
HOST_PORT=8002 GPU_MEMORY_UTILIZATION=0.60 ./scripts/02-run-vllm-rocm.sh
4. Follow container logs
docker logs -f vllm-rocm

Look for server readiness messages such as:

Starting vLLM server on http://0.0.0.0:8000
Application startup complete.
5. Check service status
HOST_PORT=8002 ./scripts/05-vllm-status.sh
6. Test the health endpoint
BASE_URL=http://localhost:8002 ./scripts/03-test-health.sh
7. Test chat completion
BASE_URL=http://localhost:8002 ./scripts/04-test-chat-completion.sh
8. Run a single-request benchmark
BASE_URL=http://localhost:8002 ./scripts/06-single-request-benchmark.sh
9. Inspect benchmark results
cat benchmark-results/single-request-results.jsonl | jq .
10. Clean up the vLLM container
./scripts/07-cleanup-vllm.sh
11. Destroy the Droplet

Destroy the GPU Droplet after the session to avoid continued billing.

Benchmark Results

The first benchmark milestone used repeated non-streaming /v1/chat/completions requests against:

Qwen/Qwen2.5-0.5B-Instruct

Summarized results are documented in:

docs/benchmark-results.md

The initial benchmark captures approximate end-to-end completion tokens/sec, not streaming time-to-first-token or sustained concurrent throughput.

Operational Lessons

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
* Kubernetes

Possible future direction if GPU credits, capacity, and project scope allow:

* Run vLLM under k3s or managed Kubernetes
* Explore GPU allocation
* Add Kubernetes service exposure
* Investigate observability and autoscaling patterns

## Notes
This is a learning project. The goal is to understand each step rather than hide everything behind a prebuilt deployment.

The project intentionally starts simple with Docker before moving into more advanced serving, benchmarking, and infrastructure topics.
