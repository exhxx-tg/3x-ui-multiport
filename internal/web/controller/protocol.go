package controller

import (
	"net/http"
	"strconv"

	"github.com/exhxx-tg/3x-ui-multiport/internal/database/model"
	xuiProtocol "github.com/exhxx-tg/3x-ui-multiport/internal/protocol"
	"github.com/exhxx-tg/3x-ui-multiport/internal/web/service"

	"github.com/gin-gonic/gin"
)

type ProtocolController struct {
	protocolService *service.ProtocolService
}

func NewProtocolController(g *gin.RouterGroup) *ProtocolController {
	a := &ProtocolController{
		protocolService: service.NewProtocolService(),
	}
	a.initRouter(g)
	return a
}

func (a *ProtocolController) initRouter(g *gin.RouterGroup) {
	g.GET("/protocols", a.listProtocols)
	g.GET("/protocols/:id", a.getProtocol)
	g.GET("/protocols/:id/status", a.getProtocolStatus)
	g.POST("/protocols/:id/start", a.startProtocol)
	g.POST("/protocols/:id/stop", a.stopProtocol)
	g.POST("/protocols/:id/restart", a.restartProtocol)
	g.GET("/protocols/:id/wrappers", a.getSupportedWrappers)
	g.GET("/protocols/:id/health", a.getProtocolHealth)
	g.GET("/protocols/detailed", a.listProtocolsDetailed)

	g.GET("/services", a.listServices)
	g.GET("/services/:id", a.getService)
	g.POST("/services", a.createService)
	g.PUT("/services/:id", a.updateService)
	g.DELETE("/services/:id", a.deleteService)

	g.GET("/wrappers", a.listWrappers)
	g.POST("/wrappers", a.createWrapper)
	g.PUT("/wrappers/:id", a.updateWrapper)
	g.DELETE("/wrappers/:id", a.deleteWrapper)

	g.GET("/protocol-configs/:protocol", a.getProtocolConfig)
	g.PUT("/protocol-configs/:protocol", a.saveProtocolConfig)
}

func (a *ProtocolController) listProtocols(c *gin.Context) {
	protocols := a.protocolService.ListProtocols()
	jsonObj(c, protocols, nil)
}

func (a *ProtocolController) getProtocol(c *gin.Context) {
	id := xuiProtocol.ProtocolID(c.Param("id"))
	info := a.protocolService.GetProtocol(id)
	if info == nil {
		jsonMsg(c, "Protocol not found", nil)
		return
	}
	jsonObj(c, info, nil)
}

func (a *ProtocolController) getProtocolStatus(c *gin.Context) {
	id := xuiProtocol.ProtocolID(c.Param("id"))
	status, err := a.protocolService.GetStatus(id)
	if err != nil {
		jsonMsg(c, "Failed to get status", err)
		return
	}
	jsonObj(c, gin.H{"id": id, "status": status}, nil)
}

func (a *ProtocolController) startProtocol(c *gin.Context) {
	id := xuiProtocol.ProtocolID(c.Param("id"))
	if err := a.protocolService.Start(id); err != nil {
		jsonMsg(c, "Failed to start protocol", err)
		return
	}
	jsonObj(c, gin.H{"id": id, "status": "running"}, nil)
}

func (a *ProtocolController) stopProtocol(c *gin.Context) {
	id := xuiProtocol.ProtocolID(c.Param("id"))
	if err := a.protocolService.Stop(id); err != nil {
		jsonMsg(c, "Failed to stop protocol", err)
		return
	}
	jsonObj(c, gin.H{"id": id, "status": "stopped"}, nil)
}

func (a *ProtocolController) restartProtocol(c *gin.Context) {
	id := xuiProtocol.ProtocolID(c.Param("id"))
	if err := a.protocolService.Restart(id); err != nil {
		jsonMsg(c, "Failed to restart protocol", err)
		return
	}
	jsonObj(c, gin.H{"id": id, "status": "running"}, nil)
}

func (a *ProtocolController) getSupportedWrappers(c *gin.Context) {
	id := xuiProtocol.ProtocolID(c.Param("id"))
	wrappers := a.protocolService.GetSupportedWrappers(id)
	jsonObj(c, wrappers, nil)
}

func (a *ProtocolController) getProtocolHealth(c *gin.Context) {
	id := xuiProtocol.ProtocolID(c.Param("id"))
	health := a.protocolService.HealthCheck(id)
	jsonObj(c, gin.H{"id": id, "healthy": health == nil, "error": func() string {
		if health != nil { return health.Error() }
		return ""
	}()}, nil)
}

