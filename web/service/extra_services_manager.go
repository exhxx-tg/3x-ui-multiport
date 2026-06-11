package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/mhsanaei/3x-ui/v3/config"
	"github.com/mhsanaei/3x-ui/v3/logger"
	"github.com/mhsanaei/3x-ui/v3/web/entity"
)

// DaemonInfo tracks the state of a running background process.
type DaemonInfo struct {
	Cmd    *exec.Cmd
	PID    int
	Port   int
	Status string // running, stopped, errored
}

// ExtraServicesManager handles the lifecycle of standalone daemons for extra protocols.
type ExtraServicesManager struct {
	registry map[string]*DaemonInfo
	mu       sync.RWMutex
}

func NewExtraServicesManager() *ExtraServicesManager {
	return &ExtraServicesManager{
		registry: make(map[string]*DaemonInfo),
	}
}

var globalExtraServicesManager = NewExtraServicesManager()

func GetExtraServicesManager() *ExtraServicesManager {
	return globalExtraServicesManager
}

// StartDaemon launches a background process for a specific protocol.
func (m *ExtraServicesManager) StartDaemon(protocol string, port int, users []entity.ExtraUser) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if info, exists := m.registry[protocol]; exists && info.Cmd != nil && info.Cmd.Process != nil {
		m.stopDaemonLocked(protocol)
	}

	configPath, err := m.generateConfigFile(protocol, port, users)
	if err != nil {
		// Some protocols might not need a config file, we'll handle that in generateConfigFile
		// If it's not required, it might return a specific error or empty path.
	}

	binaryPath := m.getBinaryPath(protocol)
	if binaryPath == "" {
		return fmt.Errorf("binary path not defined for protocol: %s", protocol)
	}
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("binary not found at %s", binaryPath)
	}

	args := m.getArgs(protocol, port, configPath)
	cmd := exec.Command(binaryPath, args...)

	logFile, err := os.OpenFile(filepath.Join(config.GetLogFolder(), fmt.Sprintf("%s.log", protocol)), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon %s: %w", protocol, err)
	}

	m.registry[protocol] = &DaemonInfo{
		Cmd:    cmd,
		PID:    cmd.Process.Pid,
		Port:   port,
		Status: "running",
	}

	logger.Infof("Started %s daemon (PID: %d) on port %d", protocol, cmd.Process.Pid, port)
	return nil
}

func (m *ExtraServicesManager) StopDaemon(protocol string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stopDaemonLocked(protocol)
}

func (m *ExtraServicesManager) stopDaemonLocked(protocol string) error {
	info, exists := m.registry[protocol]
	if !exists || info.Cmd == nil || info.Cmd.Process == nil {
		return nil
	}

	_ = info.Cmd.Process.Signal(syscall.SIGTERM)

	done := make(chan error, 1)
	go func() {
		done <- info.Cmd.Wait()
	}()

	select {
	case <-done:
		logger.Infof("Daemon %s stopped gracefully", protocol)
	case <-time.After(5 * time.Second):
		logger.Warningf("Daemon %s did not stop in time, killing...", protocol)
		_ = info.Cmd.Process.Kill()
	}

	delete(m.registry, protocol)
	return nil
}

func (m *ExtraServicesManager) RestartDaemon(protocol string, port int, users []entity.ExtraUser) error {
	if err := m.StopDaemon(protocol); err != nil {
		return err
	}
	return m.StartDaemon(protocol, port, users)
}

func (m *ExtraServicesManager) generateConfigFile(protocol string, port int, users []entity.ExtraUser) (string, error) {
	configDir := "/etc/3x-ui/extra"
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}

	filePath := filepath.Join(configDir, fmt.Sprintf("%s.conf", protocol))
	var content string

	switch protocol {
	case "SLOW-DNS":
		content = m.buildSlowDnsConfig(port, users)
	case "Psiphon":
		content = m.buildPsiphonConfig(port, users)
	case "SSL (Stunnel)":
		content = m.buildStunnelConfig(port)
	case "OpenVPN":
		content = m.buildOpenVpnConfig(port)
	case "Squid":
		content = m.buildSquidConfig(port)
	case "OHP":
		content = m.buildOhpConfig(port)
	default:
		return "", nil // No config file needed for some protocols (e.g. UDP Custom (BadVPN), Dropbear might use flags)
	}

	tmpPath := filePath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(content), 0644); err != nil {
		return "", err
	}
	if err := os.Rename(tmpPath, filePath); err != nil {
		return "", err
	}

	return filePath, nil
}

func (m *ExtraServicesManager) buildSlowDnsConfig(port int, users []entity.ExtraUser) string {
	conf := fmt.Sprintf("[global]\nport = %d\n\n[users]\n", port)
	for _, u := range users {
		if u.ProtocolType == "SLOW-DNS" {
			conf += fmt.Sprintf("%s = %s\n", u.Username, u.ConfigPayload)
		}
	}
	return conf
}

func (m *ExtraServicesManager) buildPsiphonConfig(port int, users []entity.ExtraUser) string {
	conf := fmt.Sprintf("listen_port = %d\n", port)
	for _, u := range users {
		if u.ProtocolType == "Psiphon" {
			conf += fmt.Sprintf("user %s auth %s\n", u.Username, u.ConfigPayload)
		}
	}
	return conf
}

func (m *ExtraServicesManager) buildStunnelConfig(port int) string {
	return fmt.Sprintf("[service]\nport = %d\nclient = yes\n", port)
}

func (m *ExtraServicesManager) buildOpenVpnConfig(port int) string {
	return fmt.Sprintf("port %d\nproto udp\ndev tun\n", port)
}

func (m *ExtraServicesManager) buildSquidConfig(port int) string {
	return fmt.Sprintf("http_port %d\n", port)
}

func (m *ExtraServicesManager) buildOhpConfig(port int) string {
	return fmt.Sprintf("listen %d\n", port)
}

func (m *ExtraServicesManager) getBinaryPath(protocol string) string {
	switch protocol {
	case "SLOW-DNS":
		return "/usr/local/bin/slow-dns"
	case "Psiphon":
		return "/usr/local/bin/psiphon-server"
	case "UDP Custom (BadVPN)":
		return "/usr/local/bin/badvpn-udpgw"
	case "Dropbear":
		return "/usr/local/bin/dropbear"
	case "SSL (Stunnel)":
		return "/usr/local/bin/stunnel4"
	case "OpenVPN":
		return "/usr/local/bin/openvpn"
	case "Squid":
		return "/usr/local/bin/squid"
	case "OHP":
		return "/usr/local/bin/ohp"
	default:
		return ""
	}
}

func (m *ExtraServicesManager) getArgs(protocol string, port int, configPath string) []string {
	switch protocol {
	case "UDP Custom (BadVPN)":
		return []string{"-l", fmt.Sprintf("%d", port)}
	case "Dropbear":
		return []string{"-p", fmt.Sprintf("%d", port)}
	case "SSL (Stunnel)":
		return []string{"-f", configPath}
	case "OpenVPN":
		return []string{"--config", configPath}
	case "Squid":
		return []string{"-f", configPath}
	case "OHP":
		return []string{"-c", configPath}
	case "SLOW-DNS", "Psiphon":
		return []string{"-c", configPath}
	default:
		return []string{}
	}
}

func (m *ExtraServicesManager) SyncAllDaemons() {
	// Note: This would need to be updated to use the new protocol list.
	// Implementation will be handled in ExtraProtocolsService.
}
