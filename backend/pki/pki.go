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

// The constants used to generate certificates.
const (
	CertValidityYears = 30
	CertCountry       = "US"
	CertOrganization  = "ISC Stork"
)

// Convert binary data to PEM format using provided block type.  This
// function is local and is called by genECDSAKey, createCert and
// GenCSRUsingKey.  They use it to convert binary form of different
// crypto entities like keys or certs to PEM format that is easy to
// transport or store.
func toPEM(blockType string, bytes []byte) []byte {
	b := pem.Block{Type: blockType, Bytes: bytes}
	certPEM := pem.EncodeToMemory(&b)
	return certPEM
}

// Generate an ECDSA key and convert it to PEM format. This function
// is local and is called by GenCAKeyCert, GenKeyCert, GenKeyAndCSR.
// They use it to generate ECDSA key in *ecdsa.PrivateKey and PEM
// formats.
func genECDSAKey() (*ecdsa.PrivateKey, []byte, error) {
	// Generate ECDSA key using P256 curve. ECDSA is used because
	// it is modern and much faster than RSA keys. P256 is used
	// because it is the most popular currently.
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatalf("Cannot generate ECDSA key: %v", err)
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

// Create a certificate based on a template using a parent cert, a
// public key of signee and a private parent key. Convert it to PEM
// format. This can be used generate server traffic certificate
// (typically done once during the first startup) and agent certs
// (typically done once for each agent, during its registration).
// This function is local and is called by GenCAKeyCert, GenKeyCert
// and SignCert. They are using them to generate a certificate in
// *x509.Certificate and PEM formats.
func createCert(templateCert, parentCert *x509.Certificate, publicKey *ecdsa.PublicKey, parentPrvKey *ecdsa.PrivateKey) (*x509.Certificate, []byte, error) {
	certBytes, err := x509.CreateCertificate(rand.Reader, templateCert, parentCert, publicKey, parentPrvKey)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to create certificate")
	}

	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to parse certificate")
	}

	certPEM := toPEM("CERTIFICATE", certBytes)

	return cert, certPEM, nil
}

