package monitor

import (
	"fmt"
	"time"
)

// NotifierFactory provides a way to create notifiers without importing
// service packages directly (avoids circular dependencies).
type NotifierFactory struct {
	TelegramFn func(msg string) error
	EmailFn    func(subject, body string) error
}

var globalNotifierFactory *NotifierFactory

// SetNotifierFactory configures the factory functions used to send real
// Telegram and Email alerts. Called once at startup from web.go.
func SetNotifierFactory(f *NotifierFactory) {
	globalNotifierFactory = f
}

// GetNotifierFactory returns the current factory, or nil if not set.
func GetNotifierFactory() *NotifierFactory {
	return globalNotifierFactory
}

// ---- Telegram Notifier ----

type TelegramNotifier struct {
	id   string
	name string
}

func NewTelegramNotifier(id, name string) *TelegramNotifier {
	return &TelegramNotifier{id: id, name: name}
}

func (n *TelegramNotifier) ID() string   { return n.id }
func (n *TelegramNotifier) Name() string { return n.name }
func (n *TelegramNotifier) Type() string { return "telegram" }
func (n *TelegramNotifier) Config() map[string]any {
	return map[string]any{"channel": "admins"}
}

func (n *TelegramNotifier) Send(event AlertEvent) error {
	factory := GetNotifierFactory()
	if factory == nil || factory.TelegramFn == nil {
		return fmt.Errorf("telegram notifier not configured")
	}

	msg := formatAlertMessage(event)
	return factory.TelegramFn(msg)
}

// ---- Email Notifier ----

type EmailNotifier struct {
	id   string
	name string
}

func NewEmailNotifier(id, name string) *EmailNotifier {
	return &EmailNotifier{id: id, name: name}
}

func (n *EmailNotifier) ID() string   { return n.id }
func (n *EmailNotifier) Name() string { return n.name }
func (n *EmailNotifier) Type() string { return "email" }
func (n *EmailNotifier) Config() map[string]any {
	return map[string]any{}
}

func (n *EmailNotifier) Send(event AlertEvent) error {
	factory := GetNotifierFactory()
	if factory == nil || factory.EmailFn == nil {
		return fmt.Errorf("email notifier not configured")
	}

	subject := fmt.Sprintf("[%s] %s - %s", event.Severity, event.RuleName, event.ProtocolID)
	body := formatAlertMessage(event)
	return factory.EmailFn(subject, body)
}

func formatAlertMessage(event AlertEvent) string {
	severity := string(event.Severity)
	msg := fmt.Sprintf("🔔 *Alert: %s*\n", event.RuleName)
	msg += fmt.Sprintf("  Severity: %s\n", severity)
	msg += fmt.Sprintf("  Protocol: %s\n", event.ProtocolID)
	msg += fmt.Sprintf("  Status: %s\n", event.Status)
	msg += fmt.Sprintf("  Metric: %s\n", event.Metric)
	msg += fmt.Sprintf("  Value: %.2f\n", event.Value)
	msg += fmt.Sprintf("  Threshold: %.2f\n", event.Threshold)
	if event.Message != "" {
		msg += fmt.Sprintf("  Message: %s\n", event.Message)
	}
	msg += fmt.Sprintf("  Time: %s\n", event.FiredAt.Format(time.RFC3339))
	return msg
}
