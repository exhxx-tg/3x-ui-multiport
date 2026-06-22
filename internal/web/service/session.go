package service

import (
	"fmt"
	"time"

	"github.com/exhxx-tg/3x-ui-multiport/internal/database"
	"github.com/exhxx-tg/3x-ui-multiport/internal/database/model"
)

type SessionService struct{}

func (s *SessionService) CreateSession(userID int, username, token, ip, userAgent string, maxAgeMinutes int) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not available")
	}

	now := time.Now()
	session := &model.ActiveSession{
		UserId:       userID,
		Username:     username,
		Token:        token,
		IP:           ip,
		UserAgent:    userAgent,
		LastActivity: now.UnixMilli(),
		ExpiresAt:    now.Add(time.Duration(maxAgeMinutes) * time.Minute).UnixMilli(),
		CreatedAt:    now.UnixMilli(),
	}

	return db.Create(session).Error
}

func (s *SessionService) GetActiveSessions() ([]model.ActiveSession, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var sessions []model.ActiveSession
	now := time.Now().UnixMilli()
	if err := db.Where("expires_at > ?", now).Order("last_activity DESC").Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

func (s *SessionService) RevokeSession(sessionID int) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not available")
	}

	return db.Delete(&model.ActiveSession{}, sessionID).Error
}

func (s *SessionService) RevokeAllSessions(userID int) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not available")
	}

	return db.Where("user_id = ?", userID).Delete(&model.ActiveSession{}).Error
}

func (s *SessionService) UpdateActivity(token string) {
	db := database.GetDB()
	if db == nil {
		return
	}

	now := time.Now().UnixMilli()
	db.Model(&model.ActiveSession{}).Where("token = ?", token).Update("last_activity", now)
}

func (s *SessionService) IsSessionValid(token string) bool {
	db := database.GetDB()
	if db == nil {
		return false
	}

	var count int64
	now := time.Now().UnixMilli()
	db.Model(&model.ActiveSession{}).Where("token = ? AND expires_at > ?", token, now).Count(&count)
	return count > 0
}

func (s *SessionService) CleanupExpired() {
	db := database.GetDB()
	if db == nil {
		return
	}

	now := time.Now().UnixMilli()
	db.Where("expires_at <= ?", now).Delete(&model.ActiveSession{})
}
