package agentcomm

import (
	"crypto/ecdsa"
	"crypto/x509"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/security/advancedtls"
	"isc.org/stork/pki"
	storktest "isc.org/stork/server/test/dbmodel"
)

// Generates self-signed certificates for testing.
func generateSelfSignedCerts() (caCertPEM, serverCertPEM, serverKeyPEM []byte, err error) {
	var caCert *x509.Certificate
	var caKey *ecdsa.PrivateKey
	caKey, _, caCert, caCertPEM, err = pki.GenCAKeyCert(42)
	if err != nil {
		return
	}

	serverCertPEM, serverKeyPEM, err = pki.GenKeyCert(
		"foo",
		[]string{"foobar"},
		[]net.IP{net.ParseIP("127.0.0.1")},
		24,
		caCert, caKey,
		x509.ExtKeyUsageServerAuth,
	)
	return
}

// Test that it is possible to connect to a new agent and that the
// statistics can be gathered for this agent.
func TestConnectingToAgent(t *testing.T) {
	caCertPEM, serverCertPEM, serverKeyPEM, err := generateSelfSignedCerts()
	require.NoError(t, err)

	settings := AgentsSettings{}
	fec := &storktest.FakeEventCenter{}
	agents := newConnectedAgentsImpl(&settings, fec, caCertPEM, serverCertPEM, serverKeyPEM)
	defer agents.Shutdown()

	// connect one agent and check if it is in agents map
	agent, err := agents.getConnectedAgent("127.0.0.1:8080")
	require.NoError(t, err)
	_, ok := agents.agentsStates["127.0.0.1:8080"]
	require.True(t, ok)

	// Initially, there should be no stats.
	require.NotNil(t, agent)
	require.Empty(t, agent.stats.agentCommErrors)
	require.Empty(t, agent.stats.daemonCommErrors)

	// Let's modify some stats.
	agent.stats.agentCommErrors["foo"] = 1

	// We should be able to get pointer to stats via the convenience
	// function.
	stats := agents.GetConnectedAgentStatsWrapper("127.0.0.1", 8080)
	require.NotNil(t, stats)
	defer stats.Close()
	require.Len(t, stats.GetStats().agentCommErrors, 1)
	require.Contains(t, stats.GetStats().agentCommErrors, "foo")
}

// Check if credentials for TLS can be prepared using prepareTLSCreds.
func TestPrepareTLSCreds(t *testing.T) {
	caCertPEM, serverCertPEM, serverKeyPEM, err := generateSelfSignedCerts()
	require.NoError(t, err)
	creds, err := prepareTLSCreds(caCertPEM, serverCertPEM, serverKeyPEM)
	require.NoError(t, err)
	require.NotNil(t, creds)
}

// The verification function must deny access if the extended key usage is
// missing.
func TestVerifyPeerMissingExtendedKeyUsage(t *testing.T) {
	// Arrange
	cert := &x509.Certificate{Raw: []byte("foo")}

	// Act
	rsp, err := verifyPeer(&advancedtls.HandshakeVerificationInfo{
		Leaf: cert,
	})

	// Assert
	require.Nil(t, rsp)
	require.ErrorContains(t, err, "peer certificate does not have the extended key usage set")
}

// Test that the verification function allows access if the certificate meets
// the requirements.
func TestVerifyPeerCorrectCertificate(t *testing.T) {
	// Arrange
	cert := &x509.Certificate{
		Raw:         []byte("foo"),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	// Act
	rsp, err := verifyPeer(&advancedtls.HandshakeVerificationInfo{
		Leaf: cert,
	})

	// Assert
	require.NotNil(t, rsp)
	require.NoError(t, err)
}

// Test that the agent client can be instantiated.
func TestConnectedAgentsConnectorCreateClient(t *testing.T) {
	agentConnector := &agentConnectorImpl{}
	client := agentConnector.createClient()
	require.NotNil(t, client)
}
