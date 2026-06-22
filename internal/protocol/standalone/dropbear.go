package standalone

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
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

type Dropbear struct {
	status      protocol.Status
	port        int
	configPath  string
	installPath string
	sm          *servicemanager.ServiceManager
}

type SSHUser struct {
	Username     string `json:"username"`
	Password     string `json:"password,omitempty"`
	PublicKey    string `json:"publicKey,omitempty"`
	Shell       string `json:"shell"`
	HomeDir      string `json:"homeDir"`
}

func NewDropbear() *Dropbear {
	d := &Dropbear{
		status:      protocol.StatusUnknown,
		port:        22,
		configPath:  "/etc/dropbear",
		installPath: "/usr/sbin/dropbear",
		sm:          servicemanager.Global(),
	}
	binPath := d.detectBinary()
	d.sm.Register(servicemanager.ServiceDefinition{
		Name:        "dropbear",
		DisplayName: "Dropbear SSH",
		BinaryName:  "dropbear",
		BinaryPath:  binPath,
		ConfigPath:  d.configPath,
		LogPath:     "/var/log/x-ui/dropbear.log",
		DefaultPort: 22,
		Args:        []string{"-p", fmt.Sprintf("%d", d.port), "-d", d.configPath + "/dropbear_dss_host_key", "-r", d.configPath + "/dropbear_rsa_host_key"},
		InstallHint: "apt-get install dropbear -y",
	})
	return d
}

func (d *Dropbear) detectBinary() string {
	if runtime.GOOS == "windows" {
		return ""
	}
	if _, err := os.Stat(d.installPath); err == nil {
		return d.installPath
	}
	if p, err := exec.LookPath("dropbear"); err == nil {
		return p
	}
	return "/usr/sbin/dropbear"
}

func (d *Dropbear) ID() protocol.ProtocolID { return protocol.ProtocolDropbear }
func (d *Dropbear) Info() protocol.ProtocolInfo {
	return *protocol.GetProtocolInfo(protocol.ProtocolDropbear)
}
func (d *Dropbear) ServiceName() string { return "dropbear" }

func (d *Dropbear) Status() protocol.Status {
	state := d.sm.Status("dropbear")
	switch state {
	case servicemanager.StateRunning:
		d.status = protocol.StatusRunning
	case servicemanager.StateStopped:
		d.status = protocol.StatusStopped
	case servicemanager.StateError:
		d.status = protocol.StatusError
	default:
		d.status = protocol.StatusUnknown
	}
	return d.status
}

func (d *Dropbear) IsInstalled() bool {
	return d.sm.IsInstalled("dropbear")
}

func (d *Dropbear) Install() error {
	d.status = protocol.StatusInstalling
	if d.IsInstalled() && d.hostKeysExist() {
		d.status = protocol.StatusStopped
		return nil
	}
	if err := os.MkdirAll(d.configPath, 0755); err != nil {
		d.status = protocol.StatusError
		return fmt.Errorf("failed to create config dir: %w", err)
	}
	if !d.hostKeysExist() {
		if err := d.generateHostKeys(); err != nil {
			d.status = protocol.StatusError
			return fmt.Errorf("failed to generate host keys: %w", err)
		}
	}
	if err := d.sm.Install("dropbear"); err != nil {
		l := err.Error()
		if !strings.Contains(l, "systemd not available") {
			d.status = protocol.StatusError
			return fmt.Errorf("failed to install: %w", err)
		}
	}
	d.status = protocol.StatusStopped
	return nil
}

func (d *Dropbear) Uninstall() error {
	_ = d.sm.Uninstall("dropbear")
	d.status = protocol.StatusStopped
	return nil
}

func (d *Dropbear) Start() error {
	if !d.IsInstalled() {
		if err := d.Install(); err != nil {
			return fmt.Errorf("dropbear not installed: %w", err)
		}
	}
	if err := checkPortAvailable(d.port); err != nil {
		d.status = protocol.StatusError
		return err
	}
	if err := d.sm.Start("dropbear"); err != nil {
		d.status = protocol.StatusError
		return err
	}
	d.status = protocol.StatusRunning
	return nil
}

func (d *Dropbear) Stop() error {
	if err := d.sm.Stop("dropbear"); err != nil {
		return err
	}
	d.status = protocol.StatusStopped
	return nil
}

func (d *Dropbear) Restart() error {
	_ = d.Stop()
	return d.Start()
}

