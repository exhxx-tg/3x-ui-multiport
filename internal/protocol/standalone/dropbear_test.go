package standalone

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDropbear_GenerateHostKeys(t *testing.T) {
	dir := t.TempDir()
	d := NewDropbear()
	d.configPath = dir

	if d.hostKeysExist() {
		t.Fatal("host keys should not exist before generation")
	}

	if err := d.generateHostKeys(); err != nil {
		t.Fatalf("generateHostKeys failed: %v", err)
	}

	if !d.hostKeysExist() {
		t.Fatal("host keys should exist after generation")
	}

	rsaPath := filepath.Join(dir, "dropbear_rsa_host_key")
	if _, err := os.Stat(rsaPath); os.IsNotExist(err) {
		t.Fatal("RSA host key file not created")
	}

	data, err := os.ReadFile(rsaPath)
	if err != nil {
		t.Fatalf("failed to read RSA host key: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("RSA host key should not be empty")
	}
}

func TestDropbear_Info(t *testing.T) {
	d := NewDropbear()
	info := d.Info()
	if info.ID != "dropbear" {
		t.Fatalf("expected dropbear, got %s", info.ID)
	}
	if info.Name != "Dropbear" {
		t.Fatalf("expected Dropbear, got %s", info.Name)
	}
}

func TestDropbear_ServiceName(t *testing.T) {
	d := NewDropbear()
	if name := d.ServiceName(); name != "dropbear" {
		t.Fatalf("expected dropbear, got %s", name)
	}
}

func TestDropbear_Config(t *testing.T) {
	d := NewDropbear()
	cfg, err := d.Config()
	if err != nil {
		t.Fatalf("Config failed: %v", err)
	}
	m, ok := cfg.(map[string]any)
	if !ok {
		t.Fatal("config should be a map")
	}
	if m["port"].(int) != 22 {
		t.Fatalf("expected port 22, got %d", m["port"])
	}
}

func TestDropbear_ApplyConfig(t *testing.T) {
	d := NewDropbear()
	if err := d.ApplyConfig(map[string]any{"port": 2222}); err != nil {
		t.Fatalf("ApplyConfig failed: %v", err)
	}
	if d.port != 2222 {
		t.Fatalf("expected port 2222, got %d", d.port)
	}
}

func TestDropbear_ID(t *testing.T) {
	d := NewDropbear()
	if d.ID() != "dropbear" {
		t.Fatalf("expected dropbear, got %s", d.ID())
	}
}
