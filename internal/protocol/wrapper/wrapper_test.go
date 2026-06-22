package wrapper

import (
	"testing"

	"github.com/exhxx-tg/3x-ui-multiport/internal/protocol"
)

func TestNewWebSocketWrapper(t *testing.T) {
	w := NewWebSocketWrapper()
	if w == nil {
		t.Fatal("NewWebSocketWrapper returned nil")
	}
	if w.ID() != protocol.WrapperWebSocket {
		t.Errorf("expected WrapperWebSocket, got %s", w.ID())
	}
	supported := w.SupportedProtocols()
	if len(supported) == 0 {
		t.Error("expected supported protocols")
	}
}

func TestNewTLSWrapper(t *testing.T) {
	w := NewTLSWrapper()
	if w == nil {
		t.Fatal("NewTLSWrapper returned nil")
	}
	if w.ID() != protocol.WrapperTLS {
		t.Errorf("expected WrapperTLS, got %s", w.ID())
	}
}

func TestNewHTTP2Wrapper(t *testing.T) {
	w := NewHTTP2Wrapper()
	if w == nil {
		t.Fatal("NewHTTP2Wrapper returned nil")
	}
	if w.ID() != protocol.WrapperHTTP2 {
		t.Errorf("expected WrapperHTTP2, got %s", w.ID())
	}
}

func TestNewGRPCWrapper(t *testing.T) {
	w := NewGRPCWrapper()
	if w == nil {
		t.Fatal("NewGRPCWrapper returned nil")
	}
	if w.ID() != protocol.WrapperGRPC {
		t.Errorf("expected WrapperGRPC, got %s", w.ID())
	}
}

func TestNewNaiveWrapper(t *testing.T) {
	w := NewNaiveWrapper()
	if w == nil {
		t.Fatal("NewNaiveWrapper returned nil")
	}
	if w.ID() != protocol.WrapperNaive {
		t.Errorf("expected WrapperNaive, got %s", w.ID())
	}
}

func TestWrapperLifecycle(t *testing.T) {
	w := NewWebSocketWrapper()

	if w.Status() != protocol.StatusUnknown {
		t.Errorf("expected initial status unknown, got %s", w.Status())
	}

	if err := w.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if w.Status() != protocol.StatusRunning {
		t.Errorf("expected running status after start, got %s", w.Status())
	}

	if err := w.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	if w.Status() != protocol.StatusStopped {
		t.Errorf("expected stopped status after stop, got %s", w.Status())
	}

	if err := w.Restart(); err != nil {
		t.Fatalf("Restart failed: %v", err)
	}
	if w.Status() != protocol.StatusRunning {
		t.Errorf("expected running status after restart, got %s", w.Status())
	}
}

func TestWrapperInfo(t *testing.T) {
	wrappers := []protocol.TransportWrapper{
		NewWebSocketWrapper(),
		NewTLSWrapper(),
		NewHTTP2Wrapper(),
		NewGRPCWrapper(),
		NewNaiveWrapper(),
	}

	for _, w := range wrappers {
		info := w.Info()
		if info.ID == "" {
			t.Errorf("wrapper %s has empty info", w.ID())
		}
		if info.Category != protocol.CategoryWrapper {
			t.Errorf("expected CategoryWrapper, got %s", info.Category)
		}
	}
}

func TestWrapperConfig(t *testing.T) {
	w := NewWebSocketWrapper()

	cfg, err := w.Config()
	if err != nil {
		t.Fatalf("Config failed: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
}

func TestWrapperApplyConfig(t *testing.T) {
	w := NewWebSocketWrapper()

	err := w.ApplyConfig(map[string]any{
		"port": 9090,
		"path": "/custom",
	})
	if err != nil {
		t.Fatalf("ApplyConfig failed: %v", err)
	}

	cfg, _ := w.Config()
	m, ok := cfg.(map[string]any)
	if !ok {
		t.Fatal("expected map config")
	}
	if m["path"] != "/custom" {
		t.Errorf("expected path /custom, got %v", m["path"])
	}
}

func TestWrapperApplyConfigInvalid(t *testing.T) {
	w := NewWebSocketWrapper()

	err := w.ApplyConfig("invalid")
	if err == nil {
		t.Fatal("expected error for invalid config type")
	}
}

func TestWrapperWrapConfig(t *testing.T) {
	w := NewWebSocketWrapper()

	result, err := w.WrapConfig(map[string]any{"protocol": "vmess"}, map[string]any{"path": "/ws"})
	if err != nil {
		t.Fatalf("WrapConfig failed: %v", err)
	}

	r, ok := result.(map[string]any)
	if !ok {
		t.Fatal("expected map result")
	}
	if r["type"] != string(protocol.WrapperWebSocket) {
		t.Errorf("expected type websocket, got %v", r["type"])
	}
}

func TestWrapperSupportedProtocols(t *testing.T) {
	tests := []struct {
		name     string
		wrapper  func() protocol.TransportWrapper
		expected []protocol.ProtocolID
	}{
		{"WebSocket", NewWebSocketWrapper, []protocol.ProtocolID{protocol.ProtocolVMess, protocol.ProtocolVLESS, protocol.ProtocolTrojan, protocol.ProtocolShadowsocks}},
		{"TLS", NewTLSWrapper, []protocol.ProtocolID{protocol.ProtocolVMess, protocol.ProtocolVLESS, protocol.ProtocolTrojan, protocol.ProtocolShadowsocks}},
		{"HTTP2", NewHTTP2Wrapper, []protocol.ProtocolID{protocol.ProtocolVLESS, protocol.ProtocolTrojan}},
		{"gRPC", NewGRPCWrapper, []protocol.ProtocolID{protocol.ProtocolVLESS, protocol.ProtocolTrojan}},
		{"Naive", NewNaiveWrapper, []protocol.ProtocolID{protocol.ProtocolVMess, protocol.ProtocolVLESS, protocol.ProtocolTrojan, protocol.ProtocolShadowsocks, protocol.ProtocolHysteria}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := tt.wrapper()
			supported := w.SupportedProtocols()

			if len(supported) != len(tt.expected) {
				t.Errorf("expected %d supported protocols, got %d: %v", len(tt.expected), len(supported), supported)
			}

			for _, exp := range tt.expected {
				found := false
				for _, s := range supported {
					if s == exp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("missing expected protocol %s in supported list", exp)
				}
			}
		})
	}
}

func TestWrapperPortDefaults(t *testing.T) {
	tests := []struct {
		name     string
		wrapper  func() protocol.TransportWrapper
		expectedPort int
		expectedConfigHas string
	}{
		{"WebSocket", NewWebSocketWrapper, 80, "path"},
		{"TLS", NewTLSWrapper, 443, "tls"},
		{"HTTP2", NewHTTP2Wrapper, 443, "path"},
		{"gRPC", NewGRPCWrapper, 443, "serviceName"},
		{"Naive", NewNaiveWrapper, 8080, "proxyType"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := tt.wrapper()
			cfg, _ := w.Config()
			m, ok := cfg.(map[string]any)
			if !ok {
				t.Fatal("expected map config")
			}
			if _, exists := m[tt.expectedConfigHas]; !exists {
				t.Errorf("expected config key %q", tt.expectedConfigHas)
			}
		})
	}
}
