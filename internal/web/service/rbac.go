package service

import (
	"fmt"

	"github.com/exhxx-tg/3x-ui-multiport/internal/database"
	"github.com/exhxx-tg/3x-ui-multiport/internal/database/model"
	"gorm.io/gorm"
)

type RBACService struct{}

func (s *RBACService) HasPermission(userID int, resource, action string) bool {
	db := database.GetDB()
	if db == nil {
		return false
	}
	var userRole model.UserRole
	if err := db.Where("user_id = ?", userID).First(&userRole).Error; err != nil {
		return false
	}
	var count int64
	db.Model(&model.RolePermission{}).
		Joins("JOIN permissions ON permissions.id = role_permissions.permission_id").
		Where("role_permissions.role_id = ? AND permissions.resource = ? AND permissions.action = ?",
			userRole.RoleId, resource, action).
		Count(&count)
	return count > 0
}

func (s *RBACService) RequirePermission(userID int, resource, action string) error {
	if s.HasPermission(userID, resource, action) {
		return nil
	}
	return fmt.Errorf("permission denied: %s:%s", resource, action)
}

func (s *RBACService) GetUserRole(userID int) (*model.UserRole, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not available")
	}
	var ur model.UserRole
	if err := db.Where("user_id = ?", userID).First(&ur).Error; err != nil {
		return nil, err
	}
	return &ur, nil
}

func (s *RBACService) SetUserRole(userID, roleID int) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not available")
	}
	var existing model.UserRole
	err := db.Where("user_id = ?", userID).First(&existing).Error
	if err == nil {
		return db.Model(&existing).Update("role_id", roleID).Error
	}
	if err == gorm.ErrRecordNotFound {
		return db.Create(&model.UserRole{
			UserId: userID,
			RoleId: roleID,
		}).Error
	}
	return err
}

type RoleWithPerms struct {
	Id          int              `json:"id"`
	Name        string           `json:"name"`
	Permissions []PermissionInfo `json:"permissions" gorm:"-"`
}

type PermissionInfo struct {
	Id       int    `json:"id"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

var roleNames = map[int]string{
	1: "admin",
	2: "operator",
	3: "viewer",
	4: "service",
}

func (s *RBACService) ListRoles() []RoleWithPerms {
	db := database.GetDB()
	if db == nil {
		return nil
	}
	var roles []RoleWithPerms
	for id := 1; id <= 4; id++ {
		var perms []PermissionInfo
		db.Raw(`
			SELECT p.id, p.resource, p.action
			FROM role_permissions rp
			JOIN permissions p ON p.id = rp.permission_id
			WHERE rp.role_id = ?
			ORDER BY p.resource, p.action`, id).Scan(&perms)
		roles = append(roles, RoleWithPerms{
			Id:          id,
			Name:        roleNames[id],
			Permissions: perms,
		})
	}
	return roles
}

func (s *RBACService) ListPermissions() ([]PermissionInfo, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not available")
	}
	var perms []PermissionInfo
	if err := db.Model(&model.Permission{}).Order("resource, action").Find(&perms).Error; err != nil {
		return nil, err
	}
	return perms, nil
}

func (s *RBACService) GetRolePermissions(roleID int) ([]PermissionInfo, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not available")
	}
	var perms []PermissionInfo
	if err := db.Raw(`
		SELECT p.id, p.resource, p.action
		FROM role_permissions rp
		JOIN permissions p ON p.id = rp.permission_id
		WHERE rp.role_id = ?
		ORDER BY p.resource, p.action`, roleID).Scan(&perms).Error; err != nil {
		return nil, err
	}
	return perms, nil
}

func (s *RBACService) SetRolePermissions(roleID int, permissionIDs []int) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not available")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", roleID).Delete(&model.RolePermission{}).Error; err != nil {
			return err
		}
		for _, permID := range permissionIDs {
			if err := tx.Create(&model.RolePermission{
				RoleId:       roleID,
				PermissionId: permID,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}


