package standalone

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestWireGuard_GenerateKeypair(t *testing.T) {
	w := NewWireGuard()

	kp, err := w.GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair failed: %v", err)
	}
	if kp == nil {
		t.Fatal("keypair should not be nil")
	}
	if kp.PrivateKey == "" {
		t.Fatal("private key should not be empty")
	}
	if kp.PublicKey == "" {
		t.Fatal("public key should not be empty")
	}

	enc := base64.StdEncoding.WithPadding(base64.NoPadding)
	if _, err := enc.DecodeString(kp.PrivateKey); err != nil {
		t.Fatalf("private key is not valid base64: %v", err)
	}
	if _, err := enc.DecodeString(kp.PublicKey); err != nil {
		t.Fatalf("public key is not valid base64: %v", err)
	}

	privBytes, _ := enc.DecodeString(kp.PrivateKey)
	if len(privBytes) != 32 {
		t.Fatalf("private key should be 32 bytes, got %d", len(privBytes))
	}
}

func TestWireGuard_GeneratePresharedKey(t *testing.T) {
	w := NewWireGuard()

	psk, err := w.GeneratePresharedKey()
	if err != nil {
		t.Fatalf("GeneratePresharedKey failed: %v", err)
	}
	if psk == "" {
		t.Fatal("preshared key should not be empty")
	}

	enc := base64.StdEncoding.WithPadding(base64.NoPadding)
	keyBytes, err := enc.DecodeString(psk)
	if err != nil {
		t.Fatalf("preshared key is not valid base64: %v", err)
	}
	if len(keyBytes) != 32 {
		t.Fatalf("preshared key should be 32 bytes, got %d", len(keyBytes))
	}
}

func TestWireGuard_GenerateClientConfig(t *testing.T) {
	w := NewWireGuard()
	kp, _ := w.GenerateKeypair()
	w.privateKey = kp.PrivateKey
	w.publicKey = kp.PublicKey

	peer := &WireGuardPeer{
		PublicKey:    "peer-pub-key-test",
		PrivateKey:   "peer-priv-key-test",
		PresharedKey: "psk-test",
		ClientIP:     "10.0.0.2",
		ClientName:   "test-peer",
	}

	cfg := w.GenerateClientConfig(peer, "vpn.example.com")
	if cfg == "" {
		t.Fatal("client config should not be empty")
	}
	if !strings.Contains(cfg, "PrivateKey = peer-priv-key-test") {
		t.Fatal("config should contain peer private key")
	}
	if !strings.Contains(cfg, "Endpoint = vpn.example.com:51820") {
		t.Fatal("config should contain endpoint")
	}
	if !strings.Contains(cfg, "PresharedKey = psk-test") {
		t.Fatal("config should contain preshared key")
	}
}

func TestWireGuard_GenerateConfig(t *testing.T) {
	w := NewWireGuard()
	kp, _ := w.GenerateKeypair()
	w.privateKey = kp.PrivateKey
	w.publicKey = kp.PublicKey

	cfg := w.GenerateConfig(nil)
	if cfg == "" {
		t.Fatal("config should not be empty")
	}
	if !strings.Contains(cfg, "[Interface]") {
		t.Fatal("config should have [Interface] section")
	}
	if !strings.Contains(cfg, "ListenPort") {
		t.Fatal("config should have ListenPort")
	}

	peers := []*WireGuardPeer{
		{
			PublicKey:    "peer1-pub-key",
			AllowedIPs:   "10.0.0.2/32",
			PresharedKey: "psk1",
			ClientName:   "peer1",
			ClientIP:     "10.0.0.2",
		},
		{
			PublicKey:    "peer2-pub-key",
			AllowedIPs:   "10.0.0.3/32",
			ClientName:   "peer2",
			ClientIP:     "10.0.0.3",
		},
	}

	cfg = w.GenerateConfig(peers)
	if !strings.Contains(cfg, "peer1-pub-key") {
		t.Fatal("config should contain peer1 public key")
	}
	if !strings.Contains(cfg, "peer2-pub-key") {
		t.Fatal("config should contain peer2 public key")
	}
	if !strings.Contains(cfg, "psk1") {
		t.Fatal("config should contain preshared key")
	}
}

func TestWireGuard_serverAddress(t *testing.T) {
	w := NewWireGuard()

	addr := w.serverAddress()
	expected := "10.0.0.1/24"
	if addr != expected {
		t.Fatalf("expected %q, got %q", expected, addr)
	}

	w.subnet = "192.168.100.0/24"
	addr = w.serverAddress()
	expected = "192.168.100.1/24"
	if addr != expected {
		t.Fatalf("expected %q, got %q", expected, addr)
	}
}

func TestWireGuard_nextClientIP(t *testing.T) {
	w := NewWireGuard()

	ip, err := w.nextClientIP()
	if err != nil {
		t.Fatalf("nextClientIP failed: %v", err)
	}
	if ip != "10.0.0.2" {
		t.Fatalf("expected first IP to be 10.0.0.2, got %s", ip)
	}
}

func TestWireGuard_SubnetCIDR(t *testing.T) {
	w := NewWireGuard()

	cidr, err := w.SubnetCIDR()
	if err != nil {
		t.Fatalf("SubnetCIDR failed: %v", err)
	}
	if cidr == nil {
		t.Fatal("cidr should not be nil")
	}
	if cidr.String() != "10.0.0.0/24" {
		t.Fatalf("expected 10.0.0.0/24, got %s", cidr.String())
	}
}

func TestWireGuard_KeypairsAreUnique(t *testing.T) {
	w := NewWireGuard()

	seenPriv := make(map[string]bool)
	seenPub := make(map[string]bool)

	for i := 0; i < 10; i++ {
		kp, err := w.GenerateKeypair()
		if err != nil {
			t.Fatalf("iteration %d: %v", i, err)
		}
		if seenPriv[kp.PrivateKey] {
			t.Fatalf("duplicate private key at iteration %d", i)
		}
		if seenPub[kp.PublicKey] {
			t.Fatalf("duplicate public key at iteration %d", i)
		}
		seenPriv[kp.PrivateKey] = true
		seenPub[kp.PublicKey] = true
	}
}
