package performance

import "errors"

var (
	ErrPoolQueueFull = errors.New("worker pool queue is full")
	ErrCacheMiss     = errors.New("cache miss")
)
