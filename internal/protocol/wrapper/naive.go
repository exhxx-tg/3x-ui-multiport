package wrapper

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/exhxx-tg/3x-ui-multiport/internal/protocol"
	"github.com/exhxx-tg/3x-ui-multiport/internal/servicemanager"
)

func checkPortAvailable(port int) error {
	addr := net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return nil
	}
	conn.Close()
	return fmt.Errorf("port %d is already in use", port)
}

type NaiveProxy struct {
	mu       sync.RWMutex
	id       protocol.ProtocolID
	status   protocol.Status
	port     int
	host     string
	authUser string
	authPass string
	certFile string
	keyFile  string
	sm       *servicemanager.ServiceManager
	proc     *exec.Cmd
	cancel   context.CancelFunc
}

func NewNaiveProxy() *NaiveProxy {
	n := &NaiveProxy{
		id:     protocol.WrapperNaive,
		status: protocol.StatusUnknown,
		port:   8080,
		host:   "0.0.0.0",
		sm:     servicemanager.Global(),
	}
	binPath := n.detectBinary()
	n.sm.Register(servicemanager.ServiceDefinition{
		Name:        "naiveproxy",
		DisplayName: "Naive Proxy (HTTP CONNECT tunnel)",
		BinaryName:  "naive",
		BinaryPath:  binPath,
		ConfigPath:  "/etc/naiveproxy/config.json",
		LogPath:     "/var/log/x-ui/naiveproxy.log",
		DefaultPort: 8080,
		InstallHint: "Install naiveproxy from https://github.com/klzgrad/naiveproxy",
	})
	return n
}

func (n *NaiveProxy) detectBinary() string {
	if runtime.GOOS == "windows" {
		if p, err := exec.LookPath("naive.exe"); err == nil {
			return p
		}
		return ""
	}
	if p, err := exec.LookPath("naive"); err == nil {
		return p
	}
	return "/usr/local/bin/naive"
}

func (n *NaiveProxy) ID() protocol.ProtocolID { return protocol.WrapperNaive }
func (n *NaiveProxy) Info() protocol.ProtocolInfo {
	return *protocol.GetProtocolInfo(protocol.WrapperNaive)
}

func (n *NaiveProxy) Status() protocol.Status {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.status
}

func (n *NaiveProxy) SupportedProtocols() []protocol.ProtocolID {
	return []protocol.ProtocolID{
		protocol.ProtocolVMess,
		protocol.ProtocolVLESS,
		protocol.ProtocolTrojan,
		protocol.ProtocolShadowsocks,
		protocol.ProtocolHysteria,
	}
}

func (n *NaiveProxy) Start() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.status == protocol.StatusRunning {
		return nil
	}

	binPath := n.detectBinary()
	if binPath == "" {
		n.status = protocol.StatusError
		return fmt.Errorf("naiveproxy binary not found; install from github.com/klzgrad/naiveproxy")
	}

	if err := checkPortAvailable(n.port); err != nil {
		n.status = protocol.StatusError
		return err
	}

	configDir := "/etc/naiveproxy"
	if runtime.GOOS == "windows" {
		configDir = filepath.Join(os.Getenv("PROGRAMDATA"), "naiveproxy")
	}
	if err := os.MkdirAll(configDir, 0755); err != nil {
		n.status = protocol.StatusError
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	logFile := "/var/log/x-ui/naiveproxy.log"
	if runtime.GOOS == "windows" {
		logFile = filepath.Join(configDir, "naiveproxy.log")
	}

	ctx, cancel := context.WithCancel(context.Background())
	n.cancel = cancel

	args := []string{
		"--listen", fmt.Sprintf("%s:%d", n.host, n.port),
		"--log", logFile,
	}
	if n.authUser != "" && n.authPass != "" {
		args = append(args, "--auth", fmt.Sprintf("%s:%s", n.authUser, n.authPass))
	}
	if n.certFile != "" && n.keyFile != "" {
		args = append(args, "--cert", n.certFile, "--key", n.keyFile)
	}

	n.proc = exec.CommandContext(ctx, binPath, args...)
	n.proc.Stdout = os.Stdout
	n.proc.Stderr = os.Stderr

	if err := n.proc.Start(); err != nil {
		cancel()
		n.status = protocol.StatusError
		return fmt.Errorf("failed to start naiveproxy: %w", err)
	}

	go func() {
		n.proc.Wait()
		n.mu.Lock()
		if n.status == protocol.StatusRunning {
			n.status = protocol.StatusStopped
		}
		n.mu.Unlock()
	}()

	n.status = protocol.StatusRunning
	return nil
}

func (n *NaiveProxy) Stop() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.status != protocol.StatusRunning {
		return nil
	}

	if n.cancel != nil {
		n.cancel()
	}
	if n.proc != nil && n.proc.Process != nil {
		_ = n.proc.Process.Kill()
	}

	n.status = protocol.StatusStopped
	return nil
}

func (n *NaiveProxy) Restart() error {
	_ = n.Stop()
	time.Sleep(500 * time.Millisecond)
	return n.Start()
}

func (n *NaiveProxy) Config() (any, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return map[string]any{
		"port":     n.port,
		"host":     n.host,
		"authUser": n.authUser,
		"certFile": n.certFile,
		"keyFile":  n.keyFile,
		"status":   n.status,
		"installed": n.binaryExists(),
	}, nil
}

func (n *NaiveProxy) ApplyConfig(cfg any) error {
	c, ok := cfg.(map[string]any)
	if !ok {
		return protocol.ErrInvalidConfig
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if p, ok := c["port"].(int); ok {
		n.port = p
	}
	if h, ok := c["host"].(string); ok {
		n.host = h
	}
	if u, ok := c["authUser"].(string); ok {
		n.authUser = u
	}
	if p, ok := c["authPass"].(string); ok {
		n.authPass = p
	}
	if cf, ok := c["certFile"].(string); ok {
		n.certFile = cf
	}
	if kf, ok := c["keyFile"].(string); ok {
		n.keyFile = kf
	}
	return nil
}

func (n *NaiveProxy) WrapConfig(baseCfg any, wrapperCfg any) (any, error) {
	return map[string]any{
		"type":  string(n.id),
		"base":  baseCfg,
		"proxy": wrapperCfg,
		"auth":  n.authUser != "",
		"tls":   n.certFile != "" && n.keyFile != "",
	}, nil
}

func (n *NaiveProxy) CheckHealth() error {
	addr := net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", n.port))
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return fmt.Errorf("naiveproxy port %d not reachable: %w", n.port, err)
	}
	conn.Close()
	return nil
}

func (n *NaiveProxy) binaryExists() bool {
	if runtime.GOOS == "windows" {
		if _, err := exec.LookPath("naive.exe"); err == nil {
			return true
		}
		_, err := os.Stat(filepath.Join(os.Getenv("PROGRAMDATA"), "naiveproxy", "naive.exe"))
		return err == nil
	}
	if _, err := exec.LookPath("naive"); err == nil {
		return true
	}
	_, err := os.Stat("/usr/local/bin/naive")
	return err == nil
}
