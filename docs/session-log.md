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

