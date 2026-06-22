package protocol

import "errors"

var (
	ErrProtocolNotFound = errors.New("protocol not found")
	ErrNotInstalled     = errors.New("protocol not installed")
	ErrAlreadyRunning   = errors.New("protocol already running")
	ErrNotRunning       = errors.New("protocol not running")
	ErrInvalidConfig    = errors.New("invalid protocol configuration")
	ErrPortInUse        = errors.New("port already in use")
)
