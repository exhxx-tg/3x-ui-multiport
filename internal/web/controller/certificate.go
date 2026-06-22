package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/exhxx-tg/3x-ui-multiport/internal/web/service"
)

type CertificateController struct {
	certService service.CertificateService
}

func NewCertificateController(g *gin.RouterGroup) *CertificateController {
	ctrl := &CertificateController{
		certService: service.CertificateService{},
	}
	ctrl.initRouter(g)
	return ctrl
}

func (ctrl *CertificateController) initRouter(g *gin.RouterGroup) {
	c := g.Group("/certificates")
	{
		c.GET("/list", ctrl.list)
		c.POST("/create", ctrl.create)
		c.POST("/generate-selfsigned", ctrl.generateSelfSigned)
		c.POST("/set-active/:id", ctrl.setActive)
		c.POST("/delete/:id", ctrl.delete)
		c.GET("/check-expiry", ctrl.checkExpiry)
	}
}

func (ctrl *CertificateController) list(c *gin.Context) {
	certs, err := ctrl.certService.List()
	if err != nil {
		jsonMsg(c, "Failed to list certificates", err)
		return
	}
	jsonObj(c, certs, nil)
}

func (ctrl *CertificateController) create(c *gin.Context) {
	var req struct {
		Domain    string `json:"domain" binding:"required"`
		CertPEM   string `json:"certPem" binding:"required"`
		KeyPEM    string `json:"keyPem" binding:"required"`
		AutoRenew bool   `json:"autoRenew"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		jsonMsg(c, "Invalid request", err)
		return
	}

	cert, err := ctrl.certService.Create(req.Domain, req.CertPEM, req.KeyPEM, req.AutoRenew)
	if err != nil {
		jsonMsg(c, "Failed to create certificate", err)
		return
	}
	jsonObj(c, cert, nil)
}

func (ctrl *CertificateController) generateSelfSigned(c *gin.Context) {
	var req struct {
		Domain string `json:"domain" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		jsonMsg(c, "Invalid request", err)
		return
	}

	cert, err := ctrl.certService.GenerateSelfSigned(req.Domain)
	if err != nil {
		jsonMsg(c, "Failed to generate self-signed certificate", err)
		return
	}
	jsonObj(c, cert, nil)
}

func (ctrl *CertificateController) setActive(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "Invalid certificate id", err)
		return
	}

	cert, err := ctrl.certService.Get(id)
	if err != nil {
		jsonMsg(c, "Certificate not found", err)
		return
	}

	if err := ctrl.certService.SetActive(cert.Domain); err != nil {
		jsonMsg(c, "Failed to set active certificate", err)
		return
	}
	jsonMsg(c, "Certificate set as active", nil)
}

func (ctrl *CertificateController) delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "Invalid certificate id", err)
		return
	}

	if err := ctrl.certService.Delete(id); err != nil {
		jsonMsg(c, "Failed to delete certificate", err)
		return
	}
	jsonMsg(c, "Certificate deleted", nil)
}

func (ctrl *CertificateController) checkExpiry(c *gin.Context) {
	expiring, err := ctrl.certService.CheckExpiry()
	if err != nil {
		jsonMsg(c, "Failed to check certificate expiry", err)
		return
	}
	jsonObj(c, expiring, nil)
}