func (a *ProtocolController) listProtocolsDetailed(c *gin.Context) {
	protocols := a.protocolService.ListProtocols()
	type ProtocolDetailed struct {
		xuiProtocol.ProtocolInfo
		Status  xuiProtocol.Status `json:"status"`
		Healthy bool                `json:"healthy"`
	}
	var detailed []ProtocolDetailed
	for _, p := range protocols {
		status, _ := a.protocolService.GetStatus(p.ID)
		health := a.protocolService.HealthCheck(p.ID)
		detailed = append(detailed, ProtocolDetailed{
			ProtocolInfo: p,
			Status:       status,
			Healthy:      health == nil,
		})
	}
	jsonObj(c, detailed, nil)
}

func (a *ProtocolController) listServices(c *gin.Context) {
	services, err := a.protocolService.ListServices()
	if err != nil {
		jsonMsg(c, "Failed to list services", err)
		return
	}
	jsonObj(c, services, nil)
}

func (a *ProtocolController) getService(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	svc, err := a.protocolService.GetService(id)
	if err != nil {
		jsonMsg(c, "Service not found", err)
		return
	}
	jsonObj(c, svc, nil)
}

func (a *ProtocolController) createService(c *gin.Context) {
	var svc model.Service
	if err := c.ShouldBindJSON(&svc); err != nil {
		jsonMsg(c, "Invalid request body", err)
		return
	}
	if err := a.protocolService.CreateService(&svc); err != nil {
		jsonMsg(c, "Failed to create service", err)
		return
	}
	jsonObj(c, svc, nil)
}

func (a *ProtocolController) updateService(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	var svc model.Service
	if err := c.ShouldBindJSON(&svc); err != nil {
		jsonMsg(c, "Invalid request body", err)
		return
	}
	svc.Id = id
	if err := a.protocolService.UpdateService(&svc); err != nil {
		jsonMsg(c, "Failed to update service", err)
		return
	}
	jsonObj(c, svc, nil)
}

func (a *ProtocolController) deleteService(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	if err := a.protocolService.DeleteService(id); err != nil {
		jsonMsg(c, "Failed to delete service", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (a *ProtocolController) listWrappers(c *gin.Context) {
	wrappers, err := a.protocolService.ListWrappers()
	if err != nil {
		jsonMsg(c, "Failed to list wrappers", err)
		return
	}
	jsonObj(c, wrappers, nil)
}

func (a *ProtocolController) createWrapper(c *gin.Context) {
	var w model.TransportWrapper
	if err := c.ShouldBindJSON(&w); err != nil {
		jsonMsg(c, "Invalid request body", err)
		return
	}
	if err := a.protocolService.CreateWrapper(&w); err != nil {
		jsonMsg(c, "Failed to create wrapper", err)
		return
	}
	jsonObj(c, w, nil)
}

func (a *ProtocolController) updateWrapper(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	var w model.TransportWrapper
	if err := c.ShouldBindJSON(&w); err != nil {
		jsonMsg(c, "Invalid request body", err)
		return
	}
	w.Id = id
	if err := a.protocolService.UpdateWrapper(&w); err != nil {
		jsonMsg(c, "Failed to update wrapper", err)
		return
	}
	jsonObj(c, w, nil)
}

func (a *ProtocolController) deleteWrapper(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	if err := a.protocolService.DeleteWrapper(id); err != nil {
		jsonMsg(c, "Failed to delete wrapper", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (a *ProtocolController) getProtocolConfig(c *gin.Context) {
	protoName := c.Param("protocol")
	cfg, err := a.protocolService.GetProtocolConfig(protoName)
	if err != nil {
		jsonMsg(c, "Config not found", err)
		return
	}
	jsonObj(c, cfg, nil)
}

func (a *ProtocolController) saveProtocolConfig(c *gin.Context) {
	protoName := c.Param("protocol")
	var cfg any
	if err := c.ShouldBindJSON(&cfg); err != nil {
		jsonMsg(c, "Invalid config", err)
		return
	}
	if err := a.protocolService.SaveProtocolConfig(protoName, cfg); err != nil {
		jsonMsg(c, "Failed to save config", err)
		return
	}
	jsonObj(c, gin.H{"protocol": protoName, "status": "saved"}, nil)
}
