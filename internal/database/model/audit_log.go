package model

type AuditLog struct {
	Id         int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId     int    `json:"userId" gorm:"column:user_id;index"`
	Username   string `json:"username" gorm:"column:username;size:64"`
	Action     string `json:"action" gorm:"column:action;size:128;not null"`
	Resource   string `json:"resource" gorm:"column:resource;size:128"`
	ResourceId string `json:"resourceId" gorm:"column:resource_id;size:64"`
	Detail     string `json:"detail" gorm:"column:detail;type:text"`
	Ip         string `json:"ip" gorm:"column:ip;size:45"`
	UserAgent  string `json:"userAgent" gorm:"column:user_agent;size:256"`
	Status     string `json:"status" gorm:"column:status;size:16;default:success"`
	CreatedAt  int64  `json:"createdAt" gorm:"autoCreateTime:milli"`
}

func (AuditLog) TableName() string { return "audit_logs" }
