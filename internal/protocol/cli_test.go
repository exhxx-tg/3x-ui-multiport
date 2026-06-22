package protocol

import (
	"os"
	"testing"
)

func setupCLITest(t *testing.T) *CLI {
	t.Helper()
	reg := NewRegistry()
	reg.Register(&mockProtocol{id: ProtocolVMess, cat: CategoryBase})
	reg.Register(&mockProtocol{id: ProtocolVLESS, cat: CategoryBase})
	reg.Register(&mockProtocol{id: ProtocolTrojan, cat: CategoryBase})
	reg.Register(&mockProtocol{id: ProtocolShadowsocks, cat: CategoryBase})
	reg.Register(&mockProtocol{id: ProtocolHysteria, cat: CategoryBase})
	reg.Register(&mockProtocol{id: ProtocolOpenVPN, cat: CategoryStandalone})
	reg.Register(&mockProtocol{id: ProtocolWireGuard, cat: CategoryStandalone})
	reg.Register(&mockProtocol{id: ProtocolDropbear, cat: CategoryStandalone})
	reg.Register(&mockWrapperProtocol{
		id:        WrapperWebSocket,
		status:    StatusStopped,
		supported: []ProtocolID{ProtocolVMess, ProtocolVLESS, ProtocolTrojan, ProtocolShadowsocks},
	})
	reg.Register(&mockWrapperProtocol{
		id:        WrapperTLS,
		status:    StatusStopped,
		supported: []ProtocolID{ProtocolVMess, ProtocolVLESS, ProtocolTrojan, ProtocolShadowsocks},
	})
	reg.Register(&mockWrapperProtocol{
		id:        WrapperHTTP2,
		status:    StatusStopped,
		supported: []ProtocolID{ProtocolVLESS, ProtocolTrojan},
	})
	reg.Register(&mockWrapperProtocol{
		id:        WrapperGRPC,
		status:    StatusStopped,
		supported: []ProtocolID{ProtocolVLESS, ProtocolTrojan},
	})
	reg.Register(&mockWrapperProtocol{
		id:        WrapperNaive,
		status:    StatusStopped,
		supported: []ProtocolID{ProtocolVMess, ProtocolVLESS, ProtocolTrojan, ProtocolShadowsocks, ProtocolHysteria},
	})
	mgr := NewManager(reg)
	globalManager = mgr
	return NewCLI()
}

func TestCLINew(t *testing.T) {
	old := globalManager
	globalManager = nil
	defer func() { globalManager = old }()

	cli := NewCLI()
	if cli != nil {
		t.Fatal("expected nil CLI when not initialized")
	}
}

func TestCLIList(t *testing.T) {
	cli := setupCLITest(t)
	if cli == nil {
		t.Fatal("expected non-nil CLI")
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cli.Run([]string{"list"})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}

	w.Close()
	os.Stdout = old

	var buf [4096]byte
	n, _ := r.Read(buf[:])
	output := string(buf[:n])

	if !contains(output, "ID") || !contains(output, "NAME") || !contains(output, "CATEGORY") {
		t.Errorf("list output missing headers: %s", output)
	}
	if !contains(output, "vmess") || !contains(output, "openvpn") || !contains(output, "websocket") {
		t.Errorf("list output missing known protocols: %s", output)
	}
}

func TestCLIStartStopRestart(t *testing.T) {
	cli := setupCLITest(t)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cli.Run([]string{"start", "vmess"})
	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("start failed: %v", err)
	}

	var buf [4096]byte
	n, _ := r.Read(buf[:])
	output := string(buf[:n])
	if !contains(output, "VMess") || !contains(output, "started successfully") {
		t.Errorf("start output unexpected: %s", output)
	}

	r, w, _ = os.Pipe()
	os.Stdout = w

	err = cli.Run([]string{"stop", "vmess"})
	w.Close()
	os.Stdout = old
	if err != nil {
		t.Fatalf("stop failed: %v", err)
	}

	n, _ = r.Read(buf[:])
	output = string(buf[:n])
	if !contains(output, "stopped successfully") {
		t.Errorf("stop output unexpected: %s", output)
	}
}

func TestCLIStatus(t *testing.T) {
	cli := setupCLITest(t)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cli.Run([]string{"status", "trojan"})
	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("status failed: %v", err)
	}

	var buf [4096]byte
	n, _ := r.Read(buf[:])
	output := string(buf[:n])

	if !contains(output, "trojan") {
		t.Errorf("status output missing protocol name: %s", output)
	}
	if !contains(output, "Protocol:") || !contains(output, "Status:") {
		t.Errorf("status output missing fields: %s", output)
	}
}

func TestCLIHealth(t *testing.T) {
	cli := setupCLITest(t)

	// Start the protocol first so it's healthy
	err := cli.Run([]string{"start", "shadowsocks"})
	if err != nil {
		t.Fatalf("start before health failed: %v", err)
	}

	err = cli.Run([]string{"health", "shadowsocks"})
	if err != nil {
		t.Fatalf("health command failed: %v", err)
	}
}

func TestCLIUnknownCommand(t *testing.T) {
	cli := setupCLITest(t)

	err := cli.Run([]string{"unknown"})
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
}

func TestCLIMissingID(t *testing.T) {
	cli := setupCLITest(t)

	tests := []string{"start", "stop", "restart", "status", "health"}
	for _, cmd := range tests {
		err := cli.Run([]string{cmd})
		if err == nil {
			t.Errorf("expected error for '%s' without id", cmd)
		}
	}
}

func TestCLIUnknownProtocol(t *testing.T) {
	cli := setupCLITest(t)

	err := cli.Run([]string{"start", "nonexistent"})
	if err == nil {
		t.Fatal("expected error for unknown protocol")
	}
}

func TestCLINoArgs(t *testing.T) {
	cli := setupCLITest(t)

	err := cli.Run([]string{})
	if err == nil {
		t.Fatal("expected error for empty args")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
