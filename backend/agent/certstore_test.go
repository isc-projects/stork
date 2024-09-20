package agent

import (
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/testutil"
	storkutil "isc.org/stork/util"
)

// Test that the default GRPC store is constructed properly.
func TestNewCertStoreDefault(t *testing.T) {
	// Arrange & Act
	store := NewCertStoreDefault()

	// Assert
	require.NotNil(t, store)
	require.Equal(t, KeyPEMFile, store.keyPEMPath)
	require.Equal(t, CertPEMFile, store.certPEMPath)
	require.Equal(t, RootCAFile, store.rootCAPEMPath)
	require.Equal(t, AgentTokenFile, store.agentTokenPath)
	require.Equal(t, ServerCertFingerprintFile, store.serverCertFingerprintPath)
}

// Test that the store reads and parses a proper root CA certificate.
func TestReadRootCA(t *testing.T) {
	// Arrange
	teardown, _ := GenerateSelfSignedCerts()
	defer teardown()
	store := NewCertStoreDefault()

	// Act
	pool, err := store.ReadRootCA()

	// Assert
	require.NoError(t, err)
	require.NotNil(t, pool)
}

// Test that the store reads and parses a proper TLS certificate pair.
func TestReadTLSCert(t *testing.T) {
	// Arrange
	teardown, _ := GenerateSelfSignedCerts()
	defer teardown()
	store := NewCertStoreDefault()

	// Act
	cert, err := store.ReadTLSCert()

	// Assert
	require.NoError(t, err)
	require.NotNil(t, cert)
}

// Test that the store reads and parses a proper agent token.
func TestReadAgentToken(t *testing.T) {
	// Arrange
	teardown, _ := GenerateSelfSignedCerts()
	defer teardown()
	store := NewCertStoreDefault()

	// Act
	token, err := store.ReadToken()

	// Assert
	require.NoError(t, err)
	require.NotNil(t, token)
	require.Equal(t,
		"1800000000000000000000000000000000000000000000000000000000000000",
		token,
	)
}

// Test that the store reads and parses a proper server certificate fingerprint.
func TestReadServerCertFingerprint(t *testing.T) {
	// Arrange
	teardown, _ := GenerateSelfSignedCerts()
	defer teardown()
	store := NewCertStoreDefault()

	// Act
	fingerprint, err := store.ReadServerCertFingerprint()

	// Assert
	require.NoError(t, err)
	require.NotNil(t, fingerprint)
	require.Equal(t, [32]byte{42}, fingerprint)
}

// Test that the performing the key regeneration changes the content of the
// key file and removes other store files.
func TestCreateKey(t *testing.T) {
	// Arrange
	teardown, _ := GenerateSelfSignedCerts()
	defer teardown()
	store := NewCertStoreDefault()

	initialKey, _ := os.ReadFile(KeyPEMFile)

	// Act
	err := store.CreateKey()

	// Assert
	require.NoError(t, err)
	actualHash, _ := os.ReadFile(KeyPEMFile)
	require.NotEqual(t, initialKey, actualHash)
	require.NoFileExists(t, CertPEMFile)
	require.NoFileExists(t, RootCAFile)
	require.NoFileExists(t, AgentTokenFile)
}

// Test that the generating CSR works properly.
func TestGenerateCSR(t *testing.T) {
	// Arrange
	teardown, _ := GenerateSelfSignedCerts()
	defer teardown()
	store := NewCertStoreDefault()

	testCases := []string{"hostname", "10.0.0.1"}
	for _, testCase := range testCases {
		t.Run(testCase, func(t *testing.T) {
			// Act
			csr, fingerprint, err := store.GenerateCSR(testCase)

			// Assert
			require.NoError(t, err)
			require.NotNil(t, csr)
			require.NotNil(t, fingerprint)
		})
	}
}

// Test that the generating CSR fails if the private key is missing.
func TestGenerateCSRForMissingPrivateKey(t *testing.T) {
	// Arrange
	teardown, _ := GenerateSelfSignedCerts()
	defer teardown()
	_ = os.Remove(KeyPEMFile)
	store := NewCertStoreDefault()

	// Act
	csr, _, err := store.GenerateCSR("foobar")

	// Assert
	require.ErrorContains(t, err, "could not read the private key")
	require.ErrorContains(t, err, "no such file or directory")
	require.Nil(t, csr)
}

