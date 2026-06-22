package model

type IPAccessRule struct {
	Id        int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Type      string `json:"type" gorm:"column:type;size:8;not null;index" validate:"required,oneof=allow block"`
	CIDR      string `json:"cidr" gorm:"column:cidr;size:64;not null" validate:"required"`
	Remark    string `json:"remark" gorm:"column:remark;size:256"`
	Enabled   bool   `json:"enabled" gorm:"default:true"`
	Priority  int    `json:"priority" gorm:"default:0"`
	CreatedAt int64  `json:"createdAt" gorm:"autoCreateTime:milli"`
	UpdatedAt int64  `json:"updatedAt" gorm:"autoUpdateTime:milli"`
}

func (IPAccessRule) TableName() string { return "ip_access_rules" }

type LoginAttempt struct {
	Id        int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Username  string `json:"username" gorm:"column:username;size:64;index"`
	IP        string `json:"ip" gorm:"column:ip;size:45;index"`
	Success   bool   `json:"success" gorm:"default:false"`
	UserAgent string `json:"userAgent" gorm:"column:user_agent;size:256"`
	CreatedAt int64  `json:"createdAt" gorm:"autoCreateTime:milli"`
}

func (LoginAttempt) TableName() string { return "login_attempts" }

type ActiveSession struct {
	Id           int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId       int    `json:"userId" gorm:"column:user_id;index;not null"`
	Username     string `json:"username" gorm:"column:username;size:64"`
	Token        string `json:"token" gorm:"column:token;size:512;not null;index"`
	IP           string `json:"ip" gorm:"column:ip;size:45"`
	UserAgent    string `json:"userAgent" gorm:"column:user_agent;size:256"`
	LastActivity int64  `json:"lastActivity" gorm:"column:last_activity;index"`
	ExpiresAt    int64  `json:"expiresAt" gorm:"column:expires_at"`
	CreatedAt    int64  `json:"createdAt" gorm:"autoCreateTime:milli"`
}

func (ActiveSession) TableName() string { return "active_sessions" }

type SecurityEvent struct {
	Id        int    `json:"id" gorm:"primaryKey;autoIncrement"`
	EventType string `json:"eventType" gorm:"column:event_type;size:64;not null;index"`
	Severity  string `json:"severity" gorm:"column:severity;size:16;default:info"`
	Message   string `json:"message" gorm:"column:message;type:text"`
	Detail    string `json:"detail" gorm:"column:detail;type:text"`
	IP        string `json:"ip" gorm:"column:ip;size:45"`
	UserId    int    `json:"userId" gorm:"column:user_id"`
	CreatedAt int64  `json:"createdAt" gorm:"autoCreateTime:milli"`
}

func (SecurityEvent) TableName() string { return "security_events" }

type BackupCode struct {
	Id        int    `json:"id" gorm:"primaryKey;autoIncrement"`
	CodeHash  string `json:"codeHash" gorm:"column:code_hash;size:128;not null"`
	Consumed  bool   `json:"consumed" gorm:"column:consumed;default:false"`
	UserId    int    `json:"userId" gorm:"column:user_id;not null;index"`
	CreatedAt int64  `json:"createdAt" gorm:"autoCreateTime:milli"`
}

func (BackupCode) TableName() string { return "backup_codes" }
