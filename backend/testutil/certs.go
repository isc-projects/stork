package testutil

import (
	"crypto/x509"
	"net"
	"os"

	"isc.org/stork/pki"
)

// Creates the certificate, key and CA certificate for testing
// secure database connections.
func CreateTestCerts(sb *Sandbox) (serverCert, serverKey, rootCert string, err error) {
	privateServerKey, _, rootCA, rootCAPEM, err := pki.GenCAKeyCert(42)
	if err != nil {
		return
	}

	clientCertPEM, privateClientKeyPEM, err := pki.GenKeyCert(
		"foo",
		[]string{"foobar"},
		[]net.IP{net.ParseIP("127.0.0.1")},
		24,
		rootCA, privateServerKey,
		x509.ExtKeyUsageServerAuth,
	)
	if err != nil {
		return
	}

	serverCert, err = sb.Write("server-cert.pem", string(clientCertPEM))
	if err != nil {
		return "", "", "", err
	}

	serverKey, err = sb.Write("server-key.pem", string(privateClientKeyPEM))
	if err != nil {
		return "", "", "", err
	}
	err = os.Chmod(serverKey, 0o600)
	if err != nil {
		return "", "", "", err
	}

	rootCert, err = sb.Write("root-cert.pem", string(rootCAPEM))
	if err != nil {
		return "", "", "", err
	}

	return serverCert, serverKey, rootCert, nil
}
