package service

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mhsanaei/3x-ui/v3/database"
	"github.com/mhsanaei/3x-ui/v3/logger"
	"github.com/mhsanaei/3x-ui/v3/util/common"
	"github.com/mhsanaei/3x-ui/v3/web/entity"
)

type ExtraProtocolsService struct {
	sysManager SshSswsManager
}

// MigrateDB creates the extra_users and extra_settings tables and populates default settings.
func (s *ExtraProtocolsService) MigrateDB() error {
	db := database.GetDB()

	if err := db.AutoMigrate(&entity.ExtraUser{}, &entity.ExtraSetting{}); err != nil {
		return fmt.Errorf("failed to migrate extra protocols tables: %w", err)
	}

	defaults := []entity.ExtraSetting{
		{ProtocolName: "SSH", ListeningPort: 2222, IsEnabled: false},
		{ProtocolName: "SSWS", ListeningPort: 8443, IsEnabled: false},
		{ProtocolName: "SLOW-DNS", ListeningPort: 5353, IsEnabled: false},
		{ProtocolName: "Psiphon", ListeningPort: 443, IsEnabled: false},
		{ProtocolName: "UDP Custom (BadVPN)", ListeningPort: 7300, IsEnabled: false},
		{ProtocolName: "Dropbear", ListeningPort: 2223, IsEnabled: false},
		{ProtocolName: "SSL (Stunnel)", ListeningPort: 444, IsEnabled: false},
		{ProtocolName: "OpenVPN", ListeningPort: 1194, IsEnabled: false},
		{ProtocolName: "Squid", ListeningPort: 3128, IsEnabled: false},
		{ProtocolName: "OHP", ListeningPort: 80, IsEnabled: false},
	}

	for _, setting := range defaults {
		if err := db.FirstOrCreate(&setting, entity.ExtraSetting{ProtocolName: setting.ProtocolName}).Error; err != nil {
			return fmt.Errorf("failed to insert default setting for %s: %w", setting.ProtocolName, err)
		}
	}

	return nil
}

// --- User Management ---

func (s *ExtraProtocolsService) GetUsers() ([]entity.ExtraUser, error) {
	var users []entity.ExtraUser
	if err := database.GetDB().Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (s *ExtraProtocolsService) AddUser(user *entity.ExtraUser) error {
	if user.Username == "" || user.Password == "" || user.ProtocolType == "" {
		return common.NewError("username, password and protocol type are required")
	}
	
	if user.ProtocolType == "SSH" {
		if err := s.sysManager.CreateOSUser(user.Username, user.Password); err != nil {
			return fmt.Errorf("OS user creation failed: %w", err)
		}
	} else if user.ProtocolType == "SSWS" {
		if err := s.sysManager.ManageSSWSConfig(user.Username, user.Password, "add"); err != nil {
			return fmt.Errorf("SSWS config update failed: %w", err)
		}
	}

	if user.ProtocolType != "SSH" && user.ProtocolType != "SSWS" {
		var users []entity.ExtraUser
		database.GetDB().Find(&users)
		users = append(users, *user)
		
		var setting entity.ExtraSetting
		if err := database.GetDB().Where("protocol_name = ?", user.ProtocolType).First(&setting).Error; err == nil {
			if setting.IsEnabled {
				GetExtraServicesManager().RestartDaemon(user.ProtocolType, setting.ListeningPort, users)
			}
		}
	}

	return database.GetDB().Create(user).Error
}

func (s *ExtraProtocolsService) UpdateUser(id int64, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}

	var user entity.ExtraUser
	if err := database.GetDB().First(&user, id).Error; err != nil {
		return err
	}

	if user.ProtocolType == "SSH" {
		if pass, ok := updates["password"].(string); ok {
			if err := s.sysManager.UpdateOSUserPassword(user.Username, pass); err != nil {
				return fmt.Errorf("OS password update failed: %w", err)
			}
		}
		if uname, ok := updates["username"].(string); ok && uname != user.Username {
			return fmt.Errorf("changing SSH username is not supported via UI")
		}
	} else if user.ProtocolType == "SSWS" {
		if pass, ok := updates["password"].(string); ok {
			if err := s.sysManager.ManageSSWSConfig(user.Username, pass, "update"); err != nil {
				return fmt.Errorf("SSWS config update failed: %w", err)
			}
		}
	}

	if user.ProtocolType != "SSH" && user.ProtocolType != "SSWS" {
		var users []entity.ExtraUser
		database.GetDB().Find(&users)
		
		var setting entity.ExtraSetting
		if err := database.GetDB().Where("protocol_name = ?", user.ProtocolType).First(&setting).Error; err == nil {
			if setting.IsEnabled {
				GetExtraServicesManager().RestartDaemon(user.ProtocolType, setting.ListeningPort, users)
			}
		}
	}

	return database.GetDB().Model(&entity.ExtraUser{}).Where("id = ?", id).Updates(updates).Error
}

