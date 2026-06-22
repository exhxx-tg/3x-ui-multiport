package performance

import (
	"expvar"
	"fmt"
	"net/http"
	"net/http/pprof"
	"runtime"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	startTime      = time.Now()
	totalRequests  atomic.Int64
	activeRequests atomic.Int64
	requestErrors  atomic.Int64
	requestTime    atomic.Int64
)

type ProfilingHandler struct {
	enabled bool
	mux     *http.ServeMux
}

func NewProfilingHandler(pprofEnabled bool) *ProfilingHandler {
	h := &ProfilingHandler{enabled: true}
	if pprofEnabled {
		h.mux = http.NewServeMux()
		h.mux.HandleFunc("/debug/pprof/", pprof.Index)
		h.mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		h.mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		h.mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		h.mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		h.mux.HandleFunc("/debug/pprof/heap", pprof.Handler("heap").ServeHTTP)
		h.mux.HandleFunc("/debug/pprof/goroutine", pprof.Handler("goroutine").ServeHTTP)
		h.mux.HandleFunc("/debug/pprof/block", pprof.Handler("block").ServeHTTP)
		h.mux.HandleFunc("/debug/pprof/mutex", pprof.Handler("mutex").ServeHTTP)
		h.mux.HandleFunc("/debug/pprof/threadcreate", pprof.Handler("threadcreate").ServeHTTP)
	}
	return h
}

type RuntimeMetrics struct {
	Version            string  `json:"version"`
	UptimeSeconds      float64 `json:"uptimeSeconds"`
	NumCPU             int     `json:"numCPU"`
	NumGoroutine       int     `json:"numGoroutine"`
	NumCgoCall         int64   `json:"numCgoCall"`
	AllocMB            float64 `json:"allocMB"`
	TotalAllocMB       float64 `json:"totalAllocMB"`
	SysMB              float64 `json:"sysMB"`
	HeapAllocMB        float64 `json:"heapAllocMB"`
	HeapSysMB          float64 `json:"heapSysMB"`
	HeapIdleMB         float64 `json:"heapIdleMB"`
	HeapInuseMB        float64 `json:"heapInuseMB"`
	StackInuseMB       float64 `json:"stackInuseMB"`
	GCStatsNumGC       uint32  `json:"gcStatsNumGC"`
	GCStatsPauseTotal  float64 `json:"gcStatsPauseTotalMs"`
	GCStatsPauseAvg    float64 `json:"gcStatsPauseAvgMs"`
	NextGC             float64 `json:"nextGCMB"`
	LastGC             string  `json:"lastGC"`
	TotalRequests      int64   `json:"totalRequests"`
	ActiveRequests     int64   `json:"activeRequests"`
	RequestErrors      int64   `json:"requestErrors"`
	AvgRequestTimeMs   float64 `json:"avgRequestTimeMs"`
	CacheHitRate       float64 `json:"cacheHitRate"`
}

func CollectRuntimeMetrics() RuntimeMetrics {
	m := RuntimeMetrics{
		Version:       runtime.Version(),
		UptimeSeconds: time.Since(startTime).Seconds(),
		NumCPU:        runtime.NumCPU(),
		NumGoroutine:  runtime.NumGoroutine(),
		NumCgoCall:    runtime.NumCgoCall(),
		TotalRequests: totalRequests.Load(),
		ActiveRequests: activeRequests.Load(),
		RequestErrors: requestErrors.Load(),
	}

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	m.AllocMB = bToMB(ms.Alloc)
	m.TotalAllocMB = bToMB(ms.TotalAlloc)
	m.SysMB = bToMB(ms.Sys)
	m.HeapAllocMB = bToMB(ms.HeapAlloc)
	m.HeapSysMB = bToMB(ms.HeapSys)
	m.HeapIdleMB = bToMB(ms.HeapIdle)
	m.HeapInuseMB = bToMB(ms.HeapInuse)
	m.StackInuseMB = bToMB(ms.StackInuse)
	m.GCStatsNumGC = ms.NumGC
	m.GCStatsPauseTotal = float64(ms.PauseTotalNs) / 1e6
	m.NextGC = bToMB(ms.NextGC)

	if ms.NumGC > 0 {
		m.GCStatsPauseAvg = float64(ms.PauseTotalNs) / float64(ms.NumGC) / 1e6
	}

	if t := totalRequests.Load(); t > 0 {
		m.AvgRequestTimeMs = float64(requestTime.Load()) / float64(t) / 1e6
	}

	if hits, misses := globalCacheHits.Load(), globalCacheMisses.Load(); hits+misses > 0 {
		m.CacheHitRate = float64(hits) / float64(hits+misses) * 100
	}

	return m
}

