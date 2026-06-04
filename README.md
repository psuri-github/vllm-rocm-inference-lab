# vLLM ROCm Inference Lab

This project is a hands-on learning lab for deploying and operating **vLLM on AMD ROCm GPU infrastructure**.

The first milestone uses a **DigitalOcean AMD MI300X GPU Droplet** to learn how to verify GPU access, run a ROCm-compatible vLLM container, and test the OpenAI-compatible API exposed by vLLM.

The longer-term goal is to extend this into a Kubernetes-based inference platform that explores GPU allocation, serving, autoscaling, and observability.

## Project Goals

This project is intended to build practical understanding of:

* AMD GPU infrastructure for LLM inference
* ROCm runtime verification
* Docker-based vLLM serving
* OpenAI-compatible inference APIs
* Safe workflows for ephemeral GPU cloud resources
* Future Kubernetes deployment of vLLM
* Future observability and autoscaling for inference workloads

## Current Scope

The current milestone focuses on running vLLM directly on an AMD GPU Droplet using Docker.

The project currently includes scripts to:

* Check required local tools
* Verify ROCm and AMD GPU access on a remote Droplet
* Start a vLLM ROCm Docker container
* Test the vLLM health endpoint
* Send a chat completion request to the vLLM OpenAI-compatible API

## Why AMD ROCm?

Most GPU inference tutorials assume NVIDIA CUDA. This project intentionally explores the AMD ROCm path using AMD MI300X infrastructure.

This helps build a better understanding of vendor-specific GPU enablement, including:

* AMD ROCm runtime
* AMD GPU device files such as `/dev/kfd` and `/dev/dri`
* ROCm-compatible vLLM Docker images
* GPU access from inside containers

## Repository Structure

```text
.
├── README.md
├── docs
│   └── session-log.md
└── scripts
    ├── 00-local-preflight.sh
    ├── 01-remote-gpu-checks.sh
    ├── 02-run-vllm-rocm.sh
    ├── 03-test-health.sh
    └── 04-test-chat-completion.sh
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

* `rocminfo`
* `rocm-smi`
* Docker
* `/dev/kfd`
* `/dev/dri`

This script helps confirm that the Droplet is ready for ROCm-based vLLM serving.

### `02-run-vllm-rocm.sh`

Runs on the AMD GPU Droplet.

Starts the vLLM ROCm Docker container after verifying that Docker and ROCm/GPU prerequisites are present.

The default model is:

```text
Qwen/Qwen2.5-0.5B-Instruct
```

The default Docker image is:

```text
vllm/vllm-openai-rocm:latest
```

### `03-test-health.sh`

Tests the vLLM health endpoint:

```text
/health
```

This script can run either on the Droplet or on the local machine when using SSH port forwarding.

### `04-test-chat-completion.sh`

Sends a test request to the vLLM OpenAI-compatible chat completions endpoint:

```text
/v1/chat/completions
```

This verifies that the model server is reachable and able to generate a response.

## Cost Safety

GPU Droplets are billed while they exist, even if they are powered off.

The intended workflow is:

```text
Create GPU Droplet
Run verification and vLLM tests
Save notes and results
Destroy GPU Droplet
```

Do not rely on powering off the Droplet to stop billing.

## Recommended Workflow

### 1. Run local preflight checks

```bash
./scripts/00-local-preflight.sh
```

### 2. Create AMD MI300X GPU Droplet

Create the Droplet only when ready to run a focused GPU test session.

### 3. SSH into the Droplet

```bash
ssh root@<DROPLET_PUBLIC_IP>
```

### 4. Clone this repository on the Droplet

```bash
git clone https://github.com/psuri-github/vllm-rocm-inference-lab.git
cd vllm-rocm-inference-lab
chmod +x scripts/*.sh
```

### 5. Verify GPU and ROCm environment

```bash
./scripts/01-remote-gpu-checks.sh
```

### 6. Start vLLM

```bash
./scripts/02-run-vllm-rocm.sh
```

### 7. Follow container logs

```bash
docker logs -f vllm-rocm
```

### 8. Use SSH port forwarding from local machine

From the local machine:

```bash
ssh -L 8000:localhost:8000 root@<DROPLET_PUBLIC_IP>
```

Keep this terminal open.

### 9. Test the health endpoint

From another local terminal:

```bash
./scripts/03-test-health.sh
```

### 10. Test chat completion

```bash
./scripts/04-test-chat-completion.sh
```

### 11. Record results

Update:

```text
docs/session-log.md
```

Capture what worked, what failed, and what needs to be improved.

### 12. Destroy the Droplet

After the session, destroy the GPU Droplet to avoid continued billing.

## Current Milestone

Milestone 1:

* [x] Create local preflight script
* [x] Create remote GPU verification script
* [x] Create vLLM ROCm container startup script
* [x] Create health endpoint test script
* [x] Create chat completion test script
* [ ] Run scripts on AMD MI300X GPU Droplet
* [ ] Verify vLLM starts successfully
* [ ] Verify chat completion response
* [ ] Record first session findings

## Future Milestones

### Milestone 2: Improve Docker-based serving

* Add better container lifecycle management
* Add log collection
* Add model override examples
* Add basic latency measurement
* Add cleanup script

### Milestone 3: Move to Kubernetes

* Install k3s on the GPU Droplet
* Deploy vLLM as a Kubernetes Deployment
* Request AMD GPU resources from Kubernetes
* Expose vLLM through a Kubernetes Service

### Milestone 4: Observability

* Add Prometheus
* Add Grafana
* Track GPU and inference metrics
* Document useful metrics for LLM serving

### Milestone 5: Autoscaling

* Explore KEDA or HPA-based scaling
* Compare CPU-based scaling with inference-aware scaling signals
* Document the challenges of autoscaling GPU inference workloads

## Notes

This is a learning project. The goal is to understand each step rather than hide everything behind a prebuilt deployment.

The project intentionally starts simple with Docker before moving to Kubernetes.

