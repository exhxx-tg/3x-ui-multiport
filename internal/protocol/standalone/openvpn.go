package standalone

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/exhxx-tg/3x-ui-multiport/internal/protocol"
	"github.com/exhxx-tg/3x-ui-multiport/internal/servicemanager"
)

func init() {
	protocol.RegisterStandaloneFactory(func(id protocol.ProtocolID) protocol.StandaloneService {
		switch id {
		case protocol.ProtocolOpenVPN:
			return NewOpenVPN()
		case protocol.ProtocolWireGuard:
			return NewWireGuard()
		case protocol.ProtocolDropbear:
			return NewDropbear()
		default:
			return nil
		}
	})
}

type OpenVPN struct {
	status      protocol.Status
	port        int
	configPath  string
	installPath string
	pkiDir      string
	sm          *servicemanager.ServiceManager
	pki         *PKIStore
	serverAddr  string
}

func NewOpenVPN() *OpenVPN {
	o := &OpenVPN{
		status:      protocol.StatusUnknown,
		port:        1194,
		configPath:  "/etc/openvpn/server.conf",
		installPath: "/usr/sbin/openvpn",
		pkiDir:      "/etc/openvpn/pki",
		sm:          servicemanager.Global(),
	}
	o.pki = NewPKIStore(o.pkiDir)
	binPath := o.detectBinary()
	o.sm.Register(servicemanager.ServiceDefinition{
		Name:        "openvpn",
		DisplayName: "OpenVPN",
		BinaryName:  "openvpn",
		BinaryPath:  binPath,
		ConfigPath:  o.configPath,
		LogPath:     "/var/log/x-ui/openvpn.log",
		DefaultPort: 1194,
		Args:        []string{"--config", o.configPath},
		InstallHint: "apt-get install openvpn -y",
	})
	return o
}

func (o *OpenVPN) detectBinary() string {
	if runtime.GOOS == "windows" {
		if p, err := exec.LookPath("openvpn.exe"); err == nil {
			return p
		}
		return "C:\\Program Files\\OpenVPN\\bin\\openvpn.exe"
	}
	if _, err := os.Stat(o.installPath); err == nil {
		return o.installPath
	}
	if p, err := exec.LookPath("openvpn"); err == nil {
		return p
	}
	return "/usr/sbin/openvpn"
}

func (o *OpenVPN) ID() protocol.ProtocolID { return protocol.ProtocolOpenVPN }
func (o *OpenVPN) Info() protocol.ProtocolInfo {
	return *protocol.GetProtocolInfo(protocol.ProtocolOpenVPN)
}
func (o *OpenVPN) ServiceName() string { return "openvpn" }

func (o *OpenVPN) Status() protocol.Status {
	state := o.sm.Status("openvpn")
	switch state {
	case servicemanager.StateRunning:
		o.status = protocol.StatusRunning
	case servicemanager.StateStopped:
		o.status = protocol.StatusStopped
	case servicemanager.StateError:
		o.status = protocol.StatusError
	default:
		o.status = protocol.StatusUnknown
	}
	return o.status
}

func (o *OpenVPN) IsInstalled() bool {
	return o.sm.IsInstalled("openvpn")
}

func (o *OpenVPN) Install() error {
	o.status = protocol.StatusInstalling
	if o.IsInstalled() && o.pki.CAExists() {
		o.status = protocol.StatusStopped
		return nil
	}
	configDir := filepath.Dir(o.configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		o.status = protocol.StatusError
		return fmt.Errorf("failed to create config dir: %w", err)
	}
	if !o.pki.CAExists() {
		if err := o.pki.GenerateCA("x-ui-pro-openvpn-ca"); err != nil {
			o.status = protocol.StatusError
			return fmt.Errorf("failed to generate CA: %w", err)
		}
		if err := o.pki.GenerateServerCert(); err != nil {
			o.status = protocol.StatusError
			return fmt.Errorf("failed to generate server cert: %w", err)
		}
		tlsKey, err := o.pki.GenerateTLSACryptKey()
		if err != nil {
			o.status = protocol.StatusError
			return fmt.Errorf("failed to generate tls-crypt key: %w", err)
		}
		tlsKeyPath := filepath.Join(o.pkiDir, "tls-crypt.key")
		if err := os.WriteFile(tlsKeyPath, tlsKey, 0600); err != nil {
			o.status = protocol.StatusError
			return fmt.Errorf("failed to write tls-crypt key: %w", err)
		}
	}
	cfg := o.GenerateConfig("", nil)
	if err := os.WriteFile(o.configPath, []byte(cfg), 0600); err != nil {
		o.status = protocol.StatusError
		return fmt.Errorf("failed to write config: %w", err)
	}
	if err := o.sm.Install("openvpn"); err != nil {
		l := err.Error()
		if !strings.Contains(l, "systemd not available") {
			o.status = protocol.StatusError
			return fmt.Errorf("failed to install systemd unit: %w", err)
		}
	}
	o.status = protocol.StatusStopped
	return nil
}

