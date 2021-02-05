package pki

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Convert binary data to PEM format using provided block type.
func toPEM(blockType string, bytes []byte) []byte {
	b := pem.Block{Type: blockType, Bytes: bytes}
	certPEM := pem.EncodeToMemory(&b)
	return certPEM
}

// Generate an ECDSA key and convert it to PEM format.
func genECDSAKey() (*ecdsa.PrivateKey, []byte, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatalf("cannot generate RSA key: %v", err)
		return nil, nil, err
	}

	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		log.Fatalf("Unable to marshal private key: %v", err)
		return nil, nil, err
	}

	pem := toPEM("PRIVATE KEY", privBytes)

	return priv, pem, nil
}

// Create a certificate based on a template using a parent cert, a public key of signee
// and a private parent key. Convert it to PEM format.
func createCert(templateCert, parentCert *x509.Certificate, publicKey *ecdsa.PublicKey, parentPrvKey *ecdsa.PrivateKey) (*x509.Certificate, []byte, error) {
	certBytes, err := x509.CreateCertificate(rand.Reader, templateCert, parentCert, publicKey, parentPrvKey)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to parse certificate")
	}

	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to parse certificate")
	}

	certPEM := toPEM("CERTIFICATE", certBytes)

	return cert, certPEM, nil
}

// Generate a root CA key and a CA certifacte. Return them in PEM format.
func GenCACert(serialNumber int64) (*ecdsa.PrivateKey, []byte, *x509.Certificate, []byte, error) {
	rootTemplate := x509.Certificate{
		SerialNumber: big.NewInt(serialNumber),
		Subject: pkix.Name{
			Country:      []string{"US"},
			Organization: []string{"ISC Stork"},
			CommonName:   "Root CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(30, 0, 0), // 30 years of cert validity
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	privKey, privKeyPEM, err := genECDSAKey()
	if err != nil {
		return nil, nil, nil, nil, errors.Wrapf(err, "problem with generating ECDSA key")
	}
	rootCert, rootPEM, err := createCert(&rootTemplate, &rootTemplate, &privKey.PublicKey, privKey)
	if err != nil {
		return nil, nil, nil, nil, errors.Wrapf(err, "problem with generating certificate")
	}
	return privKey, privKeyPEM, rootCert, rootPEM, nil
}

// Generate a key and a cerficate for provided DNS names and IP addresses, using provided serial number and a CA key and a CA cert.
// Return them in PEM format.
func GenKeyCert(name string, dnsNames []string, ipAddresses []net.IP, serialNumber int64, parentCert *x509.Certificate, parentKey *ecdsa.PrivateKey) ([]byte, []byte, error) {
	// check args
	if len(dnsNames) == 0 {
		return nil, nil, errors.New("DNS names cannot be empty")
	}
	if parentCert == nil {
		return nil, nil, errors.New("parent cert cannot be empty")
	}
	if parentKey == nil {
		return nil, nil, errors.New("parent key cannot be empty")
	}

	// generate a key pair
	privKey, privKeyPEM, err := genECDSAKey()
	if err != nil {
		return nil, nil, err
	}

	// prepare cert template
	template := x509.Certificate{
		SerialNumber: big.NewInt(serialNumber),
		Subject: pkix.Name{
			Country:            []string{"US"},
			Organization:       []string{"ISC Stork"},
			OrganizationalUnit: []string{name},
			CommonName:         dnsNames[0],
		},
		NotBefore:      time.Now(),
		NotAfter:       time.Now().AddDate(30, 0, 0), // 30 years of cert validity
		IsCA:           false,
		MaxPathLenZero: true,
		IPAddresses:    ipAddresses,
		DNSNames:       dnsNames,
	}

	// prepare cert by signing template and public key using parent cert and parent priv key
	_, certPEM, err := createCert(&template, parentCert, &privKey.PublicKey, parentKey)
	if err != nil {
		return nil, nil, err
	}
	return certPEM, privKeyPEM, nil
}

// Generate a CSR (Certificate Signing Request) for provided private key, DNS names and IP addresses.
func GenCSRUsingKey(name string, dnsNames []string, ipAddresses []net.IP, privKeyPEM []byte) ([]byte, [32]byte, error) {
	var fingerprint [32]byte

	if privKeyPEM == nil {
		return nil, fingerprint, errors.New("private key cannot be empty")
	}

	var commonName string
	switch {
	case len(dnsNames) > 0:
		commonName = dnsNames[0]
	case len(ipAddresses) > 0:
		commonName = ipAddresses[0].String()
	default:
		return nil, fingerprint, errors.New("DNS names and IP addresses both cannot be empty")
	}
	// parse priv key
	pemBlock, _ := pem.Decode(privKeyPEM)
	privKeyIf, err := x509.ParsePKCS8PrivateKey(pemBlock.Bytes)
	if err != nil {
		return nil, fingerprint, errors.Wrapf(err, "parsing priv key")
	}
	privKey := privKeyIf.(*ecdsa.PrivateKey)

	// generate a CSR template
	csrTemplate := x509.CertificateRequest{
		Subject: pkix.Name{
			Country:            []string{"US"},
			Organization:       []string{"ISC Stork"},
			OrganizationalUnit: []string{name},
			CommonName:         commonName,
		},
		IPAddresses: ipAddresses,
		DNSNames:    dnsNames,
	}
	// generate the CSR request
	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, &csrTemplate, privKey)
	if err != nil {
		return nil, fingerprint, err
	}
	csrPEM := toPEM("CERTIFICATE REQUEST", csrBytes)

	fingerprint = sha256.Sum256(csrBytes)

	return csrPEM, fingerprint, nil
}

// Generate an ECDSA key and a CSR for it. Return them in PEM format with fingerprint.
func GenKeyAndCSR(name string, dnsNames []string, ipAddresses []net.IP) ([]byte, []byte, [32]byte, error) {
	var fingerprint [32]byte

	// generate a key pair
	_, privKeyPEM, err := genECDSAKey()
	if err != nil {
		return nil, nil, fingerprint, err
	}

	// create CSR using priv key
	csrPEM, fingerprint, err := GenCSRUsingKey(name, dnsNames, ipAddresses, privKeyPEM)
	if err != nil {
		return nil, nil, fingerprint, err
	}

	return privKeyPEM, csrPEM, fingerprint, nil
}

// Parse a certificate in PEM format.
func ParseCert(certPEM []byte) (*x509.Certificate, error) {
	if certPEM == nil {
		return nil, errors.New("cannot parse empty cert PEM")
	}
	pemBlock, _ := pem.Decode(certPEM)
	if pemBlock == nil {
		return nil, errors.New("decoding PEM with cert failed")
	}
	cert, err := x509.ParseCertificate(pemBlock.Bytes)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing cert failed")
	}
	return cert, nil
}

