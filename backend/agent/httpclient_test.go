package agent

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"isc.org/stork/testutil"
)

// Check that HTTP client can be created.
func TestNewHTTPClient(t *testing.T) {
	// Arrange & Act
	client := NewHTTPClient()

	// Assert
	require.NotNil(t, client)
	require.Nil(t, client.basicAuth)
	require.NotNil(t, client.client)
	require.EqualValues(t, DefaultHTTPClientTimeout, client.client.Timeout)
	transport := client.getTransport()
	require.NotNil(t, transport)
	require.NotNil(t, transport.TLSClientConfig)
	require.NotNil(t, transport.TLSNextProto)
	require.Nil(t, transport.TLSClientConfig.RootCAs)
	require.Nil(t, transport.TLSClientConfig.Certificates)
	require.False(t, transport.TLSClientConfig.InsecureSkipVerify)
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

	require.Nil(t, client.basicAuth)
	require.False(t, client.HasAuthenticationCredentials())
}

// Check that HTTP client returns an error if the TLS credentials could not be
// loaded because the certificate files are missing.
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
	ServerCertFingerprintFile = path.Join(sb.BasePath, "server-cert-not-exists.sha256")

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
// when there are credentials for specific network location.
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

	// Create HTTP Client
	client := NewHTTPClient()

	// Load credentials
	client.SetBasicAuth("foo", "bar")
	require.NotNil(t, client.basicAuth)

	res, err := client.Call(ts.URL, bytes.NewBuffer([]byte{}))
	require.NoError(t, err)
	defer res.Body.Close()
}

// Test that an authorization header isn't added to the HTTP request
// if the basic auth credentials are missing.
func TestAddAuthorizationHeaderWhenBasicAuthCredentialsNonExist(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headerContent := r.Header.Get("Authorization")
		require.Empty(t, headerContent)
	}))
	defer ts.Close()

	client := NewHTTPClient()
	client.SetSkipTLSVerification(true)

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

	client := NewHTTPClient()

	// Act & Assert
	require.False(t, client.HasAuthenticationCredentials())

	// Load credentials.
	client.SetBasicAuth("foo", "bar")

	// Act & Assert
	require.True(t, client.HasAuthenticationCredentials())
}

// Test that the client returns with a timeout if the server doesn't
// respond.
func TestCallTimeout(t *testing.T) {
	wgServer := &sync.WaitGroup{}
	wgServer.Add(1)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow/blocked response.
		wgServer.Wait()
	}))
	defer func() {
		wgServer.Done()
		ts.Close()
	}()

	client := NewHTTPClient()
	// Set very short timeout for the testing purposes.
	client.SetRequestTimeout(100 * time.Millisecond)
	var (
		res        *http.Response
		err        error
		clientDone bool
		mutex      sync.RWMutex
		wgClient   sync.WaitGroup
	)
	// Ensyre that the client returned before we check an error code.
	wgClient.Add(1)
	go func() {
		// Use HTTP client to communicate with the server. This call
		// should return with a timeout because the server response
		// is blocked.
		res, err = client.Call(ts.URL, nil)
		defer func() {
			if err == nil {
				res.Body.Close()
			}
			// Indicate that the client returned.
			mutex.Lock()
			defer mutex.Unlock()
			clientDone = true
		}()
		// Indicate that the client has returned so we can now check
		// an error code returned.
		wgClient.Done()
	}()
	// The timeout is 100ms. Let's wait up to 2 seconds for the timeout.
	require.Eventually(t, func() bool {
		mutex.RLock()
		defer mutex.RUnlock()
		return clientDone
	}, 2*time.Second, 100*time.Millisecond)

	// Ensure that the client has returned and we can safely access the
	// returned error.
	wgClient.Wait()
	require.NotNil(t, err)
	require.ErrorContains(t, err, "context deadline exceeded")
}

// Test that the HTTP client can be cloned and the cloned instances differ from
// the original one.
func TestHTTPClientClone(t *testing.T) {
	// Arrange
	httpClient := NewHTTPClient()
	httpClient.SetSkipTLSVerification(true)
	httpClient.SetBasicAuth("foo", "bar")

	// Act
	clonedHTTPClient := httpClient.Clone()

	// Assert
	require.NotNil(t, clonedHTTPClient)
	require.NotEqual(t, httpClient, clonedHTTPClient)
	require.NotEqual(t, httpClient.client, clonedHTTPClient.client)

	require.Equal(t, httpClient.basicAuth, clonedHTTPClient.basicAuth)
	require.NotSame(t, httpClient.basicAuth, clonedHTTPClient.basicAuth)
	require.Equal(t, "foo", clonedHTTPClient.basicAuth.User)
	require.Equal(t, "bar", clonedHTTPClient.basicAuth.Password)

	originalTransport := httpClient.getTransport()
	clonedTransport := clonedHTTPClient.getTransport()
	require.NotEqual(t, originalTransport, clonedTransport)
	require.Equal(t, originalTransport.TLSClientConfig, clonedTransport.TLSClientConfig)
	require.NotSame(t, originalTransport.TLSClientConfig, clonedTransport.TLSClientConfig)
	require.True(t, clonedTransport.TLSClientConfig.InsecureSkipVerify)
}