// Test that the fingerprint is saved properly into the agent token file.
func TestWriteFingerprintAsToken(t *testing.T) {
	// Arrange
	teardown, _ := GenerateSelfSignedCerts()
	defer teardown()
	store := NewCertStoreDefault()

	var fingerprint [32]byte
	for i := 0; i < len(fingerprint); i++ {
		fingerprint[i] = byte(i)
	}
	expectedToken := storkutil.BytesToHex(fingerprint[:])

	// Act
	err := store.WriteFingerprintAsToken(fingerprint)

	// Assert
	require.NoError(t, err)
	actualTokenRaw, err := os.ReadFile(AgentTokenFile)
	require.NoError(t, err)
	require.Equal(t, expectedToken, string(actualTokenRaw))
}

// Test that the root CA file in the PEM format is saved properly.
func TestWriteRootCAPEM(t *testing.T) {
	// Arrange
	teardown, _ := GenerateSelfSignedCerts()
	defer teardown()
	store := NewCertStoreDefault()
	pem, _ := os.ReadFile(RootCAFile)

	// Act
	err := store.WriteRootCAPEM(pem)

	// Assert
	require.NoError(t, err)
}

// Test that the invalid root CA file cannot be saved.
func TestWriteRootCAPEMInvalid(t *testing.T) {
	// Arrange
	teardown, _ := GenerateSelfSignedCerts()
	defer teardown()
	store := NewCertStoreDefault()

	// Act
	err := store.WriteRootCAPEM([]byte("invalid"))

	// Assert
	require.ErrorContains(t, err, "content is invalid")
}

// Test that the certificate file in the PEM format is saved properly.
func TestWriteCertPEM(t *testing.T) {
	// Arrange
	teardown, _ := GenerateSelfSignedCerts()
	defer teardown()
	store := NewCertStoreDefault()
	pem, _ := os.ReadFile(CertPEMFile)

	// Act
	err := store.WriteRootCAPEM(pem)

	// Assert
	require.NoError(t, err)
}

// Test that the invalid certificate file cannot be saved.
func TestWriteCertPEMInvalid(t *testing.T) {
	// Arrange
	teardown, _ := GenerateSelfSignedCerts()
	defer teardown()
	store := NewCertStoreDefault()

	// Act
	err := store.WriteCertPEM([]byte("invalid"))

	// Assert
	require.ErrorContains(t, err, "content is invalid")
}

// Test that the server certificate fingerprint is saved properly.
func TestWriteServerCertFingerprint(t *testing.T) {
	// Arrange
	teardown, _ := GenerateSelfSignedCerts()
	defer teardown()
	store := NewCertStoreDefault()
	fingerprint := [32]byte{42, 42, 42}

	// Act
	err := store.WriteServerCertFingerprint(fingerprint)

	// Assert
	require.NoError(t, err)
	savedFingerprint, _ := store.ReadServerCertFingerprint()
	require.Equal(t, fingerprint, savedFingerprint)
}

// Test that the cert store is recognized as valid if all certificate files
// are valid.
func TestCertStoreIsValid(t *testing.T) {
	// Arrange
	teardown, _ := GenerateSelfSignedCerts()
	defer teardown()
	store := NewCertStoreDefault()

	// Act
	err := store.IsValid()

	// Assert
	require.NoError(t, err)
}

// Test that the cert store is not recognized as valid if any cert file is missing.
func TestCertStoreIsNotValidForMissingFiles(t *testing.T) {
	// Arrange
	pathGetters := map[string]func(s *CertStore) string{
		"key":                func(s *CertStore) string { return s.keyPEMPath },
		"cert":               func(s *CertStore) string { return s.certPEMPath },
		"root CA":            func(s *CertStore) string { return s.rootCAPEMPath },
		"token":              func(s *CertStore) string { return s.agentTokenPath },
		"server fingerprint": func(s *CertStore) string { return s.serverCertFingerprintPath },
	}
	for label, pathGetter := range pathGetters {
		t.Run(label, func(t *testing.T) {
			teardown, _ := GenerateSelfSignedCerts()
			defer teardown()
			store := NewCertStoreDefault()
			_ = os.Remove(pathGetter(store))

			// Act
			err := store.IsValid()

			// Assert
			require.ErrorContains(t, err, "store is not valid")
		})
	}
}

// Test that the cert store is recognized as empty if all cert files don't
// exist.
func TestCertStoreIsEmpty(t *testing.T) {
	// Arrange
	restore := RememberPaths()
	defer restore()
	sb := testutil.NewSandbox()
	defer sb.Close()

	KeyPEMFile = path.Join(sb.BasePath, "key-not-exists.pem")
	RootCAFile = path.Join(sb.BasePath, "root-ca-not-exists.pem")
	CertPEMFile = path.Join(sb.BasePath, "cert-not-exists.pem")
	AgentTokenFile = path.Join(sb.BasePath, "agent-token-not-exists.json")
	ServerCertFingerprintFile = path.Join(sb.BasePath, "server-cert-not-exists.sha256")

	store := NewCertStoreDefault()

	// Act
	isEmpty, err := store.IsEmpty()

	// Assert
	require.NoError(t, err)
	require.True(t, isEmpty)
}

