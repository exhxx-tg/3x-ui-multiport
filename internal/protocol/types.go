package protocol

const EcosystemVersion = "1.0.0"

type Category string

const (
	CategoryBase       Category = "base"
	CategoryStandalone Category = "standalone"
	CategoryWrapper    Category = "wrapper"
)

type ProtocolID string

const (
	// Category 1: Base Protocols (Xray-native)
	ProtocolVMess       ProtocolID = "vmess"
	ProtocolVLESS       ProtocolID = "vless"
	ProtocolTrojan      ProtocolID = "trojan"
	ProtocolShadowsocks ProtocolID = "shadowsocks"
	ProtocolHysteria    ProtocolID = "hysteria"

	// Category 2: Standalone Services
	ProtocolOpenVPN   ProtocolID = "openvpn"
	ProtocolWireGuard ProtocolID = "wireguard"
	ProtocolDropbear  ProtocolID = "dropbear"

	// Category 3: Transport Wrappers
	WrapperWebSocket ProtocolID = "websocket"
	WrapperTLS       ProtocolID = "tls"
	WrapperHTTP2     ProtocolID = "http2"
	WrapperGRPC      ProtocolID = "grpc"
	WrapperNaive     ProtocolID = "naive"
)

type ProtocolInfo struct {
	ID          ProtocolID
	Name        string
	Category    Category
	Description string
	Source      string
	XrayNative  bool
}

var AllProtocols = []ProtocolInfo{
	{ProtocolVMess, "VMess", CategoryBase, "Socks5-like proxy with encryption", "github.com/XTLS/Xray-core", true},
	{ProtocolVLESS, "VLESS", CategoryBase, "Lightweight VMess alternative", "github.com/XTLS/Xray-core", true},
	{ProtocolTrojan, "Trojan", CategoryBase, "TLS-based protocol mimicking HTTPS", "github.com/XTLS/Xray-core", true},
	{ProtocolShadowsocks, "Shadowsocks", CategoryBase, "Simple socks5 + stream cipher", "github.com/XTLS/Xray-core", true},
	{ProtocolHysteria, "Hysteria", CategoryBase, "UDP-based speed optimized protocol", "github.com/XTLS/Xray-core", true},
	{ProtocolOpenVPN, "OpenVPN", CategoryStandalone, "Industry-standard VPN protocol", "github.com/OpenVPN/openvpn", false},
	{ProtocolWireGuard, "WireGuard", CategoryStandalone, "Modern kernel-based VPN", "github.com/WireGuard/wireguard-go", false},
	{ProtocolDropbear, "Dropbear", CategoryStandalone, "Lightweight SSH server", "github.com/mkj/dropbear", false},
	{WrapperWebSocket, "WebSocket Wrapper", CategoryWrapper, "HTTP WebSocket tunnel", "Xray-core built-in", true},
	{WrapperTLS, "TLS/HTTPS Wrapper", CategoryWrapper, "TLS encrypted transport", "Xray-core built-in", true},
	{WrapperHTTP2, "HTTP/2 Wrapper", CategoryWrapper, "HTTP/2 multiplexed transport", "Xray-core built-in", true},
	{WrapperGRPC, "gRPC Wrapper", CategoryWrapper, "gRPC protocol wrapping", "Xray-core built-in", true},
	{WrapperNaive, "Naive Wrapper", CategoryWrapper, "HTTP CONNECT tunnel", "github.com/klzgrad/naiveproxy", false},
}

type Status string

const (
	StatusRunning    Status = "running"
	StatusStopped    Status = "stopped"
	StatusError      Status = "error"
	StatusInstalling Status = "installing"
	StatusUnknown    Status = "unknown"
)

func GetProtocolInfo(id ProtocolID) *ProtocolInfo {
	for _, p := range AllProtocols {
		if p.ID == id {
			return &p
		}
	}
	return nil
}

func GetByCategory(cat Category) []ProtocolInfo {
	var result []ProtocolInfo
	for _, p := range AllProtocols {
		if p.Category == cat {
			result = append(result, p)
		}
	}
	return result
}
