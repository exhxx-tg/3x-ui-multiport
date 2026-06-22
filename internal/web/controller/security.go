package controller

import (
	"crypto/rand"
	"math/big"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/exhxx-tg/3x-ui-multiport/internal/database"
	"github.com/exhxx-tg/3x-ui-multiport/internal/database/model"
	"github.com/exhxx-tg/3x-ui-multiport/internal/util/crypto"
	"github.com/exhxx-tg/3x-ui-multiport/internal/web/service"
)

type SecurityController struct {
	securityService service.SecurityService
	settingService  service.SettingService
}

func NewSecurityController(g *gin.RouterGroup) *SecurityController {
	ctrl := &SecurityController{
		securityService: service.SecurityService{},
		settingService:  service.SettingService{},
	}
	ctrl.initRouter(g)
	return ctrl
}

func (ctrl *SecurityController) initRouter(g *gin.RouterGroup) {
	s := g.Group("/security")
	{
		s.GET("/overview", ctrl.overview)
		s.GET("/login-attempts", ctrl.listLoginAttempts)
		s.GET("/events", ctrl.listEvents)

		s.GET("/ip-access", ctrl.listIPAccessRules)
		s.POST("/ip-access", ctrl.createIPAccessRule)
		s.POST("/ip-access/:id/update", ctrl.updateIPAccessRule)
		s.POST("/ip-access/:id/delete", ctrl.deleteIPAccessRule)

		s.GET("/sessions", ctrl.listSessions)
		s.POST("/sessions/revoke/:id", ctrl.revokeSession)
		s.POST("/sessions/revoke-all", ctrl.revokeAllSessions)

		s.POST("/2fa/generate-backup-codes", ctrl.generateBackupCodes)
		s.POST("/2fa/verify-backup-code", ctrl.verifyBackupCode)
	}
}

func (ctrl *SecurityController) overview(c *gin.Context) {
	overview, err := ctrl.securityService.GetOverview()
	if err != nil {
		jsonMsg(c, "Failed to get security overview", err)
		return
	}
	jsonObj(c, overview, nil)
}

func (ctrl *SecurityController) listLoginAttempts(c *gin.Context) {
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	rows, total, err := ctrl.securityService.ListLoginAttempts(offset, limit)
	if err != nil {
		jsonMsg(c, "Failed to list login attempts", err)
		return
	}
	jsonObj(c, gin.H{"items": rows, "total": total}, nil)
}

func (ctrl *SecurityController) listEvents(c *gin.Context) {
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	rows, total, err := ctrl.securityService.ListSecurityEvents(offset, limit)
	if err != nil {
		jsonMsg(c, "Failed to list security events", err)
		return
	}
	jsonObj(c, gin.H{"items": rows, "total": total}, nil)
}

func (ctrl *SecurityController) listIPAccessRules(c *gin.Context) {
	rules, err := ctrl.securityService.ListIPAccessRules()
	if err != nil {
		jsonMsg(c, "Failed to list IP access rules", err)
		return
	}
	jsonObj(c, rules, nil)
}

func (ctrl *SecurityController) createIPAccessRule(c *gin.Context) {
	var req struct {
		Type     string `json:"type" binding:"required"`
		CIDR     string `json:"cidr" binding:"required"`
		Remark   string `json:"remark"`
		Priority int    `json:"priority"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		jsonMsg(c, "Invalid request", err)
		return
	}

	rule, err := ctrl.securityService.AddIPAccessRule(req.Type, req.CIDR, req.Remark, req.Priority)
	if err != nil {
		jsonMsg(c, "Failed to create rule", err)
		return
	}
	jsonObj(c, rule, nil)
}

func (ctrl *SecurityController) updateIPAccessRule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "Invalid rule id", err)
		return
	}

	var req struct {
		Type     string `json:"type"`
		CIDR     string `json:"cidr"`
		Remark   string `json:"remark"`
		Enabled  *bool  `json:"enabled"`
		Priority int    `json:"priority"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		jsonMsg(c, "Invalid request", err)
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	if err := ctrl.securityService.UpdateIPAccessRule(id, req.Type, req.CIDR, req.Remark, enabled, req.Priority); err != nil {
		jsonMsg(c, "Failed to update rule", err)
		return
	}
	jsonMsg(c, "Rule updated", nil)
}

func (ctrl *SecurityController) deleteIPAccessRule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "Invalid rule id", err)
		return
	}

	if err := ctrl.securityService.DeleteIPAccessRule(id); err != nil {
		jsonMsg(c, "Failed to delete rule", err)
		return
	}
	jsonMsg(c, "Rule deleted", nil)
}

func (ctrl *SecurityController) listSessions(c *gin.Context) {
	sessionSvc := service.SessionService{}
	sessions, err := sessionSvc.GetActiveSessions()
	if err != nil {
		jsonMsg(c, "Failed to list sessions", err)
		return
	}
	jsonObj(c, sessions, nil)
}

func (ctrl *SecurityController) revokeSession(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "Invalid session id", err)
		return
	}

	sessionSvc := service.SessionService{}
	if err := sessionSvc.RevokeSession(id); err != nil {
		jsonMsg(c, "Failed to revoke session", err)
		return
	}
	jsonMsg(c, "Session revoked", nil)
}

func (ctrl *SecurityController) revokeAllSessions(c *gin.Context) {
	userID, _ := getCurrentUser(c)

	sessionSvc := service.SessionService{}
	if err := sessionSvc.RevokeAllSessions(userID); err != nil {
		jsonMsg(c, "Failed to revoke sessions", err)
		return
	}
	jsonMsg(c, "All sessions revoked", nil)
}

func (ctrl *SecurityController) generateBackupCodes(c *gin.Context) {
	userID, _ := getCurrentUser(c)
	db := database.GetDB()
	if db == nil {
		jsonMsg(c, "Database not available", nil)
		return
	}

	plainCodes := make([]string, 10)
	codeModels := make([]model.BackupCode, 10)
	for i := range plainCodes {
		code := randomBase32String(16)
		plainCodes[i] = code
		hash, err := crypto.HashPasswordAsBcrypt(code)
		if err != nil {
			codeModels[i] = model.BackupCode{CodeHash: code, UserId: userID}
			continue
		}
		codeModels[i] = model.BackupCode{CodeHash: hash, UserId: userID}
	}

	if err := db.Create(&codeModels).Error; err != nil {
		jsonMsg(c, "Failed to store backup codes", err)
		return
	}

	jsonObj(c, gin.H{"codes": plainCodes, "count": len(plainCodes)}, nil)
}

func (ctrl *SecurityController) verifyBackupCode(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		jsonMsg(c, "Invalid request", err)
		return
	}

	userID, _ := getCurrentUser(c)
	db := database.GetDB()
	if db == nil {
		jsonMsg(c, "Database not available", nil)
		return
	}

	var codes []model.BackupCode
	db.Where("user_id = ? AND consumed = ?", userID, false).Find(&codes)

	for _, bc := range codes {
		if crypto.CheckPasswordHash(bc.CodeHash, req.Code) {
			db.Model(&bc).Update("consumed", true)
			jsonObj(c, gin.H{"valid": true, "consumed": req.Code}, nil)
			return
		}
	}

	jsonObj(c, gin.H{"valid": false, "consumed": ""}, nil)
}

func randomBase32String(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			b[i] = 'A'
			continue
		}
		b[i] = charset[n.Int64()]
	}
	return string(b)
}
