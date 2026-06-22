// Package performance provides profiling, caching, concurrency, and
// benchmarking infrastructure for X-UI PRO. It is Phase 7 of the
// project: Performance Optimization & Load Testing.
//
// Architecture
//
//	profiling.go   — pprof handlers, runtime metrics, goroutine dumper
//	cache.go       — generics-based TTL cache with sharded map
//	pool.go        — sync.Pool wrappers for buffer reuse
//	workerpool.go  — bounded goroutine pool with job queue
//	benchmark.go   — load-test helpers and assertion macros
//	config.go      — tunable performance parameters
package performance
