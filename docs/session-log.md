# Session Log

This document captures hands-on test sessions for the `vllm-rocm-inference-lab` project.

The goal of these sessions is to learn how to run and operate vLLM on AMD ROCm GPU infrastructure, starting with a DigitalOcean AMD GPU Droplet and later moving toward Kubernetes-based deployment.

---

## Session 1: Initial AMD GPU Droplet Attempt

### Objective

Create an AMD GPU Droplet, clone the project repository, and run the initial script sequence to verify the environment and start vLLM.

### Result

The session did not progress past the vLLM startup phase.

### Issue Encountered

The selected Droplet image did not include the expected ROCm command-line tools.

The remote GPU verification script reported that ROCm tools such as `rocminfo` and `rocm-smi` were missing.

### Fix / Follow-up

For the next session, I changed the selected Droplet image to a ROCm-ready image:

```text
openai-gpt-oss-rocm-7
```

### Cleanup

The Droplet was destroyed after the session.

---

## Session 2: Successful vLLM ROCm Startup and API Test

### Objective

Run the prepared scripts on an AMD GPU Droplet and verify that vLLM can serve an OpenAI-compatible API using the ROCm Docker image.

### Environment

```text
Cloud provider: DigitalOcean
GPU type: AMD MI350X
GPU count: 1
VRAM: 288 GB
Image: openai-gpt-oss-rocm-7
Repository: https://github.com/psuri-github/vllm-rocm-inference-lab
```

### Scripts Run

The following scripts were run from the repository:

```bash
./00-local-preflight.sh
./01-remote-gpu-checks.sh
HOST_PORT=8002 ./02-run-vllm-rocm.sh
BASE_URL=http://localhost:8002 ./03-test-health.sh
BASE_URL=http://localhost:8002 ./04-test-chat-completion.sh
```

### Final Result

The session was successful.

vLLM started successfully in a ROCm Docker container, the health endpoint returned HTTP 200, and the OpenAI-compatible chat completion endpoint returned a valid model response.

### Health Endpoint Result

Command:

```bash
BASE_URL=http://localhost:8002 ./03-test-health.sh
```

Output:

```text
Checking vLLM health endpoint...
Base URL: http://localhost:8002
HTTP/1.1 200 OK
date: Fri, 05 Jun 2026 00:21:23 GMT
server: uvicorn
content-length: 0

OK: vLLM health endpoint is reachable.
```

### Chat Completion Result

Command:

```bash
BASE_URL=http://localhost:8002 ./04-test-chat-completion.sh
```

Output summary:

```text
Testing vLLM chat completion endpoint...
Base URL: http://localhost:8002
Model: Qwen/Qwen2.5-0.5B-Instruct

OK: chat completion request completed.
```

The response came from:

```text
Qwen/Qwen2.5-0.5B-Instruct
```

served by:

```text
vLLM 0.22.0
```

The API returned a valid `chat.completion` response with token usage information.

---

## Operational Issue 1: Host Port 8000 Already in Use

### Symptom

The first attempt to start the vLLM container failed with a Docker networking error.

Error:

```text
docker: Error response from daemon: failed to set up container networking:
driver failed programming external connectivity on endpoint vllm-rocm:
failed to bind host port for 0.0.0.0:8000:172.17.0.2:8000/tcp:
address already in use
```

### Diagnosis

The vLLM startup script originally attempted to map:

```text
host port 8000 -> container port 8000
```

I checked which process was using port 8000:

```bash
sudo ss -ltnp | grep ':8000'
```

This showed:

```text
caddy listening on host port 8000
```

I also inspected existing containers:

```bash
docker ps -a
```

The Droplet was not a blank machine. It already had preconfigured services running, including:

```text
caddy         listening on host port 8000
open-webui   mapped to 127.0.0.1:3000
rocm-gpt-oss mapped to 127.0.0.1:8001 -> container 8000
```

### Root Cause

The host port expected by my vLLM script was already occupied by Caddy.

Docker could not bind another container to the same host port.

### Workaround

I did not stop Caddy.

Instead, I relaunched my script using host port `8002`:

```bash
HOST_PORT=8002 ./02-run-vllm-rocm.sh
```

This mapped:

```text
Droplet localhost:8002 -> vLLM container port 8000
```

---

## Operational Issue 2: GPU Memory Already Consumed

### Symptom

After resolving the port conflict, the vLLM container reached the GPU but failed during engine initialization.

Root error from the logs:

```text
ValueError: Free memory on device cuda:0 (12.38/287.69 GiB) on startup
is less than desired GPU memory utilization (0.92, 264.67 GiB).
Decrease GPU memory utilization or reduce GPU memory used by other processes.
```

### Diagnosis

The error indicated that vLLM wanted approximately:

```text
264.67 GiB
```

of GPU memory, but only:

```text
12.38 GiB
```

was free.

This suggested that another process or container was already using most of the GPU memory.

