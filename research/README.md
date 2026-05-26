# Performance Research And Benchmarking

This directory contains benchmark scripts, load tests, raw results, and analysis materials for Zenthril.

The goal is to make performance and security-related claims reproducible. Every published number should be tied to a test script, a hardware profile, and raw output.

## Research Goals

1. Measure WebSocket throughput and latency under controlled load.
2. Measure encryption and ratchet-related overhead.
3. Measure PostgreSQL query behavior for core messaging flows.
4. Evaluate reliability under restart and dependency-failure scenarios.
5. Compare architectural tradeoffs with Matrix-like, Signal-like, and Discord-like systems.

## Running Go Benchmarks

From the repository root:

```bash
cd backend
go test -bench=. -benchmem ./benchmarks/...
```

Focused runs:

```bash
# Encryption-related benchmarks
go test -bench=Encryption -benchmem ./benchmarks/

# WebSocket benchmarks
go test -bench=WebSocket -benchmem ./benchmarks/

# Database benchmarks require DB_URL
DB_URL="postgres://user:pass@localhost:5432/zenthril?sslmode=disable" go test -bench=Database -benchmem ./benchmarks/
```

## Running k6 Load Tests

```bash
cd research/load_test
k6 run k6_load_test.js
```

Custom run:

```bash
k6 run --vus 100 --duration 5m --out json=../results/k6_run.json k6_load_test.js
```

## Result Storage

Raw and summarized results should be stored under `research/results/`.

Each result should include:

- Git commit hash.
- Hardware and operating system.
- Go, Node, Docker, PostgreSQL, Redis, and k6 versions.
- Environment variables that affect behavior, with secrets redacted.
- Number of users, messages, channels, and connections.
- Test duration and warm-up period.
- P50, P95, P99 latency where applicable.
- Error rate and timeout count.

## Current Example Results

Example single-node results measured on Intel Core i7-12700K with 32GB RAM:

- Peak WebSocket throughput: about 14,800 messages/sec.
- P95 latency at 500 concurrent users: about 98ms.

These numbers are useful as local research data, not as a universal production guarantee. Hardware, OS scheduling, database settings, network path, and workload shape can significantly change results.

## Metrics To Track

### WebSocket

- Active connections.
- Total connections.
- Messages per second.
- P50, P95, and P99 message latency.
- Disconnect and reconnect rate.
- Rate-limit and malformed-message rejections.

### Database

- Insert latency.
- Indexed select latency.
- Slow query count.
- Connection pool saturation.
- Error rate.

### Security And E2EE

- Encryption/decryption operation time.
- Ratchet step time.
- Skipped message key count.
- Device revocation propagation time.
- Refresh token replay detection.

### Reliability

- API startup time.
- Graceful shutdown duration.
- Health/readiness failures.
- Redis/PostgreSQL outage behavior.

## Academic Output

Recommended thesis/report sections:

1. Methodology.
2. Test environment.
3. Security model and limitations.
4. Benchmark results.
5. Comparative analysis.
6. Bottlenecks and optimization work.
7. Conclusion and future work.

See `docs/ACADEMIC_RESEARCH_PLAN.md` for the broader academic framing.
