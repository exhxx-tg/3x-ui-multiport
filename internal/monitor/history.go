package monitor

import (
	"encoding/json"
	"time"

	"github.com/exhxx-tg/3x-ui-multiport/internal/database"
	"github.com/exhxx-tg/3x-ui-multiport/internal/database/model"
)

type HistoryStore struct{}

func NewHistoryStore() *HistoryStore {
	return &HistoryStore{}
}

func (hs *HistoryStore) RecordEvent(event AlertEvent) (int64, error) {
	labelsJSON := ""
	if event.Labels != nil {
		b, err := json.Marshal(event.Labels)
		if err == nil {
			labelsJSON = string(b)
		}
	}

	resolvedAt := int64(0)
	if event.ResolvedAt != nil {
		resolvedAt = event.ResolvedAt.UnixMilli()
	}

	row := &model.AlertHistory{
		RuleId:     0,
		RuleName:   event.RuleName,
		ProtocolId: event.ProtocolID,
		Severity:   string(event.Severity),
		Status:     string(event.Status),
		Message:    event.Message,
		Metric:     event.Metric,
		Value:      event.Value,
		Threshold:  event.Threshold,
		Labels:     labelsJSON,
		ResolvedAt: resolvedAt,
	}

	db := database.GetDB()
	if db == nil {
		return 0, nil
	}
	if err := db.Create(row).Error; err != nil {
		return 0, err
	}
	return int64(row.Id), nil
}

func (hs *HistoryStore) ResolveEvent(ruleID string) error {
	db := database.GetDB()
	if db == nil {
		return nil
	}
	return db.Model(&model.AlertHistory{}).
		Where("rule_id = ? AND status = ?", ruleID, "firing").
		Updates(map[string]any{
			"status":      "resolved",
			"resolved_at": time.Now().UnixMilli(),
		}).Error
}

func (hs *HistoryStore) List(opts HistoryQuery) ([]model.AlertHistory, int64, error) {
	db := database.GetDB()
	if db == nil {
		return nil, 0, nil
	}

	query := db.Model(&model.AlertHistory{})
	if opts.ProtocolID != "" {
		query = query.Where("protocol_id = ?", opts.ProtocolID)
	}
	if opts.Severity != "" {
		query = query.Where("severity = ?", opts.Severity)
	}
	if opts.Status != "" {
		query = query.Where("status = ?", opts.Status)
	}
	if opts.Limit <= 0 {
		opts.Limit = 50
	}
	if opts.Offset < 0 {
		opts.Offset = 0
	}

	var total int64
	query.Count(&total)

	var rows []model.AlertHistory
	if err := query.Order("id DESC").Offset(opts.Offset).Limit(opts.Limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

type HistoryQuery struct {
	ProtocolID string
	Severity   string
	Status     string
	Offset     int
	Limit      int
}
