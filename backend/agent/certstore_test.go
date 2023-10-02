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
		testCase := testCase
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
		"key":     func(s *CertStore) string { return s.keyPEMPath },
		"cert":    func(s *CertStore) string { return s.certPEMPath },
		"root CA": func(s *CertStore) string { return s.rootCAPEMPath },
		"token":   func(s *CertStore) string { return s.agentTokenPath },
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

	pathPointers := map[string]*string{
		"key":     &KeyPEMFile,
		"root CA": &RootCAFile,
		"cert":    &CertPEMFile,
		"token":   &AgentTokenFile,
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
