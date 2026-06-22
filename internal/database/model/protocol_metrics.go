package model

type ProtocolMetrics struct {
	Id            int    `json:"id" gorm:"primaryKey;autoIncrement"`
	ProtocolId    string `json:"protocolId" gorm:"column:protocol_id;index"`
	UpBytes       int64  `json:"upBytes" gorm:"default:0"`
	DownBytes     int64  `json:"downBytes" gorm:"default:0"`
	Connections   int    `json:"connections" gorm:"default:0"`
	ActiveUsers   int    `json:"activeUsers" gorm:"default:0"`
	ErrorCount    int    `json:"errorCount" gorm:"default:0"`
	UptimeSeconds int64  `json:"uptimeSeconds" gorm:"default:0"`
	CollectedAt   int64  `json:"collectedAt" gorm:"index"`
}

func (ProtocolMetrics) TableName() string { return "protocol_metrics" }
