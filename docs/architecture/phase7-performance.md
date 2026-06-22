# Phase 7: Performance Optimization & Load Testing

## Overview

Phase 7 implements the systematic performance optimization layer for X-UI PRO,
targeting the resource goals:
- Memory: < 2GB peak, ~1.5GB average
- CPU: < 5% idle, < 25% normal, < 80% peak
- API response: < 200ms p95
- Concurrent connections: 5000+

## Package Structure

```
internal/performance/
  ├── doc.go           — Package overview
  ├── config.go        — Tunable performance parameters (env-driven)
  ├── profiling.go     — pprof handlers, runtime metrics, goroutine debug
  ├── cache.go         — Generic TTL cache with sharded map
  ├── pool.go          — sync.Pool wrappers for buffer reuse
  ├── workerpool.go    — Bounded goroutine pool with job queue
  ├── network.go       — Optimized HTTP transport, connection pool
  ├── xray.go          — Xray-specific buffer pools, traffic batching
  ├── benchmark.go     — Benchmark runner, result formatting
  ├── loadtest.go      — Load testing framework, connection simulator
  ├── integration.go   — Gin middleware, web server integration
  └── errors.go        — Package error definitions
```

## Components

### 1. Profiling (`profiling.go`)

- **pprof endpoints**: `/debug/pprof/*` (heap, goroutine, block, mutex, trace, profile)
- **Runtime metrics**: `/debug/runtime` — JSON with 20+ metrics
- **GC trigger**: `/debug/gc` — force GC + FreeOSMemory
- **expvar**: `/debug/vars` — standard Go expvar endpoint
- **Request tracking**: middleware counts requests, latency, errors

### 2. Caching (`cache.go`)

- Generic `Cache[K, V]` with TTL-based expiration
- `CacheManager` — centralized multi-cache registry
- Auto-cleanup via periodic goroutine
- Cache hit/miss tracking integrated with profiling

### 3. Buffer Pools (`pool.go`)

- `bytes.Buffer` pool (config generation, IO)
- JSON buffer pool (API responses)
- `map[string]any` pool (gin.H, JSON marshaling)
- `map[string]string` pool (headers, labels)
- `[]string` and `[]int` pools (temporary slices)
- All pools via `sync.Pool` — zero-alloc after warmup

### 4. Worker Pools (`workerpool.go`)

- Bounded goroutine pools with job queue
- Panic recovery per job
- Graceful shutdown via context
- Three global pools:
  - `health` — health check concurrency (8 workers)
  - `metrics` — metrics collection (4 workers)
  - `async` — general async tasks (16 workers)

### 5. Network Optimization (`network.go`)

- Optimized `http.Transport` with keepalive, connection limits
- Custom connection pool with `ConnPool`
- Default HTTP client with proper timeouts

### 6. Xray Optimization (`xray.go`)

- Config generation buffer pool
- Traffic buffer pool
- `XrayConnPool` — gRPC connection limiter
- `TrafficBatch` — batched traffic updates

### 7. Load Testing (`loadtest.go`)

- `LoadTester` — configurable concurrency, ramp-up, duration
- Default profiles: light (10), medium (50), heavy (200), stress (500)
- `ConnectionSimulator` — simulate concurrent connections
- `Throughput` measurement
- Latency percentiles (p50, p90, p99)

### 8. Benchmark Suite (`benchmark.go`)

- `RunBenchmark` — single-threaded benchmark with mem stats
- `RunParallelBenchmark` — multi-worker benchmark
- `FormatBenchmarkResults` — formatted output

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `XUI_CACHE_DISABLE` | false | Disable caching |
| `XUI_CACHE_TTL_SEC` | 30 | Default cache TTL |
| `XUI_WORKER_POOL` | 16 | Async worker pool size |
| `XUI_DB_MAX_OPEN` | 25 (pg) / 8 (sqlite) | Max open DB connections |
| `XUI_DB_MAX_IDLE` | 25 (pg) / 4 (sqlite) | Max idle DB connections |
| `XUI_PPROF_ENABLE` | false | Enable pprof endpoints |
| `XUI_GZIP_LEVEL` | 1 | Gzip compression level (1-9) |

## Integration Points

- **web.go `initRouter()`**: Registers performance routes + middleware
- **web.go `start()`**: Initializes performance system after monitor
- **web.go `stop()`**: Graceful shutdown of pools + caches

## Performance Targets

| Metric | Target | Measured |
|--------|--------|----------|
| API p95 latency | < 200ms | |
| Memory overhead | < 50MB (infra) | |
| Cache hit rate | > 80% | |
| GC pause avg | < 1ms | |
| Connections | 5000+ concurrent | |
| Idle CPU | < 5% | |
