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

// Generates the self-signed certificates and creates the HTTP client instance.
func newHTTPClientWithCerts(skipTLSVerification bool) (*HTTPClient, func(), error) {
	cleanup, err := GenerateSelfSignedCerts()
	if err != nil {
		return nil, nil, err
	}

	client, err := NewHTTPClient(skipTLSVerification)
	if err != nil {
		return nil, nil, err
	}

	return client, cleanup, nil
}

// Check that HTTP client sets the TLS credentials if available.
func TestCreateHTTPClientWithClientCerts(t *testing.T) {
	cleanup, err := GenerateSelfSignedCerts()
	require.NoError(t, err)
	defer cleanup()

	client, cleanup, err := newHTTPClientWithCerts(false)
	require.NoError(t, err)
	require.NotNil(t, client)
	defer cleanup()

	transport := client.client.Transport.(*http.Transport)
	require.NotNil(t, transport)
	require.NotNil(t, transport.TLSClientConfig)

	transportConfig := transport.TLSClientConfig
	require.False(t, transportConfig.InsecureSkipVerify)

	require.NotNil(t, transportConfig.RootCAs)
	require.NotNil(t, transportConfig.Certificates)

	require.NotNil(t, client.credentials)
}

// Check that HTTP client doesn't set the TLS credentials if missing
// (for example in the unit tests).
func TestCreateHTTPClientWithoutClientCerts(t *testing.T) {
	cleanup := RememberPaths()
	defer cleanup()
	sb := testutil.NewSandbox()
	defer sb.Close()

	KeyPEMFile = path.Join(sb.BasePath, "key-not-exists.pem")
	CertPEMFile = path.Join(sb.BasePath, "cert-not-exists.pem")
	RootCAFile = path.Join(sb.BasePath, "rootCA-not-exists.pem")
	AgentTokenFile = path.Join(sb.BasePath, "agentToken-not-exists")

	client, err := NewHTTPClient(false)
	require.NotNil(t, client)
	require.NoError(t, err)

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
	client, cleanup, err := newHTTPClientWithCerts(true)
	require.NotNil(t, client)
	require.NoError(t, err)
	defer cleanup()

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
	client, cleanup, err := newHTTPClientWithCerts(true)
	require.NotNil(t, client)
	require.NoError(t, err)
	defer cleanup()

	restorePaths := RememberPaths()
	defer restorePaths()

	// Create temp dir
	tmpDir, err := os.MkdirTemp("", "reg")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

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
	CredentialsFile = path.Join(tmpDir, "credentials.json")
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
	err = os.WriteFile(CredentialsFile, []byte(content), 0o600)
	require.NoError(t, err)

	// Create HTTP Client
	client, teardown, err := newHTTPClientWithCerts(true)
	require.NotNil(t, client.credentials)
	require.NoError(t, err)
	defer teardown()

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

	client, cleanup, err := newHTTPClientWithCerts(true)
	require.NotNil(t, client)
	require.NoError(t, err)
	defer cleanup()

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

	client, cleanup, err := newHTTPClientWithCerts(true)
	require.NotNil(t, client)
	require.NoError(t, err)
	defer cleanup()

	require.NoError(t, err)
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

	// Act
	client, cleanup, err := newHTTPClientWithCerts(false)
	require.NotNil(t, client)
	require.NoError(t, err)
	defer cleanup()

	// Assert
	require.NoError(t, err)
	require.True(t, client.HasAuthenticationCredentials())
}

// Test that the authentication credentials are not detected if the credentials
// file exists but it's empty.
func TestHasAuthenticationCredentialsEmptyFile(t *testing.T) {
	// Arrange
	restorePaths := RememberPaths()
	defer restorePaths()

	tmpDir, _ := os.MkdirTemp("", "reg")
	defer os.RemoveAll(tmpDir)

	CredentialsFile = path.Join(tmpDir, "credentials.json")

	content := `{ "basic_auth": [ ] }`

	_ = os.WriteFile(CredentialsFile, []byte(content), 0o600)

	// Act
	client, cleanup, err := newHTTPClientWithCerts(false)
	require.NotNil(t, client)
	require.NoError(t, err)
	defer cleanup()

	// Assert
	require.NoError(t, err)
	require.False(t, client.HasAuthenticationCredentials())
}

// Test that the authentication credentials are not detected if the credentials
// is missing.
func TestHasAuthenticationCredentialsMissingFile(t *testing.T) {
	// Arrange
	restorePaths := RememberPaths()
	defer restorePaths()
	sb := testutil.NewSandbox()
	defer sb.Close()

	CredentialsFile = path.Join(sb.BasePath, "credentials-not-exists.json")

	// Act
	client, cleanup, err := newHTTPClientWithCerts(false)
	require.NotNil(t, client)
	require.NoError(t, err)
	defer cleanup()

	// Assert
	require.NoError(t, err)
	require.False(t, client.HasAuthenticationCredentials())
}
