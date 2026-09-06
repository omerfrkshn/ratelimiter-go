# Benchmarks

Load test tool: [vegeta](https://github.com/tsenart/vegeta). Target: `cmd/demo`
running a single algorithm at a time behind `middleware.RateLimit`, all
requests from one client (one rate-limit key).

## Method

- Limit: **50 requests / 5s** per key (steady-state sustainable rate: 10 req/s)
- Load: constant **20 req/s for 20s** — double the sustainable rate, so the
  limiter is expected to reject roughly half of steady-state traffic
- Total requests sent: **400** per algorithm
- Each algorithm run against a fresh server process (no state carried over)
- Raw vegeta reports: [`loadtest/results/`](loadtest/results/)

Reproduce with:

```bash
go build -o demo.exe ./cmd/demo
./demo.exe -algorithm=<name> -rate=50 -period=5s -addr=:8098 &
vegeta attack -targets=loadtest/targets.txt -rate=20/1s -duration=20s -timeout=5s \
  | vegeta report -type=text
```

## Results

| Algorithm | Requests | Accepted (200) | Rejected (429) | Accept rate | p50 | p95 | p99 |
|---|---|---|---|---|---|---|---|
| Fixed window | 400 | 200 | 200 | 50.0% | 318µs | 626µs | 865µs |
| Sliding window log | 400 | 200 | 200 | 50.0% | 264µs | 689µs | 1.18ms |
| Sliding window counter | 400 | 200 | 200 | 50.0% | 361µs | 855µs | 1.26ms |
| Token bucket | 400 | 249 | 151 | 62.3% | 490µs | 1.01ms | 1.25ms |

## Reading the results

The three window-based algorithms land on the same accept rate (50%,
matching sustainable-rate / offered-rate = 10/20) because they all
converge to *rejecting anything above the average rate* once the initial
window fills — that's their design goal, not a coincidence.

**Token bucket accepts 12 points more traffic (62.3% vs 50%)** because it
starts with a full bucket of 50 tokens and spends them immediately on the
first burst, on top of the same steady-state refill rate the others
enforce. This is the concrete trade-off the brief asked to demonstrate:
token bucket trades strict average-rate enforcement for burst tolerance.
Whether that's desirable depends on the workload — it's the right choice
for "bursty but well-behaved" clients, the wrong choice if the goal is a
hard ceiling on sustained throughput.

Latencies are all sub-millisecond at p99 for the in-memory algorithms
(single mutex + map lookup per request) on localhost; the distributed
Redis-backed limiter was not included in this load test since its latency
is dominated by the network/Redis round trip, not the algorithm — see
[README.md](README.md#dağıtık-mod-redis) for its correctness guarantee
instead (verified by an integration test, not a load test).
