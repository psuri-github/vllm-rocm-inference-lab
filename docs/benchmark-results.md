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