func (o *OpenVPN) Uninstall() error {
	_ = o.sm.Uninstall("openvpn")
	o.status = protocol.StatusStopped
	return nil
}

func (o *OpenVPN) Start() error {
	if !o.IsInstalled() {
		if err := o.Install(); err != nil {
			return fmt.Errorf("openvpn not installed: %w", err)
		}
	}
	if err := checkPortAvailable(o.port); err != nil {
		o.status = protocol.StatusError
		return err
	}
	if err := o.sm.Start("openvpn"); err != nil {
		o.status = protocol.StatusError
		return err
	}
	o.status = protocol.StatusRunning
	return nil
}

func (o *OpenVPN) Stop() error {
	if err := o.sm.Stop("openvpn"); err != nil {
		return err
	}
	o.status = protocol.StatusStopped
	return nil
}

func (o *OpenVPN) Restart() error {
	_ = o.Stop()
	return o.Start()
}

func (o *OpenVPN) Config() (any, error) {
	ca, _ := o.pki.ReadCA()
	clients, _ := o.pki.ListClients()
	return map[string]any{
		"port":        o.port,
		"protocol":    "tcp",
		"configPath":  o.configPath,
		"installPath": o.installPath,
		"pkiDir":      o.pkiDir,
		"installed":   o.IsInstalled(),
		"caExists":    o.pki.CAExists(),
		"caCert":      string(ca),
		"clients":     clients,
		"pid":         o.sm.PID("openvpn"),
		"serverAddr":  o.serverAddr,
	}, nil
}

func (o *OpenVPN) ApplyConfig(cfg any) error {
	c, ok := cfg.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid config type")
	}
	if p, ok := c["port"].(int); ok {
		o.port = p
	}
	if cStr, ok := c["configPath"].(string); ok && cStr != "" {
		o.configPath = cStr
	}
	if addr, ok := c["serverAddr"].(string); ok && addr != "" {
		o.serverAddr = addr
	}
	return nil
}

func (o *OpenVPN) HealthCheck() error {
	state := o.sm.Status("openvpn")
	if state != servicemanager.StateRunning {
		return fmt.Errorf("openvpn is not running")
	}
	if !tcpPortCheck("127.0.0.1", o.port, 5*time.Second) {
		return fmt.Errorf("openvpn port %d is not responding", o.port)
	}
	return nil
}

func (o *OpenVPN) SetServerAddr(addr string) {
	o.serverAddr = addr
}

func (o *OpenVPN) AddClient(clientName string) (*ClientConfig, error) {
	if !o.pki.CAExists() {
		return nil, fmt.Errorf("PKI not initialized, install OpenVPN first")
	}
	if o.pki.ClientExists(clientName) {
		return nil, fmt.Errorf("client %q already exists", clientName)
	}
	pair, err := o.pki.GenerateClientCert(clientName)
	if err != nil {
		return nil, fmt.Errorf("failed to generate client cert: %w", err)
	}
	if err := o.pki.WriteClientCert(clientName, pair); err != nil {
		return nil, fmt.Errorf("failed to write client cert: %w", err)
	}
	cfg := o.GenerateClientConfig(clientName, pair)
	return &ClientConfig{
		ClientName: clientName,
		Config:     cfg,
		Cert:       string(pair.Cert),
		Key:        string(pair.Key),
	}, nil
}

func (o *OpenVPN) RemoveClient(clientName string) error {
	return o.pki.RemoveClient(clientName)
}

func (o *OpenVPN) ListClients() ([]string, error) {
	return o.pki.ListClients()
}

func (o *OpenVPN) GetClientConfig(clientName string) (*ClientConfig, error) {
	pair, err := o.pki.ReadClientCert(clientName)
	if err != nil {
		return nil, err
	}
	cfg := o.GenerateClientConfig(clientName, pair)
	return &ClientConfig{
		ClientName: clientName,
		Config:     cfg,
		Cert:       string(pair.Cert),
		Key:        string(pair.Key),
	}, nil
}

