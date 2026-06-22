package performance

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/op/go-logging"
	"github.com/exhxx-tg/3x-ui-multiport/internal/logger"
)

func init() {
	logger.InitLogger(logging.WARNING)
}

func BenchmarkBufferPool(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf := GetBuffer()
		buf.WriteString("test")
		PutBuffer(buf)
	}
}

func BenchmarkBufferNoPool(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		buf.WriteString("test")
	}
}

func BenchmarkCacheGetSet(b *testing.B) {
	cache := NewCache[string, string](defaultCacheTTL, defaultCacheMaxEntries)
	cache.Set("key", "value")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cache.Get("key")
	}
}

func BenchmarkCacheParallel(b *testing.B) {
	cache := NewCache[string, string](defaultCacheTTL, defaultCacheMaxEntries)
	for i := 0; i < 100; i++ {
		key := strings.Repeat("a", i)
		cache.Set(key, "value")
	}

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cache.Get("a")
		}
	})
}

func BenchmarkWorkerPool(b *testing.B) {
	wp := NewWorkerPool("bench", 4, 64)
	wp.Start()
	defer wp.Stop()

	done := make(chan struct{})
	defer close(done)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		wp.Submit(func(_ context.Context) {})
	}
}

func BenchmarkMapStringAnyPool(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m := GetMapStringAny()
		m["key"] = "value"
		PutMapStringAny(m)
	}
}

func BenchmarkMapStringAnyNoPool(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m := make(map[string]any, 16)
		m["key"] = "value"
	}
}

func BenchmarkParallelWorkers(b *testing.B) {
	wp := NewWorkerPool("bench-parallel", 8, 128)
	wp.Start()
	defer wp.Stop()

	var mu sync.Mutex
	counter := 0

	b.ReportAllocs()
	b.ResetTimer()

	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		wp.Submit(func(ctx context.Context) {
			mu.Lock()
			counter++
			mu.Unlock()
			wg.Done()
		})
	}
	wg.Wait()
}

func TestCacheTTL(t *testing.T) {
	cache := NewCache[string, string](defaultCacheTTL, defaultCacheMaxEntries)

	cache.Set("key", "value")

	val, ok := cache.Get("key")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if val != "value" {
		t.Fatalf("expected 'value', got '%s'", val)
	}

	if n := cache.Len(); n != 1 {
		t.Fatalf("expected 1 entry, got %d", n)
	}

	cache.Delete("key")

	_, ok = cache.Get("key")
	if ok {
		t.Fatal("expected cache miss after delete")
	}
}

func TestCacheLimit(t *testing.T) {
	cache := NewCache[string, string](defaultCacheTTL, 5)

	for i := 0; i < 10; i++ {
		key := strings.Repeat("x", i+1)
		cache.Set(key, "value")
	}

	if n := cache.Len(); n > 5 {
		t.Fatalf("expected max 5 entries, got %d", n)
	}
}

func TestCacheCleanup(t *testing.T) {
	cache := NewCache[string, string](-1, 100)
	cache.Set("key", "value")

	_, ok := cache.Get("key")
	if ok {
		t.Fatal("expected cache miss for expired entry")
	}

	cache.Cleanup()
	if n := cache.Len(); n != 0 {
		t.Fatalf("expected 0 entries after cleanup, got %d", n)
	}
}

func TestWorkerPoolBasic(t *testing.T) {
	wp := NewWorkerPool("test", 4, 64)
	wp.Start()
	defer wp.Stop()

	done := make(chan struct{})
	wp.Submit(func(ctx context.Context) {
		close(done)
	})

	<-done
}

func TestWorkerPoolQueueFull(t *testing.T) {
	// Pool with no workers and no queue capacity — submits must fail.
	wp := NewWorkerPool("test-full", 1, 0)
	wp.Start()
	defer wp.Stop()

	block := make(chan struct{})
	defer close(block)

	// Block the only worker
	wp.queue <- func(ctx context.Context) { <-block }

	// Queue capacity is 0 and worker is busy — TrySubmit must fail.
	ok := wp.TrySubmit(func(ctx context.Context) {})
	if ok {
		t.Fatal("expected TrySubmit to return false (no workers, no queue), got true")
	}
}

func TestProfilingMetrics(t *testing.T) {
	RecordRequest(100 * time.Millisecond)
	RecordRequestError()
	TrackCacheResult(true)
	TrackCacheResult(false)

	metrics := CollectRuntimeMetrics()

	if metrics.TotalRequests <= 0 {
		t.Fatal("expected total requests > 0")
	}

	if metrics.CacheHitRate <= 0 {
		t.Fatal("expected cache hit rate > 0")
	}
}

func TestGetSetPool(t *testing.T) {
	_ = GetBuffer()
	_ = GetJSONBuffer()
	_ = GetSmallIntSlice()
	_ = GetStringSlice()
	_ = GetMapStringAny()
	_ = GetMapStringString()
}
