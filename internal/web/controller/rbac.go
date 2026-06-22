package controller

import (
	"strconv"

	"github.com/exhxx-tg/3x-ui-multiport/internal/web/middleware"
	"github.com/exhxx-tg/3x-ui-multiport/internal/web/service"

	"github.com/gin-gonic/gin"
)

type RBACController struct {
	rbacService service.RBACService
}

func NewRBACController(g *gin.RouterGroup) *RBACController {
	a := &RBACController{}
	a.initRouter(g)
	return a
}

func (a *RBACController) initRouter(g *gin.RouterGroup) {
	g.GET("/roles", a.listRoles)
	g.GET("/roles/:id/permissions", a.getRolePermissions)
	g.PUT("/roles/:id/permissions", a.setRolePermissions)
	g.GET("/permissions", a.listPermissions)
	g.GET("/users/:userId/role", a.getUserRole)
	g.PUT("/users/:userId/role", a.setUserRole)
}

func (a *RBACController) listRoles(c *gin.Context) {
	roles := a.rbacService.ListRoles()
	jsonObj(c, roles, nil)
}

func (a *RBACController) getRolePermissions(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "invalid role id", err)
		return
	}
	perms, err := a.rbacService.GetRolePermissions(id)
	jsonObj(c, perms, err)
}

func (a *RBACController) setRolePermissions(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "invalid role id", err)
		return
	}
	var form struct {
		PermissionIDs []int `json:"permissionIds"`
	}
	if err := c.ShouldBind(&form); err != nil {
		jsonMsg(c, "invalid request", err)
		return
	}
	err = a.rbacService.SetRolePermissions(id, form.PermissionIDs)
	jsonMsg(c, "permissions updated", err)
}

func (a *RBACController) listPermissions(c *gin.Context) {
	perms, err := a.rbacService.ListPermissions()
	jsonObj(c, perms, err)
}

func (a *RBACController) getUserRole(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("userId"))
	if err != nil {
		jsonMsg(c, "invalid user id", err)
		return
	}
	ur, err := a.rbacService.GetUserRole(userID)
	if err != nil {
		jsonMsg(c, "user has no role assigned — defaulting to admin", nil)
		jsonObj(c, gin.H{"userId": userID, "roleId": 1, "roleName": "admin"}, nil)
		return
	}
	roleName := "unknown"
	switch ur.RoleId {
	case 1:
		roleName = "admin"
	case 2:
		roleName = "operator"
	case 3:
		roleName = "viewer"
	case 4:
		roleName = "service"
	}
	jsonObj(c, gin.H{"userId": ur.UserId, "roleId": ur.RoleId, "roleName": roleName, "createdAt": ur.CreatedAt}, nil)
}

func (a *RBACController) setUserRole(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("userId"))
	if err != nil {
		jsonMsg(c, "invalid user id", err)
		return
	}
	var form struct {
		RoleId int `json:"roleId"`
	}
	if err := c.ShouldBind(&form); err != nil {
		jsonMsg(c, "invalid request", err)
		return
	}
	err = a.rbacService.SetUserRole(userID, form.RoleId)
	jsonMsg(c, "user role updated", err)
}

func init() {
	// Ensure rbac endpoints are registered for permission middleware
	_ = middleware.RequirePermission
}
