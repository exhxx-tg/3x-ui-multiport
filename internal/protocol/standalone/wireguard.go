package standalone

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/exhxx-tg/3x-ui-multiport/internal/protocol"
	"github.com/exhxx-tg/3x-ui-multiport/internal/servicemanager"
	"golang.org/x/crypto/curve25519"
)

type WireGuard struct {
	status        protocol.Status
	port          int
	configPath    string
	interfaceName string
	privateKey    string
	publicKey     string
	subnet        string
	sm            *servicemanager.ServiceManager
}

type WireGuardPeer struct {
	PublicKey    string `json:"publicKey"`
	PrivateKey   string `json:"privateKey,omitempty"`
	PresharedKey string `json:"presharedKey,omitempty"`
	AllowedIPs   string `json:"allowedIPs"`
	ClientIP     string `json:"clientIP"`
	ClientName   string `json:"clientName"`
	Endpoint     string `json:"endpoint,omitempty"`
	DNS          string `json:"dns,omitempty"`
}

type WireGuardKeypair struct {
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
}

type wireGuardPeersFile struct {
	Peers []*WireGuardPeer `json:"peers"`
}

func NewWireGuard() *WireGuard {
	w := &WireGuard{
		status:        protocol.StatusUnknown,
		port:          51820,
		configPath:    "/etc/wireguard/wg0.conf",
		interfaceName: "wg0",
		subnet:        "10.0.0.0/24",
		sm:            servicemanager.Global(),
	}
	binPath := w.detectBinary()
	w.sm.Register(servicemanager.ServiceDefinition{
		Name:        "wg-quick@wg0",
		DisplayName: "WireGuard",
		BinaryName:  "wg",
		BinaryPath:  binPath,
		ConfigPath:  w.configPath,
		LogPath:     "/var/log/x-ui/wireguard.log",
		DefaultPort: 51820,
		InstallHint: "apt-get install wireguard -y",
	})
	return w
}

func (w *WireGuard) detectBinary() string {
	if runtime.GOOS == "windows" {
		if p, err := exec.LookPath("wireguard.exe"); err == nil {
			return p
		}
		return ""
	}
	if p, err := exec.LookPath("wg"); err == nil {
		return p
	}
	return "/usr/bin/wg"
}

func (w *WireGuard) ID() protocol.ProtocolID { return protocol.ProtocolWireGuard }
func (w *WireGuard) Info() protocol.ProtocolInfo {
	return *protocol.GetProtocolInfo(protocol.ProtocolWireGuard)
}
func (w *WireGuard) ServiceName() string { return "wg-quick@wg0" }

func (w *WireGuard) Status() protocol.Status {
	state := w.sm.Status("wg-quick@wg0")
	switch state {
	case servicemanager.StateRunning:
		w.status = protocol.StatusRunning
	case servicemanager.StateStopped:
		w.status = protocol.StatusStopped
	case servicemanager.StateError:
		w.status = protocol.StatusError
	default:
		w.status = protocol.StatusUnknown
	}
	return w.status
}

func (w *WireGuard) IsInstalled() bool {
	return w.sm.IsInstalled("wg-quick@wg0") || w.sm.IsInstalled("wireguard")
}

func (w *WireGuard) Install() error {
	w.status = protocol.StatusInstalling
	if w.IsInstalled() && w.privateKey != "" {
		w.status = protocol.StatusStopped
		return nil
	}
	configDir := filepath.Dir(w.configPath)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		w.status = protocol.StatusError
		return fmt.Errorf("failed to create config dir: %w", err)
	}
	kp, err := w.GenerateKeypair()
	if err != nil {
		w.status = protocol.StatusError
		return fmt.Errorf("failed to generate keys: %w", err)
	}
	w.privateKey = kp.PrivateKey
	w.publicKey = kp.PublicKey
	cfg := w.GenerateConfig(nil)
	if err := os.WriteFile(w.configPath, []byte(cfg), 0600); err != nil {
		w.status = protocol.StatusError
		return fmt.Errorf("failed to write config: %w", err)
	}
	if err := w.sm.Install("wg-quick@wg0"); err != nil {
		l := err.Error()
		if !strings.Contains(l, "systemd not available") && !strings.Contains(l, "wg-quick") {
			w.status = protocol.StatusError
			return fmt.Errorf("failed to install: %w", err)
		}
	}
	w.status = protocol.StatusStopped
	return nil
}

func (w *WireGuard) Uninstall() error {
	_ = w.sm.Uninstall("wg-quick@wg0")
	w.status = protocol.StatusStopped
	return nil
}

func (w *WireGuard) Start() error {
	if w.privateKey == "" {
		kp, err := w.GenerateKeypair()
		if err != nil {
			return fmt.Errorf("failed to generate keys: %w", err)
		}
		w.privateKey = kp.PrivateKey
		w.publicKey = kp.PublicKey
	}
	if err := checkPortAvailable(w.port); err != nil {
		w.status = protocol.StatusError
		return err
	}
	if runtime.GOOS != "windows" {
		if err := exec.Command("wg-quick", "up", w.interfaceName).Run(); err != nil {
			w.status = protocol.StatusError
			return fmt.Errorf("failed to start wireguard: %w", err)
		}
	} else {
		if !w.IsInstalled() {
			return fmt.Errorf("wireguard is not installed")
		}
	}
	w.status = protocol.StatusRunning
	return nil
}

