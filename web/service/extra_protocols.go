package service

import (
	"encoding/json"
	"fmt"
	"net"
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
		{ProtocolName: "SSH", ListeningPort: 22, IsEnabled: false},
		{ProtocolName: "SSWS", ListeningPort: 80, IsEnabled: false},
		{ProtocolName: "SLOW-DNS", ListeningPort: 5353, IsEnabled: false},
		{ProtocolName: "Psiphon", ListeningPort: 3001, IsEnabled: false},
		{ProtocolName: "UDP Custom (BadVPN)", ListeningPort: 7300, IsEnabled: false},
		{ProtocolName: "Dropbear", ListeningPort: 143, IsEnabled: false},
		{ProtocolName: "SSL (Stunnel)", ListeningPort: 443, IsEnabled: false},
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
		account := sshAccount(serverHost, port, user.Username, user.Password)
		config = joinConfigLines(
			"HTTP Custom - SSH",
			fmt.Sprintf("Host:Port@User:Pass: %s", account),
			"SNI: none",
		)
		details["SSH Account"] = account
	case "DROPBEAR":
		account := sshAccount(serverHost, port, user.Username, user.Password)
		config = joinConfigLines(
			"HTTP Custom - Dropbear SSH",
			fmt.Sprintf("Host:Port@User:Pass: %s", account),
			fmt.Sprintf("Alternative 22: %s", sshAccount(serverHost, 22, user.Username, user.Password)),
			"SNI: none",
		)
		details["SSH Account"] = account
		details["Alternative SSH Account"] = sshAccount(serverHost, 22, user.Username, user.Password)
	case "SSWS", "SSH-WS":
		path := firstPayloadValue(payload, "path", "wsPath", "websocketPath")
		if path == "" {
			path = "/"
		}
		hostHeader := firstPayloadValue(payload, "host", "sni", "bugHost")
		if hostHeader == "" {
			hostHeader = serverHost
		}
		wsPort := payloadInt(payload, 80, "port", "wsPort", "remoteProxyPort")
		payloadText := fmt.Sprintf("GET %s HTTP/1.1[crlf]Host: %s[crlf]Upgrade: websocket[crlf]Connection: Upgrade[crlf][crlf]", path, hostHeader)
		remoteProxy := sshAccount(serverHost, wsPort, user.Username, user.Password)
		config = joinConfigLines(
			"HTTP Custom - SSH-WS (Payload)",
			fmt.Sprintf("SSH Account / Remote Proxy: %s", remoteProxy),
			fmt.Sprintf("Proxy Host: %s", serverHost),
			fmt.Sprintf("Proxy Port: %d", wsPort),
			fmt.Sprintf("Payload: %s", payloadText),
		)
		details["Remote Proxy"] = remoteProxy
		details["Payload"] = payloadText
		details["WebSocket Path"] = path
		details["Host Header"] = hostHeader
	case "SLOW-DNS":
		domain := firstPayloadValue(payload, "domain", "ns", "nameserver")
		if domain == "" {
			domain = readFirstExistingFile("/etc/dnstt/nameserver", "/etc/3x-ui/extra/dnstt.nameserver")
		}
		if domain == "" {
			domain = generatedNameserver(serverHost)
		}
		publicKey := firstPayloadValue(payload, "publicKey", "pubKey", "dnsttKey", "key")
		if publicKey == "" {
			publicKey = readFirstExistingFile("/etc/dnstt/server.pub", "/etc/3x-ui/extra/dnstt.pub")
		}
		if publicKey == "" {
			publicKey = "GENERATED_PUBKEY_NOT_SET"
		}
		account := sshAccount(serverHost, 22, user.Username, user.Password)
		config = joinConfigLines(
			"HTTP Custom - SlowDNS",
			fmt.Sprintf("SSH Account: %s", account),
			fmt.Sprintf("Nameserver: %s", domain),
			fmt.Sprintf("Public Key: %s", publicKey),
			fmt.Sprintf("DNS Server: %s", serverHost),
			"Mode: DNSTT / SlowDNS",
		)
		details["SSH Account"] = account
		details["DNSTT Domain/NS"] = domain
		details["DNSTT Public Key"] = publicKey
		details["Client Command"] = fmt.Sprintf("dnstt-client -udp %s:%d -pubkey %s %s 127.0.0.1:22", serverHost, port, publicKey, domain)
	case "PSIPHON":
		serverEntry := firstPayloadValue(payload, "serverEntry", "entry", "config")
		if serverEntry == "" {
			serverEntry = readFirstExistingFile("/etc/psiphon/server-entry.dat", "/etc/psiphon/server-entry.json", "/etc/3x-ui/extra/psiphon-server-entry.dat")
		}
		if serverEntry == "" {
			serverEntry = "PSIPHON_SERVER_ENTRY_NOT_GENERATED_RUN_setup_extra_protocols.sh"
		}
		config = joinConfigLines(
			"HTTP Custom - Psiphon",
			"Server Entry:",
			serverEntry,
		)
		details["Server Entry"] = serverEntry
	case "UDP CUSTOM (BADVPN)", "UDP CUSTOM":
		sshPort := payloadInt(payload, 22, "sshPort", "accountPort")
		udpGatewayPort := payloadInt(payload, 7300, "udpGatewayPort", "udpgwPort", "gatewayPort")
		account := sshAccount(serverHost, sshPort, user.Username, user.Password)
		config = joinConfigLines(
			"HTTP Custom - UDP Custom",
			fmt.Sprintf("SSH Account: %s", account),
			fmt.Sprintf("UDP Gateway Port: %d", udpGatewayPort),
		)
		details["SSH Account"] = account
		details["UDP Gateway Port"] = fmt.Sprintf("%d", udpGatewayPort)
		details["BadVPN Gateway"] = fmt.Sprintf("%s:%d", serverHost, udpGatewayPort)
	case "SSL (STUNNEL)", "STUNNEL":
		sni := firstPayloadValue(payload, "sni", "host")
		if sni == "" {
			sni = serverHost
		}
		sslPort := payloadInt(payload, 443, "sslPort", "stunnelPort", "port")
		account := sshAccount(serverHost, sslPort, user.Username, user.Password)
		config = joinConfigLines(
			"HTTP Custom - SSL/Stunnel",
			fmt.Sprintf("Host:Port@User:Pass: %s", account),
			fmt.Sprintf("SNI: %s", sni),
			"TLS Mode: SSL/TLS Direct",
		)
		details["TLS/SNI"] = sni
		details["SSH Account"] = account
	case "OPENVPN":
		openVPNPort := payloadInt(payload, 1194, "openvpnPort", "vpnPort", "port")
		config = buildOpenVPNClientTemplate(serverHost, openVPNPort, user.Username, user.Password)
		details["Remote"] = fmt.Sprintf("%s:%d", serverHost, openVPNPort)
		details["Cipher"] = "AES-128-CBC"
	case "SQUID":
		config = joinConfigLines(
			"HTTP Custom - Squid Proxy",
			fmt.Sprintf("Proxy Host: %s", serverHost),
			fmt.Sprintf("Proxy Port: %d", port),
			fmt.Sprintf("Username: %s", user.Username),
			fmt.Sprintf("Password: %s", user.Password),
		)
	case "OHP":
		config = joinConfigLines(
			"HTTP Custom - OHP",
			fmt.Sprintf("SSH Account: %s", sshAccount(serverHost, 22, user.Username, user.Password)),
			fmt.Sprintf("OHP Port: %d", port),
		)
	default:
		config = joinConfigLines(
			"HTTP Custom - Generic SSH",
			fmt.Sprintf("SSH Account: %s", sshAccount(serverHost, port, user.Username, user.Password)),
		)
	}

	details["Config"] = config
	return config, orderedDetails(details)
}

