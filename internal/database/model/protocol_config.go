package model

type ProtocolConfig struct {
	Id         int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Protocol   string `json:"protocol" gorm:"column:protocol;uniqueIndex;not null"`
	Config     string `json:"config" gorm:"type:text"`
	Version    string `json:"version" gorm:"default:1.0.0"`
	Enabled    bool   `json:"enabled" gorm:"default:true"`
	CreatedAt  int64  `json:"createdAt" gorm:"autoCreateTime:milli"`
	UpdatedAt  int64  `json:"updatedAt" gorm:"autoUpdateTime:milli"`
}

func (ProtocolConfig) TableName() string { return "protocol_configs" }
