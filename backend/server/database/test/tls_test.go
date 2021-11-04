package dbtest

import (
	"crypto/tls"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	dbops "isc.org/stork/server/database"
	testutil "isc.org/stork/testutil"
)

// Creates a certificate, key and the CA certificate for testing
// secure database connections.
func createTestCerts(t *testing.T, sb *testutil.Sandbox) (serverCert, serverKey, rootCert string) {
	const CACERT = `-----BEGIN CERTIFICATE-----
MIIBjDCCATKgAwIBAgIBATAKBggqhkjOPQQDAjAzMQswCQYDVQQGEwJVUzESMBAG
A1UEChMJSVNDIFN0b3JrMRAwDgYDVQQDEwdSb290IENBMCAXDTIxMDkwNjEyMjU0
N1oYDzIwNTEwOTA2MTIyNTQ3WjAzMQswCQYDVQQGEwJVUzESMBAGA1UEChMJSVND
IFN0b3JrMRAwDgYDVQQDEwdSb290IENBMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcD
QgAEJmbefKWfxvdpSnd8+NZVxjObDW4bc/1ANu2TjpP2dsaGXFI+4Jd0HHvdZoHB
hOg2iwdY6i/aJjTftpaDQHwCBKM1MDMwEgYDVR0TAQH/BAgwBgEB/wIBATAdBgNV
HQ4EFgQUU07u+8zyLNobqvJi4rtpsSrayu8wCgYIKoZIzj0EAwIDSAAwRQIhAPAf
YfThoFyxzukrwN16eMP8lX8tVwhyNMZ0aRu3S4vdAiBAcDx0tFt+rWIyFz7eCkeB
fVkdWL4LIJypZP53JBCFYg==
-----END CERTIFICATE-----`

	const SRVKEY = `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgwxFLpLqRpR46bS46
27ukTFCwOcL6I6NNEpfWSE8R+1yhRANCAAQMJcAWsP3nDDZdXYkeZI+D+IFozFbW
HJ/kNaPkCQjuBN2t02BZu6bdr2p5rXcK2mMbxvvjJhSXrBS0/jpsJKZs
-----END PRIVATE KEY-----`

	const SRVCERT = `-----BEGIN CERTIFICATE-----
MIICxDCCAmmgAwIBAgIBAjAKBggqhkjOPQQDAjAzMQswCQYDVQQGEwJVUzESMBAG
A1UEChMJSVNDIFN0b3JrMRAwDgYDVQQDEwdSb290IENBMCAXDTIxMDkwNjEyMjU1
MFoYDzIwNTEwOTA2MTIyNTUwWjBGMQswCQYDVQQGEwJVUzESMBAGA1UEChMJSVND
IFN0b3JrMQ8wDQYDVQQLEwZzZXJ2ZXIxEjAQBgNVBAMTCWxvY2FsaG9zdDBZMBMG
ByqGSM49AgEGCCqGSM49AwEHA0IABAwlwBaw/ecMNl1diR5kj4P4gWjMVtYcn+Q1
o+QJCO4E3a3TYFm7pt2vanmtdwraYxvG++MmFJesFLT+OmwkpmyjggFXMIIBUzAf
BgNVHSMEGDAWgBRTTu77zPIs2huq8mLiu2mxKtrK7zCCAS4GA1UdEQSCASUwggEh
gglsb2NhbGhvc3SCBXR5Y2hvggV0eWNob4IFdHljaG+CDWlwNi1sb2NhbGhvc3SC
BXR5Y2hvggV0eWNob4IFdHljaG+CBXR5Y2hvggV0eWNob4IFdHljaG+CBXR5Y2hv
ggV0eWNob4cEfwAAAYcEwKgBY4cEwKh6AYcErBEAAYcQAAAAAAAAAAAAAAAAAAAA
AYcQIAEEcGOJAAAAAAAAAAAOLYcQ/Qq4KPD/AAAAAAAAAAAOLYcQ/Qq4KPD/AAD0
y69/jhT1zYcQ/Qq4KPD/AAAWRgqhR1EjJIcQIAEEcGOJAAAMjU3ezH+W/ocQIAEE
cGOJAABrU7hrjAdOgIcQ/oAAAAAAAADxjY42pn4t9IcQ/oAAAAAAAAAAQiX//obP
5DAKBggqhkjOPQQDAgNJADBGAiEAywycleZPDX5adSLRCghFA8476nVYmGlkwA7+
hbkkHg8CIQDEfP1HGySpXF5AhAK5RSIxSJTvVhzSSMKtAEmqG2BgYw==
-----END CERTIFICATE-----`

	serverCert, err := sb.Write("server-cert.pem", SRVCERT)
	require.NoError(t, err)

	serverKey, err = sb.Write("server-key.pem", SRVKEY)
	require.NoError(t, err)
	err = os.Chmod(serverKey, 0600)
	require.NoError(t, err)

	rootCert, err = sb.Write("root-cert.pem", CACERT)
	require.NoError(t, err)

	return serverCert, serverKey, rootCert
}

