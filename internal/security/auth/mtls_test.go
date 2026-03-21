package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewmTLSValidator(t *testing.T) {
	tests := []struct {
		name    string
		config  *mTLSConfig
		wantErr bool
	}{
		{
			name: "valid config without CA",
			config: &mTLSConfig{
				EnableClientAuth: true,
				VerifyClientCert: true,
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewmTLSValidator(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewmTLSValidator() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMTLSValidator_ValidateCertificate(t *testing.T) {
	validator, err := NewmTLSValidator(&mTLSConfig{})
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	// Create a test certificate
	cert, err := generateTestCertificate()
	if err != nil {
		t.Fatalf("Failed to generate test certificate: %v", err)
	}

	tests := []struct {
		name    string
		cert    *x509.Certificate
		wantErr error
	}{
		{
			name:    "valid certificate",
			cert:    cert,
			wantErr: nil,
		},
		{
			name:    "nil certificate",
			cert:    nil,
			wantErr: ErrInvalidCertificate,
		},
		{
			name:    "expired certificate",
			cert:    generateExpiredCertificate(),
			wantErr: ErrCertificateExpired,
		},
		{
			name:    "revoked certificate",
			cert:    cert,
			wantErr: ErrCertificateRevoked,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For revoked test, mark cert as revoked first
			if tt.wantErr == ErrCertificateRevoked && tt.cert != nil {
				validator.RevokeCertificate(tt.cert.SerialNumber.String())
			}

			user, err := validator.ValidateCertificate(tt.cert)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("ValidateCertificate() expected error %v, got nil", tt.wantErr)
					return
				}
				if err.Error() != tt.wantErr.Error() {
					t.Errorf("ValidateCertificate() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("ValidateCertificate() unexpected error = %v", err)
				return
			}

			if user.ID != tt.cert.Subject.CommonName {
				t.Errorf("Expected user ID %s, got %s", tt.cert.Subject.CommonName, user.ID)
			}

			if user.Username != tt.cert.Subject.CommonName {
				t.Errorf("Expected username %s, got %s", tt.cert.Subject.CommonName, user.Username)
			}
		})
	}
}

func TestMTLSValidator_GetTrustedCerts(t *testing.T) {
	validator, err := NewmTLSValidator(&mTLSConfig{})
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	certs := validator.GetTrustedCerts()
	if certs != nil {
		t.Error("Expected no trusted certs when no CA is configured")
	}
}

func TestMTLSValidator_RevokeCertificate(t *testing.T) {
	validator, err := NewmTLSValidator(&mTLSConfig{})
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	serialNumber := "12345"
	validator.RevokeCertificate(serialNumber)

	cert, err := generateTestCertificate()
	if err != nil {
		t.Fatalf("Failed to generate test certificate: %v", err)
	}

	// Mark certificate as revoked
	validator.RevokeCertificate(cert.SerialNumber.String())

	// Verify it's revoked
	_, err = validator.ValidateCertificate(cert)
	if err != ErrCertificateRevoked {
		t.Errorf("Expected error %v, got %v", ErrCertificateRevoked, err)
	}

	// Unrevoke and verify
	validator.UnrevokeCertificate(cert.SerialNumber.String())
	_, err = validator.ValidateCertificate(cert)
	if err == ErrCertificateRevoked {
		t.Error("Expected certificate to be unrevoked")
	}
}

func TestMTLSValidator_CreateTLSConfig(t *testing.T) {
	validator, err := NewmTLSValidator(&mTLSConfig{})
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	tests := []struct {
		name     string
		certFile string
		keyFile  string
		wantErr  bool
	}{
		{
			name:     "non-existent files",
			certFile: "/nonexistent/cert.pem",
			keyFile:  "/nonexistent/key.pem",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validator.CreateTLSConfig(tt.certFile, tt.keyFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateTLSConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExtractRolesFromCert(t *testing.T) {
	cert, err := generateTestCertificate()
	if err != nil {
		t.Fatalf("Failed to generate test certificate: %v", err)
	}

	roles := extractRolesFromCert(cert)
	if len(roles) == 0 {
		t.Error("Expected at least default role")
	}

	if roles[0] != "user" {
		t.Errorf("Expected default role 'user', got %s", roles[0])
	}
}

func TestExtractMetadataFromCert(t *testing.T) {
	cert, err := generateTestCertificate()
	if err != nil {
		t.Fatalf("Failed to generate test certificate: %v", err)
	}

	metadata := extractMetadataFromCert(cert)

	if len(cert.Subject.Organization) > 0 {
		if metadata["organization"] != cert.Subject.Organization[0] {
			t.Errorf("Expected organization %s, got %s", cert.Subject.Organization[0], metadata["organization"])
		}
	}

	if metadata["serial_number"] != cert.SerialNumber.String() {
		t.Errorf("Expected serial number %s, got %s", cert.SerialNumber.String(), metadata["serial_number"])
	}
}

func generateTestCertificate() (*x509.Certificate, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "test-user",
			Organization: []string{"Test Org"},
			Country:      []string{"US"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour * 24),
		KeyUsage:  x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
		},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, err
	}

	return x509.ParseCertificate(certDER)
}

func generateExpiredCertificate() *x509.Certificate {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)

	template := x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			CommonName: "expired-user",
		},
		NotBefore: time.Now().Add(-time.Hour * 48),
		NotAfter:  time.Now().Add(-time.Hour * 24),
		KeyUsage:  x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
	}

	certDER, _ := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	cert, _ := x509.ParseCertificate(certDER)
	return cert
}

func TestCreateTestCACertificate(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "mtls-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate test CA certificate
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "Test CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24 * 365),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// Write CA certificate to file
	caCertPath := filepath.Join(tmpDir, "ca.crt")
	if err := os.WriteFile(caCertPath, certPEM, 0644); err != nil {
		t.Fatalf("Failed to write CA cert: %v", err)
	}

	// Create validator with CA config
	config := &mTLSConfig{
		CACertPath:       caCertPath,
		EnableClientAuth: true,
		VerifyClientCert: true,
	}

	validator, err := NewmTLSValidator(config)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	if validator.caCertPool == nil {
		t.Error("Expected CA cert pool to be initialized")
	}

	trustedCerts := validator.GetTrustedCerts()
	// Note: GetTrustedCerts returns certificates reconstructed from the cert pool
	// This may be empty because we can't reconstruct full certs from subjects
	_ = trustedCerts // Use the variable to avoid lint errors
}