func (w *WireGuard) Stop() error {
	if runtime.GOOS != "windows" {
		_ = exec.Command("wg-quick", "down", w.interfaceName).Run()
	}
	w.status = protocol.StatusStopped
	return nil
}

func (w *WireGuard) Restart() error {
	_ = w.Stop()
	return w.Start()
}

func (w *WireGuard) Config() (any, error) {
	peers, _ := w.ListPeers()
	return map[string]any{
		"port":          w.port,
		"interfaceName": w.interfaceName,
		"configPath":    w.configPath,
		"privateKey":    w.privateKey,
		"publicKey":     w.publicKey,
		"subnet":        w.subnet,
		"installed":     w.IsInstalled(),
		"peers":         peers,
	}, nil
}

func (w *WireGuard) ApplyConfig(cfg any) error {
	c, ok := cfg.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid config type")
	}
	if p, ok := c["port"].(int); ok {
		w.port = p
	}
	if iface, ok := c["interfaceName"].(string); ok && iface != "" {
		w.interfaceName = iface
	}
	if pk, ok := c["privateKey"].(string); ok && pk != "" {
		w.privateKey = pk
	}
	if subnet, ok := c["subnet"].(string); ok && subnet != "" {
		w.subnet = subnet
	}
	return nil
}

func (w *WireGuard) HealthCheck() error {
	if runtime.GOOS == "windows" {
		if w.status != protocol.StatusRunning {
			return fmt.Errorf("wireguard is not running")
		}
		return nil
	}
	cmd := exec.Command("wg", "show", w.interfaceName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("wireguard interface %s is not active: %w", w.interfaceName, err)
	}
	return nil
}

func (w *WireGuard) GenerateKeypair() (*WireGuardKeypair, error) {
	private := make([]byte, 32)
	if _, err := rand.Read(private); err != nil {
		return nil, fmt.Errorf("failed to generate random: %w", err)
	}
	private[0] &= 248
	private[31] &= 127
	private[31] |= 64
	public, err := curve25519.X25519(private, curve25519.Basepoint)
	if err != nil {
		return nil, fmt.Errorf("failed to derive public key: %w", err)
	}
	enc := base64.StdEncoding.WithPadding(base64.NoPadding)
	return &WireGuardKeypair{
		PrivateKey: enc.EncodeToString(private),
		PublicKey:  enc.EncodeToString(public),
	}, nil
}

func (w *WireGuard) GeneratePresharedKey() (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("failed to generate preshared key: %w", err)
	}
	return base64.StdEncoding.WithPadding(base64.NoPadding).EncodeToString(key), nil
}

func (w *WireGuard) AddPeer(clientName string) (*WireGuardPeer, error) {
	kp, err := w.GenerateKeypair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate peer keys: %w", err)
	}
	psk, err := w.GeneratePresharedKey()
	if err != nil {
		return nil, err
	}
	nextIP, err := w.nextClientIP()
	if err != nil {
		return nil, err
	}
	peer := &WireGuardPeer{
		PublicKey:    kp.PublicKey,
		PrivateKey:   kp.PrivateKey,
		PresharedKey: psk,
		AllowedIPs:   nextIP + "/32",
		ClientIP:     nextIP,
		ClientName:   clientName,
	}
	if err := w.savePeer(peer); err != nil {
		return nil, err
	}
	if err := w.syncConfig(); err != nil {
		return nil, err
	}
	return peer, nil
}

func (w *WireGuard) RemovePeer(clientName string) error {
	peers, err := w.loadPeers()
	if err != nil {
		return err
	}
	var filtered []*WireGuardPeer
	for _, p := range peers {
		if p.ClientName != clientName {
			filtered = append(filtered, p)
		}
	}
	if err := w.savePeers(filtered); err != nil {
		return err
	}
	return w.syncConfig()
}

func (w *WireGuard) ListPeers() ([]*WireGuardPeer, error) {
	return w.loadPeers()
}

func (w *WireGuard) GetPeerConfig(clientName string, serverEndpoint string) (string, error) {
	peers, err := w.loadPeers()
	if err != nil {
		return "", err
	}
	for _, p := range peers {
		if p.ClientName == clientName {
			return w.GenerateClientConfig(p, serverEndpoint), nil
		}
	}
	return "", fmt.Errorf("peer %q not found", clientName)
}