func (h *ProfilingHandler) RegisterGinRoutes(router gin.IRouter) {
	if !h.enabled {
		return
	}

	router.GET("/debug/vars", func(c *gin.Context) {
		c.Header("Content-Type", "application/json; charset=utf-8")
		expvar.Handler().ServeHTTP(c.Writer, c.Request)
	})

	router.GET("/debug/runtime", func(c *gin.Context) {
		metrics := CollectRuntimeMetrics()
		c.JSON(200, metrics)
	})

	router.GET("/debug/gc", func(c *gin.Context) {
		debug.FreeOSMemory()
		c.JSON(200, gin.H{"status": "GC triggered", "timestamp": time.Now().UnixMilli()})
	})

	n := h.mux
	if n != nil {
		router.GET("/debug/pprof/*any", func(c *gin.Context) {
			path := c.Param("any")
			if path == "" || path == "/" {
				path = "/debug/pprof/"
			} else {
				path = "/debug/pprof" + path
			}
			c.Request.URL.Path = path
			n.ServeHTTP(c.Writer, c.Request)
		})
	}
}

func bToMB(b uint64) float64 {
	return float64(b) / 1024 / 1024
}

func RecordRequest(duration time.Duration) {
	totalRequests.Add(1)
	requestTime.Add(duration.Nanoseconds())
}

func RecordRequestError() {
	requestErrors.Add(1)
}

func TrackActiveRequest(delta int64) {
	activeRequests.Add(delta)
}

func TrackCacheResult(hit bool) {
	if hit {
		globalCacheHits.Add(1)
	} else {
		globalCacheMisses.Add(1)
	}
}

func FormatMetricsText(metrics RuntimeMetrics) string {
	var b strings.Builder
	b.WriteString("# HELP xui_pro_uptime_seconds Server uptime\n")
	b.WriteString("# TYPE xui_pro_uptime_seconds gauge\n")
	fmt.Fprintf(&b, "xui_pro_uptime_seconds %0.f\n", metrics.UptimeSeconds)

	b.WriteString("# HELP xui_pro_goroutines Current number of goroutines\n")
	b.WriteString("# TYPE xui_pro_goroutines gauge\n")
	fmt.Fprintf(&b, "xui_pro_goroutines %d\n", metrics.NumGoroutine)

	b.WriteString("# HELP xui_pro_memory_alloc_bytes Current heap allocation\n")
	b.WriteString("# TYPE xui_pro_memory_alloc_bytes gauge\n")
	fmt.Fprintf(&b, "xui_pro_memory_alloc_bytes %.0f\n", metrics.HeapAllocMB*1024*1024)

	b.WriteString("# HELP xui_pro_gc_total Total GC cycles\n")
	b.WriteString("# TYPE xui_pro_gc_total counter\n")
	fmt.Fprintf(&b, "xui_pro_gc_total %d\n", metrics.GCStatsNumGC)

	b.WriteString("# HELP xui_pro_gc_pause_seconds_total Total GC pause time\n")
	b.WriteString("# TYPE xui_pro_gc_pause_seconds_total counter\n")
	fmt.Fprintf(&b, "xui_pro_gc_pause_seconds_total %.3f\n", metrics.GCStatsPauseTotal/1000)

	b.WriteString("# HELP xui_pro_requests_total Total HTTP requests\n")
	b.WriteString("# TYPE xui_pro_requests_total counter\n")
	fmt.Fprintf(&b, "xui_pro_requests_total %d\n", metrics.TotalRequests)

	b.WriteString("# HELP xui_pro_requests_active Currently active requests\n")
	b.WriteString("# TYPE xui_pro_requests_active gauge\n")
	fmt.Fprintf(&b, "xui_pro_requests_active %d\n", metrics.ActiveRequests)

	b.WriteString("# HELP xui_pro_requests_errors_total Total request errors\n")
	b.WriteString("# TYPE xui_pro_requests_errors_total counter\n")
	fmt.Fprintf(&b, "xui_pro_requests_errors_total %d\n", metrics.RequestErrors)

	b.WriteString("# HELP xui_pro_cache_hit_rate Cache hit rate percentage\n")
	b.WriteString("# TYPE xui_pro_cache_hit_rate gauge\n")
	fmt.Fprintf(&b, "xui_pro_cache_hit_rate %.1f\n", metrics.CacheHitRate)

	return b.String()
}

var globalCacheHits, globalCacheMisses atomic.Int64