From the earlier `docker ps -a` output, the likely existing GPU-consuming container was:

```text
rocm-gpt-oss
```

### Fix

I stopped the pre-existing model/UI containers:

```bash
docker stop rocm-gpt-oss
docker stop open-webui
```

Then I checked GPU usage:

```bash
rocm-smi
```

The output showed the GPU was idle:

```text
VRAM%  0%
GPU%   0%
```

I then removed the failed vLLM container:

```bash
docker rm -f vllm-rocm
```

and relaunched vLLM:

```bash
HOST_PORT=8002 ./02-run-vllm-rocm.sh
```

### Result

After freeing GPU memory, the vLLM container started successfully.

---

## Key Learnings

### 1. Preconfigured GPU images may not be blank machines

The selected ROCm-ready image already had services running, including Caddy, Open WebUI, and a pre-existing model container.

This affected both network port availability and GPU memory availability.

### 2. Port conflicts are easy to diagnose with `ss`

The command:

```bash
sudo ss -ltnp | grep ':8000'
```

quickly identified that Caddy was already listening on port 8000.

### 3. GPU memory must be checked before starting vLLM

vLLM may request a large fraction of GPU memory at startup.

If another container is already using the GPU, vLLM can fail during engine initialization even though ROCm and Docker are working.

### 4. `rocm-smi` is essential for AMD GPU debugging

`rocm-smi` helped confirm when the GPU was idle and ready for the vLLM workload.

### 5. Making ports configurable was useful

Because `02-run-vllm-rocm.sh` supports `HOST_PORT`, I was able to work around the port conflict without editing the script.

Example:

```bash
HOST_PORT=8002 ./02-run-vllm-rocm.sh
```

---

## Session 2 Final Status

```text
ROCm tools available: yes
Docker usable: yes
AMD GPU visible: yes
vLLM container started: yes
Health endpoint working: yes
Chat completion working: yes
Droplet destroyed after session: yes
```

---
## Session 3: Retesting Improved vLLM Startup Script

### Objective

Validate the improved `02-run-vllm-rocm.sh` script on an AMD GPU Droplet.

The script had been updated to include:

* Configurable host port via `HOST_PORT`
* Configurable GPU memory utilization via `GPU_MEMORY_UTILIZATION`
* Docker availability and usability checks
* ROCm/GPU prerequisite checks
* AMD GPU device file checks
* Host port availability check using `ss`
* GPU status printing before and immediately after starting vLLM

### Environment

```text
Cloud provider: DigitalOcean
Image: ROCm-based AMD GPU image
OS: Ubuntu 24.04.4 LTS
GPU: AMD GPU exposed through ROCm
Container image: vllm/vllm-openai-rocm:latest
Model: Qwen/Qwen2.5-0.5B-Instruct
vLLM version observed in logs: 0.22.1
```

### Script Sequence

The following scripts were run from the `scripts/` directory:

```bash
./00-local-preflight.sh
./01-remote-gpu-checks.sh
./02-run-vllm-rocm.sh
HOST_PORT=8002 GPU_MEMORY_UTILIZATION=0.60 ./02-run-vllm-rocm.sh
BASE_URL=http://localhost:8002 ./03-test-health.sh
BASE_URL=http://localhost:8002 ./04-test-chat-completion.sh
```

---

### Remote GPU Check Result

The remote GPU check script completed successfully enough to proceed.

Important observations:

```text
OS: Ubuntu 24.04.4 LTS
rocm-smi: found
docker: found
docker usability check: passed
/dev/kfd: exists
/dev/dri: exists
```

`rocminfo` was not installed:

```text
MISSING: rocminfo
```

However, the session still succeeded because `rocm-smi`, Docker, the AMD GPU device files, and the ROCm-compatible vLLM container were sufficient for this test.

`rocm-smi` showed the GPU was visible and idle before launch:

```text
VRAM%  0%
GPU%   0%
```

### Note on `rocminfo`

`rocminfo` is useful for deeper ROCm inspection, but this run showed it was not strictly required for the current Docker-based vLLM serving test.

---

## Operational Issue: Host Port 8000 Already in Use

### Symptom

The first attempt to run `02-run-vllm-rocm.sh` with the default host port failed before starting vLLM:

```text
ERROR: host port 8000 is still in use after removing vllm-rocm.
Another process or container is using this port.
```

This confirmed that the improved script correctly detected a host port conflict before attempting `docker run`.

### Diagnosis

I inspected the process using port 8000:

```bash
sudo ss -ltnp | grep ':8000'
```

Output showed Docker proxy processes bound to port 8000:

```text
LISTEN 0 4096 0.0.0.0:8000 0.0.0.0:* users:(("docker-proxy",pid=3291,fd=8))
LISTEN 0 4096 [::]:8000 [::]:* users:(("docker-proxy",pid=3297,fd=8))
```

I then checked running containers:

```bash
docker ps -a
```

