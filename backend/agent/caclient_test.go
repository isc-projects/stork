package agent

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/testutil"
	storkutil "isc.org/stork/util"
)

// Check that HTTP client can be created.
func TestNewHTTPClient(t *testing.T) {
	// Arrange & Act
	client := NewHTTPClient()

	// Assert
	require.NotNil(t, client)
	require.Nil(t, client.credentials)
	require.NotNil(t, client.client)
	require.NotNil(t, client.transport)
	require.NotNil(t, client.transport.TLSClientConfig)
	require.NotNil(t, client.transport.TLSNextProto)
	require.Nil(t, client.transport.TLSClientConfig.RootCAs)
	require.Nil(t, client.transport.TLSClientConfig.Certificates)
	require.False(t, client.transport.TLSClientConfig.InsecureSkipVerify)
}

// Check that HTTP client can load the GRPC TLS credentials if available.
func TestLoadGRPCCertificates(t *testing.T) {
	// Arrange
	cleanup, _ := GenerateSelfSignedCerts()
	defer cleanup()

	client := NewHTTPClient()

	// Act
	ok, err := client.LoadGRPCCertificates()

	// Assert
	require.NoError(t, err)
	require.True(t, ok)

	transport := client.client.Transport.(*http.Transport)
	require.NotNil(t, transport)
	require.NotNil(t, transport.TLSClientConfig)

	transportConfig := transport.TLSClientConfig
	require.False(t, transportConfig.InsecureSkipVerify)

	require.NotNil(t, transportConfig.RootCAs)
	require.NotNil(t, transportConfig.Certificates)

	require.Nil(t, client.credentials)
}

// Check that HTTP client returns an error if the TLS credentials could not be
// / loaded because the certificate files are missing.
func TestLoadGRPCCertificatesMissingCerts(t *testing.T) {
	// Arrange
	cleanup := RememberPaths()
	defer cleanup()
	sb := testutil.NewSandbox()
	defer sb.Close()

	KeyPEMFile = path.Join(sb.BasePath, "key-not-exists.pem")
	CertPEMFile = path.Join(sb.BasePath, "cert-not-exists.pem")
	RootCAFile = path.Join(sb.BasePath, "rootCA-not-exists.pem")
	AgentTokenFile = path.Join(sb.BasePath, "agentToken-not-exists")

	client := NewHTTPClient()

	// Act
	ok, err := client.LoadGRPCCertificates()

	// Assert
	require.NoError(t, err)
	require.False(t, ok)

	transport := client.client.Transport.(*http.Transport)
	require.NotNil(t, transport)
	require.NotNil(t, transport.TLSClientConfig)

	transportConfig := transport.TLSClientConfig
	require.False(t, transportConfig.InsecureSkipVerify)

	require.Nil(t, transportConfig.RootCAs)
	require.Nil(t, transportConfig.Certificates)
}

// Check that HTTP client may be set to skip a server
// credentials validation.
func TestCreateHTTPClientSkipVerification(t *testing.T) {
	// Arrange
	client := NewHTTPClient()

	// Act
	client.SetSkipTLSVerification(true)

	// Assert
	transport := client.client.Transport.(*http.Transport)
	require.NotNil(t, transport)
	require.NotNil(t, transport.TLSClientConfig)

	transportConfig := transport.TLSClientConfig
	require.True(t, transportConfig.InsecureSkipVerify)
}

// Test that an authorization header is added to the HTTP request
// when the credentials file contains the credentials for specific
// network location.
func TestAddAuthorizationHeaderWhenBasicAuthCredentialsExist(t *testing.T) {
	// Prepare test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headerContent := r.Header.Get("Authorization")
		require.NotEmpty(t, headerContent)
		require.True(t, strings.HasPrefix(headerContent, "Basic "))
		secret := strings.TrimPrefix(headerContent, "Basic ")
		rawCredentials, err := base64.StdEncoding.DecodeString(secret)
		require.NoError(t, err)
		parts := strings.Split(string(rawCredentials), ":")
		require.Len(t, parts, 2)
		user := parts[0]
		password := parts[1]
		require.EqualValues(t, "foo", user)
		require.EqualValues(t, "bar", password)
	}))
	defer ts.Close()

	serverURL := ts.URL
	serverIP, serverPort, _ := storkutil.ParseURL(serverURL)

	// Create credentials file
	restorePaths := RememberPaths()
	defer restorePaths()
	sb := testutil.NewSandbox()
	defer sb.Close()

	content := fmt.Sprintf(`{
		"basic_auth": [
			{
				"ip": "%s",
				"port": %d,
				"user": "foo",
				"password": "bar"
			}
		]
	}`, serverIP, serverPort)

	CredentialsFile, _ = sb.Write("credentials.json", content)

	// Create HTTP Client
	client := NewHTTPClient()

	// Load credentials
	ok, err := client.LoadCredentials()
	require.NoError(t, err)
	require.True(t, ok)
	require.NotNil(t, client.credentials)

	res, err := client.Call(ts.URL, bytes.NewBuffer([]byte{}))
	require.NoError(t, err)
	defer res.Body.Close()
}

