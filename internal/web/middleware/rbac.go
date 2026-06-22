package middleware

import (
	"net/http"
	"strings"

	"github.com/exhxx-tg/3x-ui-multiport/internal/logger"
	"github.com/exhxx-tg/3x-ui-multiport/internal/web/service"
	"github.com/exhxx-tg/3x-ui-multiport/internal/web/session"

	"github.com/gin-gonic/gin"
)

var rbacService service.RBACService

func RequirePermission(resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := session.GetLoginUser(c)
		if user == nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if err := rbacService.RequirePermission(user.Id, resource, action); err != nil {
			logger.Debugf("rbac: denied %s:%s for user %d (%s)", resource, action, user.Id, user.Username)
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Next()
	}
}

func RequireAnyPermission(perms ...struct{ Resource, Action string }) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := session.GetLoginUser(c)
		if user == nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		for _, p := range perms {
			if rbacService.HasPermission(user.Id, p.Resource, p.Action) {
				c.Next()
				return
			}
		}
		logger.Debugf("rbac: denied any of %d permissions for user %d (%s)", len(perms), user.Id, user.Username)
		c.AbortWithStatus(http.StatusForbidden)
	}
}

func httpMethodToAction(method string) string {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return "read"
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return "write"
	case http.MethodDelete:
		return "delete"
	default:
		return "read"
	}
}

func RequireResource(resource string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := session.GetLoginUser(c)
		if user == nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		action := httpMethodToAction(c.Request.Method)
		if err := rbacService.RequirePermission(user.Id, resource, action); err != nil {
			logger.Debugf("rbac: denied %s:%s for user %d (%s)", resource, action, user.Id, user.Username)
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Next()
	}
}

func pathToResource(path string) string {
	trimmed := strings.TrimPrefix(path, "/panel/api/")
	trimmed = strings.TrimPrefix(trimmed, "/")
	if idx := strings.Index(trimmed, "/"); idx > 0 {
		trimmed = trimmed[:idx]
	}
	switch trimmed {
	case "inbounds", "clients", "server", "nodes",
		"protocols", "services", "wrappers", "audit",
		"backup", "cert", "certificates":
		return trimmed
	case "hosts":
		return "inbounds"
	case "setting", "xray":
		return "settings"
	case "monitor", "notifiers":
		return "monitoring"
	case "rbac":
		return "roles"
	case "backuptotgbot":
		return "backup"
	default:
		return "inbounds"
	}
}

func isControlPath(path string) bool {
	return strings.HasSuffix(path, "/start") ||
		strings.HasSuffix(path, "/stop") ||
		strings.HasSuffix(path, "/restart")
}

func RBACMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := session.GetLoginUser(c)
		if user == nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		resource := pathToResource(c.Request.URL.Path)
		action := httpMethodToAction(c.Request.Method)
		if c.Request.Method == http.MethodPost && isControlPath(c.Request.URL.Path) {
			action = "control"
		}
		if err := rbacService.RequirePermission(user.Id, resource, action); err != nil {
			logger.Debugf("rbac: denied %s:%s for user %d (%s) on %s", resource, action, user.Id, user.Username, c.Request.URL.Path)
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Next()
	}
}

func RequireResourceWithControl(resource string, controlPaths ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := session.GetLoginUser(c)
		if user == nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		action := httpMethodToAction(c.Request.Method)

		for _, cp := range controlPaths {
			if strings.HasSuffix(c.Request.URL.Path, cp) && c.Request.Method == http.MethodPost {
				action = "control"
				break
			}
		}

		if err := rbacService.RequirePermission(user.Id, resource, action); err != nil {
			logger.Debugf("rbac: denied %s:%s for user %d (%s)", resource, action, user.Id, user.Username)
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Next()
	}
}
