package monitor

import "time"

type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityWarning  Severity = "warning"
	SeverityInfo     Severity = "info"
)

type AlertStatus string

const (
	AlertStatusFiring   AlertStatus = "firing"
	AlertStatusResolved AlertStatus = "resolved"
	AlertStatusAcked    AlertStatus = "acknowledged"
)

type CheckResult struct {
	ProtocolID string        `json:"protocolId"`
	Healthy    bool          `json:"healthy"`
	Latency    time.Duration `json:"latency"`
	Error      string        `json:"error,omitempty"`
	CheckedAt  time.Time     `json:"checkedAt"`
	Details    map[string]any `json:"details,omitempty"`
}

type ProtocolMetrics struct {
	ProtocolID    string    `json:"protocolId"`
	UpBytes       int64     `json:"upBytes"`
	DownBytes     int64     `json:"downBytes"`
	Connections   int       `json:"connections"`
	ActiveUsers   int       `json:"activeUsers"`
	ErrorCount    int       `json:"errorCount"`
	UptimeSeconds int64     `json:"uptimeSeconds"`
	CollectedAt   time.Time `json:"collectedAt"`
}

type HealthSummary struct {
	TotalProtocols int            `json:"totalProtocols"`
	Healthy        int            `json:"healthy"`
	Unhealthy      int            `json:"unhealthy"`
	Unknown        int            `json:"unknown"`
	Results        []CheckResult  `json:"results"`
	GeneratedAt    time.Time      `json:"generatedAt"`
}
