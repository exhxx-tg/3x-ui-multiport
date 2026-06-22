package protocol

import (
	"os"
	"testing"
)

func TestIntegrationFullLifecycle(t *testing.T) {
	setupCLITest(t)

	cli := NewCLI()
	if cli == nil {
		t.Fatal("expected non-nil CLI after setup")
	}

	// List all protocols
	err := cli.Run([]string{"list"})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}

	// Start multiple protocols
	for _, id := range []ProtocolID{ProtocolVMess, ProtocolVLESS, ProtocolTrojan, ProtocolShadowsocks} {
		err := cli.Run([]string{"start", string(id)})
		if err != nil {
			t.Fatalf("start %s failed: %v", id, err)
		}
	}

	// Check status of one
	err = cli.Run([]string{"status", "vmess"})
	if err != nil {
		t.Fatalf("status failed: %v", err)
	}

	// Health check
	err = cli.Run([]string{"health", "vmess"})
	if err != nil {
		t.Fatalf("health failed: %v", err)
	}

	// Restart
	err = cli.Run([]string{"restart", "vmess"})
	if err != nil {
		t.Fatalf("restart failed: %v", err)
	}

	// Stop one
	err = cli.Run([]string{"stop", "vmess"})
	if err != nil {
		t.Fatalf("stop failed: %v", err)
	}

	// List again after lifecycle
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = cli.Run([]string{"list"})
	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("list after lifecycle failed: %v", err)
	}

	var buf [8192]byte
	n, _ := r.Read(buf[:])
	output := string(buf[:n])

	if !contains(output, "vmess") || !contains(output, "vless") || !contains(output, "trojan") {
		t.Errorf("list after lifecycle missing protocols: %s", output)
	}
}

func TestIntegrationStandaloneCLI(t *testing.T) {
	setupCLITest(t)
	cli := NewCLI()

	// Test standalone protocol operations
	for _, id := range []ProtocolID{ProtocolOpenVPN, ProtocolWireGuard, ProtocolDropbear} {
		err := cli.Run([]string{"start", string(id)})
		if err != nil {
			t.Fatalf("start %s failed: %v", id, err)
		}

		err = cli.Run([]string{"status", string(id)})
		if err != nil {
			t.Fatalf("status %s failed: %v", id, err)
		}

		err = cli.Run([]string{"stop", string(id)})
		if err != nil {
			t.Fatalf("stop %s failed: %v", id, err)
		}
	}
}

func TestIntegrationWrapperCLI(t *testing.T) {
	setupCLITest(t)
	cli := NewCLI()

	wrappers := []ProtocolID{WrapperWebSocket, WrapperTLS, WrapperHTTP2, WrapperGRPC, WrapperNaive}
	for _, id := range wrappers {
		err := cli.Run([]string{"start", string(id)})
		if err != nil {
			t.Fatalf("start %s failed: %v", id, err)
		}

		err = cli.Run([]string{"status", string(id)})
		if err != nil {
			t.Fatalf("status %s failed: %v", id, err)
		}

		err = cli.Run([]string{"stop", string(id)})
		if err != nil {
			t.Fatalf("stop %s failed: %v", id, err)
		}
	}
}

func TestRunCLICommandEntryPoint(t *testing.T) {
	// This is the main entry point function used by main.go
	// We test it by setting up the global manager first
	setupCLITest(t)

	// Test that RunCLICommand works with various args
	err := RunCLICommand([]string{"list"})
	if err != nil {
		t.Fatalf("RunCLICommand list failed: %v", err)
	}

	err = RunCLICommand([]string{"start", "vmess"})
	if err != nil {
		t.Fatalf("RunCLICommand start failed: %v", err)
	}

	err = RunCLICommand([]string{"stop", "vmess"})
	if err != nil {
		t.Fatalf("RunCLICommand stop failed: %v", err)
	}
}

func TestIntegrationHealthUnhealthy(t *testing.T) {
	setupCLITest(t)
	cli := NewCLI()

	// Protocol starts stopped (unknown status), so health should fail
	err := cli.Run([]string{"health", "openvpn"})
	if err == nil {
		t.Fatal("expected health to fail for stopped protocol")
	}
}

func TestIntegrationRestartAll(t *testing.T) {
	setupCLITest(t)
	cli := NewCLI()

	// Start all, then restart all
	bases := []ProtocolID{ProtocolVMess, ProtocolVLESS, ProtocolTrojan, ProtocolShadowsocks, ProtocolHysteria}
	for _, id := range bases {
		if err := cli.Run([]string{"start", string(id)}); err != nil {
			t.Fatalf("start %s failed: %v", id, err)
		}
	}

	for _, id := range bases {
		if err := cli.Run([]string{"restart", string(id)}); err != nil {
			t.Fatalf("restart %s failed: %v", id, err)
		}
	}
}