// Test that the cert store is not recognized as empty is any cert file exists.
func TestCertStoreIsNotEmpty(t *testing.T) {
	// Arrange
	_, thisFilePath, _, ok := runtime.Caller(0)
	require.True(t, ok)

	restore := RememberPaths()
	defer restore()
	sb := testutil.NewSandbox()
	defer sb.Close()

	KeyPEMFile = path.Join(sb.BasePath, "key-not-exists.pem")
	RootCAFile = path.Join(sb.BasePath, "root-ca-not-exists.pem")
	CertPEMFile = path.Join(sb.BasePath, "cert-not-exists.pem")
	AgentTokenFile = path.Join(sb.BasePath, "agent-token-not-exists.json")
	ServerCertFingerprintFile = path.Join(sb.BasePath, "server-cert-not-exists.sha256")

	pathPointers := map[string]*string{
		"key":                &KeyPEMFile,
		"root CA":            &RootCAFile,
		"cert":               &CertPEMFile,
		"token":              &AgentTokenFile,
		"server fingerprint": &ServerCertFingerprintFile,
	}

	for label, pathPointer := range pathPointers {
		t.Run(label, func(t *testing.T) {
			restore := RememberPaths()
			defer restore()
			*pathPointer = thisFilePath

			store := NewCertStoreDefault()

			// Act
			isEmpty, err := store.IsEmpty()

			// Assert
			require.NoError(t, err)
			require.False(t, isEmpty)
		})
	}
}

// Test that the server cert fingerprint is removed properly.
func TestRemoveServerCertFingerprint(t *testing.T) {
	// Arrange
	teardown, _ := GenerateSelfSignedCerts()
	defer teardown()
	store := NewCertStoreDefault()

	// Act
	// The fingerprint file exists.
	errExists := store.RemoveServerCertFingerprint()
	// The fingerprint file does not exist. Removing it should not cause an
	// error.
	errNotExists := store.RemoveServerCertFingerprint()

	// Assert
	require.NoError(t, errExists)
	require.NoError(t, errNotExists)
}

// Test that the fingerprint of the CA certificate is read properly.
func TestReadCACertFingerprint(t *testing.T) {
	// Arrange
	teardown, _ := GenerateSelfSignedCerts()
	defer teardown()
	store := NewCertStoreDefault()

	// Act
	actualFingerprint, err := store.ReadRootCAFingerprint()

	// Assert
	require.NoError(t, err)
	require.NotEqual(t, [32]byte{}, actualFingerprint)
}

// Test that the fingerprint is zero if the CA certificate is missing.
func TestReadCACertFingerprintForMissingCA(t *testing.T) {
	// Arrange
	teardown, _ := GenerateSelfSignedCerts()
	defer teardown()
	sb := testutil.NewSandbox()
	defer sb.Close()

	RootCAFile = path.Join(sb.BasePath, "root-ca-not-exists.pem")
	store := NewCertStoreDefault()

	// Act
	fingerprint, err := store.ReadRootCAFingerprint()

	// Assert
	require.NoError(t, err)
	require.Equal(t, [32]byte{}, fingerprint)
}

// Test that the fingerprint cannot be read from the invalid CA cert.
func TestReadCACertFingerprintForInvalidCA(t *testing.T) {
	// Arrange
	teardown, _ := GenerateSelfSignedCerts()
	defer teardown()
	store := NewCertStoreDefault()
	_ = os.WriteFile(RootCAFile, []byte("invalid"), 0o644)

	// Act
	fingerprint, err := store.ReadRootCAFingerprint()

	// Assert
	require.ErrorContains(t, err, "could not calculate fingerprint for Root CA cert")
	require.Zero(t, fingerprint)
}

// Test that the existence of the server certificate fingerprint file is
// recognized properly.
func TestIsServerCertFingerprintFileExist(t *testing.T) {
	// Arrange
	teardown, _ := GenerateSelfSignedCerts()
	defer teardown()
	store := NewCertStoreDefault()

	t.Run("exists", func(t *testing.T) {
		// Act
		exists, err := store.IsServerCertFingerprintFileExist()

		// Assert
		require.NoError(t, err)
		require.True(t, exists)
	})

	t.Run("not exists", func(t *testing.T) {
		// Arrange
		_ = os.Remove(ServerCertFingerprintFile)

		// Act
		exists, err := store.IsServerCertFingerprintFileExist()

		// Assert
		require.NoError(t, err)
		require.False(t, exists)
	})
}
