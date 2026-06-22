package monitor

import "errors"

var (
	ErrRuleNotFound     = errors.New("alert rule not found")
	ErrNotifierNotFound = errors.New("notifier not found")
	ErrCheckFailed      = errors.New("health check failed")
)
