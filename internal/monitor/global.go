package monitor

import (
	"fmt"
	"sync"
	"time"

	"github.com/exhxx-tg/3x-ui-multiport/internal/database"
	"github.com/exhxx-tg/3x-ui-multiport/internal/database/model"
	xuiProtocol "github.com/exhxx-tg/3x-ui-multiport/internal/protocol"
	"github.com/exhxx-tg/3x-ui-multiport/internal/servicemanager"
)

var (
	globalHealthChecker   *HealthChecker
	globalMetricsCollector *MetricsCollector
	globalRuleEngine      *RuleEngine
	globalHistoryStore    *HistoryStore
	globalInitOnce        sync.Once
)

func InitGlobal(registry *xuiProtocol.Registry, sm *servicemanager.ServiceManager) {
	globalInitOnce.Do(func() {
		globalHealthChecker = NewHealthChecker(registry, sm)
		globalMetricsCollector = NewMetricsCollector()
		globalRuleEngine = NewRuleEngine()
		globalHistoryStore = NewHistoryStore()

		globalRuleEngine.RegisterNotifier(NewLogNotifier("log-default", "Default Logger"))

		// Load existing alert rules from database into the rule engine
		loadAlertRulesFromDB()
	})
}

func loadAlertRulesFromDB() {
	db := database.GetDB()
	if db == nil {
		return
	}

	var dbRules []model.AlertRule
	if err := db.Find(&dbRules).Error; err != nil {
		return
	}

	for _, r := range dbRules {
		rule := &AlertRule{
			ID:           fmt.Sprintf("rule-%d", r.Id),
			DBID:         r.Id,
			Name:         r.Name,
			Description:  r.Description,
			ProtocolID:   r.ProtocolId,
			Metric:       r.Metric,
			Condition:    r.Condition,
			Threshold:    r.Threshold,
			Duration:     time.Duration(r.Duration) * time.Second,
			Severity:     Severity(r.Severity),
			Enabled:      r.Enabled,
			Cooldown:     time.Duration(r.Cooldown) * time.Second,
			AutoRecovery: r.AutoRecovery,
			CreatedAt:    time.UnixMilli(r.CreatedAt),
			UpdatedAt:    time.UnixMilli(r.UpdatedAt),
		}
		globalRuleEngine.AddRule(rule)
	}
}

func GlobalHealthChecker() *HealthChecker {
	return globalHealthChecker
}

func GlobalMetricsCollector() *MetricsCollector {
	return globalMetricsCollector
}

func GlobalRuleEngine() *RuleEngine {
	return globalRuleEngine
}

func GlobalHistoryStore() *HistoryStore {
	return globalHistoryStore
}
