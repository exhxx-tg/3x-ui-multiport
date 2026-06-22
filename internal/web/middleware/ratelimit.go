package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitorInfo
	maxReqs  int
	interval time.Duration
}

type visitorInfo struct {
	count    int
	lastSeen time.Time
	resetAt  time.Time
}

var globalRateLimiter = newRateLimiter(100, time.Minute)

var rateLimitExemptPaths = []string{
	"/healthz",
	"/readyz",
	"/metrics",
}

func newRateLimiter(maxReqs int, interval time.Duration) *rateLimiter {
	rl := &rateLimiter{
		visitors: make(map[string]*visitorInfo),
		maxReqs:  maxReqs,
		interval: interval,
	}
	go rl.cleanup()
	return rl
}

func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, info := range rl.visitors {
			if now.After(info.resetAt) {
				delete(rl.visitors, key)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	info, exists := rl.visitors[ip]

	if !exists {
		rl.visitors[ip] = &visitorInfo{
			count:    1,
			lastSeen: now,
			resetAt:  now.Add(rl.interval),
		}
		return true
	}

	if now.After(info.resetAt) {
		info.count = 1
		info.lastSeen = now
		info.resetAt = now.Add(rl.interval)
		return true
	}

	info.count++
	info.lastSeen = now

	return info.count <= rl.maxReqs
}

func isExemptPath(path string) bool {
	for _, exempt := range rateLimitExemptPaths {
		if path == exempt {
			return true
		}
	}
	return false
}

func RateLimitMiddleware(maxRequests int, duration time.Duration) gin.HandlerFunc {
	rl := newRateLimiter(maxRequests, duration)

	return func(c *gin.Context) {
		if isExemptPath(c.Request.URL.Path) {
			c.Next()
			return
		}

		ip := c.GetHeader("X-Forwarded-For")
		if ip == "" {
			ip = c.GetHeader("X-Real-IP")
		}
		if ip == "" {
			ip = c.ClientIP()
		}

		if !rl.allow(ip) {
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"msg":     "rate limit exceeded. try again later.",
			})
			return
		}

		c.Next()
	}
}

func ConfigurableRateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isExemptPath(c.Request.URL.Path) {
			c.Next()
			return
		}

		ip := c.GetHeader("X-Forwarded-For")
		if ip == "" {
			ip = c.GetHeader("X-Real-IP")
		}
		if ip == "" {
			ip = c.ClientIP()
		}

		if !globalRateLimiter.allow(ip) {
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"msg":     "rate limit exceeded. try again later.",
			})
			return
		}

		c.Next()
	}
}

func UpdateRateLimit(maxReqs int, interval time.Duration) {
	globalRateLimiter.mu.Lock()
	defer globalRateLimiter.mu.Unlock()
	globalRateLimiter.maxReqs = maxReqs
	globalRateLimiter.interval = interval
}
