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

// Creates a new HTTP client with the default configuration.
// It is intended to be used in the tests.
func newHTTPClientWithDefaults() *httpClient {
	return NewHTTPClient(HTTPClientConfig{})
}

// Check that HTTP client can be created.
func TestNewHTTPClient(t *testing.T) {
	// Arrange & Act
	client := newHTTPClientWithDefaults()

	// Assert
	require.NotNil(t, client)
	require.Zero(t, client.basicAuth)
	require.NotNil(t, client.client)
	require.EqualValues(t, defaultHTTPClientTimeout, client.client.Timeout)
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

	// Act
	config := HTTPClientConfig{}
	ok, err := config.LoadGRPCCertificates()
	client := NewHTTPClient(config)

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

	require.Zero(t, client.basicAuth)
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

	config := HTTPClientConfig{}

	// Act
	ok, err := config.LoadGRPCCertificates()
	client := NewHTTPClient(config)

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
	// Arrange & Act
	config := HTTPClientConfig{SkipTLSVerification: true}
	client := NewHTTPClient(config)

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

	// Create HTTP Client Configuration with credentials.
	config := HTTPClientConfig{
		BasicAuth: basicAuthCredentials{User: "foo", Password: "bar"},
	}
	client := NewHTTPClient(config)

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

	client := NewHTTPClient(HTTPClientConfig{SkipTLSVerification: true})

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

	client := newHTTPClientWithDefaults()
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

	clientWithoutCredentials := NewHTTPClient(HTTPClientConfig{})
	clientWithCredentials := NewHTTPClient(HTTPClientConfig{
		BasicAuth: basicAuthCredentials{User: "foo", Password: "bar"},
	})

	// Act & Assert
	require.False(t, clientWithoutCredentials.HasAuthenticationCredentials())
	require.True(t, clientWithCredentials.HasAuthenticationCredentials())
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

	client := NewHTTPClient(HTTPClientConfig{
		// Set very short timeout for the testing purposes.
		Timeout: 100 * time.Millisecond,
	})
	var (
		res        *http.Response
		err        error
		clientDone bool
		mutex      sync.RWMutex
		wgClient   sync.WaitGroup
	)
	// Ensure that the client returned before we check an error code.
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

// Test that the HTTP client configuration can be copied and the copied
// instances differ from the original one.
func TestHTTPClientConfigCopy(t *testing.T) {
	// Arrange
	original := HTTPClientConfig{
		SkipTLSVerification: true,
		BasicAuth: basicAuthCredentials{
			User: "foo", Password: "bar",
		},
	}

	// Act
	copy := original

	// Assert
	require.NotNil(t, copy)
	require.Equal(t, original, copy)
	require.NotSame(t, &original, &copy)

	require.Equal(t, original.BasicAuth, copy.BasicAuth)
	require.NotSame(t, &original.BasicAuth, &copy.BasicAuth)

	require.Equal(t, original.TLSCert, copy.TLSCert)
	require.Same(t, original.TLSCert, copy.TLSCert)

	require.Equal(t, original.TLSRootCA, copy.TLSRootCA)
	require.Same(t, original.TLSRootCA, copy.TLSRootCA)

	require.Equal(t, original.SkipTLSVerification, copy.SkipTLSVerification)
}
