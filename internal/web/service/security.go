package service

import (
	"fmt"
	"time"

	"github.com/exhxx-tg/3x-ui-multiport/internal/database"
	"github.com/exhxx-tg/3x-ui-multiport/internal/database/model"
)

type SecurityService struct{}

type SecurityOverview struct {
	TotalSessions       int                `json:"totalSessions"`
	LoginAttempts24h    int                `json:"loginAttempts24h"`
	FailedLogins24h     int                `json:"failedLogins24h"`
	ActiveRules         int                `json:"activeRules"`
	BlockedIPs          int                `json:"blockedIPs"`
	AllowedIPs          int                `json:"allowedIPs"`
	ExpiringCerts       int                `json:"expiringCerts"`
	RecentEvents        []SecurityEventRow `json:"recentEvents"`
	RecentLoginAttempts []LoginAttemptRow  `json:"recentLoginAttempts"`
}

type SecurityEventRow struct {
	Id        int    `json:"id"`
	EventType string `json:"eventType"`
	Severity  string `json:"severity"`
	Message   string `json:"message"`
	IP        string `json:"ip"`
	CreatedAt int64  `json:"createdAt"`
}

type LoginAttemptRow struct {
	Id        int    `json:"id"`
	Username  string `json:"username"`
	IP        string `json:"ip"`
	Success   bool   `json:"success"`
	CreatedAt int64  `json:"createdAt"`
}

func (s *SecurityService) GetOverview() (*SecurityOverview, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not available")
	}

	overview := &SecurityOverview{}

	cutoff24h := time.Now().Add(-24 * time.Hour).UnixMilli()

	var c int64
	db.Model(&model.ActiveSession{}).Where("expires_at > ?", time.Now().UnixMilli()).Count(&c)
	overview.TotalSessions = int(c)

	db.Model(&model.LoginAttempt{}).Where("created_at > ?", cutoff24h).Count(&c)
	overview.LoginAttempts24h = int(c)

	db.Model(&model.LoginAttempt{}).Where("created_at > ? AND success = ?", cutoff24h, false).Count(&c)
	overview.FailedLogins24h = int(c)

	db.Model(&model.IPAccessRule{}).Where("enabled = ?", true).Count(&c)
	overview.ActiveRules = int(c)

	db.Model(&model.IPAccessRule{}).Where("enabled = ? AND type = ?", true, "block").Count(&c)
	overview.BlockedIPs = int(c)

	db.Model(&model.IPAccessRule{}).Where("enabled = ? AND type = ?", true, "allow").Count(&c)
	overview.AllowedIPs = int(c)

	expiryThreshold := time.Now().Add(30 * 24 * time.Hour).UnixMilli()
	db.Model(&model.Certificate{}).Where("not_after > 0 AND not_after <= ?", expiryThreshold).Count(&c)
	overview.ExpiringCerts = int(c)

	var events []SecurityEventRow
	db.Raw(`
		SELECT id, event_type, severity, message, ip, created_at
		FROM security_events
		ORDER BY id DESC
		LIMIT 10
	`).Scan(&events)
	overview.RecentEvents = events

	var attempts []LoginAttemptRow
	db.Raw(`
		SELECT id, username, ip, success, created_at
		FROM login_attempts
		ORDER BY id DESC
		LIMIT 10
	`).Scan(&attempts)
	overview.RecentLoginAttempts = attempts

	return overview, nil
}

func (s *SecurityService) RecordLoginAttempt(username, ip, userAgent string, success bool) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not available")
	}

	attempt := &model.LoginAttempt{
		Username:  username,
		IP:        ip,
		Success:   success,
		UserAgent: userAgent,
		CreatedAt: time.Now().UnixMilli(),
	}
	return db.Create(attempt).Error
}

func (s *SecurityService) RecordSecurityEvent(eventType, severity, message, detail, ip string, userID int) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not available")
	}

	event := &model.SecurityEvent{
		EventType: eventType,
		Severity:  severity,
		Message:   message,
		Detail:    detail,
		IP:        ip,
		UserId:    userID,
		CreatedAt: time.Now().UnixMilli(),
	}
	return db.Create(event).Error
}

func (s *SecurityService) ListLoginAttempts(offset, limit int) ([]LoginAttemptRow, int64, error) {
	db := database.GetDB()
	if db == nil {
		return nil, 0, fmt.Errorf("database not available")
	}

	var total int64
	db.Model(&model.LoginAttempt{}).Count(&total)

	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	var rows []LoginAttemptRow
	if err := db.Raw(`
		SELECT id, username, ip, success, created_at
		FROM login_attempts
		ORDER BY id DESC
		LIMIT ? OFFSET ?
	`, limit, offset).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}

func (s *SecurityService) ListSecurityEvents(offset, limit int) ([]SecurityEventRow, int64, error) {
	db := database.GetDB()
	if db == nil {
		return nil, 0, fmt.Errorf("database not available")
	}

	var total int64
	db.Model(&model.SecurityEvent{}).Count(&total)

	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	var rows []SecurityEventRow
	if err := db.Raw(`
		SELECT id, event_type, severity, message, ip, created_at
		FROM security_events
		ORDER BY id DESC
		LIMIT ? OFFSET ?
	`, limit, offset).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}

func (s *SecurityService) AddIPAccessRule(ruleType, cidr, remark string, priority int) (*model.IPAccessRule, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not available")
	}

	rule := &model.IPAccessRule{
		Type:      ruleType,
		CIDR:      cidr,
		Remark:    remark,
		Enabled:   true,
		Priority:  priority,
		CreatedAt: time.Now().UnixMilli(),
		UpdatedAt: time.Now().UnixMilli(),
	}

	if err := db.Create(rule).Error; err != nil {
		return nil, err
	}

	RefreshIPAccessGlobal()
	return rule, nil
}

func (s *SecurityService) ListIPAccessRules() ([]model.IPAccessRule, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var rules []model.IPAccessRule
	if err := db.Order("priority ASC, id ASC").Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

func (s *SecurityService) UpdateIPAccessRule(id int, ruleType, cidr, remark string, enabled bool, priority int) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not available")
	}

	updates := map[string]any{
		"updated_at": time.Now().UnixMilli(),
	}
	if ruleType != "" {
		updates["type"] = ruleType
	}
	if cidr != "" {
		updates["cidr"] = cidr
	}
	if remark != "" {
		updates["remark"] = remark
	}
	updates["enabled"] = enabled
	updates["priority"] = priority

	if err := db.Model(&model.IPAccessRule{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return err
	}

	RefreshIPAccessGlobal()
	return nil
}

func (s *SecurityService) DeleteIPAccessRule(id int) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not available")
	}

	if err := db.Delete(&model.IPAccessRule{}, id).Error; err != nil {
		return err
	}

	RefreshIPAccessGlobal()
	return nil
}

var RefreshIPAccessFunc func()

func RefreshIPAccessGlobal() {
	if RefreshIPAccessFunc != nil {
		RefreshIPAccessFunc()
	}
}
