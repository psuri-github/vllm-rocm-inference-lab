# Benchmark Results

These benchmarks were run as part of a hands-on vLLM ROCm inference lab using an AMD GPU Droplet.

The goal is to understand serving behavior across different request patterns:

* single request
* repeated sequential requests
* same-prompt concurrency
* sequential prompt-suite requests
* mixed-prompt concurrency
* capped-output versus natural-completion behavior

These are learning benchmarks for this specific setup. They should not be treated as general-purpose vLLM, AMD GPU, or ROCm performance claims.

## Measurement Notes
`completion_tokens_per_sec` is calculated per request as:

completion_tokens / elapsed_seconds

For concurrent benchmarks, `aggregate_completion_tokens_per_sec` is calculated as:

total_completion_tokens / benchmark_wall_clock_seconds

`finish_reason=length` means generation stopped because the request reached the configured `max_tokens` limit.

`finish_reason=stop` means the model stopped naturally or reached a stop condition before hitting the configured `max_tokens` limit.

The benchmark scripts use non-streaming `/v1/chat/completions` requests. These measurements do not capture streaming time-to-first-token, p95/p99 latency over repeated trials, or sustained production throughput.


## MAX_TOKENS=100: Capped-Output Benchmarks

### Single Request Benchmark: Qwen/Qwen2.5-0.5B-Instruct

| Date       | Runs | Model                      | Max Tokens | Prompt Tokens | Completion Tokens | Avg Elapsed ms | Min ms | Max ms | Avg Completion Tokens/sec | Finish Reason |
| ---------- | ---: | -------------------------- | ---------: | ------------: | ----------------: | -------------: | -----: | -----: | ------------------------: | ------------- |
| 2026-06-08 |   11 | Qwen/Qwen2.5-0.5B-Instruct |        100 |            38 |               100 |         134.09 |    133 |    135 |                    745.67 | length        |

### Test Setup

* Endpoint: `/v1/chat/completions`
* Base URL: `http://localhost:8002`
* Serving stack: vLLM ROCm Docker container
* Prompt: `Explain GPU inference in one short paragraph.`
* `max_tokens`: `100`
* Temperature: `0`

#### Notes

This benchmark used repeated non-streaming `/v1/chat/completions` requests against vLLM running with the ROCm Docker image on an AMD GPU Droplet.

`completion_tokens_per_sec` is an approximate end-to-end value calculated as:

```text
completion_tokens / elapsed_seconds
```

This is not yet a streaming time-to-first-token or sustained throughput benchmark.

All runs ended with `finish_reason=length`, meaning generation stopped because the request used `max_tokens=100`.

The raw result file initially contained pretty-printed JSON records and later switched to compact JSONL after updating the benchmark script to use `jq -c -n`. Going forward, benchmark result records should be written as one JSON object per line.

### Sequential Repeated-Request Benchmark: Qwen/Qwen2.5-0.5B-Instruct

| Date       | Runs | Model                      | Prompt Tokens | Completion Tokens / Run | Avg Elapsed ms | Min ms | Max ms | Avg Completion Tokens/sec | Finish Reason |
|------------|-----:|----------------------------|--------------:|------------------------:|---------------:|-------:|-------:|--------------------------:|---------------|
| 2026-06-09 | 11   | Qwen/Qwen2.5-0.5B-Instruct |            38 |                     100 |         154.55 |    153 |    158 |                    647.23 |        length |

#### Notes

This benchmark ran repeated sequential non-streaming `/v1/chat/completions` requests against the same vLLM ROCm endpoint.

The model, prompt, `max_tokens`, and base URL were kept constant across all runs.

The results were stable across 11 runs, with elapsed request time ranging from 153 ms to 158 ms. Each run generated 100 completion tokens and ended with `finish_reason=length`, meaning generation stopped because the configured `max_tokens=100` limit was reached.

`completion_tokens_per_sec` is an approximate end-to-end value calculated as:

```text
completion_tokens / elapsed_seconds
```

### Start-Gated Concurrent Request Benchmark

