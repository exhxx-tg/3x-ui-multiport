package standalone

import (
	"strings"
	"testing"
)

func TestOpenVPN_GenerateConfig(t *testing.T) {
	o := NewOpenVPN()

	cfg := o.GenerateConfig("", nil)
	if cfg == "" {
		t.Fatal("config should not be empty")
	}

	checks := []string{
		"port 1194",
		"proto tcp",
		"dev tun",
		"server 10.8.0.0 255.255.255.0",
		"dh none",
		"auth SHA256",
		"cipher AES-256-GCM",
		"tls-version-min 1.2",
		"keepalive 10 120",
		"max-clients 100",
		"redirect-gateway def1 bypass-dhcp",
		"dhcp-option DNS 8.8.8.8",
		"verb 3",
	}
	for _, check := range checks {
		if !strings.Contains(cfg, check) {
			t.Fatalf("config should contain %q", check)
		}
	}
}

func TestOpenVPN_GenerateConfig_CustomPort(t *testing.T) {
	o := NewOpenVPN()
	o.port = 443

	cfg := o.GenerateConfig("", nil)
	if !strings.Contains(cfg, "port 443") {
		t.Fatal("config should have custom port 443")
	}
}

func TestOpenVPN_GenerateClientConfig_NeedsCA(t *testing.T) {
	o := NewOpenVPN()
	dir := t.TempDir()
	o.pkiDir = dir
	o.pki = NewPKIStore(dir)

	if err := o.pki.GenerateCA("test-ca"); err != nil {
		t.Fatalf("GenerateCA failed: %v", err)
	}
	if err := o.pki.GenerateServerCert(); err != nil {
		t.Fatalf("GenerateServerCert failed: %v", err)
	}

	pair, err := o.pki.GenerateClientCert("test-client")
	if err != nil {
		t.Fatalf("GenerateClientCert failed: %v", err)
	}
	if err := o.pki.WriteClientCert("test-client", pair); err != nil {
		t.Fatalf("WriteClientCert failed: %v", err)
	}

	o.serverAddr = "vpn.example.com"
	cfg := o.GenerateClientConfig("test-client", pair)

	checks := []string{
		"client",
		"dev tun",
		"remote vpn.example.com 1194",
		"remote-cert-tls server",
		"cipher AES-256-GCM",
		"<ca>",
		"</ca>",
		"<cert>",
		"</cert>",
		"<key>",
		"</key>",
	}
	for _, check := range checks {
		if !strings.Contains(cfg, check) {
			t.Fatalf("client config should contain %q", check)
		}
	}
}

func TestOpenVPN_AddClient(t *testing.T) {
	o := NewOpenVPN()
	dir := t.TempDir()
	o.pkiDir = dir
	o.pki = NewPKIStore(dir)

	if err := o.pki.GenerateCA("test-ca"); err != nil {
		t.Fatalf("GenerateCA failed: %v", err)
	}
	if err := o.pki.GenerateServerCert(); err != nil {
		t.Fatalf("GenerateServerCert failed: %v", err)
	}

	cc, err := o.AddClient("client-a")
	if err != nil {
		t.Fatalf("AddClient failed: %v", err)
	}
	if cc == nil {
		t.Fatal("client config should not be nil")
	}
	if cc.ClientName != "client-a" {
		t.Fatalf("expected client-a, got %s", cc.ClientName)
	}
	if cc.Config == "" {
		t.Fatal("client config should not be empty")
	}

	if !o.pki.ClientExists("client-a") {
		t.Fatal("client should exist after AddClient")
	}
}

func TestOpenVPN_ListClients(t *testing.T) {
	o := NewOpenVPN()
	dir := t.TempDir()
	o.pkiDir = dir
	o.pki = NewPKIStore(dir)

	if err := o.pki.GenerateCA("test-ca"); err != nil {
		t.Fatalf("GenerateCA failed: %v", err)
	}
	if err := o.pki.GenerateServerCert(); err != nil {
		t.Fatalf("GenerateServerCert failed: %v", err)
	}

	clients, err := o.ListClients()
	if err != nil {
		t.Fatalf("ListClients failed: %v", err)
	}
	if len(clients) != 0 {
		t.Fatalf("expected 0 clients, got %d", len(clients))
	}

	o.AddClient("client-1")
	o.AddClient("client-2")

	clients, err = o.ListClients()
	if err != nil {
		t.Fatalf("ListClients failed: %v", err)
	}
	if len(clients) != 2 {
		t.Fatalf("expected 2 clients, got %d", len(clients))
	}
}

func TestOpenVPN_RemoveClient(t *testing.T) {
	o := NewOpenVPN()
	dir := t.TempDir()
	o.pkiDir = dir
	o.pki = NewPKIStore(dir)

	if err := o.pki.GenerateCA("test-ca"); err != nil {
		t.Fatalf("GenerateCA failed: %v", err)
	}
	if err := o.pki.GenerateServerCert(); err != nil {
		t.Fatalf("GenerateServerCert failed: %v", err)
	}

	o.AddClient("to-remove")
	if err := o.RemoveClient("to-remove"); err != nil {
		t.Fatalf("RemoveClient failed: %v", err)
	}

	clients, _ := o.ListClients()
	if len(clients) != 0 {
		t.Fatal("client should be removed")
	}
}

func TestOpenVPN_GetClientConfig(t *testing.T) {
	o := NewOpenVPN()
	dir := t.TempDir()
	o.pkiDir = dir
	o.pki = NewPKIStore(dir)

	if err := o.pki.GenerateCA("test-ca"); err != nil {
		t.Fatalf("GenerateCA failed: %v", err)
	}
	if err := o.pki.GenerateServerCert(); err != nil {
		t.Fatalf("GenerateServerCert failed: %v", err)
	}

	cc, _ := o.AddClient("get-test")
	o.serverAddr = "example.com"

	loaded, err := o.GetClientConfig("get-test")
	if err != nil {
		t.Fatalf("GetClientConfig failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("loaded config should not be nil")
	}
	if loaded.ClientName != "get-test" {
		t.Fatalf("expected get-test, got %s", loaded.ClientName)
	}
	if cc.Cert != loaded.Cert {
		t.Fatal("loaded cert should match")
	}
}

func TestOpenVPN_Info(t *testing.T) {
	o := NewOpenVPN()
	info := o.Info()
	if info.ID != "openvpn" {
		t.Fatalf("expected openvpn, got %s", info.ID)
	}
	if info.Name != "OpenVPN" {
		t.Fatalf("expected OpenVPN, got %s", info.Name)
	}
}
