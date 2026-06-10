# Benchmark Results

## Single Request Benchmark: Qwen/Qwen2.5-0.5B-Instruct

| Date       | Runs | Model                      | Max Tokens | Prompt Tokens | Completion Tokens | Avg Elapsed ms | Min ms | Max ms | Avg Completion Tokens/sec | Finish Reason |
| ---------- | ---: | -------------------------- | ---------: | ------------: | ----------------: | -------------: | -----: | -----: | ------------------------: | ------------- |
| 2026-06-08 |   11 | Qwen/Qwen2.5-0.5B-Instruct |        100 |            38 |               100 |         134.09 |    133 |    135 |                    745.67 | length        |

## Test Setup

* Endpoint: `/v1/chat/completions`
* Base URL: `http://localhost:8002`
* Serving stack: vLLM ROCm Docker container
* Prompt: `Explain GPU inference in one short paragraph.`
* `max_tokens`: `100`
* Temperature: `0`

## Notes

This benchmark used repeated non-streaming `/v1/chat/completions` requests against vLLM running with the ROCm Docker image on an AMD GPU Droplet.

`completion_tokens_per_sec` is an approximate end-to-end value calculated as:

```text
completion_tokens / elapsed_seconds
```

This is not yet a streaming time-to-first-token or sustained throughput benchmark.

All runs ended with `finish_reason=length`, meaning generation stopped because the request used `max_tokens=100`.

The raw result file initially contained pretty-printed JSON records and later switched to compact JSONL after updating the benchmark script to use `jq -c -n`. Going forward, benchmark result records should be written as one JSON object per line.

## Sequential Repeated-Request Benchmark: Qwen/Qwen2.5-0.5B-Instruct

| Date       | Runs | Model                      | Prompt Tokens | Completion Tokens / Run | Avg Elapsed ms | Min ms | Max ms | Avg Completion Tokens/sec | Finish Reason |
|------------|-----:|----------------------------|--------------:|------------------------:|---------------:|-------:|-------:|--------------------------:|---------------|
| 2026-06-09 | 11   | Qwen/Qwen2.5-0.5B-Instruct |            38 |                     100 |         154.55 |    153 |    158 |                    647.23 |        length |

### Notes

This benchmark ran repeated sequential non-streaming `/v1/chat/completions` requests against the same vLLM ROCm endpoint.

The model, prompt, `max_tokens`, and base URL were kept constant across all runs.

The results were stable across 11 runs, with elapsed request time ranging from 153 ms to 158 ms. Each run generated 100 completion tokens and ended with `finish_reason=length`, meaning generation stopped because the configured `max_tokens=100` limit was reached.

`completion_tokens_per_sec` is an approximate end-to-end value calculated as:

```text
completion_tokens / elapsed_seconds
```

## Start-Gated Concurrent Request Benchmark

| Date       | Concurrency | Successful Requests | Failed Requests | Wall-clock ms | Total Completion Tokens | Aggregate Completion Tokens/sec | Avg Request Latency ms | Min ms | Max ms |
|------------|------------:|--------------------:|----------------:|--------------:|------------------------:|--------------------------------:|-----------------------:|-------:|-------:|
| 2026-06-10 |           2 |                   2 |               0 |           173 |                     200 |                         1156.07 |                 152.00 |    152 |    152 |
| 2026-06-10 |           4 |                   4 |               0 |           192 |                     400 |                         2083.33 |                 169.00 |    169 |    169 |
| 2026-06-10 |           6 |                   6 |               0 |           228 |                     600 |                         2631.58 |                 204.33 |    204 |    205 |
| 2026-06-10 |           8 |                   8 |               0 |           231 |                     800 |                         3463.20 |                 206.63 |    205 |    208 |
| 2026-06-10 |          10 |                  10 |               0 |           217 |                    1000 |                         4608.29 |                 192.20 |    190 |    193 |

### Notes

This benchmark used a start-gated concurrent launcher. Worker processes were created first and then released together by creating a shared start-gate file. Each request wrote to a separate result file, and the parent benchmark script combined successful results after all workers completed.

The same model, prompt, and `max_tokens=100` setting were used across all requests.

The results show successful concurrent serving through concurrency 10 with no request failures. Aggregate completion throughput increased as concurrency increased, reaching approximately 4.6K completion tokens/sec at concurrency 10.

Average per-request latency increased compared with sequential single-request runs, which is expected under concurrent load. This benchmark does not yet measure p95/p99 latency across repeated trials or sustained throughput over a longer time window.

## Sequential Prompt-Suite Benchmark

| Date       | Prompt ID      | Prompt Tokens | Completion Tokens | Total Tokens | Elapsed ms | Completion Tokens/sec | Finish Reason |
|------------|----------------|--------------:|------------------:|-------------:|-----------:|----------------------:|---------------|
| 2026-06-10 | short_explain  |            38 |               100 |          138 |        135 |                740.74 | length        |
| 2026-06-10 | long_explain   |            53 |               100 |          153 |        134 |                746.27 | length        |
| 2026-06-10 | debugging      |            51 |               100 |          151 |        135 |                740.74 | length        |
| 2026-06-10 | summarization  |            88 |               100 |          188 |        137 |                729.93 | stop          |
| 2026-06-10 | code_explain   |            79 |               100 |          179 |        135 |                740.74 | length        |
| 2026-06-10 | step_by_step   |            48 |               100 |          148 |        133 |                751.88 | length        |

### Notes

This benchmark ran one request per prompt from `benchmarks/prompts.jsonl` using the same model, endpoint, and `max_tokens=100`.

The run completed successfully for all 6 prompts with no failures. Prompt sizes ranged from 38 to 88 prompt tokens, while elapsed request time remained tightly grouped between 133 ms and 137 ms.

Most prompts ended with `finish_reason=length`, meaning generation reached the configured `max_tokens=100` limit. The summarization prompt returned `finish_reason=stop`.

This benchmark is sequential. It measures different prompt shapes one after another, not concurrent mixed-prompt serving.
