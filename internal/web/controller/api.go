package controller

import (
	"net/http"
	"strings"

	"github.com/exhxx-tg/3x-ui-multiport/internal/web/middleware"
	"github.com/exhxx-tg/3x-ui-multiport/internal/web/service"
	"github.com/exhxx-tg/3x-ui-multiport/internal/web/service/panel"
	"github.com/exhxx-tg/3x-ui-multiport/internal/web/service/tgbot"
	"github.com/exhxx-tg/3x-ui-multiport/internal/web/session"

	"github.com/gin-gonic/gin"
)

// APIController handles the main API routes for the 3x-ui panel, including inbounds and server management.
type APIController struct {
	BaseController
	inboundController     *InboundController
	serverController      *ServerController
	nodeController        *NodeController
	hostController        *HostController
	settingController     *SettingController
	xraySettingController *XraySettingController
	protocolController    *ProtocolController
	rbacController        *RBACController
	settingService        service.SettingService
	userService           panel.UserService
	apiTokenService       panel.ApiTokenService
	Tgbot                 tgbot.Tgbot
}

// NewAPIController creates a new APIController instance and initializes its routes.
func NewAPIController(g *gin.RouterGroup) *APIController {
	a := &APIController{}
	a.initRouter(g)
	return a
}

func (a *APIController) checkAPIAuth(c *gin.Context) {
	// A verified client certificate (a completed mTLS handshake) authenticates
	// the caller, equivalent to a valid bearer token. api_authed must be set so
	// the CSRF middleware lets cert-authed mutations through.
	if c.Request.TLS != nil && len(c.Request.TLS.VerifiedChains) > 0 {
		if u, err := a.userService.GetFirstUser(); err == nil {
			session.SetAPIAuthUser(c, u)
		}
		c.Set("api_authed", true)
		c.Next()
		return
	}
	auth := c.GetHeader("Authorization")
	if after, ok := strings.CutPrefix(auth, "Bearer "); ok {
		tok := after
		if a.apiTokenService.Match(tok) {
			if u, err := a.userService.GetFirstUser(); err == nil {
				session.SetAPIAuthUser(c, u)
			}
			c.Set("api_authed", true)
			c.Next()
			return
		}
	}
	if !session.IsLogin(c) {
		if c.GetHeader("X-Requested-With") == "XMLHttpRequest" {
			c.AbortWithStatus(http.StatusUnauthorized)
		} else {
			c.AbortWithStatus(http.StatusNotFound)
		}
		return
	}
	c.Next()
}

// initRouter sets up the API routes for inbounds, server, and other endpoints.
func (a *APIController) initRouter(g *gin.RouterGroup) {
	// Main API group
	api := g.Group("/panel/api")
	api.Use(a.checkAPIAuth)
	// Decode + verify the node config envelope (zstd + X-Config-Sha256) and
	// advertise support, before CSRF/handlers read the body.
	api.Use(middleware.ConfigEnvelopeMiddleware())
	api.Use(middleware.CSRFMiddleware())
	// Audit logging for all state-changing API calls
	api.Use(middleware.AuditMiddleware())
	// Auto RBAC — maps URL paths to resources and HTTP methods to actions
	api.Use(middleware.RBACMiddleware())

	// Inbounds API
	inbounds := api.Group("/inbounds")
	a.inboundController = NewInboundController(inbounds)

	clients := api.Group("/clients")
	NewClientController(clients)
	NewGroupController(clients)

	// Server API
	server := api.Group("/server")
	a.serverController = NewServerController(server)

	// Nodes API — multi-panel management
	nodes := api.Group("/nodes")
	a.nodeController = NewNodeController(nodes)

	// Hosts API — per-inbound override endpoints for subscription links
	hosts := api.Group("/hosts")
	a.hostController = NewHostController(hosts)

	// Protocol Ecosystem API — unified management of all 13 protocols, services, and wrappers
	protocols := api.Group("/protocols")
	a.protocolController = NewProtocolController(protocols)

	// Monitoring & Alerting API (controller creates its own /monitor sub-group)
	NewMonitorController(api)

	// Audit Log API
	NewAuditController(api)

	// Settings + Xray config management
	a.settingController = NewSettingController(api)
	a.xraySettingController = NewXraySettingController(api)

	// RBAC & Security
	rbacGroup := api.Group("/rbac")
	a.rbacController = NewRBACController(rbacGroup)

	// Security Dashboard — IP access control, sessions, login attempts, 2FA
	NewSecurityController(api)

	// Backup & Restore
	NewBackupController(api)

	// Certificate Management
	NewCertificateController(api)

	// Extra routes
	api.POST("/backuptotgbot", a.BackuptoTgbot)
}

// BackuptoTgbot sends a backup of the panel data to Telegram bot admins.
func (a *APIController) BackuptoTgbot(c *gin.Context) {
	a.Tgbot.SendBackupToAdmins()
}