// Test that an authorization header isn't added to the HTTP request
// when the credentials file doesn't exist.
func TestAddAuthorizationHeaderWhenBasicAuthCredentialsNonExist(t *testing.T) {
	restorePaths := RememberPaths()
	defer restorePaths()
	sb := testutil.NewSandbox()
	defer sb.Close()

	CredentialsFile = path.Join(sb.BasePath, "credentials-not-exists.json")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headerContent := r.Header.Get("Authorization")
		require.Empty(t, headerContent)
	}))
	defer ts.Close()

	client := NewHTTPClient()
	client.SetSkipTLSVerification(true)

	// Load credentials
	ok, err := client.LoadCredentials()
	require.NoError(t, err)
	require.False(t, ok)
	require.Nil(t, client.credentials)

	res, err := client.Call(ts.URL, bytes.NewBuffer([]byte{}))
	require.NoError(t, err)
	defer res.Body.Close()
}

// Test that missing body in request is accepted.
func TestCallWithMissingBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.EqualValues(t, http.NoBody, r.Body)
	}))
	defer ts.Close()

	client := NewHTTPClient()
	res, err := client.Call(ts.URL, nil)
	require.NoError(t, err)
	defer res.Body.Close()
}

// Test that the authentication credentials are detected properly.
func TestHasAuthenticationCredentials(t *testing.T) {
	// Arrange
	restorePaths := RememberPaths()
	defer restorePaths()

	tmpDir, _ := os.MkdirTemp("", "reg")
	defer os.RemoveAll(tmpDir)

	CredentialsFile = path.Join(tmpDir, "credentials.json")

	content := `{
		"basic_auth": [
			{
				"ip": "10.0.0.1",
				"port": 42,
				"user": "foo",
				"password": "bar"
			}
		]
	}`

	_ = os.WriteFile(CredentialsFile, []byte(content), 0o600)

	client := NewHTTPClient()

	// Act & Assert
	require.False(t, client.HasAuthenticationCredentials())

	// Load credentials.
	ok, err := client.LoadCredentials()
	require.NoError(t, err)
	require.True(t, ok)

	// Act & Assert
	require.True(t, client.HasAuthenticationCredentials())
}

// Test that the loading credentials returns no error if the credentials
// file has no entries.
func TestLoadCredentialsNoEntries(t *testing.T) {
	// Arrange
	restorePaths := RememberPaths()
	defer restorePaths()

	tmpDir, _ := os.MkdirTemp("", "reg")
	defer os.RemoveAll(tmpDir)

	CredentialsFile = path.Join(tmpDir, "credentials.json")
	content := `{ "basic_auth": [ ] }`
	_ = os.WriteFile(CredentialsFile, []byte(content), 0o600)

	client := NewHTTPClient()

	// Act
	ok, err := client.LoadCredentials()

	// Assert
	require.NoError(t, err)
	require.True(t, ok)
	require.False(t, client.HasAuthenticationCredentials())
}

// Test that the loading credentials returns an error if the credentials
// file is empty.
func TestLoadCredentialsEmptyFile(t *testing.T) {
	// Arrange
	restorePaths := RememberPaths()
	defer restorePaths()

	tmpDir, _ := os.MkdirTemp("", "reg")
	defer os.RemoveAll(tmpDir)

	CredentialsFile = path.Join(tmpDir, "credentials.json")
	content := ""
	_ = os.WriteFile(CredentialsFile, []byte(content), 0o600)

	client := NewHTTPClient()

	// Act
	ok, err := client.LoadCredentials()

	// Assert
	require.Error(t, err)
	require.False(t, ok)
	require.False(t, client.HasAuthenticationCredentials())
}

// Test that loading the authentication credentials returns no error if the
// credentials is missing.
func TestLoadCredentialsCredentialsMissingFile(t *testing.T) {
	// Arrange
	restorePaths := RememberPaths()
	defer restorePaths()

	CredentialsFile = "/not/exist/file.json"

	client := NewHTTPClient()

	// Act
	ok, err := client.LoadCredentials()

	// Assert
	require.NoError(t, err)
	require.False(t, ok)
	require.False(t, client.HasAuthenticationCredentials())
}