The existing container was:

```text
NAMES: rocm
PORTS: 0.0.0.0:8000->8000/tcp, 0.0.0.0:8888->8888/tcp, 0.0.0.0:30000->30000/tcp
```

### Root Cause

The ROCm image already had a container named `rocm` running, and that container was using host port `8000`.

### Fix

I did not stop the existing `rocm` container. Instead, I used the script’s configurable port support:

```bash
HOST_PORT=8002 GPU_MEMORY_UTILIZATION=0.60 ./02-run-vllm-rocm.sh
```

This mapped:

```text
Droplet host port 8002 -> vLLM container port 8000
```

---

## vLLM Startup Result

The vLLM ROCm image was not already present locally, so Docker pulled it:

```text
Unable to find image 'vllm/vllm-openai-rocm:latest' locally
latest: Pulling from vllm/vllm-openai-rocm
...
Status: Downloaded newer image for vllm/vllm-openai-rocm:latest
```

The container started successfully:

```text
Container started: vllm-rocm
```

The script printed GPU status before and immediately after container launch.

At that exact moment, `rocm-smi` still showed low GPU activity:

```text
VRAM%  0%
GPU%   0%
```

This is expected because `docker run -d` starts the container in the background, and model loading continues after the command returns.

---

## vLLM Logs

I followed the container logs:

```bash
docker logs -f vllm-rocm
```

Important observations from the logs:

```text
vLLM version: 0.22.1
Model: Qwen/Qwen2.5-0.5B-Instruct
gpu_memory_utilization: 0.6
```

The logs showed that vLLM resolved the model architecture:

```text
Resolved architecture: Qwen2ForCausalLM
```

The model loaded successfully:

```text
Model loading took 0.93 GiB memory and 2.851300 seconds
```

vLLM reported available KV cache memory:

```text
Available KV cache memory: 168.58 GiB
GPU KV cache size: 14,730,720 tokens
Maximum concurrency for 32,768 tokens per request: 449.55x
```

The server started successfully:

```text
Starting vLLM server on http://0.0.0.0:8000
Application startup complete.
```

The logs also showed that the following routes were available:

```text
/health
/metrics
/v1/models
/v1/chat/completions
/v1/completions
```

---

## Health Endpoint Test

Command:

```bash
BASE_URL=http://localhost:8002 ./03-test-health.sh
```

Result:

```text
Checking vLLM health endpoint...
Base URL: http://localhost:8002
HTTP/1.1 200 OK
server: uvicorn
content-length: 0

OK: vLLM health endpoint is reachable.
```

This confirmed that the vLLM server was reachable through host port `8002`.

---

## Chat Completion Test

Command:

```bash
BASE_URL=http://localhost:8002 ./04-test-chat-completion.sh
```

Result:

```text
Testing vLLM chat completion endpoint...
Base URL: http://localhost:8002
Model: Qwen/Qwen2.5-0.5B-Instruct
OK: chat completion request completed.
```

The API returned a valid OpenAI-compatible `chat.completion` response.

Important response fields:

```text
object: chat.completion
model: Qwen/Qwen2.5-0.5B-Instruct
system_fingerprint: vllm-0.22.1-37ab569a
prompt_tokens: 38
completion_tokens: 100
total_tokens: 138
finish_reason: length
```

The `finish_reason` was `length`, which means the response stopped because the script requested:

```json
"max_tokens": 100
```

---

## Key Learnings

### 1. The improved startup script caught a real issue

The host port check correctly detected that port `8000` was already in use before Docker attempted to start the container.

This prevented a lower-level Docker bind error and gave a clearer remediation path.

### 2. Preconfigured GPU images may already have containers running

The Droplet image already had a container named `rocm` using port `8000`.

Before starting new inference workloads, it is useful to inspect:

```bash
docker ps -a
sudo ss -ltnp
rocm-smi
```

### 3. Configurable ports are useful

Because `HOST_PORT` was configurable, I was able to avoid the conflict without editing the script:

```bash
HOST_PORT=8002 GPU_MEMORY_UTILIZATION=0.60 ./02-run-vllm-rocm.sh
```

### 4. `docker run -d` returns before the model is fully ready

The script’s “immediately after starting vLLM” GPU status may not show final memory usage because model loading continues in the background.

The correct readiness signal came from the vLLM logs and the `/health` endpoint.

### 5. vLLM exposes useful operational endpoints

The logs showed several useful routes, including:

```text
/health
/metrics
/v1/models
/v1/chat/completions
```

The `/metrics` endpoint will be useful for a future observability milestone.

---

## Session 3 Final Status

```text
Remote GPU checks: passed with note that rocminfo was missing
Docker usable: yes
AMD GPU device files present: yes
Port conflict detected: yes
Port workaround successful: yes
vLLM ROCm image pulled: yes
vLLM container started: yes
Health endpoint reachable: yes
Chat completion successful: yes
```

---