| Date       | Concurrency | Successful Requests | Failed Requests | Wall-clock ms | Total Completion Tokens | Aggregate Completion Tokens/sec | Avg Request Latency ms | Min ms | Max ms |
|------------|------------:|--------------------:|----------------:|--------------:|------------------------:|--------------------------------:|-----------------------:|-------:|-------:|
| 2026-06-10 |           2 |                   2 |               0 |           173 |                     200 |                         1156.07 |                 152.00 |    152 |    152 |
| 2026-06-10 |           4 |                   4 |               0 |           192 |                     400 |                         2083.33 |                 169.00 |    169 |    169 |
| 2026-06-10 |           6 |                   6 |               0 |           228 |                     600 |                         2631.58 |                 204.33 |    204 |    205 |
| 2026-06-10 |           8 |                   8 |               0 |           231 |                     800 |                         3463.20 |                 206.63 |    205 |    208 |
| 2026-06-10 |          10 |                  10 |               0 |           217 |                    1000 |                         4608.29 |                 192.20 |    190 |    193 |

#### Notes

This benchmark used a start-gated concurrent launcher. Worker processes were created first and then released together by creating a shared start-gate file. Each request wrote to a separate result file, and the parent benchmark script combined successful results after all workers completed.

The same model, prompt, and `max_tokens=100` setting were used across all requests.

In this run, all requests completed successfully through concurrency 10 with no failures. The measured aggregate completion-token rate increased as concurrency increased, reaching approximately 4.6K completion tokens/sec for this specific model, prompt, machine, and benchmark harness.

Average per-request latency increased compared with sequential single-request runs, which is expected under concurrent load. This benchmark does not yet measure p95/p99 latency across repeated trials or sustained throughput over a longer time window.

### Sequential Prompt-Suite Benchmark

| Date       | Prompt ID      | Prompt Tokens | Completion Tokens | Total Tokens | Elapsed ms | Completion Tokens/sec | Finish Reason |
|------------|----------------|--------------:|------------------:|-------------:|-----------:|----------------------:|---------------|
| 2026-06-10 | short_explain  |            38 |               100 |          138 |        135 |                740.74 | length        |
| 2026-06-10 | long_explain   |            53 |               100 |          153 |        134 |                746.27 | length        |
| 2026-06-10 | debugging      |            51 |               100 |          151 |        135 |                740.74 | length        |
| 2026-06-10 | summarization  |            88 |               100 |          188 |        137 |                729.93 | stop          |
| 2026-06-10 | code_explain   |            79 |               100 |          179 |        135 |                740.74 | length        |
| 2026-06-10 | step_by_step   |            48 |               100 |          148 |        133 |                751.88 | length        |

#### Notes

This benchmark ran one request per prompt from `benchmarks/prompts.jsonl` using the same model, endpoint, and `max_tokens=100`.

The run completed successfully for all 6 prompts with no failures. Prompt sizes ranged from 38 to 88 prompt tokens, while elapsed request time remained tightly grouped between 133 ms and 137 ms.

Most prompts ended with `finish_reason=length`, meaning generation reached the configured `max_tokens=100` limit. The summarization prompt returned `finish_reason=stop`.

This benchmark is sequential. It measures different prompt shapes one after another, not concurrent mixed-prompt serving.

### Mixed-Prompt Concurrent Benchmark

| Date       | Total Requests | Prompt Definitions | Requests / Prompt | Failed Requests | Wall-clock ms | Total Completion Tokens | Aggregate Completion Tokens/sec | Avg Request Latency ms | Min ms | Max ms |
|------------|---------------:|-------------------:|------------------:|----------------:|--------------:|------------------------:|--------------------------------:|-----------------------:|-------:|-------:|
| 2026-06-10 |             12 |                  6 |                 2 |               0 |           234 |                    1200 |                         5128.21 |                 197.75 |    196 |    199 |


#### Per-Prompt Summary

| Prompt ID      | Count | Avg Prompt Tokens | Avg Elapsed ms | Min ms | Max ms | Avg Completion Tokens/sec |
|----------------|------:|------------------:|---------------:|-------:|-------:|--------------------------:|
| code_explain   |     2 |                79 |          198.0 |    197 |    199 |                    505.06 |
| debugging      |     2 |                51 |          198.0 |    197 |    199 |                    505.06 |
| long_explain   |     2 |                53 |          197.5 |    196 |    199 |                    506.36 |
| short_explain  |     2 |                38 |          198.5 |    198 |    199 |                    503.78 |
| step_by_step   |     2 |                48 |          197.5 |    197 |    198 |                    506.33 |
| summarization  |     2 |                88 |          197.0 |    197 |    197 |                    507.61 |

#### Notes

This benchmark used a mixed-prompt concurrent workload. Six prompt definitions were loaded from `benchmarks/prompts.jsonl`, and 12 concurrent requests were launched using round-robin prompt assignment.

