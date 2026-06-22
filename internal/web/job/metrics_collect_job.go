package job

import (
	"time"

	"github.com/exhxx-tg/3x-ui-multiport/internal/database"
	"github.com/exhxx-tg/3x-ui-multiport/internal/database/model"
	"github.com/exhxx-tg/3x-ui-multiport/internal/logger"
	"github.com/exhxx-tg/3x-ui-multiport/internal/monitor"
	xuiProtocol "github.com/exhxx-tg/3x-ui-multiport/internal/protocol"
)

type MetricsCollectJob struct {
	collector *monitor.MetricsCollector
}

func NewMetricsCollectJob() *MetricsCollectJob {
	return &MetricsCollectJob{
		collector: monitor.GlobalMetricsCollector(),
	}
}

func (j *MetricsCollectJob) Run() {
	if j.collector == nil {
		logger.Warning("metrics collect job: collector not initialized")
		return
	}

	db := database.GetDB()
	if db == nil {
		return
	}

	for _, info := range xuiProtocol.AllProtocols {
		hc := monitor.GlobalHealthChecker()
		if hc == nil {
			continue
		}
		lastResult := hc.GetLastResult(info.ID)

		errCount := 0
		connections := 0
		up := int64(0)
		down := int64(0)

		if lastResult != nil {
			if !lastResult.Healthy {
				errCount = 1
			}
			if v, ok := lastResult.Details["connections"]; ok {
				if c, ok := v.(int); ok {
					connections = c
				}
			}
		}

		metrics := j.collector.Collect(info.ID, up, down, connections, 0, errCount)

		dbRow := &model.ProtocolMetrics{
			ProtocolId:    string(info.ID),
			UpBytes:       metrics.UpBytes,
			DownBytes:     metrics.DownBytes,
			Connections:   metrics.Connections,
			ActiveUsers:   metrics.ActiveUsers,
			ErrorCount:    metrics.ErrorCount,
			UptimeSeconds: metrics.UptimeSeconds,
			CollectedAt:   time.Now().UnixMilli(),
		}
		if err := db.Create(dbRow).Error; err != nil {
			logger.Warningf("failed to persist metrics for %s: %v", info.ID, err)
		}
	}

	logger.Debug("metrics collection completed")
}
