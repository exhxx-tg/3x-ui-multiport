package middleware

import (
	"bytes"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/exhxx-tg/3x-ui-multiport/internal/logger"
	"github.com/exhxx-tg/3x-ui-multiport/internal/web/session"
	"github.com/exhxx-tg/3x-ui-multiport/internal/web/service"
)

// AuditMiddleware logs state-changing API requests to the audit log.
// It should be mounted on the API group that requires authentication.
func AuditMiddleware() gin.HandlerFunc {
	var auditSvc service.AuditService

	return func(c *gin.Context) {
		// Only log mutating methods
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		// Read the request body (store for details)
		var bodyBytes []byte
		if c.Request.Body != nil {
			bodyBytes, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		// Process the request
		start := time.Now()
		c.Next()

		// Determine status based on response code
		status := "success"
		if c.Writer.Status() >= 400 {
			status = "failure"
		}

		// Get user info
		user := session.GetLoginUser(c)
		userID := 0
		username := "unknown"
		if user != nil {
			userID = user.Id
			username = user.Username
		}

		// Truncate body for storage
		detail := string(bodyBytes)
		if len(detail) > 500 {
			detail = detail[:500] + "..."
		}

		// Get client IP
		ip := c.GetHeader("X-Forwarded-For")
		if ip == "" {
			ip = c.GetHeader("X-Real-IP")
		}
		if ip == "" {
			ip = c.ClientIP()
		}

		// Extract resource from path
		path := c.Request.URL.Path
		resource := "api"
		if len(path) > 11 { // strip /panel/api/
			resource = path[11:]
		}

		auditSvc.Record(
			userID,
			username,
			c.Request.Method,
			resource,
			"",
			detail,
			ip,
			c.Request.UserAgent(),
			status,
		)

		logger.Debugf("audit: %s %s by %s (%dms)", c.Request.Method, path, username, time.Since(start).Milliseconds())
	}
}
