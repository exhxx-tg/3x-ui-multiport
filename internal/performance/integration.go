package performance

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/exhxx-tg/3x-ui-multiport/internal/logger"
)

var globalProfilingHandler *ProfilingHandler
var globalPerfConfig Config

func Initialize(cfg Config) {
	globalPerfConfig = cfg

	if cfg.CacheEnabled {
		InitCacheManager(cfg)
		logger.Infof("performance: cache manager initialized (ttl=%v, maxEntries=%d)", cfg.CacheDefaultTTL, cfg.CacheMaxEntries)
	}

	InitGlobalWorkerPools(cfg)

	globalProfilingHandler = NewProfilingHandler(cfg.PprofEnabled)
	logger.Infof("performance: profiling handler initialized (pprof=%v, metrics=%v)", cfg.PprofEnabled, cfg.MetricsEnabled)
}

func RegisterRoutes(router gin.IRouter) {
	if globalProfilingHandler != nil {
		globalProfilingHandler.RegisterGinRoutes(router)
	}
}

func GetProfilingHandler() *ProfilingHandler {
	return globalProfilingHandler
}

func GetPerfConfig() Config {
	return globalPerfConfig
}

func StartPeriodicCleanup(interval time.Duration) {
	if globalCacheManager == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			globalCacheManager.CleanupAll()
		}
	}()
	logger.Infof("performance: periodic cache cleanup started (interval=%v)", interval)
}

func Shutdown() {
	StopAllWorkerPools()
	if globalCacheManager != nil {
		globalCacheManager.ClearAll()
	}
	logger.Info("performance: shut down all pools and caches")
}

func NewRequestTimer() *RequestTimer {
	return &RequestTimer{start: time.Now()}
}

type RequestTimer struct {
	start time.Time
}

func (rt *RequestTimer) Finish() {
	RecordRequest(time.Since(rt.start))
}

func (rt *RequestTimer) FinishWithError() {
	RecordRequest(time.Since(rt.start))
	RecordRequestError()
}

func GinPerformanceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		TrackActiveRequest(1)
		rt := NewRequestTimer()

		c.Next()

		rt.Finish()
		TrackActiveRequest(-1)

		if c.Writer.Status() >= 500 {
			RecordRequestError()
		}
	}
}
