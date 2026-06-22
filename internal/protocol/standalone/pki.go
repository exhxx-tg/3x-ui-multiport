package standalone

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const certValidity = 10 * 365 * 24 * time.Hour

type PKIStore struct {
	BaseDir string
}

type CertPair struct {
	Cert []byte
	Key  []byte
}

func NewPKIStore(baseDir string) *PKIStore {
	return &PKIStore{BaseDir: baseDir}
}

func (p *PKIStore) EnsureDir() error {
	return os.MkdirAll(p.BaseDir, 0700)
}

func (p *PKIStore) CAExists() bool {
	_, err := os.Stat(filepath.Join(p.BaseDir, "ca.crt"))
	return err == nil
}

func (p *PKIStore) GenerateCA(commonName string) error {
	if err := p.EnsureDir(); err != nil {
		return err
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate CA key: %w", err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("failed to generate serial: %w", err)
	}
	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: commonName},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(certValidity),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, key.Public(), key)
	if err != nil {
		return fmt.Errorf("failed to create CA cert: %w", err)
	}
	if err := p.writePEM("ca.crt", "CERTIFICATE", der); err != nil {
		return err
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return err
	}
	return p.writePEM("ca.key", "EC PRIVATE KEY", keyDER)
}

func (p *PKIStore) GenerateServerCert() error {
	caCert, caKey, err := p.loadCA()
	if err != nil {
		return fmt.Errorf("failed to load CA: %w", err)
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate server key: %w", err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return err
	}
	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "server"},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(certValidity),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, key.Public(), caKey)
	if err != nil {
		return fmt.Errorf("failed to create server cert: %w", err)
	}
	if err := p.writePEM("server.crt", "CERTIFICATE", der); err != nil {
		return err
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return err
	}
	return p.writePEM("server.key", "EC PRIVATE KEY", keyDER)
}

func (p *PKIStore) GenerateClientCert(clientName string) (*CertPair, error) {
	caCert, caKey, err := p.loadCA()
	if err != nil {
		return nil, fmt.Errorf("failed to load CA: %w", err)
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate client key: %w", err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, err
	}
	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: clientName},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(certValidity),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, key.Public(), caKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create client cert: %w", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	return &CertPair{Cert: certPEM, Key: keyPEM}, nil
}

func (p *PKIStore) WriteClientCert(clientName string, pair *CertPair) error {
	dir := filepath.Join(p.BaseDir, "clients", clientName)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "client.crt"), pair.Cert, 0600); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "client.key"), pair.Key, 0600)
}

func (p *PKIStore) ReadClientCert(clientName string) (*CertPair, error) {
	dir := filepath.Join(p.BaseDir, "clients", clientName)
	cert, err := os.ReadFile(filepath.Join(dir, "client.crt"))
	if err != nil {
		return nil, err
	}
	key, err := os.ReadFile(filepath.Join(dir, "client.key"))
	if err != nil {
		return nil, err
	}
	return &CertPair{Cert: cert, Key: key}, nil
}

func (p *PKIStore) ClientExists(clientName string) bool {
	_, err := os.Stat(filepath.Join(p.BaseDir, "clients", clientName, "client.crt"))
	return err == nil
}

func (p *PKIStore) ListClients() ([]string, error) {
	dir := filepath.Join(p.BaseDir, "clients")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

func (p *PKIStore) RemoveClient(clientName string) error {
	return os.RemoveAll(filepath.Join(p.BaseDir, "clients", clientName))
}

func (p *PKIStore) ReadCA() ([]byte, error) {
	return os.ReadFile(filepath.Join(p.BaseDir, "ca.crt"))
}

func (p *PKIStore) GenerateTLSACryptKey() ([]byte, error) {
	key := make([]byte, 256)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate tls-crypt key: %w", err)
	}
	var b strings.Builder
	b.WriteString("#\n# 2048 bit OpenVPN static key (tls-crypt)\n#\n")
	for i := 0; i < len(key); i += 16 {
		end := i + 16
		if end > len(key) {
			end = len(key)
		}
		fmt.Fprintf(&b, "%x\n", key[i:end])
	}
	return []byte(b.String()), nil
}

func (p *PKIStore) writePEM(filename, blockType string, der []byte) error {
	path := filepath.Join(p.BaseDir, filename)
	return os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der}), 0600)
}

func (p *PKIStore) loadCA() (*x509.Certificate, crypto.Signer, error) {
	certPEM, err := os.ReadFile(filepath.Join(p.BaseDir, "ca.crt"))
	if err != nil {
		return nil, nil, err
	}
	keyPEM, err := os.ReadFile(filepath.Join(p.BaseDir, "ca.key"))
	if err != nil {
		return nil, nil, err
	}
	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return nil, nil, fmt.Errorf("invalid CA cert PEM")
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, nil, fmt.Errorf("invalid CA key PEM")
	}
	key, err := x509.ParseECPrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}
	return cert, key, nil
}
