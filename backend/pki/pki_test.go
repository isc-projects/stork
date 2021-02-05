package pki

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Check if GenCACert works.
func TestGenCACert(t *testing.T) {
	now := time.Now()

	// generate root key and cert
	privKey, privKeyPEM, rootCert, rootPEM, err := GenCACert(1)
	require.NoError(t, err)
	require.NotEmpty(t, privKey)
	require.NotEmpty(t, privKeyPEM)
	require.NotEmpty(t, rootCert)
	require.NotEmpty(t, rootPEM)

	// check root cert

	// check serial number
	require.EqualValues(t, *big.NewInt(1), *rootCert.SerialNumber)
	// check validity dates
	require.WithinDuration(t, now, rootCert.NotBefore, time.Second*10)
	require.WithinDuration(t, now.AddDate(30, 0, 0), rootCert.NotAfter, time.Second*10)
	require.True(t, rootCert.IsCA)

	// check cert PEM
	pemBlock, _ := pem.Decode(rootPEM)
	rootCert2, err := x509.ParseCertificate(pemBlock.Bytes)
	require.NoError(t, err)
	require.EqualValues(t, rootCert.Raw, rootCert2.Raw)
}

// Check if GenKeyCert works.
func TestGenKeyCert(t *testing.T) {
	// prepare arguments
	name := "name"
	ipAddresses := []net.IP{net.ParseIP("192.0.2.1")}
	dnsNames := []string{"name"}
	serialNumber := int64(1)
	parentKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(serialNumber),
		Subject: pkix.Name{
			Country:            []string{"US"},
			Organization:       []string{"ISC Stork"},
			OrganizationalUnit: []string{name},
			CommonName:         "root ca",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(30, 0, 0), // 30 years of cert validity
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}
	certBytes, err := x509.CreateCertificate(rand.Reader, template, template, &parentKey.PublicKey, parentKey)
	require.NoError(t, err)
	parentCert, err := x509.ParseCertificate(certBytes)
	require.NoError(t, err)

	// empty DNS names
	_, _, err = GenKeyCert(name, nil, ipAddresses, 1, parentCert, parentKey)
	require.EqualError(t, err, "DNS names cannot be empty")

	// empty parent key
	_, _, err = GenKeyCert(name, dnsNames, ipAddresses, 1, parentCert, nil)
	require.EqualError(t, err, "parent key cannot be empty")

	// empty parent cert
	_, _, err = GenKeyCert(name, dnsNames, ipAddresses, 1, nil, parentKey)
	require.EqualError(t, err, "parent cert cannot be empty")

	// it should be ok
	certPEM, privKeyPEM, err := GenKeyCert(name, dnsNames, ipAddresses, 1, parentCert, parentKey)
	require.NoError(t, err)
	require.NotEmpty(t, certPEM)
	require.NotEmpty(t, privKeyPEM)

	// check cert PEM
	pemBlock, _ := pem.Decode(certPEM)
	cert, err := x509.ParseCertificate(pemBlock.Bytes)
	require.NoError(t, err)
	require.EqualValues(t, dnsNames[0], cert.DNSNames[0])
	require.True(t, ipAddresses[0].Equal(cert.IPAddresses[0]))
	require.False(t, cert.IsCA)
	err = cert.CheckSignatureFrom(parentCert)
	require.NoError(t, err)
}

// Check if GenCSRUsingKey works.
func TestGenCSRUsingKey(t *testing.T) {
	// prepare arguments
	name := "name"
	dnsNames := []string{"name"}
	ipAddresses := []net.IP{net.ParseIP("192.0.2.1")}
	parentKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	privBytes, err := x509.MarshalPKCS8PrivateKey(parentKey)
	require.NoError(t, err)
	b := pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}
	privKeyPEM := pem.EncodeToMemory(&b)

	// empty DNS names and IP addresses
	_, _, err = GenCSRUsingKey(name, nil, nil, privKeyPEM)
	require.EqualError(t, err, "DNS names and IP addresses both cannot be empty")

	// empty private key
	_, _, err = GenCSRUsingKey(name, dnsNames, ipAddresses, nil)
	require.EqualError(t, err, "private key cannot be empty")

	// it should be ok
	csrPEM, fingerprint, err := GenCSRUsingKey(name, dnsNames, ipAddresses, privKeyPEM)
	require.NoError(t, err)
	require.NotEmpty(t, csrPEM)
	require.NotEmpty(t, fingerprint)

	// check csr PEM
	pemBlock, _ := pem.Decode(csrPEM)
	csr, err := x509.ParseCertificateRequest(pemBlock.Bytes)
	require.NoError(t, err)
	require.EqualValues(t, dnsNames[0], csr.DNSNames[0])
	require.True(t, ipAddresses[0].Equal(csr.IPAddresses[0]))
}

