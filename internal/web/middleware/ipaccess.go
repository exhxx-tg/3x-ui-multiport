package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/exhxx-tg/3x-ui-multiport/internal/database"
	"github.com/exhxx-tg/3x-ui-multiport/internal/database/model"
)

type IPAccessManager struct {
	mu          sync.RWMutex
	allowCIDRs  []*net.IPNet
	blockCIDRs  []*net.IPNet
	allowOnly   bool
	initialized bool
}

var globalIPAccess = &IPAccessManager{}

func (m *IPAccessManager) refresh() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.allowCIDRs = nil
	m.blockCIDRs = nil
	m.allowOnly = false

	db := database.GetDB()
	if db == nil {
		return
	}

	var rules []model.IPAccessRule
	if err := db.Where("enabled = ?", true).Order("priority ASC").Find(&rules).Error; err != nil {
		return
	}

	hasAllowRule := false
	for _, rule := range rules {
		_, cidr, err := net.ParseCIDR(rule.CIDR)
		if err != nil {
			continue
		}
		switch rule.Type {
		case "allow":
			m.allowCIDRs = append(m.allowCIDRs, cidr)
			hasAllowRule = true
		case "block":
			m.blockCIDRs = append(m.blockCIDRs, cidr)
		}
	}

	m.allowOnly = hasAllowRule
	m.initialized = true
}

func (m *IPAccessManager) isAllowed(ip string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.initialized {
		return true
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	for _, block := range m.blockCIDRs {
		if block.Contains(parsedIP) {
			return false
		}
	}

	if !m.allowOnly {
		return true
	}

	for _, allow := range m.allowCIDRs {
		if allow.Contains(parsedIP) {
			return true
		}
	}

	return false
}

func IPAccessMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.GetHeader("X-Forwarded-For")
		if ip == "" {
			ip = c.GetHeader("X-Real-IP")
		}
		if ip == "" {
			ip = c.ClientIP()
		}

		ip = strings.TrimSpace(ip)
		if idx := strings.Index(ip, ","); idx > 0 {
			ip = strings.TrimSpace(ip[:idx])
		}

		if !globalIPAccess.isAllowed(ip) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"msg":     "access denied from your IP address",
			})
			return
		}

		c.Next()
	}
}

func RefreshIPAccess() {
	globalIPAccess.refresh()
}

func init() {
	RefreshIPAccess()
}
