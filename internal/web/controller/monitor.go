package controller

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/exhxx-tg/3x-ui-multiport/internal/database"
	"github.com/exhxx-tg/3x-ui-multiport/internal/database/model"
	"github.com/exhxx-tg/3x-ui-multiport/internal/monitor"
	xuiProtocol "github.com/exhxx-tg/3x-ui-multiport/internal/protocol"
)

type MonitorController struct{}

func NewMonitorController(g *gin.RouterGroup) *MonitorController {
	ctrl := &MonitorController{}
	ctrl.initRouter(g)
	return ctrl
}

func (ctrl *MonitorController) initRouter(g *gin.RouterGroup) {
	m := g.Group("/monitor")
	{
		m.GET("/health", ctrl.healthSummary)
		m.GET("/health/:protocol", ctrl.protocolHealth)
		m.POST("/health/check", ctrl.runHealthCheck)

		m.GET("/metrics", ctrl.allMetrics)
		m.GET("/metrics/:protocol", ctrl.protocolMetrics)

		m.GET("/alerts/rules", ctrl.listRules)
		m.POST("/alerts/rules", ctrl.createRule)
		m.GET("/alerts/rules/:id", ctrl.getRule)
		m.PUT("/alerts/rules/:id", ctrl.updateRule)
		m.DELETE("/alerts/rules/:id", ctrl.deleteRule)
		m.POST("/alerts/rules/:id/test", ctrl.testRule)

		m.GET("/alerts/history", ctrl.alertHistory)
		m.POST("/alerts/history/:id/ack", ctrl.ackAlert)
	}
	g.GET("/notifiers", ctrl.listNotifiers)
}

func (ctrl *MonitorController) healthSummary(c *gin.Context) {
	hc := monitor.GlobalHealthChecker()
	if hc == nil {
		jsonMsg(c, "Health checker not initialized", fmt.Errorf("not ready"))
		return
	}
	results := hc.GetAllResults()
	summary := monitor.HealthSummary{
		TotalProtocols: len(results),
		Results:        results,
		GeneratedAt:    time.Now(),
	}
	for _, r := range results {
		if r.Healthy {
			summary.Healthy++
		} else if r.Error != "" || !r.Healthy {
			summary.Unhealthy++
		} else {
			summary.Unknown++
		}
	}
	jsonObj(c, summary, nil)
}

func (ctrl *MonitorController) protocolHealth(c *gin.Context) {
	hc := monitor.GlobalHealthChecker()
	if hc == nil {
		jsonMsg(c, "Health checker not initialized", fmt.Errorf("not ready"))
		return
	}
	id := xuiProtocol.ProtocolID(c.Param("protocol"))
	last := hc.GetLastResult(id)
	if last == nil {
		jsonMsg(c, "No health data for protocol", fmt.Errorf("no data"))
		return
	}
	jsonObj(c, last, nil)
}

