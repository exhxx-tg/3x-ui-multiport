package monitor

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type AlertRule struct {
	ID             string            `json:"id"`
	DBID           int               `json:"dbId,omitempty"`
	Name           string            `json:"name"`
	Description    string            `json:"description"`
	ProtocolID     string            `json:"protocolId"`
	Metric         string            `json:"metric"`
	Condition      string            `json:"condition"`
	Threshold      float64           `json:"threshold"`
	Duration       time.Duration     `json:"duration"`
	Severity       Severity          `json:"severity"`
	Enabled        bool              `json:"enabled"`
	Cooldown       time.Duration     `json:"cooldown"`
	Channels       []string          `json:"channels"`
	AutoRecovery   bool              `json:"autoRecovery"`
	Labels         map[string]string `json:"labels,omitempty"`
	LastFiredAt    time.Time         `json:"lastFiredAt,omitempty"`
	CreatedAt      time.Time         `json:"createdAt"`
	UpdatedAt      time.Time         `json:"updatedAt"`
}

type RuleEngine struct {
	mu        sync.RWMutex
	rules     map[string]*AlertRule
	evalHist  map[string]time.Time
	notifiers map[string]Notifier
}

type Notifier interface {
	ID() string
	Name() string
	Send(alert AlertEvent) error
	Type() string
	Config() map[string]any
}

type AlertEvent struct {
	RuleID      string            `json:"ruleId"`
	RuleName    string            `json:"ruleName"`
	ProtocolID  string            `json:"protocolId"`
	Severity    Severity          `json:"severity"`
	Status      AlertStatus       `json:"status"`
	Message     string            `json:"message"`
	Metric      string            `json:"metric"`
	Value       float64           `json:"value"`
	Threshold   float64           `json:"threshold"`
	FiredAt     time.Time         `json:"firedAt"`
	ResolvedAt  *time.Time        `json:"resolvedAt,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	HistoryID   int64             `json:"historyId"`
}

func NewRuleEngine() *RuleEngine {
	return &RuleEngine{
		rules:     make(map[string]*AlertRule),
		evalHist:  make(map[string]time.Time),
		notifiers: make(map[string]Notifier),
	}
}

func (e *RuleEngine) RegisterNotifier(n Notifier) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.notifiers[n.ID()] = n
}

func (e *RuleEngine) UnregisterNotifier(id string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.notifiers, id)
}

func (e *RuleEngine) Notifier(id string) Notifier {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.notifiers[id]
}

func (e *RuleEngine) ListNotifiers() []Notifier {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]Notifier, 0, len(e.notifiers))
	for _, n := range e.notifiers {
		out = append(out, n)
	}
	return out
}

func (e *RuleEngine) AddRule(rule *AlertRule) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rules[rule.ID] = rule
}

func (e *RuleEngine) RemoveRule(id string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.rules, id)
}

func (e *RuleEngine) GetRule(id string) *AlertRule {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.rules[id]
}

func (e *RuleEngine) ListRules() []*AlertRule {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]*AlertRule, 0, len(e.rules))
	for _, r := range e.rules {
		out = append(out, r)
	}
	return out
}

func (e *RuleEngine) UpdateRule(rule *AlertRule) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.rules[rule.ID]; !ok {
		return ErrRuleNotFound
	}
	rule.UpdatedAt = time.Now()
	e.rules[rule.ID] = rule
	return nil
}

func (e *RuleEngine) Evaluate(value float64, labels map[string]string) []AlertEvent {
	e.mu.RLock()
	rules := make([]*AlertRule, 0, len(e.rules))
	for _, r := range e.rules {
		rules = append(rules, r)
	}
	e.mu.RUnlock()

	var events []AlertEvent
	now := time.Now()

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		if lastFire, ok := e.evalHist[rule.ID]; ok {
			if now.Sub(lastFire) < rule.Cooldown {
				continue
			}
		}

		if !e.matchesCondition(value, rule.Condition, rule.Threshold) {
			continue
		}

		event := AlertEvent{
			RuleID:     rule.ID,
			RuleName:   rule.Name,
			ProtocolID: rule.ProtocolID,
			Severity:   rule.Severity,
			Status:     AlertStatusFiring,
			Message:    fmt.Sprintf("[%s] %s = %.2f (threshold: %.2f)", rule.ProtocolID, rule.Metric, value, rule.Threshold),
			Metric:     rule.Metric,
			Value:      value,
			Threshold:  rule.Threshold,
			FiredAt:    now,
			Labels:     mergeLabels(rule.Labels, labels),
		}
		events = append(events, event)
		e.evalHist[rule.ID] = now
	}

	return events
}

func (e *RuleEngine) matchesCondition(value float64, condition string, threshold float64) bool {
	switch condition {
	case "gt", ">":
		return value > threshold
	case "gte", ">=":
		return value >= threshold
	case "lt", "<":
		return value < threshold
	case "lte", "<=":
		return value <= threshold
	case "eq", "==":
		return value == threshold
	case "neq", "!=":
		return value != threshold
	default:
		return false
	}
}

func (e *RuleEngine) SendAlert(event AlertEvent) error {
	e.mu.RLock()
	rule := e.rules[event.RuleID]
	e.mu.RUnlock()

	if rule == nil {
		return ErrRuleNotFound
	}

	for _, ch := range rule.Channels {
		n := e.Notifier(ch)
		if n == nil {
			continue
		}
		if err := n.Send(event); err != nil {
			return fmt.Errorf("notifier %s failed: %w", ch, err)
		}
	}
	return nil
}

func (e *RuleEngine) ResolveAlert(ruleID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	rule, ok := e.rules[ruleID]
	if !ok {
		return ErrRuleNotFound
	}

	rule.LastFiredAt = time.Time{}
	delete(e.evalHist, ruleID)
	return nil
}

func mergeLabels(a, b map[string]string) map[string]string {
	out := make(map[string]string, len(a)+len(b))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

type WebhookNotifier struct {
	id     string
	name   string
	url    string
	secret string
}

func NewWebhookNotifier(id, name, url, secret string) *WebhookNotifier {
	return &WebhookNotifier{id: id, name: name, url: url, secret: secret}
}

func (w *WebhookNotifier) ID() string                   { return w.id }
func (w *WebhookNotifier) Name() string                 { return w.name }
func (w *WebhookNotifier) Type() string                 { return "webhook" }
func (w *WebhookNotifier) Config() map[string]any {
	return map[string]any{"url": w.url}
}

func (w *WebhookNotifier) Send(event AlertEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return postJSON(w.url, payload, w.secret)
}

type LogNotifier struct {
	id   string
	name string
}

func NewLogNotifier(id, name string) *LogNotifier {
	return &LogNotifier{id: id, name: name}
}

func (l *LogNotifier) ID() string   { return l.id }
func (l *LogNotifier) Name() string { return l.name }
func (l *LogNotifier) Type() string { return "log" }
func (l *LogNotifier) Config() map[string]any { return nil }

func (l *LogNotifier) Send(event AlertEvent) error {
	logEntry, _ := json.Marshal(event)
	fmt.Println("alert:", string(logEntry))
	return nil
}
