# Sessions log

## Session 1 
 
Date: 
 
Droplet: 
- Region: 
- GPU: 
- Image: 
- Start time: 
- End time: 
 
Commands run: 
 
Results: not able to run script 02 onwards 
 
Errors: No rocm tools found 
 
Fixes: Changed Image deployed for container to openai-gpt-oss-rocm-7 
 
Did I destroy the Droplet? Yes 

FUNDS REMAINING : $249.63

## Session 2 
 
Date: 
 
Droplet: 
- Region: 
- GPU: 
- Image: 
- Start time: 
- End time: 
 
Commands run: ran all script 00-* to 04* 
 
Results: after all intermediate errors fixed

oot@120b---ROCm-7-gpu-mi350x1-288gb-devcloud-atl1:~/vllm-rocm-inference-lab/scripts# curl -i http://localhost:8002/health
HTTP/1.1 200 OK
date: Fri, 05 Jun 2026 00:20:35 GMT
server: uvicorn
content-length: 0

root@120b---ROCm-7-gpu-mi350x1-288gb-devcloud-atl1:~/vllm-rocm-inference-lab/scripts# BASE_URL=http://localhost:8002 ./03-test-health.sh
Checking vLLM health endpoint...
Base URL: http://localhost:8002
HTTP/1.1 200 OK
date: Fri, 05 Jun 2026 00:21:23 GMT
server: uvicorn
content-length: 0


OK: vLLM health endpoint is reachable.
root@120b---ROCm-7-gpu-mi350x1-288gb-devcloud-atl1:~/vllm-rocm-inference-lab/scripts# BASE_URL=http://localhost:8002 ./04-test-chat-completion.sh
Testing vLLM chat completion endpoint...
Base URL: http://localhost:8002
Model:    Qwen/Qwen2.5-0.5B-Instruct
{"id":"chatcmpl-9a1b79a43a0afc18","object":"chat.completion","created":1780618908,"model":"Qwen/Qwen2.5-0.5B-Instruct","choices":[{"index":0,"message":{"role":"assistant","content":"GPU (Graphics Processing Unit) inference refers to the use of graphics processing units (GPUs) for performing machine learning tasks on large datasets or complex models. GPUs are designed to handle parallel computations and can significantly speed up the training process for deep neural networks, which are commonly used in image recognition, computer vision, and other fields requiring high computational power.\n\nWhen using GPUs for inference, the model is first converted into a format that can be executed directly on the GPU. This involves converting the model's weights","refusal":null,"annotations":null,"audio":null,"function_call":null,"tool_calls":[],"reasoning":null},"logprobs":null,"finish_reason":"length","stop_reason":null,"token_ids":null,"routed_experts":null}],"service_tier":null,"system_fingerprint":"vllm-0.22.0-d254c4e0","usage":{"prompt_tokens":38,"total_tokens":138,"completion_tokens":100,"prompt_tokens_details":null},"prompt_logprobs":null,"prompt_token_ids":null,"prompt_text":null,"kv_transfer_params":null}
OK: chat completion request completed.
 
 
Errors:
FIRST ERROR
-----------
Initially could not connect to port 8000 since it was being used by another process. Discovered this from the logs [docker logs vllm-rocm]
got below error docker: Error response from daemon: failed to set up container networking: driver failed programming external connectivity on endpoint vllm-rocm (66a9a68ac1cd4d67f443553f2efb52157e58131b6cd8362e0f0aab990082bb77): failed to bind host port for 0.0.0.0:8000:172.17.0.2:8000/tcp: address already in use
 
Debugging done:
Ran below two commands to gather data
1. sudo ss -ltnp | grep ':8000'
2. docker ps -a

Those two commands told the docker is not a blank machine and it already had the below services running
caddy          listening on host port 8000
open-webui     mapped to 127.0.0.1:3000
rocm-gpt-oss   mapped to 127.0.0.1:8001 -> container 8000

The script tried connecting 0.0.0.0:8000 -> container:8000 but Caddy was already listening on port 8000 Hence Docker could not use port 8000


FIRST WORKAROUND : Do not kill Caddy, use port 8002 instead of 8000. So launched script with command "HOST_PORT=8002 ./02-run-vllm-rocm.sh"

 
SECOND ERROR
------------
vLLM container reached the GPU, but failed because almost all GPU memory is already being used by another process/container. The root error from logs
Free memory on device cuda:0 (12.38/287.69 GiB) on startup is less than desired GPU memory utilization (0.92, 264.67 GiB). Decrease GPU memory utilization or reduce GPU memory used by other processes.

Debugging Done
From the logs it was evident that vLLM wanted about 264.67 GiB of GPU memory, but only 12.38 GiB was free. The container rocm-gpt-oss was already using the GPU

SECOND WORKAROUND : stop the existing model container rocm-gpt-oss and open-webui. After stopping the containers confirmed GPU memory is free by running command "rocm-smi"

Once memory free-up was confirmed remove the failed container with command "docker rm -f vllm-rocm" and then re-launch the container using command "HOST_PORT=8002 ./02-run-vllm-rocm.sh"

Fixes: invoked 02-run-vllm-rocm.sh script on HOST_PORT 8002 and freed GPU Memory by stopping rocm-gpt-oss and open-webui
 
Did I destroy the Droplet? Yes 

FUNDS REMAINING : $247.56