func joinConfigLines(lines ...string) string {
	cleaned := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		if strings.TrimSpace(line) != "" {
			cleaned = append(cleaned, line)
		}
	}
	return strings.Join(cleaned, "\n")
}

func sshAccount(serverHost string, port int, username, password string) string {
	return fmt.Sprintf("%s:%d@%s:%s", serverHost, port, username, password)
}

func payloadInt(payload map[string]string, fallback int, keys ...string) int {
	for _, key := range keys {
		value := strings.TrimSpace(payload[key])
		if value == "" {
			continue
		}
		var parsed int
		if _, err := fmt.Sscanf(value, "%d", &parsed); err == nil && parsed > 0 && parsed <= 65535 {
			return parsed
		}
	}
	return fallback
}

func generatedNameserver(serverHost string) string {
	serverHost = strings.TrimSpace(strings.Trim(serverHost, "[]"))
	if serverHost == "" || serverHost == "YOUR_SERVER_IP" || net.ParseIP(serverHost) != nil {
		return "GENERATED_NS_NOT_SET"
	}
	return "ns." + serverHost
}

func buildOpenVPNClientTemplate(serverHost string, port int, username, password string) string {
	ca := readFirstExistingFile("/etc/openvpn/3x-ui/ca.crt", "/etc/openvpn/ca.crt", "/etc/3x-ui/extra/openvpn-ca.crt")
	if ca == "" {
		ca = "-----BEGIN CERTIFICATE-----\nPASTE_CA_CERTIFICATE_HERE\n-----END CERTIFICATE-----"
	}
	tlsAuth := readFirstExistingFile("/etc/openvpn/3x-ui/ta.key", "/etc/openvpn/ta.key", "/etc/3x-ui/extra/openvpn-ta.key")

	lines := []string{
		"# OpenVPN client template",
		"# Save this as client.ovpn. OpenVPN will prompt for PAM/Linux credentials.",
		fmt.Sprintf("# Username: %s", username),
		fmt.Sprintf("# Password: %s", password),
		"client",
		"dev tun",
		"proto udp",
		fmt.Sprintf("remote %s %d", serverHost, port),
		"remote-random",
		"resolv-retry infinite",
		"nobind",
		"persist-key",
		"persist-tun",
		"remote-cert-tls server",
		"auth-user-pass",
		"cipher AES-128-CBC",
		"data-ciphers AES-256-GCM:AES-128-GCM:CHACHA20-POLY1305:AES-128-CBC",
		"data-ciphers-fallback AES-128-CBC",
		"auth SHA256",
		"key-direction 1",
		"redirect-gateway def1",
		"dhcp-option DNS 1.1.1.1",
		"verb 3",
		"<ca>",
		ca,
		"</ca>",
	}
	if tlsAuth != "" {
		lines = append(lines, "<tls-auth>", tlsAuth, "</tls-auth>")
	}
	return joinConfigLines(lines...)
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
		return 22
	case "SSWS", "SSH-WS":
		return 80
	case "SLOW-DNS":
		return 5353
	case "PSIPHON":
		return 3001
	case "UDP CUSTOM (BADVPN)", "UDP CUSTOM":
		return 7300
	case "DROPBEAR":
		return 143
	case "SSL (STUNNEL)", "STUNNEL":
		return 443
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

	if err := s.ensureLinuxAccount(user.Username, user.Password, user.ExpiryDate); err != nil {
		return fmt.Errorf("Linux user sync failed: %w", err)
	}

	if user.ProtocolType == "SSWS" {
		if err := s.sysManager.ManageSSWSConfig(user.Username, user.Password, "add"); err != nil {
			return fmt.Errorf("SSWS config update failed: %w", err)
		}
	}

	s.restartProtocolIfEnabled(user.ProtocolType, []entity.ExtraUser{*user})

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

	newUsername := user.Username
	if uname, ok := updates["username"].(string); ok && strings.TrimSpace(uname) != "" {
		newUsername = strings.TrimSpace(uname)
	}
	newPassword := user.Password
	if pass, ok := updates["password"].(string); ok && pass != "" {
		newPassword = pass
	}
	newExpiry := user.ExpiryDate
	if expiry, ok := normalizeExpiryUpdate(updates["expiryDate"]); ok {
		newExpiry = expiry
	}
	newProtocol := user.ProtocolType
	if protocol, ok := updates["protocolType"].(string); ok && strings.TrimSpace(protocol) != "" {
		newProtocol = strings.TrimSpace(protocol)
	}

	if err := s.ensureLinuxAccount(newUsername, newPassword, newExpiry); err != nil {
		return fmt.Errorf("Linux user sync failed: %w", err)
	}
	if newUsername != user.Username {
		if err := s.sysManager.DeleteOSUser(user.Username); err != nil {
			logger.Warningf("old Linux user deletion failed after rename from %s to %s: %v", user.Username, newUsername, err)
		}
	}

	if newProtocol == "SSWS" {
		if err := s.sysManager.ManageSSWSConfig(newUsername, newPassword, "update"); err != nil {
			return fmt.Errorf("SSWS config update failed: %w", err)
		}
	} else if user.ProtocolType == "SSWS" && newProtocol != "SSWS" {
		_ = s.sysManager.ManageSSWSConfig(user.Username, user.Password, "delete")
	}

	if err := database.GetDB().Model(&entity.ExtraUser{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return err
	}
	s.restartProtocolIfEnabled(user.ProtocolType, nil)
	if newProtocol != user.ProtocolType {
		s.restartProtocolIfEnabled(newProtocol, nil)
	}
	return nil
}

func (s *ExtraProtocolsService) DeleteUser(id int64) error {
	var user entity.ExtraUser
	if err := database.GetDB().First(&user, id).Error; err != nil {
		return err
	}

	if err := s.sysManager.DeleteOSUser(user.Username); err != nil {
		return fmt.Errorf("Linux user deletion failed: %w", err)
	}

	if user.ProtocolType == "SSWS" {
		if err := s.sysManager.ManageSSWSConfig(user.Username, user.Password, "delete"); err != nil {
			return fmt.Errorf("SSWS config deletion failed: %w", err)
		}
	}

	if err := database.GetDB().Delete(&entity.ExtraUser{}, id).Error; err != nil {
		return err
	}
	s.restartProtocolIfEnabled(user.ProtocolType, nil)
	return nil
}

func (s *ExtraProtocolsService) ensureLinuxAccount(username, password string, expiry int64) error {
	if err := s.sysManager.CreateOSUser(username, password); err != nil {
		return err
	}
	return s.setLinuxAccountExpiry(username, expiry)
}

func (s *ExtraProtocolsService) setLinuxAccountExpiry(username string, expiry int64) error {
	if !usernameRegex.MatchString(username) {
		return fmt.Errorf("invalid username")
	}
	if expiry <= 0 {
		if out, err := exec.Command("usermod", "-e", "", username).CombinedOutput(); err != nil {
			return fmt.Errorf("usermod clear expiry failed: %w: %s", err, string(out))
		}
		return nil
	}
	expiryDate := time.Unix(expiry, 0)
	if expiry >= 1_000_000_000_000 {
		expiryDate = time.UnixMilli(expiry)
	}
	if out, err := exec.Command("usermod", "-e", expiryDate.Format("2006-01-02"), username).CombinedOutput(); err != nil {
		return fmt.Errorf("usermod expiry failed: %w: %s", err, string(out))
	}
	return nil
}

func normalizeExpiryUpdate(value any) (int64, bool) {
	switch v := value.(type) {
	case nil:
		return 0, false
	case int64:
		return v, true
	case int:
		return int64(v), true
	case float64:
		return int64(v), true
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return 0, false
		}
		var parsed int64
		if _, err := fmt.Sscanf(v, "%d", &parsed); err == nil {
			return parsed, true
		}
	}
	return 0, false
}

func (s *ExtraProtocolsService) restartProtocolIfEnabled(protocol string, extraUsers []entity.ExtraUser) {
	var setting entity.ExtraSetting
	if err := database.GetDB().Where("protocol_name = ?", protocol).First(&setting).Error; err != nil || !setting.IsEnabled {
		return
	}
	var users []entity.ExtraUser
	database.GetDB().Find(&users)
	users = append(users, extraUsers...)
	if err := GetExtraServicesManager().RestartDaemon(protocol, setting.ListeningPort, users); err != nil {
		logger.Errorf("Failed to restart %s daemon after user sync: %v", protocol, err)
	}
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