func (ctrl *MonitorController) runHealthCheck(c *gin.Context) {
	hc := monitor.GlobalHealthChecker()
	if hc == nil {
		jsonMsg(c, "Health checker not initialized", fmt.Errorf("not ready"))
		return
	}

	var req struct {
		Protocol string `json:"protocol"`
	}
	if err := c.ShouldBindJSON(&req); err == nil && req.Protocol != "" {
		id := xuiProtocol.ProtocolID(req.Protocol)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		result := hc.CheckProtocol(ctx, id)
		jsonObj(c, result, nil)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	summary := hc.CheckAll(ctx)
	jsonObj(c, summary, nil)
}

func (ctrl *MonitorController) allMetrics(c *gin.Context) {
	mc := monitor.GlobalMetricsCollector()
	if mc == nil {
		jsonMsg(c, "Metrics collector not initialized", fmt.Errorf("not ready"))
		return
	}
	jsonObj(c, mc.GetAllLatest(), nil)
}

func (ctrl *MonitorController) protocolMetrics(c *gin.Context) {
	mc := monitor.GlobalMetricsCollector()
	if mc == nil {
		jsonMsg(c, "Metrics collector not initialized", fmt.Errorf("not ready"))
		return
	}
	id := xuiProtocol.ProtocolID(c.Param("protocol"))
	m := mc.GetLatest(id)
	if m == nil {
		jsonMsg(c, "No metrics for protocol", fmt.Errorf("no data"))
		return
	}
	jsonObj(c, m, nil)
}

func (ctrl *MonitorController) listRules(c *gin.Context) {
	re := monitor.GlobalRuleEngine()
	if re == nil {
		jsonMsg(c, "Rule engine not initialized", fmt.Errorf("not ready"))
		return
	}
	rules := re.ListRules()
	var result []gin.H
	for _, r := range rules {
		result = append(result, gin.H{
			"id":           r.ID,
			"dbId":         r.DBID,
			"name":         r.Name,
			"description":  r.Description,
			"protocolId":   r.ProtocolID,
			"metric":       r.Metric,
			"condition":    r.Condition,
			"threshold":    r.Threshold,
			"duration":     int64(r.Duration.Seconds()),
			"severity":     r.Severity,
			"enabled":      r.Enabled,
			"cooldown":     int64(r.Cooldown.Seconds()),
			"channels":     r.Channels,
			"autoRecovery": r.AutoRecovery,
			"lastFiredAt":  r.LastFiredAt.UnixMilli(),
			"createdAt":    r.CreatedAt.UnixMilli(),
			"updatedAt":    r.UpdatedAt.UnixMilli(),
		})
	}
	jsonObj(c, result, nil)
}

func (ctrl *MonitorController) createRule(c *gin.Context) {
	re := monitor.GlobalRuleEngine()
	if re == nil {
		jsonMsg(c, "Rule engine not initialized", fmt.Errorf("not ready"))
		return
	}

	var req struct {
		Name         string   `json:"name" binding:"required"`
		Description  string   `json:"description"`
		ProtocolID   string   `json:"protocolId" binding:"required"`
		Metric       string   `json:"metric" binding:"required"`
		Condition    string   `json:"condition" binding:"required"`
		Threshold    float64  `json:"threshold"`
		Duration     int64    `json:"duration"`
		Severity     string   `json:"severity"`
		Enabled      bool     `json:"enabled"`
		Cooldown     int64    `json:"cooldown"`
		Channels     []string `json:"channels"`
		AutoRecovery bool     `json:"autoRecovery"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonMsg(c, "Invalid request", err)
		return
	}

	if req.Severity == "" {
		req.Severity = string(monitor.SeverityWarning)
	}
	if req.Cooldown <= 0 {
		req.Cooldown = 300
	}
	if req.Duration <= 0 {
		req.Duration = 30
	}

	rule := &monitor.AlertRule{
		ID:           fmt.Sprintf("rule-%d", time.Now().UnixNano()),
		Name:         req.Name,
		Description:  req.Description,
		ProtocolID:   req.ProtocolID,
		Metric:       req.Metric,
		Condition:    req.Condition,
		Threshold:    req.Threshold,
		Duration:     time.Duration(req.Duration) * time.Second,
		Severity:     monitor.Severity(req.Severity),
		Enabled:      req.Enabled,
		Cooldown:     time.Duration(req.Cooldown) * time.Second,
		Channels:     req.Channels,
		AutoRecovery: req.AutoRecovery,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	db := database.GetDB()
	if db != nil {
		dbRule := &model.AlertRule{
			Name:         req.Name,
			Description:  req.Description,
			ProtocolId:   req.ProtocolID,
			Metric:       req.Metric,
			Condition:    req.Condition,
			Threshold:    req.Threshold,
			Duration:     req.Duration,
			Severity:     req.Severity,
			Enabled:      req.Enabled,
			Cooldown:     req.Cooldown,
			Channels:     joinChannels(req.Channels),
			AutoRecovery: req.AutoRecovery,
		}
		if err := db.Create(dbRule).Error; err != nil {
			jsonMsg(c, "Failed to save rule", err)
			return
		}
		rule.ID = fmt.Sprintf("rule-%d", dbRule.Id)
		rule.DBID = dbRule.Id
	}

	re.AddRule(rule)
	jsonObj(c, rule, nil)
}

func (ctrl *MonitorController) getRule(c *gin.Context) {
	re := monitor.GlobalRuleEngine()
	if re == nil {
		c.Status(http.StatusNotFound)
		return
	}
	id := c.Param("id")
	rule := re.GetRule(id)
	if rule == nil {
		c.Status(http.StatusNotFound)
		return
	}
	jsonObj(c, rule, nil)
}

func (ctrl *MonitorController) updateRule(c *gin.Context) {
	re := monitor.GlobalRuleEngine()
	if re == nil {
		jsonMsg(c, "Rule engine not initialized", fmt.Errorf("not ready"))
		return
	}

	id := c.Param("id")
	var req struct {
		Name         string   `json:"name"`
		Description  string   `json:"description"`
		ProtocolID   string   `json:"protocolId"`
		Metric       string   `json:"metric"`
		Condition    string   `json:"condition"`
		Threshold    float64  `json:"threshold"`
		Duration     int64    `json:"duration"`
		Severity     string   `json:"severity"`
		Enabled      *bool    `json:"enabled"`
		Cooldown     int64    `json:"cooldown"`
		Channels     []string `json:"channels"`
		AutoRecovery *bool    `json:"autoRecovery"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonMsg(c, "Invalid request", err)
		return
	}

	rule := re.GetRule(id)
	if rule == nil {
		c.Status(http.StatusNotFound)
		return
	}

	if req.Name != "" {
		rule.Name = req.Name
	}
	if req.Description != "" {
		rule.Description = req.Description
	}
	if req.ProtocolID != "" {
		rule.ProtocolID = req.ProtocolID
	}
	if req.Metric != "" {
		rule.Metric = req.Metric
	}
	if req.Condition != "" {
		rule.Condition = req.Condition
	}
	if req.Threshold != 0 {
		rule.Threshold = req.Threshold
	}
	if req.Duration > 0 {
		rule.Duration = time.Duration(req.Duration) * time.Second
	}
	if req.Severity != "" {
		rule.Severity = monitor.Severity(req.Severity)
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
	if req.Cooldown > 0 {
		rule.Cooldown = time.Duration(req.Cooldown) * time.Second
	}
	if req.Channels != nil {
		rule.Channels = req.Channels
	}
	if req.AutoRecovery != nil {
		rule.AutoRecovery = *req.AutoRecovery
	}
	rule.UpdatedAt = time.Now()

	re.UpdateRule(rule)

	// Persist changes to database
	db := database.GetDB()
	if db != nil && rule.DBID > 0 {
		updates := map[string]any{
			"name":          rule.Name,
			"description":   rule.Description,
			"protocol_id":   rule.ProtocolID,
			"metric":        rule.Metric,
			"condition":     rule.Condition,
			"threshold":     rule.Threshold,
			"duration":      int64(rule.Duration.Seconds()),
			"severity":      string(rule.Severity),
			"enabled":       rule.Enabled,
			"cooldown":      int64(rule.Cooldown.Seconds()),
			"channels":      joinChannels(rule.Channels),
			"auto_recovery": rule.AutoRecovery,
		}
		db.Model(&model.AlertRule{}).Where("id = ?", rule.DBID).Updates(updates)
	}

	jsonObj(c, rule, nil)
}

func (ctrl *MonitorController) deleteRule(c *gin.Context) {
	re := monitor.GlobalRuleEngine()
	if re == nil {
		c.Status(http.StatusNoContent)
		return
	}
	id := c.Param("id")

	// Get the rule from the engine before removing it
	rule := re.GetRule(id)
	if rule == nil {
		c.Status(http.StatusNotFound)
		return
	}

	re.RemoveRule(id)

	db := database.GetDB()
	if db != nil {
		// Try by DBID first, fall back to parsing the string ID
		if rule.DBID > 0 {
			db.Delete(&model.AlertRule{}, rule.DBID)
		} else {
			var dbID int
			if _, err := fmt.Sscanf(id, "rule-%d", &dbID); err == nil {
				db.Delete(&model.AlertRule{}, dbID)
			}
		}
	}
	c.Status(http.StatusNoContent)
}

func (ctrl *MonitorController) testRule(c *gin.Context) {
	re := monitor.GlobalRuleEngine()
	if re == nil {
		c.Status(http.StatusNotFound)
		return
	}

	id := c.Param("id")
	rule := re.GetRule(id)
	if rule == nil {
		c.Status(http.StatusNotFound)
		return
	}

	event := monitor.AlertEvent{
		RuleID:     rule.ID,
		RuleName:   rule.Name,
		ProtocolID: rule.ProtocolID,
		Severity:   rule.Severity,
		Status:     monitor.AlertStatusFiring,
		Message:    fmt.Sprintf("[TEST] %s = %.2f (threshold: %.2f)", rule.Metric, rule.Threshold, rule.Threshold),
		Metric:     rule.Metric,
		Value:      rule.Threshold,
		Threshold:  rule.Threshold,
		FiredAt:    time.Now(),
		Labels:     rule.Labels,
	}

	if err := re.SendAlert(event); err != nil {
		jsonMsg(c, "Test alert failed", err)
		return
	}
	jsonObj(c, gin.H{"status": "sent", "channels": rule.Channels}, nil)
}

func (ctrl *MonitorController) alertHistory(c *gin.Context) {
	hs := monitor.GlobalHistoryStore()
	if hs == nil {
		jsonMsg(c, "History store not initialized", fmt.Errorf("not ready"))
		return
	}

	opts := monitor.HistoryQuery{
		ProtocolID: c.Query("protocolId"),
		Severity:   c.Query("severity"),
		Status:     c.Query("status"),
	}

	if offset, err := strconv.Atoi(c.Query("offset")); err == nil {
		opts.Offset = offset
	}
	if limit, err := strconv.Atoi(c.Query("limit")); err == nil {
		opts.Limit = limit
	}

	rows, total, err := hs.List(opts)
	if err != nil {
		jsonMsg(c, "Failed to list history", err)
		return
	}
	jsonObj(c, gin.H{"items": rows, "total": total}, nil)
}

func (ctrl *MonitorController) ackAlert(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	db := database.GetDB()
	if db == nil {
		jsonMsg(c, "Database not available", fmt.Errorf("no db"))
		return
	}
	if err := db.Model(&model.AlertHistory{}).Where("id = ?", id).
		Update("status", "acknowledged").Error; err != nil {
		jsonMsg(c, "Failed to acknowledge", err)
		return
	}
	jsonObj(c, gin.H{"status": "acknowledged"}, nil)
}

func (ctrl *MonitorController) listNotifiers(c *gin.Context) {
	re := monitor.GlobalRuleEngine()
	if re == nil {
		jsonObj(c, []gin.H{}, nil)
		return
	}
	notifiers := re.ListNotifiers()
	var result []gin.H
	for _, n := range notifiers {
		result = append(result, gin.H{
			"id":     n.ID(),
			"name":   n.Name(),
			"type":   n.Type(),
			"config": n.Config(),
		})
	}
	jsonObj(c, result, nil)
}

func joinChannels(chs []string) string {
	if len(chs) == 0 {
		return ""
	}
	b := make([]byte, 0, len(chs)*10)
	for i, ch := range chs {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, ch...)
	}
	return string(b)
}