func (w *WireGuard) GenerateClientConfig(peer *WireGuardPeer, serverEndpoint string) string {
	if serverEndpoint == "" {
		serverEndpoint = "CHANGE_SERVER_ADDRESS"
	}
	endpoint := fmt.Sprintf("%s:%d", serverEndpoint, w.port)
	var b strings.Builder
	b.WriteString("[Interface]\n")
	b.WriteString(fmt.Sprintf("PrivateKey = %s\n", peer.PrivateKey))
	b.WriteString(fmt.Sprintf("Address = %s\n", peer.ClientIP+"/24"))
	b.WriteString("DNS = 1.1.1.1, 8.8.8.8\n")
	b.WriteString("MTU = 1420\n\n")
	b.WriteString("[Peer]\n")
	b.WriteString(fmt.Sprintf("PublicKey = %s\n", w.publicKey))
	if peer.PresharedKey != "" {
		b.WriteString(fmt.Sprintf("PresharedKey = %s\n", peer.PresharedKey))
	}
	b.WriteString(fmt.Sprintf("Endpoint = %s\n", endpoint))
	b.WriteString("AllowedIPs = 0.0.0.0/0, ::/0\n")
	b.WriteString("PersistentKeepalive = 25\n")
	return b.String()
}

func (w *WireGuard) GenerateConfig(peers []*WireGuardPeer) string {
	if w.privateKey == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString("[Interface]\n")
	b.WriteString(fmt.Sprintf("PrivateKey = %s\n", w.privateKey))
	b.WriteString(fmt.Sprintf("Address = %s\n", w.serverAddress()))
	b.WriteString(fmt.Sprintf("ListenPort = %d\n", w.port))
	b.WriteString("MTU = 1420\n")
	b.WriteString("SaveConfig = false\n\n")
	if peers == nil {
		peers, _ = w.loadPeers()
	}
	for _, peer := range peers {
		if peer.PublicKey == "" {
			continue
		}
		b.WriteString("[Peer]\n")
		b.WriteString(fmt.Sprintf("PublicKey = %s\n", peer.PublicKey))
		if peer.PresharedKey != "" {
			b.WriteString(fmt.Sprintf("PresharedKey = %s\n", peer.PresharedKey))
		}
		b.WriteString(fmt.Sprintf("AllowedIPs = %s\n", peer.AllowedIPs))
		b.WriteString("\n")
	}
	return b.String()
}

func (w *WireGuard) SubnetCIDR() (*net.IPNet, error) {
	_, cidr, err := net.ParseCIDR(w.subnet)
	return cidr, err
}

func (w *WireGuard) serverAddress() string {
	_, cidr, err := net.ParseCIDR(w.subnet)
	if err != nil {
		return "10.0.0.1/24"
	}
	ones, _ := cidr.Mask.Size()
	ip := cidr.IP.To4()
	if ip == nil {
		return "10.0.0.1/24"
	}
	ip[3] = 1
	return fmt.Sprintf("%s/%d", ip.String(), ones)
}

func (w *WireGuard) nextClientIP() (string, error) {
	peers, err := w.loadPeers()
	if err != nil {
		return "", err
	}
	used := make(map[string]bool)
	for _, p := range peers {
		used[p.ClientIP] = true
	}
	_, cidr, err := net.ParseCIDR(w.subnet)
	if err != nil {
		return "", fmt.Errorf("invalid subnet: %w", err)
	}
	ip := cidr.IP.To4()
	if ip == nil {
		return "", fmt.Errorf("invalid subnet (not IPv4)")
	}
	for i := 2; i < 254; i++ {
		candidate := make(net.IP, 4)
		copy(candidate, ip)
		candidate[3] = byte(i)
		addr := candidate.String()
		if !used[addr] {
			return addr, nil
		}
	}
	return "", fmt.Errorf("no available IPs in subnet %s", w.subnet)
}

func (w *WireGuard) peersFilePath() string {
	return filepath.Join(filepath.Dir(w.configPath), "wg0_peers.json")
}

func (w *WireGuard) loadPeers() ([]*WireGuardPeer, error) {
	data, err := os.ReadFile(w.peersFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var store wireGuardPeersFile
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}
	return store.Peers, nil
}

func (w *WireGuard) savePeer(peer *WireGuardPeer) error {
	peers, err := w.loadPeers()
	if err != nil {
		return err
	}
	peers = append(peers, peer)
	return w.savePeers(peers)
}

func (w *WireGuard) savePeers(peers []*WireGuardPeer) error {
	store := wireGuardPeersFile{Peers: peers}
	data, err := json.Marshal(store)
	if err != nil {
		return err
	}
	return os.WriteFile(w.peersFilePath(), data, 0600)
}

func (w *WireGuard) syncConfig() error {
	peers, err := w.loadPeers()
	if err != nil {
		return err
	}
	cfg := w.GenerateConfig(peers)
	if err := os.WriteFile(w.configPath, []byte(cfg), 0600); err != nil {
		return fmt.Errorf("failed to write wireguard config: %w", err)
	}
	if w.status == protocol.StatusRunning && runtime.GOOS != "windows" {
		_ = exec.Command("wg", "syncconf", w.interfaceName, w.configPath).Run()
	}
	return nil
}
