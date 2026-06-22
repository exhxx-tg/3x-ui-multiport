package performance

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
	"strings"

	"github.com/exhxx-tg/3x-ui-multiport/internal/logger"
)

type BenchmarkResult struct {
	Name           string        `json:"name"`
	Operations     int64         `json:"operations"`
	Duration       time.Duration `json:"duration"`
	OpsPerSecond   float64       `json:"opsPerSecond"`
	AllocBytes     uint64        `json:"allocBytes"`
	AllocsPerOp    uint64        `json:"allocsPerOp"`
	BytesPerOp     uint64        `json:"bytesPerOp"`
	Goroutines     int           `json:"goroutines"`
	NumGC          uint32        `json:"numGC"`
}

func RunBenchmark(name string, fn func() error, iterations int) BenchmarkResult {
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	start := time.Now()
	for i := 0; i < iterations; i++ {
		if err := fn(); err != nil {
			logger.Warningf("benchmark [%s]: iteration %d failed: %v", name, i, err)
		}
	}
	elapsed := time.Since(start)

	runtime.ReadMemStats(&m2)

	allocBytes := m2.TotalAlloc - m1.TotalAlloc

	return BenchmarkResult{
		Name:         name,
		Operations:   int64(iterations),
		Duration:     elapsed,
		OpsPerSecond: float64(iterations) / elapsed.Seconds(),
		AllocBytes:   allocBytes,
		AllocsPerOp:  allocBytes / uint64(iterations),
		BytesPerOp:   (m2.TotalAlloc - m1.TotalAlloc) / uint64(iterations),
		Goroutines:   runtime.NumGoroutine(),
		NumGC:        m2.NumGC - m1.NumGC,
	}
}

func RunParallelBenchmark(name string, fn func(ctx context.Context) error, workers, iterations int) BenchmarkResult {
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	start := time.Now()
	totalOps := int64(0)
	var mu sync.Mutex
	var wg sync.WaitGroup

	perWorker := iterations / workers
	remainder := iterations % workers

	for i := 0; i < workers; i++ {
		wg.Add(1)
		count := perWorker
		if i < remainder {
			count++
		}
		go func(n int) {
			defer wg.Done()
			ctx := context.Background()
			for j := 0; j < n; j++ {
				if err := fn(ctx); err != nil {
					continue
				}
			}
			mu.Lock()
			totalOps += int64(n)
			mu.Unlock()
		}(count)
	}
	wg.Wait()
	elapsed := time.Since(start)

	runtime.ReadMemStats(&m2)
	allocBytes := m2.TotalAlloc - m1.TotalAlloc

	return BenchmarkResult{
		Name:         name + fmt.Sprintf("(x%d workers)", workers),
		Operations:   totalOps,
		Duration:     elapsed,
		OpsPerSecond: float64(totalOps) / elapsed.Seconds(),
		AllocBytes:   allocBytes,
		AllocsPerOp:  allocBytes / uint64(totalOps),
		BytesPerOp:   (m2.TotalAlloc - m1.TotalAlloc) / uint64(totalOps),
		Goroutines:   runtime.NumGoroutine(),
		NumGC:        m2.NumGC - m1.NumGC,
	}
}

func FormatBenchmarkResults(results []BenchmarkResult) string {
	var b strings.Builder
	b.WriteString("\n═══════════════════════════════════════════════════════════════\n")
	b.WriteString("  BENCHMARK RESULTS\n")
	b.WriteString("═══════════════════════════════════════════════════════════════\n\n")

	for _, r := range results {
		b.WriteString(fmt.Sprintf("  %-40s\n", r.Name))
		b.WriteString(fmt.Sprintf("  %-20s %10d ops in %v\n", "", r.Operations, r.Duration.Round(time.Millisecond)))
		b.WriteString(fmt.Sprintf("  %-20s %10.0f ops/s\n", "", r.OpsPerSecond))

		if r.AllocsPerOp > 0 {
			b.WriteString(fmt.Sprintf("  %-20s %10d allocs/op\n", "", r.AllocsPerOp))
		}
		if r.BytesPerOp > 0 {
			b.WriteString(fmt.Sprintf("  %-20s %10d bytes/op\n", "", r.BytesPerOp))
		}
		if r.AllocBytes > 0 {
			b.WriteString(fmt.Sprintf("  %-20s %10s total allocated\n", "", formatBytes(r.AllocBytes)))
		}
		b.WriteString(fmt.Sprintf("  %-20s %10d goroutines, %d GC cycles\n\n", "", r.Goroutines, r.NumGC))
	}

	b.WriteString("═══════════════════════════════════════════════════════════════\n")
	return b.String()
}

func formatBytes(b uint64) string {
	switch {
	case b >= 1024*1024*1024:
		return fmt.Sprintf("%.2f GB", float64(b)/1024/1024/1024)
	case b >= 1024*1024:
		return fmt.Sprintf("%.2f MB", float64(b)/1024/1024)
	case b >= 1024:
		return fmt.Sprintf("%.2f KB", float64(b)/1024)
	default:
		return fmt.Sprintf("%d B", b)
	}
}