// Test the require mode without CA certificate. It disables all
// verifications.
func TestGetTLSConfigRequire(t *testing.T) {
	sb := testutil.NewSandbox()
	defer sb.Close()

	serverCert, serverKey, _ := createTestCerts(t, sb)

	tlsConfig, err := dbops.GetTLSConfig("require", "localhost", serverCert, serverKey, "")
	require.NoError(t, err)
	require.NotNil(t, tlsConfig)

	require.True(t, tlsConfig.InsecureSkipVerify)
	require.Nil(t, tlsConfig.VerifyConnection)
	require.Empty(t, tlsConfig.ServerName)
	require.Len(t, tlsConfig.Certificates, 1)
	require.Equal(t, tls.RenegotiateFreelyAsClient, tlsConfig.Renegotiation)
}

// Test the require mode when CA certificate is specified. It should
// fall back to the verify-ca mode behavior.
func TestGetTLSConfigRequireVerifyCA(t *testing.T) {
	sb := testutil.NewSandbox()
	defer sb.Close()

	serverCert, serverKey, rootCert := createTestCerts(t, sb)

	tlsConfig, err := dbops.GetTLSConfig("require", "localhost", serverCert, serverKey, rootCert)
	require.NoError(t, err)
	require.NotNil(t, tlsConfig)

	require.True(t, tlsConfig.InsecureSkipVerify)
	require.NotNil(t, tlsConfig.VerifyConnection)
	require.Empty(t, tlsConfig.ServerName)
	require.Len(t, tlsConfig.Certificates, 1)
	require.Equal(t, tls.RenegotiateFreelyAsClient, tlsConfig.Renegotiation)
}

// Test the verify-ca mode. It should setup the custom verification
// function under tlsConfig.VerifyConnection.
func TestGetTLSConfigVerifyCA(t *testing.T) {
	sb := testutil.NewSandbox()
	defer sb.Close()

	serverCert, serverKey, rootCert := createTestCerts(t, sb)

	tlsConfig, err := dbops.GetTLSConfig("verify-ca", "localhost", serverCert, serverKey, rootCert)
	require.NoError(t, err)
	require.NotNil(t, tlsConfig)

	require.True(t, tlsConfig.InsecureSkipVerify)
	require.NotNil(t, tlsConfig.VerifyConnection)
	require.Empty(t, tlsConfig.ServerName)
	require.Len(t, tlsConfig.Certificates, 1)
	require.Equal(t, tls.RenegotiateFreelyAsClient, tlsConfig.Renegotiation)
}

// Test the verify-full mode. It should set the tlsConfig.InsecureSkipVerify
// flag to false.
func TestGetTLSConfigVerifyFull(t *testing.T) {
	sb := testutil.NewSandbox()
	defer sb.Close()

	serverCert, serverKey, rootCert := createTestCerts(t, sb)

	tlsConfig, err := dbops.GetTLSConfig("verify-full", "bull", serverCert, serverKey, rootCert)
	require.NoError(t, err)
	require.NotNil(t, tlsConfig)

	require.False(t, tlsConfig.InsecureSkipVerify)
	require.Nil(t, tlsConfig.VerifyConnection)
	require.Equal(t, "bull", tlsConfig.ServerName)
	require.Len(t, tlsConfig.Certificates, 1)
	require.Equal(t, tls.RenegotiateFreelyAsClient, tlsConfig.Renegotiation)
}

// Test disabling the TLS. There should be no TLS config returned.
func TestGetTLSConfigDisable(t *testing.T) {
	tlsConfig, err := dbops.GetTLSConfig("disable", "localhost", "", "", "")
	require.NoError(t, err)
	require.Nil(t, tlsConfig)
}

// Test that specifying an unsupported mode should result in an error.
func TestGetTLSConfigUnsupportedMode(t *testing.T) {
	sb := testutil.NewSandbox()
	defer sb.Close()

	tlsConfig, err := dbops.GetTLSConfig("unsupported", "localhost", "", "", "")
	require.Error(t, err)
	require.Nil(t, tlsConfig)
}

// Test that the key file is expected to have appropriate permissions.
func TestGetTLSConfigWrongKeyPermissions(t *testing.T) {
	sb := testutil.NewSandbox()
	defer sb.Close()

	serverCert, serverKey, rootCert := createTestCerts(t, sb)
	err := os.Chmod(serverKey, 0644)
	require.NoError(t, err)

	tlsConfig, err := dbops.GetTLSConfig("verify-ca", "localhost", serverCert, serverKey, rootCert)
	require.Error(t, err)
	require.Nil(t, tlsConfig)
}
