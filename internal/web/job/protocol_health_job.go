package job

import (
	"context"
	"fmt"
	"time"

	"github.com/exhxx-tg/3x-ui-multiport/internal/logger"
	"github.com/exhxx-tg/3x-ui-multiport/internal/monitor"
	xuiProtocol "github.com/exhxx-tg/3x-ui-multiport/internal/protocol"
)

type ProtocolHealthJob struct {
	checker  *monitor.HealthChecker
	interval time.Duration
	stopCh   chan struct{}
}

func NewProtocolHealthJob() *ProtocolHealthJob {
	hc := monitor.GlobalHealthChecker()
	interval := time.Second * 30
	if hc != nil {
		interval = hc.Interval()
	}
	return &ProtocolHealthJob{
		checker:  hc,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

func (j *ProtocolHealthJob) Run() {
	if j.checker == nil {
		logger.Warning("protocol health job: health checker not initialized")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	summary := j.checker.CheckAll(ctx)
	logger.Infof("protocol health check: %d/%d healthy, %d unhealthy, %d unknown",
		summary.Healthy, summary.TotalProtocols, summary.Unhealthy, summary.Unknown)

	// Evaluate alert rules against health check results
	re := monitor.GlobalRuleEngine()
	hs := monitor.GlobalHistoryStore()
	mc := monitor.GlobalMetricsCollector()

	for _, result := range summary.Results {
		if !result.Healthy && result.Error != "" {
			logger.Warningf("protocol [%s] unhealthy: %s", result.ProtocolID, result.Error)
		}

		// Evaluate rules for this protocol
		if re != nil {
			value := 0.0
			if result.Healthy {
				value = 1.0
			}
			events := re.Evaluate(value, map[string]string{
				"protocol":  result.ProtocolID,
				"component": "health_check",
			})
			for _, ev := range events {
				if hs != nil {
					id, err := hs.RecordEvent(ev)
					if err != nil {
						logger.Warningf("failed to record alert event: %v", err)
					} else {
						ev.HistoryID = id
					}
				}
				if err := re.SendAlert(ev); err != nil {
					logger.Warningf("failed to send alert for rule %s: %v", ev.RuleID, err)
				}
			}
		}

		// Track error count in metrics collector
		if mc != nil && !result.Healthy {
			mc.IncErrors(xuiProtocol.ProtocolID(result.ProtocolID))
		}
	}

	// Auto-recover unhealthy protocols if any alert rule has auto-recovery enabled
	unhealthy := j.checker.GetUnhealthyProtocols()
	if len(unhealthy) > 0 && re != nil {
		// Check if any enabled rule has auto-recovery for these protocols
		for _, rule := range re.ListRules() {
			if !rule.Enabled || !rule.AutoRecovery {
				continue
			}
			for _, pid := range unhealthy {
				if string(pid) == rule.ProtocolID {
					logger.Infof("auto-recovery triggered by rule [%s] for protocol [%s]", rule.Name, pid)
					recovered := j.checker.AutoRecover()
					if recovered > 0 {
						logger.Infof("auto-recovery: %d protocol(s) restarted", recovered)
						// Resolve firing alerts for recovered protocols
						if hs != nil {
							hs.ResolveEvent(fmt.Sprintf("rule-%s", rule.ID))
						}
					}
					break
				}
			}
		}
	}
}
