package agent

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// Check that HTTP client sets the TLS credentials if available.
func TestCreateHTTPClientWithClientCerts(t *testing.T) {
	cleanup, err := GenerateSelfSignedCerts()
	require.NoError(t, err)
	defer cleanup()

	client := NewHTTPClient(false)
	require.NotNil(t, client)

	transport := client.client.Transport.(*http.Transport)
	require.NotNil(t, transport)
	require.NotNil(t, transport.TLSClientConfig)

	transportConfig := transport.TLSClientConfig
	require.False(t, transportConfig.InsecureSkipVerify)

	require.NotNil(t, transportConfig.RootCAs)
	require.NotNil(t, transportConfig.Certificates)
}

// Check that HTTP client doesn't set the TLS credentials if missing
// (for example in the unit tests).
func TestCreateHTTPClientWithoutClientCerts(t *testing.T) {
	cleanup := RememberCertPaths()
	defer cleanup()

	KeyPEMFile = "/not/exists/path"
	CertPEMFile = "/not/exists/path"
	RootCAFile = "/not/exists/path"
	AgentTokenFile = "/not/exists/path"

	client := NewHTTPClient(false)
	require.NotNil(t, client)

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
	client := NewHTTPClient(true)
	require.NotNil(t, client)

	transport := client.client.Transport.(*http.Transport)
	require.NotNil(t, transport)
	require.NotNil(t, transport.TLSClientConfig)

	transportConfig := transport.TLSClientConfig
	require.True(t, transportConfig.InsecureSkipVerify)
}
