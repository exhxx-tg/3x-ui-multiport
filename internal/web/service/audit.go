package service

import (
	"fmt"
	"time"

	"github.com/exhxx-tg/3x-ui-multiport/internal/database"
	"github.com/exhxx-tg/3x-ui-multiport/internal/database/model"
)

// AuditService provides methods to record and query audit log entries.
type AuditService struct{}

// Record writes an audit log entry to the database.
func (s *AuditService) Record(userID int, username, action, resource, resourceID, detail, ip, userAgent, status string) {
	db := database.GetDB()
	if db == nil {
		return
	}
	entry := &model.AuditLog{
		UserId:     userID,
		Username:   username,
		Action:     action,
		Resource:   resource,
		ResourceId: resourceID,
		Detail:     detail,
		Ip:         ip,
		UserAgent:  truncateString(userAgent, 256),
		Status:     status,
		CreatedAt:  time.Now().UnixMilli(),
	}
	if err := db.Create(entry).Error; err != nil {
		fmt.Printf("audit: failed to record event: %v\n", err)
	}
}

// List returns audit log entries with pagination and optional filters.
func (s *AuditService) List(offset, limit int, action, resource, username, status string) ([]model.AuditLog, int64, error) {
	db := database.GetDB()
	if db == nil {
		return nil, 0, fmt.Errorf("database not available")
	}

	query := db.Model(&model.AuditLog{})
	if action != "" {
		query = query.Where("action = ?", action)
	}
	if resource != "" {
		query = query.Where("resource = ?", resource)
	}
	if username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	var rows []model.AuditLog
	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func truncateString(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen]
	}
	return s
}
