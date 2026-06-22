package service

import (
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
	"time"

	"github.com/exhxx-tg/3x-ui-multiport/internal/database"
	"github.com/exhxx-tg/3x-ui-multiport/internal/database/model"
)

type CertificateService struct{}

func (s *CertificateService) List() ([]model.Certificate, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not available")
	}
	var certs []model.Certificate
	if err := db.Order("id DESC").Find(&certs).Error; err != nil {
		return nil, err
	}
	return certs, nil
}

func (s *CertificateService) Get(id int) (*model.Certificate, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not available")
	}
	var cert model.Certificate
	if err := db.First(&cert, id).Error; err != nil {
		return nil, err
	}
	return &cert, nil
}

func (s *CertificateService) Create(domain, certPEM, keyPEM string, autoRenew bool) (*model.Certificate, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not available")
	}

	certDir := filepath.Join(".", "certs")
	if err := os.MkdirAll(certDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create cert dir: %w", err)
	}

	certFile := filepath.Join(certDir, domain+"-cert.pem")
	keyFile := filepath.Join(certDir, domain+"-key.pem")

	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	if err := os.WriteFile(certFile, []byte(certPEM), 0644); err != nil {
		return nil, fmt.Errorf("failed to write cert file: %w", err)
	}
	if err := os.WriteFile(keyFile, []byte(keyPEM), 0600); err != nil {
		return nil, fmt.Errorf("failed to write key file: %w", err)
	}

	dbCert := &model.Certificate{
		Domain:      domain,
		Issuer:      cert.Issuer.CommonName,
		Fingerprint: fmt.Sprintf("%X", cert.SerialNumber),
		CertFile:    certFile,
		KeyFile:     keyFile,
		NotBefore:   cert.NotBefore.UnixMilli(),
		NotAfter:    cert.NotAfter.UnixMilli(),
		AutoRenew:   autoRenew,
		RenewStatus: "ok",
		Provider:    "manual",
		CreatedAt:   time.Now().UnixMilli(),
		UpdatedAt:   time.Now().UnixMilli(),
	}

	if err := db.Create(dbCert).Error; err != nil {
		return nil, fmt.Errorf("failed to save certificate: %w", err)
	}

	return dbCert, nil
}

func (s *CertificateService) Delete(id int) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not available")
	}

	cert, err := s.Get(id)
	if err != nil {
		return err
	}

	os.Remove(cert.CertFile)
	os.Remove(cert.KeyFile)

	return db.Delete(&model.Certificate{}, id).Error
}

func (s *CertificateService) GenerateSelfSigned(domain string) (*model.Certificate, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not available")
	}

	certDir := filepath.Join(".", "certs")
	if err := os.MkdirAll(certDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create cert dir: %w", err)
	}

	priv, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   domain,
			Organization: []string{"X-UI PRO"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{domain},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, fmt.Errorf("failed to create cert: %w", err)
	}

	certFile := filepath.Join(certDir, domain+"-selfsigned-cert.pem")
	keyFile := filepath.Join(certDir, domain+"-selfsigned-key.pem")

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	privBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes})

	if err := os.WriteFile(certFile, certPEM, 0644); err != nil {
		return nil, fmt.Errorf("failed to write cert: %w", err)
	}
	if err := os.WriteFile(keyFile, keyPEM, 0600); err != nil {
		return nil, fmt.Errorf("failed to write key: %w", err)
	}

	dbCert := &model.Certificate{
		Domain:      domain,
		Issuer:      domain,
		Fingerprint: fmt.Sprintf("%X", serialNumber),
		CertFile:    certFile,
		KeyFile:     keyFile,
		NotBefore:   notBefore.UnixMilli(),
		NotAfter:    notAfter.UnixMilli(),
		AutoRenew:   false,
		RenewStatus: "ok",
		Provider:    "selfsigned",
		CreatedAt:   time.Now().UnixMilli(),
		UpdatedAt:   time.Now().UnixMilli(),
	}

	if err := db.Create(dbCert).Error; err != nil {
		return nil, fmt.Errorf("failed to save certificate record: %w", err)
	}

	return dbCert, nil
}

func (s *CertificateService) SetActive(domain string) error {
	settingSvc := SettingService{}
	cert, err := s.FindByDomain(domain)
	if err != nil {
		return err
	}

	if err := settingSvc.SetCertFile(cert.CertFile); err != nil {
		return err
	}
	return settingSvc.SetKeyFile(cert.KeyFile)
}

func (s *CertificateService) FindByDomain(domain string) (*model.Certificate, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var cert model.Certificate
	if err := db.Where("domain = ?", domain).First(&cert).Error; err != nil {
		return nil, err
	}
	return &cert, nil
}

func (s *CertificateService) CheckExpiry() ([]model.Certificate, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var expiring []model.Certificate
	threshold := time.Now().Add(30 * 24 * time.Hour).UnixMilli()

	if err := db.Where("not_after > 0 AND not_after <= ?", threshold).Find(&expiring).Error; err != nil {
		return nil, err
	}

	return expiring, nil
}
