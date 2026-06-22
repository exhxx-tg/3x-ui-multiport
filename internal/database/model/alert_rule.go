package model

type AlertRule struct {
	Id          int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string `json:"name" gorm:"not null" validate:"required"`
	Description string `json:"description" gorm:"type:text"`
	ProtocolId  string `json:"protocolId" gorm:"column:protocol_id;index"`
	Metric      string `json:"metric" gorm:"not null"`
	Condition   string `json:"condition" gorm:"not null"`
	Threshold   float64 `json:"threshold" gorm:"default:0"`
	Duration    int64  `json:"duration" gorm:"default:30"`
	Severity    string `json:"severity" gorm:"default:warning"`
	Enabled     bool   `json:"enabled" gorm:"default:true"`
	Cooldown    int64  `json:"cooldown" gorm:"default:300"`
	Channels    string `json:"channels" gorm:"type:text"`
	AutoRecovery bool  `json:"autoRecovery" gorm:"default:false"`
	LastFiredAt int64  `json:"lastFiredAt" gorm:"default:0"`
	CreatedAt   int64  `json:"createdAt" gorm:"autoCreateTime:milli"`
	UpdatedAt   int64  `json:"updatedAt" gorm:"autoUpdateTime:milli"`
}

func (AlertRule) TableName() string { return "alert_rules" }