func (o *OpenVPN) GenerateConfig(serverAddr string, clients []map[string]any) string {
	addr := serverAddr
	if addr == "" {
		addr = o.serverAddr
	}
	var b strings.Builder
	b.WriteString("server 10.8.0.0 255.255.255.0\n")
	b.WriteString(fmt.Sprintf("port %d\n", o.port))
	b.WriteString("proto tcp\n")
	b.WriteString("dev tun\n")
	b.WriteString(fmt.Sprintf("ca %s\n", filepath.Join(o.pkiDir, "ca.crt")))
	b.WriteString(fmt.Sprintf("cert %s\n", filepath.Join(o.pkiDir, "server.crt")))
	b.WriteString(fmt.Sprintf("key %s\n", filepath.Join(o.pkiDir, "server.key")))
	b.WriteString("dh none\n")
	b.WriteString("auth SHA256\n")
	b.WriteString("cipher AES-256-GCM\n")
	b.WriteString("ncp-ciphers AES-256-GCM:AES-128-GCM:CHACHA20-POLY1305\n")
	b.WriteString("tls-version-min 1.2\n")
	b.WriteString("tls-cipher TLS-ECDHE-ECDSA-WITH-AES-256-GCM-SHA384\n")
	b.WriteString(fmt.Sprintf("tls-crypt %s\n", filepath.Join(o.pkiDir, "tls-crypt.key")))
	b.WriteString("keepalive 10 120\n")
	b.WriteString("max-clients 100\n")
	b.WriteString("push \"redirect-gateway def1 bypass-dhcp\"\n")
	b.WriteString("push \"dhcp-option DNS 8.8.8.8\"\n")
	b.WriteString("push \"dhcp-option DNS 1.1.1.1\"\n")
	b.WriteString("client-to-client\n")
	b.WriteString("duplicate-cn\n")
	b.WriteString("topology subnet\n")
	b.WriteString("status /var/log/x-ui/openvpn-status.log\n")
	b.WriteString("log-append /var/log/x-ui/openvpn.log\n")
	b.WriteString("verb 3\n")
	b.WriteString("mute 20\n")
	b.WriteString("explicit-exit-notify 1\n")
	return b.String()
}

func (o *OpenVPN) GenerateClientConfig(clientName string, pair *CertPair) string {
	addr := o.serverAddr
	if addr == "" {
		addr = "CHANGE_SERVER_ADDRESS"
	}
	var b strings.Builder
	b.WriteString("client\n")
	b.WriteString("dev tun\n")
	b.WriteString("proto tcp\n")
	b.WriteString(fmt.Sprintf("remote %s %d\n", addr, o.port))
	b.WriteString("resolv-retry infinite\n")
	b.WriteString("nobind\n")
	b.WriteString("persist-key\n")
	b.WriteString("persist-tun\n")
	b.WriteString("remote-cert-tls server\n")
	b.WriteString("auth SHA256\n")
	b.WriteString("cipher AES-256-GCM\n")
	b.WriteString("ncp-ciphers AES-256-GCM:AES-128-GCM:CHACHA20-POLY1305\n")
	b.WriteString("tls-version-min 1.2\n")
	b.WriteString("key-direction 1\n")
	b.WriteString("verb 3\n")
	b.WriteString("\n<ca>\n")
	ca, err := o.pki.ReadCA()
	if err == nil {
		b.WriteString(string(ca))
	}
	b.WriteString("</ca>\n")
	b.WriteString("\n<cert>\n")
	b.WriteString(string(pair.Cert))
	b.WriteString("</cert>\n")
	b.WriteString("\n<key>\n")
	b.WriteString(string(pair.Key))
	b.WriteString("</key>\n")
	tlsKeyPath := filepath.Join(o.pkiDir, "tls-crypt.key")
	if tlsKey, err := os.ReadFile(tlsKeyPath); err == nil {
		b.WriteString("\n<tls-crypt>\n")
		b.WriteString(string(tlsKey))
		b.WriteString("</tls-crypt>\n")
	}
	return b.String()
}

type ClientConfig struct {
	ClientName string `json:"clientName"`
	Config     string `json:"config"`
	Cert       string `json:"cert"`
	Key        string `json:"key"`
}