// Sign a cerificate for a given CSR in PEM format using provided serial number, a CA key and a CA cert.
// It returns PEM of signed CSR, fingerprint of signed CSR, parameters error and inner execution error.
func SignCert(csrPEM []byte, serialNumber int64, parentCertPEM []byte, parentKeyPEM []byte) ([]byte, [32]byte, error, error) {
	var fingerprint [32]byte
	// check args
	if parentKeyPEM == nil {
		return nil, fingerprint, errors.New("parent key PEM cannot be empty"), nil
	}
	if parentCertPEM == nil {
		return nil, fingerprint, errors.New("parent cert PEM cannot be empty"), nil
	}
	if csrPEM == nil {
		return nil, fingerprint, errors.New("CSR PEM cannot be empty"), nil
	}

	// parse and check CSR
	pemBlock, _ := pem.Decode(csrPEM)
	if pemBlock == nil {
		return nil, fingerprint, errors.New("decoding PEM with CSR failed"), nil
	}
	csr, err := x509.ParseCertificateRequest(pemBlock.Bytes)
	if err != nil {
		return nil, fingerprint, errors.Wrapf(err, "parsing CSR failed"), nil
	}
	if err = csr.CheckSignature(); err != nil {
		return nil, fingerprint, errors.Wrapf(err, "checking CSR signature failed"), nil
	}

	// parse CA cert and key
	pemBlock, _ = pem.Decode(parentKeyPEM)
	parentKeyIf, err := x509.ParsePKCS8PrivateKey(pemBlock.Bytes)
	if err != nil {
		return nil, fingerprint, nil, errors.Wrapf(err, "parsing CA keys")
	}
	parentKey := parentKeyIf.(*ecdsa.PrivateKey)
	parentCert, err := ParseCert(parentCertPEM)
	if err != nil {
		return nil, fingerprint, nil, errors.Wrapf(err, "parsing CA cert")
	}

	// prepare certificate with information from CSR and from CA cert
	template := x509.Certificate{
		Signature:          csr.Signature,
		SignatureAlgorithm: csr.SignatureAlgorithm,
		PublicKeyAlgorithm: csr.PublicKeyAlgorithm,
		PublicKey:          csr.PublicKey,

		SerialNumber: big.NewInt(serialNumber),
		Issuer:       parentCert.Subject,
		Subject:      csr.Subject,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(30, 0, 0), // 30 years of cert validity
		IPAddresses:  csr.IPAddresses,
		DNSNames:     csr.DNSNames,
	}

	cert, pem, err := createCert(&template, parentCert, csr.PublicKey.(*ecdsa.PublicKey), parentKey)
	if err != nil {
		return nil, fingerprint, nil, errors.Wrapf(err, "signing agent cert failed")
	}
	fingerprint = sha256.Sum256(cert.Raw)
	return pem, fingerprint, nil, nil
}
