package service

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

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

func (s *ExtraProtocolsService) GetUsers(serverHost ...string) ([]entity.ExtraUser, error) {
	var users []entity.ExtraUser
	if err := database.GetDB().Find(&users).Error; err != nil {
		return nil, err
	}
	host := "YOUR_SERVER_IP"
	if len(serverHost) > 0 && strings.TrimSpace(serverHost[0]) != "" {
		host = strings.TrimSpace(serverHost[0])
	}
	settings, _ := s.GetSettings()
	ports := make(map[string]int, len(settings))
	for _, setting := range settings {
		ports[setting.ProtocolName] = setting.ListeningPort
	}
	for i := range users {
		users[i].ConfigString, users[i].FormattedDetails = s.BuildConnectionDetails(users[i], host, ports)
	}
	return users, nil
}

func (s *ExtraProtocolsService) BuildConnectionDetails(user entity.ExtraUser, serverHost string, ports map[string]int) (string, map[string]string) {
	protocol := strings.TrimSpace(user.ProtocolType)
	port := ports[protocol]
	if port == 0 {
		port = defaultExtraProtocolPort(protocol)
	}
	payload := parseExtraPayload(user.ConfigPayload)
	details := map[string]string{
		"Protocol": protocol,
		"Server":   serverHost,
		"Port":     fmt.Sprintf("%d", port),
		"Username": user.Username,
		"Password": user.Password,
		"Status":   user.Status,
		"Expiry":   formatExtraExpiry(user.ExpiryDate),
	}

	var config string
	switch strings.ToUpper(protocol) {
	case "SSH":
		config = fmt.Sprintf("%s:%d@%s:%s", serverHost, port, user.Username, user.Password)
		details["Connection"] = config
	case "DROPBEAR":
		config = fmt.Sprintf("dropbear://%s:%s@%s:%d", user.Username, user.Password, serverHost, port)
		details["Connection"] = fmt.Sprintf("%s:%d@%s:%s", serverHost, port, user.Username, user.Password)
	case "SSWS", "SSH-WS":
		path := firstPayloadValue(payload, "path", "wsPath", "websocketPath")
		if path == "" {
			path = "/"
		}
		hostHeader := firstPayloadValue(payload, "host", "sni", "bugHost")
		if hostHeader == "" {
			hostHeader = serverHost
		}
		payloadText := fmt.Sprintf("GET %s HTTP/1.1[crlf]Host: %s[crlf]Upgrade: websocket[crlf][crlf]", path, hostHeader)
		config = fmt.Sprintf("sshws://%s:%s@%s:%d?path=%s&host=%s", user.Username, user.Password, serverHost, port, path, hostHeader)
		details["Payload"] = payloadText
		details["WebSocket Path"] = path
		details["Host Header"] = hostHeader
	case "SLOW-DNS":
		domain := firstPayloadValue(payload, "domain", "ns", "nameserver")
		if domain == "" {
			domain = "your-ns-domain.example.com"
		}
		publicKey := firstPayloadValue(payload, "publicKey", "pubKey", "dnsttKey", "key")
		if publicKey == "" {
			publicKey = readFirstExistingFile("/etc/dnstt/server.pub", "/etc/3x-ui/extra/dnstt.pub")
		}
		if publicKey == "" {
			publicKey = "DNSTT_PUBLIC_KEY_NOT_SET"
		}
		config = fmt.Sprintf("dnstt://%s:%s@%s:%d?domain=%s&pubkey=%s", user.Username, user.Password, serverHost, port, domain, publicKey)
		details["DNSTT Domain/NS"] = domain
		details["DNSTT Public Key"] = publicKey
		details["Client Command"] = fmt.Sprintf("dnstt-client -udp %s:%d -pubkey %s %s 127.0.0.1:2222", serverHost, port, publicKey, domain)
	case "PSIPHON":
		serverEntry := firstPayloadValue(payload, "serverEntry", "entry", "config")
		if serverEntry == "" {
			serverEntry = fmt.Sprintf("psiphon://%s:%s@%s:%d", user.Username, user.Password, serverHost, port)
		}
		config = serverEntry
		details["Server Entry"] = serverEntry
	case "UDP CUSTOM (BADVPN)", "UDP CUSTOM":
		config = fmt.Sprintf("udp-custom://%s:%s@%s:%d", user.Username, user.Password, serverHost, port)
		details["BadVPN Gateway"] = fmt.Sprintf("%s:%d", serverHost, port)
	case "SSL (STUNNEL)", "STUNNEL":
		sni := firstPayloadValue(payload, "sni", "host")
		if sni == "" {
			sni = serverHost
		}
		config = fmt.Sprintf("stunnel://%s:%s@%s:%d?sni=%s", user.Username, user.Password, serverHost, port, sni)
		details["TLS/SNI"] = sni
		details["SSH over TLS"] = fmt.Sprintf("%s:%d@%s:%s", serverHost, port, user.Username, user.Password)
	case "OPENVPN":
		config = fmt.Sprintf("openvpn://%s:%s@%s:%d", user.Username, user.Password, serverHost, port)
	case "SQUID":
		config = fmt.Sprintf("http-proxy://%s:%s@%s:%d", user.Username, user.Password, serverHost, port)
	case "OHP":
		config = fmt.Sprintf("ohp://%s:%s@%s:%d", user.Username, user.Password, serverHost, port)
	default:
		config = fmt.Sprintf("%s:%d@%s:%s", serverHost, port, user.Username, user.Password)
	}

	details["Config"] = config
	return config, orderedDetails(details)
}