func (d *Dropbear) Config() (any, error) {
	users, _ := d.ListUsers()
	return map[string]any{
		"port":        d.port,
		"configPath":  d.configPath,
		"installPath": d.installPath,
		"installed":   d.IsInstalled(),
		"hostKeysExist": d.hostKeysExist(),
		"users":       users,
		"pid":         d.sm.PID("dropbear"),
	}, nil
}

func (d *Dropbear) ApplyConfig(cfg any) error {
	c, ok := cfg.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid config type")
	}
	if p, ok := c["port"].(int); ok {
		d.port = p
	}
	return nil
}

func (d *Dropbear) HealthCheck() error {
	state := d.sm.Status("dropbear")
	if state != servicemanager.StateRunning {
		return fmt.Errorf("dropbear is not running")
	}
	if !tcpPortCheck("127.0.0.1", d.port, 5*time.Second) {
		return fmt.Errorf("dropbear port %d is not responding", d.port)
	}
	return nil
}

func (d *Dropbear) hostKeysExist() bool {
	rsaPath := filepath.Join(d.configPath, "dropbear_rsa_host_key")
	_, err := os.Stat(rsaPath)
	return err == nil
}

func (d *Dropbear) generateHostKeys() error {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key: %w", err)
	}

	privDER := x509.MarshalPKCS1PrivateKey(key)
	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privDER,
	})

	rsaKeyPath := filepath.Join(d.configPath, "dropbear_rsa_host_key")
	if err := os.WriteFile(rsaKeyPath, privPEM, 0600); err != nil {
		return fmt.Errorf("failed to write RSA host key: %w", err)
	}

	return nil
}

func (d *Dropbear) AddUser(username, password string) error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("user management not supported on windows")
	}
	if d.userExists(username) {
		return fmt.Errorf("user %q already exists", username)
	}
	homeDir := fmt.Sprintf("/home/%s", username)
	if err := os.MkdirAll(homeDir, 0755); err != nil {
		return fmt.Errorf("failed to create home dir: %w", err)
	}
	sshDir := filepath.Join(homeDir, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("failed to create .ssh dir: %w", err)
	}
	if err := exec.Command("useradd", "-m", "-d", homeDir, "-s", "/bin/bash", username).Run(); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	if password != "" {
		cmd := exec.Command("chpasswd")
		cmd.Stdin = strings.NewReader(fmt.Sprintf("%s:%s", username, password))
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set password: %w", err)
		}
	}
	return nil
}

func (d *Dropbear) RemoveUser(username string) error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("user management not supported on windows")
	}
	_ = exec.Command("userdel", "-r", username).Run()
	return nil
}

func (d *Dropbear) AddPublicKey(username, publicKey string) error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("user management not supported on windows")
	}
	homeDir := fmt.Sprintf("/home/%s", username)
	sshDir := filepath.Join(homeDir, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return err
	}
	authKeys := filepath.Join(sshDir, "authorized_keys")
	f, err := os.OpenFile(authKeys, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString(publicKey + "\n"); err != nil {
		return err
	}
	if err := os.Chown(sshDir, d.uidForUser(username), d.gidForUser(username)); err != nil {
		return err
	}
	if err := os.Chown(authKeys, d.uidForUser(username), d.gidForUser(username)); err != nil {
		return err
	}
	return nil
}

func (d *Dropbear) ListUsers() ([]SSHUser, error) {
	if runtime.GOOS == "windows" {
		return nil, nil
	}
	entries, err := os.ReadDir("/home")
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var users []SSHUser
	for _, e := range entries {
		if e.IsDir() {
			users = append(users, SSHUser{
				Username: e.Name(),
				HomeDir:  "/home/" + e.Name(),
				Shell:    "/bin/bash",
			})
		}
	}
	return users, nil
}

func (d *Dropbear) userExists(username string) bool {
	err := exec.Command("id", "-u", username).Run()
	return err == nil
}

func (d *Dropbear) uidForUser(username string) int {
	out, err := exec.Command("id", "-u", username).Output()
	if err != nil {
		return -1
	}
	var uid int
	fmt.Sscanf(string(out), "%d", &uid)
	return uid
}

func (d *Dropbear) gidForUser(username string) int {
	out, err := exec.Command("id", "-g", username).Output()
	if err != nil {
		return -1
	}
	var gid int
	fmt.Sscanf(string(out), "%d", &gid)
	return gid
}
