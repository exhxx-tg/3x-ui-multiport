package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/exhxx-tg/3x-ui-multiport/internal/web/service"
)

type BackupController struct {
	backupService service.BackupService
}

func NewBackupController(g *gin.RouterGroup) *BackupController {
	ctrl := &BackupController{
		backupService: service.BackupService{},
	}
	ctrl.initRouter(g)
	return ctrl
}

func (ctrl *BackupController) initRouter(g *gin.RouterGroup) {
	b := g.Group("/backup")
	{
		b.GET("/list", ctrl.list)
		b.POST("/create", ctrl.create)
		b.POST("/restore/:id", ctrl.restore)
		b.POST("/delete/:id", ctrl.delete)
		b.GET("/export/audit", ctrl.exportAudit)
	}
}

func (ctrl *BackupController) list(c *gin.Context) {
	backups, err := ctrl.backupService.ListBackups()
	if err != nil {
		jsonMsg(c, "Failed to list backups", err)
		return
	}
	jsonObj(c, backups, nil)
}

func (ctrl *BackupController) create(c *gin.Context) {
	var req struct {
		Description string `json:"description"`
		Encrypt     bool   `json:"encrypt"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		req.Description = ""
		req.Encrypt = false
	}

	userID, _ := getCurrentUser(c)

	backup, err := ctrl.backupService.CreateBackup(req.Description, userID, req.Encrypt)
	if err != nil {
		jsonMsg(c, "Failed to create backup", err)
		return
	}
	jsonObj(c, backup, nil)
}

func (ctrl *BackupController) restore(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "Invalid backup id", err)
		return
	}

	if err := ctrl.backupService.RestoreBackup(id); err != nil {
		jsonMsg(c, "Failed to restore backup", err)
		return
	}
	jsonMsg(c, "Backup restored successfully. Panel will restart.", nil)
}

func (ctrl *BackupController) delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "Invalid backup id", err)
		return
	}

	if err := ctrl.backupService.DeleteBackup(id); err != nil {
		jsonMsg(c, "Failed to delete backup", err)
		return
	}
	jsonMsg(c, "Backup deleted", nil)
}

func (ctrl *BackupController) exportAudit(c *gin.Context) {
	format := c.DefaultQuery("format", "csv")

	filename, data, err := ctrl.backupService.ExportAuditLogs(format)
	if err != nil {
		jsonMsg(c, "Failed to export audit logs", err)
		return
	}

	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "application/octet-stream")
	c.Data(200, "application/octet-stream", data)
}