// Generate a root CA key and a CA certificate. serialNumber is a
// unique number per generated certificate. The function returns
// generated private key as a pointer to ecdsa.Private and as slice of
// bytes in PEM format. Generated certificate is returned as a pointer
// to x509.Certificate and as a slice of bytes in PEM format. It also
// returns an error in case anything goes wrong. This function is
// public and is used in server by certs module by setupRootKeyAndCert
// to prepare CA key and cert.
func GenCAKeyCert(serialNumber int64) (*ecdsa.PrivateKey, []byte, *x509.Certificate, []byte, error) {
	rootTemplate := x509.Certificate{
		SerialNumber: big.NewInt(serialNumber),
		Subject: pkix.Name{
			Country:      []string{CertCountry},
			Organization: []string{CertOrganization},
			CommonName:   "Root CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(CertValidityYears, 0, 0), // 30 years of cert validity
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}

	privKey, privKeyPEM, err := genECDSAKey()
	if err != nil {
		return nil, nil, nil, nil, errors.Wrapf(err, "problem generating ECDSA key")
	}
	rootCert, rootPEM, err := createCert(&rootTemplate, &rootTemplate, &privKey.PublicKey, privKey)
	if err != nil {
		return nil, nil, nil, nil, errors.Wrapf(err, "problem generating certificate")
	}
	return privKey, privKeyPEM, rootCert, rootPEM, nil
}

// Generate a key and a certificate for provided DNS names and IP
// addresses, using provided serial number and a CA key and a CA cert.
// Return them in PEM format. This function is public and is used in
// server by certs module in setupServerKeyAndCert to generate server
// key and cert using CA key and cert.
// It requires to provide the certificate usage: server or client authentication.
func GenKeyCert(name string, dnsNames []string, ipAddresses []net.IP, serialNumber int64, parentCert *x509.Certificate, parentKey *ecdsa.PrivateKey, usage x509.ExtKeyUsage) ([]byte, []byte, error) {
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

	// Check the extended key usage. The application supports only server and
	// client authentication.
	switch usage {
	case x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth:
	default:
		return nil, nil, errors.New("invalid extended key usage, must be server or client authentication")
	}

	// prepare cert template
	template := x509.Certificate{
		SerialNumber: big.NewInt(serialNumber),
		Subject: pkix.Name{
			Country:            []string{CertCountry},
			Organization:       []string{CertOrganization},
			OrganizationalUnit: []string{name},
			CommonName:         dnsNames[0],
		},
		NotBefore:      time.Now(),
		NotAfter:       time.Now().AddDate(CertValidityYears, 0, 0), // 30 years of cert validity
		IsCA:           false,
		MaxPathLenZero: true,
		IPAddresses:    ipAddresses,
		DNSNames:       dnsNames,
		ExtKeyUsage:    []x509.ExtKeyUsage{usage},
	}

	// prepare cert by signing template and public key using parent cert and parent priv key
	_, certPEM, err := createCert(&template, parentCert, &privKey.PublicKey, parentKey)
	if err != nil {
		return nil, nil, err
	}
	return certPEM, privKeyPEM, nil
}

// Generate a CSR (Certificate Signing Request) for provided private
// key, DNS names and IP addresses.  An agent generates CSR with its
// own parameter that will be sent for the server to sign. This
// function is public and is used locally by GenKeyAndCSR function and
// by an agent in register module by generateCerts function to
// generate CSR (using existing agent key) that is sent to server for
// signing.
func GenCSRUsingKey(name string, dnsNames []string, ipAddresses []net.IP, privKeyPEM []byte) ([]byte, [sha256.Size]byte, error) {
	var fingerprint [sha256.Size]byte

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
		return nil, fingerprint, errors.New("both DNS names and IP addresses cannot be empty")
	}
	// parse priv key
	privKey, err := ParsePrivateKey(privKeyPEM)
	if err != nil {
		return nil, fingerprint, err
	}

	// generate a CSR template
	csrTemplate := x509.CertificateRequest{
		Subject: pkix.Name{
			Country:            []string{CertCountry},
			Organization:       []string{CertOrganization},
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

// Generate a private key and return it in PEM format. This function is public.
func GenKey() ([]byte, error) {
	_, privKeyPEM, err := genECDSAKey()
	return privKeyPEM, err
}

// Parse a certificate in PEM format. Return it in *x509.Certificate
// form.  This function is used locally by SignCert and by an agent in
// register module by checkAndStoreCerts function to verify received
// signed cert.
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

// Parse a private key in PEM format. Return it in *ecdsa.PrivateKey
// form.
func ParsePrivateKey(privKeyPEM []byte) (*ecdsa.PrivateKey, error) {
	pemBlock, _ := pem.Decode(privKeyPEM)
	if pemBlock == nil {
		return nil, errors.New("decoding PEM with private key failed")
	}
	privKeyIf, err := x509.ParsePKCS8PrivateKey(pemBlock.Bytes)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing private key")
	}
	return privKeyIf.(*ecdsa.PrivateKey), nil
}

// Sign a certificate for a given CSR in PEM format using provided
// serial number, a CA key and a CA cert.  It returns PEM of signed
// CSR, fingerprint of signed CSR, parameters error and inner
// execution error. This is public function that will be used by the
// server in restservice module by CreateMachine function to sign a
// CSR received from an agent.
func SignCert(csrPEM []byte, serialNumber int64, parentCertPEM []byte, parentKeyPEM []byte) ([]byte, [sha256.Size]byte, error, error) {
	var fingerprint [sha256.Size]byte
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
	parentKey, err := ParsePrivateKey(parentKeyPEM)
	if err != nil {
		return nil, fingerprint, nil, errors.Wrapf(err, "parsing CA keys")
	}
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

		// The application is allowed to sign only server certificates for
		// agents.
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},

		SerialNumber: big.NewInt(serialNumber),
		Issuer:       parentCert.Subject,
		Subject:      csr.Subject,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(CertValidityYears, 0, 0), // 30 years of cert validity
		IPAddresses:  csr.IPAddresses,
		DNSNames:     csr.DNSNames,
	}

	publicKey, ok := csr.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fingerprint, nil, errors.New("unrecognized public key type")
	}

	cert, pem, err := createCert(&template, parentCert, publicKey, parentKey)
	if err != nil {
		return nil, fingerprint, nil, errors.Wrapf(err, "signing agent cert failed")
	}
	fingerprint = CalculateFingerprint(cert)
	return pem, fingerprint, nil, nil
}

// Calculates the fingerprint of the certificate. It is used to ensure all
// components will be calculated in the same way.
func CalculateFingerprint(cert *x509.Certificate) [sha256.Size]byte {
	return sha256.Sum256(cert.Raw)
}

// Calculates the fingerprint of the certificate in PEM format. It is used to
// ensure all components will be calculated in the same way.
func CalculateFingerprintFromPEM(certPEM []byte) ([sha256.Size]byte, error) {
	cert, err := ParseCert(certPEM)
	if err != nil {
		return [sha256.Size]byte{}, errors.Wrapf(err, "parsing cert failed")
	}
	return CalculateFingerprint(cert), nil
}

// Returns true, if the certificate was self-generated by Stork.
func IsInternalCert(cert *x509.Certificate) bool {
	return len(cert.Subject.Organization) == 1 &&
		cert.Subject.Organization[0] == CertOrganization
}

// Returns true, if the certificate was self-signed.
func IsSelfSigned(cert *x509.Certificate) bool {
	return cert.Issuer.String() == cert.Subject.String()
}
