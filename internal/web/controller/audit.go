package controller

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/exhxx-tg/3x-ui-multiport/internal/database"
	"github.com/exhxx-tg/3x-ui-multiport/internal/web/service"
	"github.com/exhxx-tg/3x-ui-multiport/internal/web/session"
)

type AuditController struct {
	BaseController
	auditService service.AuditService
}

func NewAuditController(g *gin.RouterGroup) *AuditController {
	ctrl := &AuditController{
		auditService: service.AuditService{},
	}
	ctrl.initRouter(g)
	return ctrl
}

func (ctrl *AuditController) initRouter(g *gin.RouterGroup) {
	g.GET("/audit/logs", ctrl.listLogs)
	g.POST("/audit/logs/clear", ctrl.clearLogs)
}

func (ctrl *AuditController) listLogs(c *gin.Context) {
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	action := c.Query("action")
	resource := c.Query("resource")
	username := c.Query("username")
	status := c.Query("status")

	rows, total, err := ctrl.auditService.List(offset, limit, action, resource, username, status)
	if err != nil {
		jsonMsg(c, "Failed to list audit logs", err)
		return
	}
	jsonObj(c, gin.H{"items": rows, "total": total}, nil)
}

func (ctrl *AuditController) clearLogs(c *gin.Context) {
	db := database.GetDB()
	if db == nil {
		jsonMsg(c, "Database not available", fmt.Errorf("no db"))
		return
	}
	if err := db.Exec("DELETE FROM audit_logs").Error; err != nil {
		jsonMsg(c, "Failed to clear audit logs", err)
		return
	}
	jsonMsg(c, "Audit logs cleared", nil)
}

func getCurrentUser(c *gin.Context) (int, string) {
	user := session.GetLoginUser(c)
	if user == nil {
		return 0, "unknown"
	}
	return user.Id, user.Username
}

func getClientIP(c *gin.Context) string {
	ip := c.GetHeader("X-Forwarded-For")
	if ip == "" {
		ip = c.GetHeader("X-Real-IP")
	}
	if ip == "" {
		ip = c.ClientIP()
	}
	return ip
}
