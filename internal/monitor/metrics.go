package monitor

import (
	"sync"
	"time"

	"github.com/exhxx-tg/3x-ui-multiport/internal/logger"
	xuiProtocol "github.com/exhxx-tg/3x-ui-multiport/internal/protocol"
)

// protocolStartTimes tracks when each protocol started for uptime calculation.
var protocolStartTimes sync.Map

type MetricsCollector struct {
	mu             sync.RWMutex
	protocols      map[string]*ProtocolMetrics
	retentionCount int
}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		protocols:      make(map[string]*ProtocolMetrics),
		retentionCount: 1440,
	}
}

func (mc *MetricsCollector) Collect(id xuiProtocol.ProtocolID, up, down int64, conns, users, errors int) ProtocolMetrics {
	metrics := ProtocolMetrics{
		ProtocolID:  string(id),
		UpBytes:     up,
		DownBytes:   down,
		Connections: conns,
		ActiveUsers: users,
		ErrorCount:  errors,
		CollectedAt: time.Now(),
	}

	// Track start time on first collection for uptime calculation.
	startKey := string(id)
	if _, loaded := protocolStartTimes.LoadOrStore(startKey, time.Now()); !loaded {
		metrics.UptimeSeconds = 0
	} else if start, ok := protocolStartTimes.Load(startKey); ok {
		metrics.UptimeSeconds = int64(time.Since(start.(time.Time)).Seconds())
	}

	mc.mu.Lock()
	mc.protocols[string(id)] = &metrics
	mc.mu.Unlock()

	logger.Debugf("metrics [%s]: up=%d down=%d conns=%d users=%d errors=%d",
		id, up, down, conns, users, errors)

	return metrics
}

func (mc *MetricsCollector) GetLatest(id xuiProtocol.ProtocolID) *ProtocolMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	m, ok := mc.protocols[string(id)]
	if !ok {
		return nil
	}
	cp := *m
	return &cp
}

func (mc *MetricsCollector) GetAllLatest() map[string]*ProtocolMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	out := make(map[string]*ProtocolMetrics, len(mc.protocols))
	for k, v := range mc.protocols {
		cp := *v
		out[k] = &cp
	}
	return out
}

func (mc *MetricsCollector) IncErrors(id xuiProtocol.ProtocolID) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	if m, ok := mc.protocols[string(id)]; ok {
		m.ErrorCount++
	}
}

func (mc *MetricsCollector) SetConnections(id xuiProtocol.ProtocolID, n int) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	if m, ok := mc.protocols[string(id)]; ok {
		m.Connections = n
	}
}

func (mc *MetricsCollector) SetUsers(id xuiProtocol.ProtocolID, n int) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	if m, ok := mc.protocols[string(id)]; ok {
		m.ActiveUsers = n
	}
}

// ResetUptime resets the uptime counter for a protocol (e.g. after restart).
func ResetUptime(id xuiProtocol.ProtocolID) {
	protocolStartTimes.Store(string(id), time.Now())
}

func (mc *MetricsCollector) AddTraffic(id xuiProtocol.ProtocolID, up, down int64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	if m, ok := mc.protocols[string(id)]; ok {
		m.UpBytes += up
		m.DownBytes += down
	} else {
		mc.protocols[string(id)] = &ProtocolMetrics{
			ProtocolID:  string(id),
			UpBytes:     up,
			DownBytes:   down,
			CollectedAt: time.Now(),
		}
	}
}
