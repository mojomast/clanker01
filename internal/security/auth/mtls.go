package auth

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

var (
	ErrInvalidCertificate = errors.New("invalid certificate")
	ErrUntrustedCA        = errors.New("untrusted certificate authority")
	ErrCertificateExpired = errors.New("certificate expired")
	ErrCertificateRevoked = errors.New("certificate revoked")
)

// mTLSConfig holds mTLS configuration
type mTLSConfig struct {
	CACertPath       string
	EnableClientAuth bool
	VerifyClientCert bool
}

// mTLSValidator implements certificate-based authentication
type mTLSValidator struct {
	config      *mTLSConfig
	caCertPool  *x509.CertPool
	certRevoked map[string]bool
	mu          sync.RWMutex
}

// NewmTLSValidator creates a new mTLS certificate validator
func NewmTLSValidator(config *mTLSConfig) (*mTLSValidator, error) {
	if config == nil {
		return nil, errors.New("mTLS config is required")
	}

	validator := &mTLSValidator{
		config:      config,
		certRevoked: make(map[string]bool),
	}

	if config.CACertPath != "" {
		caCert, err := os.ReadFile(config.CACertPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}

		validator.caCertPool = x509.NewCertPool()
		if !validator.caCertPool.AppendCertsFromPEM(caCert) {
			return nil, errors.New("failed to parse CA certificate")
		}
	}

	return validator, nil
}

// ValidateCertificate validates a client certificate and returns user information
func (m *mTLSValidator) ValidateCertificate(cert *x509.Certificate) (*User, error) {
	if cert == nil {
		return nil, ErrInvalidCertificate
	}

	// Check if certificate is revoked
	m.mu.RLock()
	revoked := m.certRevoked[cert.SerialNumber.String()]
	m.mu.RUnlock()

	if revoked {
		return nil, ErrCertificateRevoked
	}

	// Check certificate expiration
	now := time.Now()
	if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
		return nil, ErrCertificateExpired
	}

	// Extract user information from certificate
	user := &User{
		ID:       cert.Subject.CommonName,
		Username: cert.Subject.CommonName,
		Roles:    extractRolesFromCert(cert),
		Metadata: extractMetadataFromCert(cert),
	}

	// Validate certificate chain if CA pool is configured
	if m.caCertPool != nil {
		opts := x509.VerifyOptions{
			Roots: m.caCertPool,
		}

		if _, err := cert.Verify(opts); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrUntrustedCA, err)
		}
	}

	return user, nil
}

// GetTrustedCerts returns the list of trusted CA certificates.
// TODO: x509.CertPool.Subjects() is deprecated and returns raw subject bytes,
// not DER-encoded certificates. There is no public API on CertPool to enumerate
// the full certificates it contains. To properly support this, the validator
// should maintain its own []*x509.Certificate slice populated at construction time.
func (m *mTLSValidator) GetTrustedCerts() []*x509.Certificate {
	if m.caCertPool == nil {
		return nil
	}

	// Cannot reliably reconstruct certificates from CertPool.Subjects() —
	// those are raw subject bytes, not full DER certificates.
	// Return an empty slice rather than silently returning incorrect data.
	return []*x509.Certificate{}
}

// RevokeCertificate revokes a certificate by its serial number
func (m *mTLSValidator) RevokeCertificate(serialNumber string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.certRevoked[serialNumber] = true
}

// UnrevokeCertificate removes a certificate from the revocation list
func (m *mTLSValidator) UnrevokeCertificate(serialNumber string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.certRevoked, serialNumber)
}

// CreateTLSConfig creates a TLS configuration for mTLS
func (m *mTLSValidator) CreateTLSConfig(certFile, keyFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load server certificate: %w", err)
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	if m.config.EnableClientAuth && m.caCertPool != nil {
		if m.config.VerifyClientCert {
			config.ClientAuth = tls.RequireAndVerifyClientCert
		} else {
			config.ClientAuth = tls.VerifyClientCertIfGiven
		}
		config.ClientCAs = m.caCertPool
	}

	return config, nil
}

// extractRolesFromCert extracts roles from certificate extensions
func extractRolesFromCert(cert *x509.Certificate) []string {
	var roles []string

	for _, ext := range cert.Extensions {
		if ext.Id.Equal([]int{2, 5, 29, 17}) { // Subject Alternative Name
			roles = append(roles, parseSANExtension(ext.Value)...)
		}
	}

	if len(roles) == 0 {
		roles = []string{"user"}
	}

	return roles
}

// extractMetadataFromCert extracts metadata from certificate fields
func extractMetadataFromCert(cert *x509.Certificate) map[string]string {
	metadata := make(map[string]string)

	if cert.Subject.Organization != nil && len(cert.Subject.Organization) > 0 {
		metadata["organization"] = cert.Subject.Organization[0]
	}

	if cert.Subject.OrganizationalUnit != nil && len(cert.Subject.OrganizationalUnit) > 0 {
		metadata["department"] = cert.Subject.OrganizationalUnit[0]
	}

	if len(cert.Subject.Locality) > 0 {
		metadata["locality"] = cert.Subject.Locality[0]
	}

	if len(cert.Subject.Country) > 0 {
		metadata["country"] = cert.Subject.Country[0]
	}

	metadata["serial_number"] = cert.SerialNumber.String()
	metadata["issuer"] = cert.Issuer.CommonName

	return metadata
}

// parseSANExtension parses Subject Alternative Name extension
// TODO: Implement proper ASN.1 decoding of SAN extension (OID 2.5.29.17).
// The value is a DER-encoded ASN.1 SEQUENCE of GeneralName entries.
// Use encoding/asn1.Unmarshal to decode the structure and extract
// dNSName, rFC822Name, uniformResourceIdentifier, etc.
// See RFC 5280 Section 4.2.1.6 for the full specification.
func parseSANExtension(value []byte) []string {
	// Placeholder — real implementation requires ASN.1 decoding
	return nil
}
