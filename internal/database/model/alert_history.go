package model

type AlertHistory struct {
	Id         int    `json:"id" gorm:"primaryKey;autoIncrement"`
	RuleId     int    `json:"ruleId" gorm:"column:rule_id;index"`
	RuleName   string `json:"ruleName" gorm:"column:rule_name"`
	ProtocolId string `json:"protocolId" gorm:"column:protocol_id;index"`
	Severity   string `json:"severity" gorm:"default:warning"`
	Status     string `json:"status" gorm:"default:firing"`
	Message    string `json:"message" gorm:"type:text"`
	Metric     string `json:"metric"`
	Value      float64 `json:"value"`
	Threshold  float64 `json:"threshold"`
	Labels     string `json:"labels" gorm:"type:text"`
	ResolvedAt int64  `json:"resolvedAt" gorm:"default:0"`
	CreatedAt  int64  `json:"createdAt" gorm:"autoCreateTime:milli"`
}

func (AlertHistory) TableName() string { return "alert_history" }