func parseExtraPayload(raw string) map[string]string {
	result := map[string]string{}
	if strings.TrimSpace(raw) == "" {
		return result
	}
	var anyMap map[string]any
	if err := json.Unmarshal([]byte(raw), &anyMap); err == nil {
		for key, value := range anyMap {
			result[key] = fmt.Sprint(value)
		}
		return result
	}
	for _, part := range strings.FieldsFunc(raw, func(r rune) bool { return r == '\n' || r == ';' || r == ',' }) {
		key, value, ok := strings.Cut(part, "=")
		if ok {
			result[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	return result
}

func firstPayloadValue(payload map[string]string, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(payload[key]); value != "" {
			return value
		}
	}
	return ""
}

func formatExtraExpiry(expiry int64) string {
	if expiry <= 0 {
		return "Never"
	}
	if expiry < 1_000_000_000_000 {
		return time.Unix(expiry, 0).Format("2006-01-02 15:04:05")
	}
	return time.UnixMilli(expiry).Format("2006-01-02 15:04:05")
}

func defaultExtraProtocolPort(protocol string) int {
	switch strings.ToUpper(strings.TrimSpace(protocol)) {
	case "SSH":
		return 2222
	case "SSWS", "SSH-WS":
		return 8443
	case "SLOW-DNS":
		return 5353
	case "PSIPHON":
		return 443
	case "UDP CUSTOM (BADVPN)", "UDP CUSTOM":
		return 7300
	case "DROPBEAR":
		return 2223
	case "SSL (STUNNEL)", "STUNNEL":
		return 444
	case "OPENVPN":
		return 1194
	case "SQUID":
		return 3128
	case "OHP":
		return 80
	default:
		return 0
	}
}

func readFirstExistingFile(paths ...string) string {
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err == nil && strings.TrimSpace(string(content)) != "" {
			return strings.TrimSpace(string(content))
		}
	}
	return ""
}

func orderedDetails(details map[string]string) map[string]string {
	// JSON objects are unordered, but sorting here gives deterministic maps for tests/logging before encoding.
	keys := make([]string, 0, len(details))
	for key := range details {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	ordered := make(map[string]string, len(details))
	for _, key := range keys {
		ordered[key] = details[key]
	}
	return ordered
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