// Check if GenKeyAndCSR works.
func TestGenKeyAndCSR(t *testing.T) {
	// prepare arguments
	name := "name"
	dnsNames := []string{"name"}
	ipAddresses := []net.IP{net.ParseIP("192.0.2.1")}

	// empty DNS names and IP addresses
	_, _, _, err := GenKeyAndCSR(name, nil, nil) // nolint:dogsled // it does not matter in tests, we just ignore not checked results
	require.EqualError(t, err, "DNS names and IP addresses both cannot be empty")

	// it should be ok
	privKeyPEM, csrPEM, fingerprint, err := GenKeyAndCSR(name, dnsNames, ipAddresses)
	require.NoError(t, err)
	require.NotEmpty(t, privKeyPEM)
	require.NotEmpty(t, csrPEM)
	require.NotEmpty(t, fingerprint)

	// check csr PEM
	pemBlock, _ := pem.Decode(csrPEM)
	csr, err := x509.ParseCertificateRequest(pemBlock.Bytes)
	require.NoError(t, err)
	require.EqualValues(t, dnsNames[0], csr.DNSNames[0])
	require.True(t, ipAddresses[0].Equal(csr.IPAddresses[0]))

	// check private key PEM
	pemBlock, _ = pem.Decode(privKeyPEM)
	privKeyIf, err := x509.ParsePKCS8PrivateKey(pemBlock.Bytes)
	require.NoError(t, err)
	privKey := privKeyIf.(*ecdsa.PrivateKey)
	require.NotNil(t, privKey)
}

// Check if ParseCert works.
func TestParseCert(t *testing.T) {
	var certPEM []byte

	// empty PEM
	_, err := ParseCert(certPEM)
	require.EqualError(t, err, "cannot parse empty cert PEM")

	// wrong form of PEM
	certPEM = []byte("123")
	_, err = ParseCert(certPEM)
	require.EqualError(t, err, "decoding PEM with cert failed")
}

// Check if SignCert works.
func TestSignCert(t *testing.T) {
	// prepare arguments
	serialNumber := int64(2)

	// prepare CA key and cert
	_, parentKeyPEM, _, parentCertPEM, err := GenCACert(1)
	require.NoError(t, err)

	// prepare CSR
	name := "name"
	dnsNames := []string{"name"}
	ipAddresses := []net.IP{net.ParseIP("192.0.2.1")}
	_, csrPEM, _, err := GenKeyAndCSR(name, dnsNames, ipAddresses)
	require.NoError(t, err)

	// empty CSR
	_, _, paramsErr, innerErr := SignCert(nil, serialNumber, parentCertPEM, parentKeyPEM)
	require.EqualError(t, paramsErr, "CSR PEM cannot be empty")
	require.NoError(t, innerErr)

	// empty parentKeyPEM
	_, _, paramsErr, innerErr = SignCert(csrPEM, serialNumber, parentCertPEM, nil)
	require.EqualError(t, paramsErr, "parent key PEM cannot be empty")
	require.NoError(t, innerErr)

	// empty parentCertPEM
	_, _, paramsErr, innerErr = SignCert(csrPEM, serialNumber, nil, parentKeyPEM)
	require.EqualError(t, paramsErr, "parent cert PEM cannot be empty")
	require.NoError(t, innerErr)

	// it should be ok
	certPEM, fingerprint, paramsErr, innerErr := SignCert(csrPEM, serialNumber, parentCertPEM, parentKeyPEM)
	require.NoError(t, paramsErr)
	require.NoError(t, innerErr)
	require.NotEmpty(t, certPEM)
	require.NotEmpty(t, fingerprint)

	// check cert PEM
	pemBlock, _ := pem.Decode(certPEM)
	cert, err := x509.ParseCertificate(pemBlock.Bytes)
	require.NoError(t, err)
	require.False(t, cert.IsCA)
	require.EqualValues(t, dnsNames[0], cert.DNSNames[0])
	require.True(t, ipAddresses[0].Equal(cert.IPAddresses[0]))
}
