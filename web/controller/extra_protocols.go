package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mhsanaei/3x-ui/v3/web/entity"
	"github.com/mhsanaei/3x-ui/v3/web/service"
)

type ExtraProtocolsController struct {
	BaseController
	service service.ExtraProtocolsService
}

func NewExtraProtocolsController(g *gin.RouterGroup) *ExtraProtocolsController {
	c := &ExtraProtocolsController{}
	c.initRouter(g)
	return c
}

func (c *ExtraProtocolsController) initRouter(g *gin.RouterGroup) {
	extra := g.Group("/extra")
	{
		extra.GET("/users", c.GetUsers)
		extra.POST("/users", c.AddUser)
		extra.PUT("/users/:id", c.UpdateUser)
		extra.DELETE("/users/:id", c.DeleteUser)
		extra.GET("/settings", c.GetSettings)
		extra.PUT("/settings", c.UpdateSetting)
	}
}

func (c *ExtraProtocolsController) GetUsers(ctx *gin.Context) {
	users, err := c.service.GetUsers()
	jsonObj(ctx, "get users success", users, err)
}

func (c *ExtraProtocolsController) AddUser(ctx *gin.Context) {
	var user entity.ExtraUser
	if err := ctx.ShouldBindJSON(&user); err != nil {
		jsonMsg(ctx, "bind user failed", err)
		return
	}
	if err := c.service.AddUser(&user); err != nil {
		jsonMsg(ctx, "add user failed", err)
		return
	}
	jsonMsg(ctx, "add user success", nil)
}

func (c *ExtraProtocolsController) UpdateUser(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonMsg(ctx, "invalid user id", err)
		return
	}

	var updates map[string]any
	if err := ctx.ShouldBindJSON(&updates); err != nil {
		jsonMsg(ctx, "bind updates failed", err)
		return
	}

	if err := c.service.UpdateUser(id, updates); err != nil {
		jsonMsg(ctx, "update user failed", err)
		return
	}
	jsonMsg(ctx, "update user success", nil)
}

func (c *ExtraProtocolsController) DeleteUser(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonMsg(ctx, "invalid user id", err)
		return
	}
	if err := c.service.DeleteUser(id); err != nil {
		jsonMsg(ctx, "delete user failed", err)
		return
	}
	jsonMsg(ctx, "delete user success", nil)
}

func (c *ExtraProtocolsController) GetSettings(ctx *gin.Context) {
	settings, err := c.service.GetSettings()
	jsonObj(ctx, "get settings success", settings, err)
}

func (c *ExtraProtocolsController) UpdateSetting(ctx *gin.Context) {
	var req struct {
		ProtocolName  string `json:"protocolName"`
		ListeningPort int    `json:"listeningPort"`
		IsEnabled     bool   `json:"isEnabled"`
		BannerText    string `json:"bannerText"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		jsonMsg(ctx, "bind setting failed", err)
		return
	}
	if err := c.service.UpdateSetting(req.ProtocolName, req.ListeningPort, req.IsEnabled, req.BannerText); err != nil {
		jsonMsg(ctx, "update setting failed", err)
		return
	}
	jsonMsg(ctx, "update setting success", nil)
}