func (s *ExtraProtocolsService) DeleteUser(id int64) error {
	var user entity.ExtraUser
	if err := database.GetDB().First(&user, id).Error; err != nil {
		return err
	}

	if user.ProtocolType == "SSH" {
		if err := s.sysManager.DeleteOSUser(user.Username); err != nil {
			return fmt.Errorf("OS user deletion failed: %w", err)
		}
	} else if user.ProtocolType == "SSWS" {
		if err := s.sysManager.ManageSSWSConfig(user.Username, user.Password, "delete"); err != nil {
			return fmt.Errorf("SSWS config deletion failed: %w", err)
		}
	}

	return database.GetDB().Delete(&entity.ExtraUser{}, id).Error
}

// --- Settings Management ---

func (s *ExtraProtocolsService) GetSettings() ([]entity.ExtraSetting, error) {
	var settings []entity.ExtraSetting
	if err := database.GetDB().Find(&settings).Error; err != nil {
		return nil, err
	}
	return settings, nil
}

func (s *ExtraProtocolsService) UpdateSetting(name string, port int, enabled bool, bannerText string) error {
	err := database.GetDB().Model(&entity.ExtraSetting{}).
		Where("protocol_name = ?", name).
		Updates(map[string]any{
			"listening_port": port,
			"is_enabled":     enabled,
			"banner_text":    bannerText,
		}).Error
	if err != nil {
		return err
	}

	if name != "SSH" && name != "SSWS" {
		var users []entity.ExtraUser
		database.GetDB().Find(&users)
		if enabled {
			if err := GetExtraServicesManager().StartDaemon(name, port, users); err != nil {
				logger.Errorf("Failed to start %s daemon after setting update: %v", name, err)
			}
		} else {
			if err := GetExtraServicesManager().StopDaemon(name); err != nil {
				logger.Errorf("Failed to stop %s daemon after setting update: %v", name, err)
			}
		}
	}

	if name == "Banner" || bannerText != "" {
		if err := s.updateBanner(bannerText); err != nil {
			logger.Errorf("Failed to update connection banner: %v", err)
		}
	}

	return nil
}

func (s *ExtraProtocolsService) updateBanner(text string) error {
	if err := os.WriteFile("/etc/issue.net", []byte(text), 0644); err != nil {
		return fmt.Errorf("failed to write /etc/issue.net: %w", err)
	}

	configs := []string{"/etc/ssh/sshd_config", "/etc/dropbear/dropbear.conf"}
	for _, path := range configs {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}

		content, err := os.ReadFile(path)
		if err != nil {
			logger.Warningf("Could not read config %s: %v", path, err)
			continue
		}

		strContent := string(content)
		if !strings.Contains(strContent, "Banner /etc/issue.net") {
			strContent += "\nBanner /etc/issue.net\n"
			if err := os.WriteFile(path, []byte(strContent), 0644); err != nil {
				logger.Errorf("Failed to update %s: %v", path, err)
			}
		}
	}

	_ = exec.Command("systemctl", "reload", "ssh").Run()
	_ = exec.Command("systemctl", "reload", "dropbear").Run()

	return nil
}

// Helper to get port for a specific protocol
func (s *ExtraProtocolsService) GetPort(protocol string) (int, error) {
	var setting entity.ExtraSetting
	if err := database.GetDB().Where("protocol_name = ?", protocol).First(&setting).Error; err != nil {
		return 0, err
	}
	return setting.ListeningPort, nil
}
