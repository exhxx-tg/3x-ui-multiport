package standalone

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPKIStore_GenerateCA(t *testing.T) {
	dir := t.TempDir()
	pki := NewPKIStore(dir)

	if err := pki.GenerateCA("test-ca"); err != nil {
		t.Fatalf("GenerateCA failed: %v", err)
	}

	if !pki.CAExists() {
		t.Fatal("CAExists should be true after generation")
	}

	caPath := filepath.Join(dir, "ca.crt")
	keyPath := filepath.Join(dir, "ca.key")

	if _, err := os.Stat(caPath); os.IsNotExist(err) {
		t.Fatal("ca.crt not created")
	}
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Fatal("ca.key not created")
	}
}

func TestPKIStore_GenerateServerCert(t *testing.T) {
	dir := t.TempDir()
	pki := NewPKIStore(dir)

	if err := pki.GenerateCA("test-ca"); err != nil {
		t.Fatalf("GenerateCA failed: %v", err)
	}
	if err := pki.GenerateServerCert(); err != nil {
		t.Fatalf("GenerateServerCert failed: %v", err)
	}

	serverCert := filepath.Join(dir, "server.crt")
	serverKey := filepath.Join(dir, "server.key")

	if _, err := os.Stat(serverCert); os.IsNotExist(err) {
		t.Fatal("server.crt not created")
	}
	if _, err := os.Stat(serverKey); os.IsNotExist(err) {
		t.Fatal("server.key not created")
	}
}

func TestPKIStore_GenerateClientCert(t *testing.T) {
	dir := t.TempDir()
	pki := NewPKIStore(dir)

	if err := pki.GenerateCA("test-ca"); err != nil {
		t.Fatalf("GenerateCA failed: %v", err)
	}
	if err := pki.GenerateServerCert(); err != nil {
		t.Fatalf("GenerateServerCert failed: %v", err)
	}

	pair, err := pki.GenerateClientCert("test-client")
	if err != nil {
		t.Fatalf("GenerateClientCert failed: %v", err)
	}
	if pair == nil {
		t.Fatal("pair should not be nil")
	}
	if len(pair.Cert) == 0 {
		t.Fatal("client cert should not be empty")
	}
	if len(pair.Key) == 0 {
		t.Fatal("client key should not be empty")
	}

	if err := pki.WriteClientCert("test-client", pair); err != nil {
		t.Fatalf("WriteClientCert failed: %v", err)
	}

	if !pki.ClientExists("test-client") {
		t.Fatal("ClientExists should be true after writing")
	}

	loaded, err := pki.ReadClientCert("test-client")
	if err != nil {
		t.Fatalf("ReadClientCert failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("loaded pair should not be nil")
	}
	if string(loaded.Cert) != string(pair.Cert) {
		t.Fatal("loaded cert doesn't match generated cert")
	}
}

func TestPKIStore_ListClients(t *testing.T) {
	dir := t.TempDir()
	pki := NewPKIStore(dir)

	if err := pki.GenerateCA("test-ca"); err != nil {
		t.Fatalf("GenerateCA failed: %v", err)
	}
	if err := pki.GenerateServerCert(); err != nil {
		t.Fatalf("GenerateServerCert failed: %v", err)
	}

	clients, err := pki.ListClients()
	if err != nil {
		t.Fatalf("ListClients failed: %v", err)
	}
	if len(clients) != 0 {
		t.Fatalf("expected 0 clients, got %d", len(clients))
	}

	pair, _ := pki.GenerateClientCert("client-a")
	pki.WriteClientCert("client-a", pair)
	pair2, _ := pki.GenerateClientCert("client-b")
	pki.WriteClientCert("client-b", pair2)

	clients, err = pki.ListClients()
	if err != nil {
		t.Fatalf("ListClients failed: %v", err)
	}
	if len(clients) != 2 {
		t.Fatalf("expected 2 clients, got %d", len(clients))
	}
}

func TestPKIStore_RemoveClient(t *testing.T) {
	dir := t.TempDir()
	pki := NewPKIStore(dir)

	if err := pki.GenerateCA("test-ca"); err != nil {
		t.Fatalf("GenerateCA failed: %v", err)
	}
	if err := pki.GenerateServerCert(); err != nil {
		t.Fatalf("GenerateServerCert failed: %v", err)
	}

	pair, _ := pki.GenerateClientCert("to-remove")
	pki.WriteClientCert("to-remove", pair)

	if !pki.ClientExists("to-remove") {
		t.Fatal("client should exist")
	}

	if err := pki.RemoveClient("to-remove"); err != nil {
		t.Fatalf("RemoveClient failed: %v", err)
	}

	if pki.ClientExists("to-remove") {
		t.Fatal("client should not exist after removal")
	}
}

func TestPKIStore_GenerateTLSACryptKey(t *testing.T) {
	dir := t.TempDir()
	pki := NewPKIStore(dir)

	key, err := pki.GenerateTLSACryptKey()
	if err != nil {
		t.Fatalf("GenerateTLSACryptKey failed: %v", err)
	}
	if len(key) == 0 {
		t.Fatal("key should not be empty")
	}
}
