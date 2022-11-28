package dbops_test

import (
	"crypto/tls"
	"os"
	"os/user"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	dbops "isc.org/stork/server/database"
	testutil "isc.org/stork/testutil"
)

// Checks if the ~/.postgresql/postgresql.crt file exists in the
// current user's home directory.
func certExistsInHomeDir() bool {
	user, _ := user.Current()

	sslCert := filepath.Join(user.HomeDir, ".postgresql", "postgresql.crt")

	if _, err := os.Stat(sslCert); err == nil {
		return true
	}
	return false
}

// Test that the server ignores cert and key files when the SSL
// mode is set to 'disable'.
func TestGetTLSConfigDisableWithNonBlankFiles(t *testing.T) {
	sb := testutil.NewSandbox()
	defer sb.Close()

	serverCert, serverKey, rootCert, err := testutil.CreateTestCerts(sb)
	require.NoError(t, err)

	tlsConfig, err := dbops.GetTLSConfig("disable", "localhost", serverCert, serverKey, rootCert)
	require.NoError(t, err)
	require.Nil(t, tlsConfig)
}

// Test the require mode without CA certificate. It disables all
// verifications.
func TestGetTLSConfigRequire(t *testing.T) {
	sb := testutil.NewSandbox()
	defer sb.Close()

	serverCert, serverKey, _, err := testutil.CreateTestCerts(sb)
	require.NoError(t, err)

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

	serverCert, serverKey, rootCert, err := testutil.CreateTestCerts(sb)
	require.NoError(t, err)

	tlsConfig, err := dbops.GetTLSConfig("require", "localhost", serverCert, serverKey, rootCert)
	require.NoError(t, err)
	require.NotNil(t, tlsConfig)

	require.True(t, tlsConfig.InsecureSkipVerify)
	require.NotNil(t, tlsConfig.VerifyConnection)
	require.Empty(t, tlsConfig.ServerName)
	require.Len(t, tlsConfig.Certificates, 1)
	require.Equal(t, tls.RenegotiateFreelyAsClient, tlsConfig.Renegotiation)
}

// Test the require mode with blank cert, key and CA cert files locations.
func TestGetTLSConfigRequireCertKeyUnspecified(t *testing.T) {
	// The test doesn't make sense if the certificate file is in
	// the user's home directory because the server will pick
	// this cert for use.
	if certExistsInHomeDir() {
		t.Skipf("Certificate file %s exists in the home dir", "postgresql.crt")
	}

	sb := testutil.NewSandbox()
	defer sb.Close()

	tlsConfig, err := dbops.GetTLSConfig("require", "localhost", "", "", "")
	require.NoError(t, err)
	require.NotNil(t, tlsConfig)

	require.True(t, tlsConfig.InsecureSkipVerify)
	require.Nil(t, tlsConfig.VerifyConnection)
	require.Empty(t, tlsConfig.ServerName)
	require.Empty(t, tlsConfig.Certificates)
	require.Equal(t, tls.RenegotiateFreelyAsClient, tlsConfig.Renegotiation)
}

// Test the require mode with non-existing cert file.
func TestGetTLSConfigRequireCertDoesNotExist(t *testing.T) {
	tlsConfig, err := dbops.GetTLSConfig("require", "localhost", "nonexist", "", "")
	require.Error(t, err)
	require.Nil(t, tlsConfig)
}

// Test the require mode with non-existing key file.
func TestGetTLSConfigRequireKeyDoesNotExist(t *testing.T) {
	sb := testutil.NewSandbox()
	defer sb.Close()

	serverCert, _, _, err := testutil.CreateTestCerts(sb)
	require.NoError(t, err)

	tlsConfig, err := dbops.GetTLSConfig("require", "localhost", serverCert, "nonexist", "")
	require.Error(t, err)
	require.Nil(t, tlsConfig)
}

// Test the verify-ca mode. It should setup the custom verification
// function under tlsConfig.VerifyConnection.
func TestGetTLSConfigVerifyCA(t *testing.T) {
	sb := testutil.NewSandbox()
	defer sb.Close()

	serverCert, serverKey, rootCert, err := testutil.CreateTestCerts(sb)
	require.NoError(t, err)

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

	serverCert, serverKey, rootCert, err := testutil.CreateTestCerts(sb)
	require.NoError(t, err)

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

	serverCert, serverKey, rootCert, err := testutil.CreateTestCerts(sb)
	require.NoError(t, err)
	err = os.Chmod(serverKey, 0o644)
	require.NoError(t, err)

	tlsConfig, err := dbops.GetTLSConfig("verify-ca", "localhost", serverCert, serverKey, rootCert)
	require.Error(t, err)
	require.Nil(t, tlsConfig)
}