Each prompt was used twice. All 12 requests completed successfully with no failures.

The benchmark used a start-gate file to launch all workers first and release them together. Each worker wrote to a separate temporary result file, and the parent script combined successful results into a single JSONL output file.

Compared with the sequential prompt-suite benchmark, per-request latency increased under concurrency, but aggregate completion throughput increased significantly. Prompt token counts ranged from 38 to 88, but request latency remained tightly grouped between 196 ms and 199 ms in this run.

This benchmark measures an all-at-once mixed-prompt concurrency scenario. It does not yet measure sustained traffic over time or p95/p99 latency across repeated trials.

## MAX_TOKENS=256: Higher-Cap Benchmarks

### Single Request Benchmark

| Date       | Max Tokens | Prompt Tokens | Completion Tokens | Total Tokens | Elapsed ms | Completion Tokens/sec | Finish Reason |
|------------|-----------:|--------------:|------------------:|-------------:|-----------:|----------------------:|---------------|
| 2026-06-11 |        256 |            38 |               232 |          270 |        291 |                797.25 | stop          |

#### Notes

For the default single prompt, `MAX_TOKENS=256` was enough for the model to complete naturally. The response ended with `finish_reason=stop`.

### Sequential Prompt-Suite Benchmark

| Date       | Prompt ID     | Prompt Tokens | Completion Tokens | Total Tokens | Elapsed ms | Completion Tokens/sec | Finish Reason |
|------------|---------------|--------------:|------------------:|-------------:|-----------:|----------------------:|---------------|
| 2026-06-11 | short_explain |            38 |               232 |          270 |        291 |                797.25 | stop          |
| 2026-06-11 | long_explain  |            53 |               256 |          309 |        318 |                805.03 | length        |
| 2026-06-11 | debugging     |            51 |               256 |          307 |        317 |                807.57 | length        |
| 2026-06-11 | summarization |            88 |                84 |          172 |        110 |                763.64 | stop          |
| 2026-06-11 | code_explain  |            79 |               256 |          335 |        318 |                805.03 | length        |
| 2026-06-11 | step_by_step  |            48 |               256 |          304 |        381 |                671.92 | length        |

#### Summary

| Runs | Avg Elapsed ms | Min ms | Max ms | Avg Prompt Tokens | Avg Completion Tokens/sec |
|-----:|---------------:|-------:|-------:|------------------:|--------------------------:|
|    6 |         289.17 |    110 |    381 |              59.5 |                    775.07 |

#### Notes

With `MAX_TOKENS=256`, two prompts stopped naturally:

- `short_explain`
- `summarization`

Four prompts still reached the configured token limit:

- `long_explain`
- `debugging`
- `code_explain`
- `step_by_step`

This means `MAX_TOKENS=256` is a higher-cap benchmark, but not a fully natural-completion benchmark for this prompt set.

### Mixed-Prompt Concurrent Benchmark

| Date       | Max Tokens | Total Requests | Prompt Definitions | Requests / Prompt | Failed Requests | Wall-clock ms | Total Completion Tokens | Aggregate Completion Tokens/sec | Avg Request Latency ms | Min ms | Max ms |
|------------|-----------:|---------------:|-------------------:|------------------:|----------------:|--------------:|------------------------:|--------------------------------:|-----------------------:|-------:|-------:|
| 2026-06-11 |        256 |             12 |                  6 |                 2 |               0 |           500 |                    2748 |                         5496.00 |                 420.33 |    223 |    470 |

#### Notes

With `MAX_TOKENS=256`, mixed-prompt concurrency completed successfully with no failures.

In this run, the benchmark generated 2748 completion tokens across 12 concurrent requests, with a measured aggregate completion-token rate of approximately 5496 tokens/sec for this specific setup.

However, because several prompts still reached `finish_reason=length` in the sequential prompt-suite run, this setting should not be treated as fully natural-completion for this prompt set.

## MAX_TOKENS=1024: Natural-Completion Benchmarks

### Single Request Benchmark

| Date       | Max Tokens | Prompt Tokens | Completion Tokens | Total Tokens | Elapsed ms | Completion Tokens/sec | Finish Reason |
|------------|-----------:|--------------:|------------------:|-------------:|-----------:|----------------------:|---------------|
| 2026-06-11 |       1024 |            38 |               232 |          270 |        292 |                794.52 | stop          |

