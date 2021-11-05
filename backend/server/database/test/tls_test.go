package dbtest

import (
	"crypto/tls"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	dbops "isc.org/stork/server/database"
	testutil "isc.org/stork/testutil"
)

// Creates the certificate, key and CA certificate for testing
// secure database connections.
func createTestCerts(t *testing.T, sb *testutil.Sandbox) (serverCert, serverKey, rootCert string) {
	serverCert, err := sb.Write("server-cert.pem", string(testutil.GetCertPEMContent()))
	require.NoError(t, err)

	serverKey, err = sb.Write("server-key.pem", string(testutil.GetKeyPEMContent()))
	require.NoError(t, err)
	err = os.Chmod(serverKey, 0600)
	require.NoError(t, err)

	rootCert, err = sb.Write("root-cert.pem", string(testutil.GetCACertPEMContent()))
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
