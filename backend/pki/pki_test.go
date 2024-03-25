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

// Check if GenCACert function works properly, i.e. returns non-empty
// cert that has its fields set up to reasonable values.
func TestGenCAKeyCert(t *testing.T) {
	now := time.Now()

	// generate root key and cert
	privKey, privKeyPEM, rootCert, rootPEM, err := GenCAKeyCert(1)
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
	require.WithinDuration(t, now.AddDate(CertValidityYears, 0, 0), rootCert.NotAfter, time.Second*10)
	require.True(t, rootCert.IsCA)

	// check cert PEM
	pemBlock, _ := pem.Decode(rootPEM)
	rootCert2, err := x509.ParseCertificate(pemBlock.Bytes)
	require.NoError(t, err)
	require.EqualValues(t, rootCert.Raw, rootCert2.Raw)
}

// Test if GenKeyCert checks arguments passed to it and if returned
// key and cert looks reasonably.
func TestGenKeyCert(t *testing.T) {
	// prepare arguments
	name := "name"
	ipAddresses := []net.IP{net.ParseIP("192.0.2.1")}
	dnsNames := []string{"name"}
	serialNumber := int64(1)
	keyUsage := x509.ExtKeyUsageClientAuth
	parentKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(serialNumber),
		Subject: pkix.Name{
			Country:            []string{CertCountry},
			Organization:       []string{CertOrganization},
			OrganizationalUnit: []string{name},
			CommonName:         "root ca",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(CertValidityYears, 0, 0), // 30 years of cert validity
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}
	certBytes, err := x509.CreateCertificate(rand.Reader, template, template, &parentKey.PublicKey, parentKey)
	require.NoError(t, err)
	parentCert, err := x509.ParseCertificate(certBytes)
	require.NoError(t, err)

	// empty DNS names
	_, _, err = GenKeyCert(name, nil, ipAddresses, 1, parentCert, parentKey, keyUsage)
	require.EqualError(t, err, "DNS names cannot be empty")

	// empty parent key
	_, _, err = GenKeyCert(name, dnsNames, ipAddresses, 1, parentCert, nil, keyUsage)
	require.EqualError(t, err, "parent key cannot be empty")

	// empty parent cert
	_, _, err = GenKeyCert(name, dnsNames, ipAddresses, 1, nil, parentKey, keyUsage)
	require.EqualError(t, err, "parent cert cannot be empty")

	// it should be ok
	certPEM, privKeyPEM, err := GenKeyCert(name, dnsNames, ipAddresses, 1, parentCert, parentKey, keyUsage)
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

// Test if GenCSRUsingKey checks arguments passed to it and if
// returned CSR looks reasonably.
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
	require.EqualError(t, err, "both DNS names and IP addresses cannot be empty")

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

func TestGenKey(t *testing.T) {
	// Arrange & Act
	privKeyPEM, err := GenKey()

	// Assert
	require.NoError(t, err)
	require.NotNil(t, privKeyPEM)
	pemBlock, _ := pem.Decode(privKeyPEM)
	privKeyIf, err := x509.ParsePKCS8PrivateKey(pemBlock.Bytes)
	require.NoError(t, err)
	privKey := privKeyIf.(*ecdsa.PrivateKey)
	require.NotNil(t, privKey)
}

// Test if ParseCert checks arguments passed to it and if returned
// cert looks reasonably.
func TestParseCert(t *testing.T) {
	var certPEM []byte

	// empty PEM
	_, err := ParseCert(certPEM)
	require.EqualError(t, err, "cannot parse empty cert PEM")

	// wrong form of PEM
	certPEM = []byte("123")
	_, err = ParseCert(certPEM)
	require.EqualError(t, err, "decoding PEM with cert failed")

	// should all be ok
	certPEM = []byte(`-----BEGIN CERTIFICATE-----
MIIFFjCCAv6gAwIBAgIBATANBgkqhkiG9w0BAQsFADAzMQswCQYDVQQGEwJVUzES
MBAGA1UEChMJSVNDIFN0b3JrMRAwDgYDVQQDEwdSb290IENBMB4XDTIwMTIwODA4
MDc1M1oXDTMwMTIwODA4MDgwM1owMzELMAkGA1UEBhMCVVMxEjAQBgNVBAoTCUlT
QyBTdG9yazEQMA4GA1UEAxMHUm9vdCBDQTCCAiIwDQYJKoZIhvcNAQEBBQADggIP
ADCCAgoCggIBALgcYkndHQGFmLk8yi8+yetaCBI1cLG/nm+hwjh5C2rh3lqqDziG
qRmcITxkEbCFujbxJzlaXop1MeXwg2YJMky3WM1GWomVKv3jOVR+GkQG70pp0qpt
JmU2CuXoNhwMFA0H22CG8pPRiilUGPI7RLXaLWpA8D+AslfPHR9TG00HbJ86Bi3g
m4/uPiGdcHS6Q+wmKQRsKs6wAKSmlCrvmaKfmVOkxpuKyuKgjmIKoCwY3gYL1T8L
idvVePvbP/Z2SRQOVbSV8eMaYuk+uFwGKq8thLHs8bIEKhrIGlzDss6ZlPotTi2V
I6e6lb06oFuCSfhBaiHPw2sldwYvE/I8MkWUAuWtBgNvVE/e64FgJb1lGIzJpYMj
5jUp9Z13INsXy9zA8nKyZAK4fI6vlQGRg3bERn+S4Q6HXQor9Ma8QWxsqbdiC9dt
pxpzyx11tWg0jEgzCEBfk9IZjlGqyCdX5Z9pshHkQZ9VeK+DG0s6tYEm7BO1ssQD
+qbJS2PJq4Cwe82a6gO+lDz8A+xiXk8dJeTb8hf/c1NY192rqSLewI8oaHOLKEQg
XNSPEEkQqtIqn92Y5oKhLYKmYkwfOgldpj0XQQ3YwUnsOCfy2wRVNRg6VYnbjca8
rSy58t2MfovKWz9UcKhpnXefSdMgR7VhGv0ekDddGIfONn153uyjN/LpAgMBAAGj
NTAzMBIGA1UdEwEB/wQIMAYBAf8CAQEwHQYDVR0OBBYEFILkrDPZAlboeF+nav7C
Rf7nN1W+MA0GCSqGSIb3DQEBCwUAA4ICAQCDfvIgo70Y0Mi+Rs0mF6114z2gGQ7a
7/VnxV9w9uIjuaARq42E2DemFs5e72tPIfT9UWncgs5ZfyO5w2tjRpUOaVCSS5VY
93qzXBfTsqgkrkVRwec4qqZxpNqpsL9u2ZIfsSJ3BJWFV3Zq/3cOrDulfR5bk0G4
hYo/GDyLHjNalBFpetJSIk7l0VOkr2CBUvxKBOP0U1IQGXd+NL/8zW6UB6OitqNL
/tO+JztOpjo6ZYKJGZvxyL/3FUsiHmd8UwqAjnFjQRd3w0gseyqWDgILXQaDXQ5D
vs2oK+HheJv4h6CdrcIdWlWRKoZP3odZyWB0l31kpMbgYC/tMPYebG6mjPx+/S4m
7L+K27zmm2wItUaWI12ky2FPgeW78ALoKDYWmQ+CnpBNE1iFUf4qRzmypu77DmmM
bLgLFj8Bb50j0/zciPO7+D1h6hCPxwXdfQk0tnWBqjImmK3enbkEsw77kF8MkNjr
Hka0EeTt0hyEFKGgJ7jVdbjLFnRzre63q1GuQbLkOibyjf9WS/1ljv1Ps82aWeE+
rh78iXtpm8c/2IqrI37sLbAIs08iPj8ULV57RbcZI7iTYFIjKwPlWL8O2U1mopYP
RXkm1+W4cMzZS14MLfmacBHnI7Z4mRKvc+zEdco/l4omlszafmUXxnCOmqZlhqbm
/p0vFt1oteWWSQ==
-----END CERTIFICATE-----`)
	cert, err := ParseCert(certPEM)
	require.NoError(t, err)
	require.NotNil(t, cert)
	require.NotEmpty(t, *cert.SerialNumber)
	require.EqualValues(t, "ISC Stork", cert.Subject.Organization[0])
}

// Test if SignCert checks arguments passed to it and if returned
// signed cert looks reasonably.
func TestSignCert(t *testing.T) {
	// prepare arguments
	serialNumber := int64(2)

	// prepare CA key and cert
	_, parentKeyPEM, _, parentCertPEM, err := GenCAKeyCert(1)
	require.NoError(t, err)

	// prepare CSR
	name := "name"
	dnsNames := []string{"name"}
	ipAddresses := []net.IP{net.ParseIP("192.0.2.1")}
	privKeyPEM, err := GenKey()
	require.NoError(t, err)
	csrPEM, _, err := GenCSRUsingKey(name, dnsNames, ipAddresses, privKeyPEM)
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