#### Notes

For the default single prompt, increasing from `MAX_TOKENS=256` to `MAX_TOKENS=1024` did not change the completion length. The model still stopped naturally at 232 completion tokens.

### Sequential Prompt-Suite Benchmark

| Date       | Prompt ID     | Prompt Tokens | Completion Tokens | Total Tokens | Elapsed ms | Completion Tokens/sec | Finish Reason |
|------------|---------------|--------------:|------------------:|-------------:|-----------:|----------------------:|---------------|
| 2026-06-11 | short_explain |            38 |               232 |          270 |        291 |                797.25 | stop          |
| 2026-06-11 | long_explain  |            53 |               522 |          575 |        639 |                816.90 | stop          |
| 2026-06-11 | debugging     |            51 |               626 |          677 |        764 |                819.37 | stop          |
| 2026-06-11 | summarization |            88 |               100 |          188 |        129 |                775.19 | stop          |
| 2026-06-11 | code_explain  |            79 |               403 |          482 |        495 |                814.14 | stop          |
| 2026-06-11 | step_by_step  |            48 |               420 |          468 |        514 |                817.12 | stop          |

#### Summary

| Runs | Avg Elapsed ms | Min ms | Max ms | Avg Prompt Tokens | Avg Completion Tokens/sec |
|-----:|---------------:|-------:|-------:|------------------:|--------------------------:|
|    6 |         472.00 |    129 |    764 |              59.5 |                    806.66 |

#### Notes

With `MAX_TOKENS=1024`, all six prompts ended with `finish_reason=stop`.

This suggests that, for the current prompt suite, MAX_TOKENS=1024 is a more appropriate cap when the goal is to observe natural completion behavior rather than capped-output behavior.

The longer prompts generated substantially more output than in the `MAX_TOKENS=256` run:

- `long_explain`: 522 completion tokens
- `debugging`: 626 completion tokens
- `code_explain`: 403 completion tokens
- `step_by_step`: 420 completion tokens

### Mixed-Prompt Concurrent Benchmark

| Date       | Max Tokens | Total Requests | Prompt Definitions | Requests / Prompt | Failed Requests | Wall-clock ms | Total Completion Tokens | Aggregate Completion Tokens/sec | Avg Request Latency ms | Min ms | Max ms |
|------------|-----------:|---------------:|-------------------:|------------------:|----------------:|--------------:|------------------------:|--------------------------------:|-----------------------:|-------:|-------:|
| 2026-06-11 |       1024 |             12 |                  6 |                 2 |               0 |          1230 |                    4610 |                         3747.97 |                 696.67 |    154 |   1210 |

#### Notes

With `MAX_TOKENS=1024`, the mixed-prompt concurrent benchmark generated 4610 completion tokens across 12 concurrent requests.

Compared with the `MAX_TOKENS=256` mixed-prompt run:

| Max Tokens | Wall-clock ms | Total Completion Tokens | Aggregate Completion Tokens/sec | Avg Request Latency ms |
|-----------:|--------------:|------------------------:|--------------------------------:|-----------------------:|
|        256 |           500 |                    2748 |                         5496.00 |                 420.33 |
|       1024 |          1230 |                    4610 |                         3747.97 |                 696.67 |

The `MAX_TOKENS=1024` run produced longer natural completions, which increased wall-clock time and average request latency. Aggregate tokens/sec was lower than the `MAX_TOKENS=256` run because the workload included longer completions and a longer tail.

This does not mean `MAX_TOKENS=1024` is worse. It means the benchmark workload changed from a partially capped workload to a more natural-completion workload.

---

## Current Recommendation

Use different `MAX_TOKENS` settings depending on the benchmark goal:

|                      Goal                             | Recommended Setting |
|-------------------------------------------------------|---------------------|
|                 Quick smoke test                      |  `MAX_TOKENS=100`   |
|           Controlled capped-output benchmark          |  `MAX_TOKENS=100`   |
|           Higher-cap exploratory benchmark            |  `MAX_TOKENS=256`   |
| Natural-completion benchmark for current prompt suite |  `MAX_TOKENS=1024`  |

For future prompt-suite and mixed-prompt benchmarks, prefer:

```bash
MAX_TOKENS=1024
```

when the goal is to observe natural completion behavior for the current prompt suite.

For quick operational tests, continue to use smaller values such as:

```bash
MAX_TOKENS=100
```

to reduce runtime and cost.
