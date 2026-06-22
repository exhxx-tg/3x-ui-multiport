package performance

import (
	"os"
	"strconv"
	"time"
)

const (
	defaultCacheTTL           = 30 * time.Second
	defaultCacheCleanupPeriod = 60 * time.Second
	defaultCacheMaxEntries    = 10000
	defaultWorkerPoolSize     = 16
	defaultWorkerQueueDepth   = 512
	defaultHealthConcurrency  = 8
	defaultDBMaxOpenConns     = 25
	defaultDBMaxIdleConns     = 10
	defaultDBConnMaxLifetime  = 5 * time.Minute
)

type Config struct {
	CacheEnabled        bool          `json:"cacheEnabled"`
	CacheDefaultTTL     time.Duration `json:"cacheDefaultTTL"`
	CacheCleanupPeriod  time.Duration `json:"cacheCleanupPeriod"`
	CacheMaxEntries     int           `json:"cacheMaxEntries"`
	WorkerPoolSize      int           `json:"workerPoolSize"`
	WorkerQueueDepth    int           `json:"workerQueueDepth"`
	HealthConcurrency   int           `json:"healthConcurrency"`
	DBMaxOpenConns      int           `json:"dbMaxOpenConns"`
	DBMaxIdleConns      int           `json:"dbMaxIdleConns"`
	DBConnMaxLifetime   time.Duration `json:"dbConnMaxLifetime"`
	ProfilingEnabled    bool          `json:"profilingEnabled"`
	PprofEnabled        bool          `json:"pprofEnabled"`
	MetricsEnabled      bool          `json:"metricsEnabled"`
	GzipCompressionLevel int          `json:"gzipCompressionLevel"`
}

func DefaultConfig() Config {
	return Config{
		CacheEnabled:         true,
		CacheDefaultTTL:      defaultCacheTTL,
		CacheCleanupPeriod:   defaultCacheCleanupPeriod,
		CacheMaxEntries:      defaultCacheMaxEntries,
		WorkerPoolSize:       defaultWorkerPoolSize,
		WorkerQueueDepth:     defaultWorkerQueueDepth,
		HealthConcurrency:    defaultHealthConcurrency,
		DBMaxOpenConns:       defaultDBMaxOpenConns,
		DBMaxIdleConns:       defaultDBMaxIdleConns,
		DBConnMaxLifetime:    defaultDBConnMaxLifetime,
		ProfilingEnabled:     true,
		PprofEnabled:         false,
		MetricsEnabled:       true,
		GzipCompressionLevel: 1,
	}
}

func ConfigFromEnv() Config {
	cfg := DefaultConfig()

	if v, _ := strconv.ParseBool(os.Getenv("XUI_CACHE_DISABLE")); v {
		cfg.CacheEnabled = false
	}
	if v, err := strconv.Atoi(os.Getenv("XUI_CACHE_TTL_SEC")); err == nil && v > 0 {
		cfg.CacheDefaultTTL = time.Duration(v) * time.Second
	}
	if v, err := strconv.Atoi(os.Getenv("XUI_WORKER_POOL")); err == nil && v > 0 {
		cfg.WorkerPoolSize = v
	}
	if v, err := strconv.Atoi(os.Getenv("XUI_DB_MAX_OPEN")); err == nil && v > 0 {
		cfg.DBMaxOpenConns = v
	}
	if v, err := strconv.Atoi(os.Getenv("XUI_DB_MAX_IDLE")); err == nil && v >= 0 {
		cfg.DBMaxIdleConns = v
	}
	if v, _ := strconv.ParseBool(os.Getenv("XUI_PPROF_ENABLE")); v {
		cfg.PprofEnabled = true
	}
	if v, err := strconv.Atoi(os.Getenv("XUI_GZIP_LEVEL")); err == nil && v >= 0 && v <= 9 {
		cfg.GzipCompressionLevel = v
	}

	return cfg
}
