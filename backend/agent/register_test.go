package agent

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/pki"
	"isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
	storktestdbmodel "isc.org/stork/server/test/dbmodel"
	"isc.org/stork/testutil"
	storkutil "isc.org/stork/util"
)

// Check if registration works in basic situation.
func TestRegisterBasic(t *testing.T) {
	// prepare temp dir for cert files
	tmpDir, err := os.MkdirTemp("", "reg")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// redefined consts with paths to cert files
	restoreCerts := RememberPaths()
	defer restoreCerts()
	KeyPEMFile = path.Join(tmpDir, "certs/key.pem")
	CertPEMFile = path.Join(tmpDir, "certs/cert.pem")
	RootCAFile = path.Join(tmpDir, "certs/ca.pem")
	AgentTokenFile = path.Join(tmpDir, "tokens/agent-token.txt")
	ServerCertFingerprintFile = path.Join(tmpDir, "tokens/server-cert.sha256")

	// generate the CA and server certs
	rootKey, rootKeyPEM, rootCert, rootCertPEM, err := pki.GenCAKeyCert(1)
	require.NoError(t, err)
	serverCertPEM, serverKeyPEM, err := pki.GenKeyCert(
		"server", []string{"localhost"}, []net.IP{net.ParseIP("127.0.0.1")},
		1, rootCert, rootKey, x509.ExtKeyUsageClientAuth,
	)
	require.NoError(t, err)
	serverCert, err := pki.ParseCert(serverCertPEM)
	require.NoError(t, err)
	serverCertFingerprint := pki.CalculateFingerprint(serverCert)

	// register arguments
	serverToken := "serverToken"
	regenKey := false
	retry := false
	agentAddr := "127.0.0.1"
	agentPort, err := testutil.GetFreeLocalTCPPort()
	require.NoError(t, err)

	fec := &storktestdbmodel.FakeEventCenter{}
	var agentAPI agentcomm.ConnectedAgents

	// internal http server for testing
	require.NoError(t, err)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("URL: %v\n", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		fmt.Printf("BODY: %v\n", string(body))
		var req map[string]interface{}
		err = json.Unmarshal(body, &req)
		require.NoError(t, err)

		if r.URL.Path == "/api/machines" {
			require.EqualValues(t, req["address"].(string), agentAddr)
			require.EqualValues(t, int(req["agentPort"].(float64)), agentPort)
			serverTokenReceived := req["serverToken"].(string)
			agentToken := req["agentToken"].(string)
			caCertFingerprint := req["caCertFingerprint"].(string)

			require.NotEmpty(t, agentToken)
			require.NotEmpty(t, caCertFingerprint)
			if serverToken != "" {
				require.EqualValues(t, serverToken, serverTokenReceived)
			}

			agentCSR := []byte(req["agentCSR"].(string))
			require.NotEmpty(t, agentCSR)

			agentCertPEM, _, paramsErr, innerErr := pki.SignCert(agentCSR, 2, rootCertPEM, rootKeyPEM)
			require.NoError(t, paramsErr)
			require.NoError(t, innerErr)

			w.WriteHeader(http.StatusOK)
			resp := map[string]interface{}{
				"id":                    10,
				"serverCACert":          string(rootCertPEM),
				"agentCert":             string(agentCertPEM),
				"serverCertFingerprint": storkutil.BytesToHex(serverCertFingerprint[:]),
			}
			json.NewEncoder(w).Encode(resp)

			// Initialize the GRPC API.
			agentAPI = agentcomm.NewConnectedAgents(
				&agentcomm.AgentsSettings{}, fec,
				rootCertPEM, serverCertPEM, serverKeyPEM,
			)
		}

		if strings.HasSuffix(r.URL.Path, "/ping") {
			// The create machine endpoint must be called before ping.
			require.NotNil(t, agentAPI)

			serverTokenReceived := req["serverToken"].(string)
			agentToken := req["agentToken"].(string)

			require.NotEmpty(t, agentToken)
			if serverToken != "" {
				require.EqualValues(t, serverToken, serverTokenReceived)

				// Send ping request to the agent. There should be a temporary
				// Ping handler running.
				err = agentAPI.Ping(context.Background(), &dbmodel.Machine{
					Address:   agentAddr,
					AgentPort: int64(agentPort),
				})
				require.NoError(t, err)
			}

			w.WriteHeader(http.StatusOK)
			resp := map[string]interface{}{
				"id": 10,
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer ts.Close()

	serverURL := ts.URL

	// register with server token
	err = Register(serverURL, serverToken, agentAddr, agentPort, regenKey, retry, newHTTPClientWithDefaults())
	require.NoError(t, err)

	// verify the server cert fingerprint is written to the file
	fingerprintFromFile, err := os.ReadFile(ServerCertFingerprintFile)
	require.NoError(t, err)
	require.Equal(t, storkutil.BytesToHex(serverCertFingerprint[:]), string(fingerprintFromFile))

	// register with agent token
	serverToken = ""
	err = Register(serverURL, serverToken, agentAddr, agentPort, regenKey, retry, newHTTPClientWithDefaults())
	require.NoError(t, err)
}

// Check if registration fails with an expected error when the agent port is
// already in use.
func TestRegisterBusyPort(t *testing.T) {
	// prepare temp dir for cert files
	tmpDir, err := os.MkdirTemp("", "reg")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// redefined consts with paths to cert files
	restoreCerts := RememberPaths()
	defer restoreCerts()
	KeyPEMFile = path.Join(tmpDir, "certs/key.pem")
	CertPEMFile = path.Join(tmpDir, "certs/cert.pem")
	RootCAFile = path.Join(tmpDir, "certs/ca.pem")
	AgentTokenFile = path.Join(tmpDir, "tokens/agent-token.txt")
	ServerCertFingerprintFile = path.Join(tmpDir, "tokens/server-cert.sha256")

	// generate the CA and server certs
	rootKey, rootKeyPEM, rootCert, rootCertPEM, err := pki.GenCAKeyCert(1)
	require.NoError(t, err)
	serverCertPEM, serverKeyPEM, err := pki.GenKeyCert(
		"server", []string{"localhost"}, []net.IP{net.ParseIP("127.0.0.1")},
		1, rootCert, rootKey, x509.ExtKeyUsageClientAuth,
	)
	require.NoError(t, err)
	serverCert, err := pki.ParseCert(serverCertPEM)
	require.NoError(t, err)
	serverCertFingerprint := pki.CalculateFingerprint(serverCert)

	// register arguments
	serverToken := "serverToken"
	regenKey := false
	retry := false
	agentAddr := "127.0.0.1"
	agentPort, err := testutil.GetFreeLocalTCPPort()
	require.NoError(t, err)

	fec := &storktestdbmodel.FakeEventCenter{}
	var agentAPI agentcomm.ConnectedAgents

	// Start a listener on the agent port.
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", agentAddr, agentPort))
	require.NoError(t, err)
	defer ln.Close()

	// internal http server for testing
	require.NoError(t, err)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("URL: %v\n", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		fmt.Printf("BODY: %v\n", string(body))
		var req map[string]interface{}
		err = json.Unmarshal(body, &req)
		require.NoError(t, err)

		if r.URL.Path == "/api/machines" {
			require.EqualValues(t, req["address"].(string), agentAddr)
			require.EqualValues(t, int(req["agentPort"].(float64)), agentPort)
			serverTokenReceived := req["serverToken"].(string)
			agentToken := req["agentToken"].(string)
			caCertFingerprint := req["caCertFingerprint"].(string)

			require.NotEmpty(t, agentToken)
			require.NotEmpty(t, caCertFingerprint)
			if serverToken != "" {
				require.EqualValues(t, serverToken, serverTokenReceived)
			}

			agentCSR := []byte(req["agentCSR"].(string))
			require.NotEmpty(t, agentCSR)

			agentCertPEM, _, paramsErr, innerErr := pki.SignCert(agentCSR, 2, rootCertPEM, rootKeyPEM)
			require.NoError(t, paramsErr)
			require.NoError(t, innerErr)

			w.WriteHeader(http.StatusOK)
			resp := map[string]interface{}{
				"id":                    10,
				"serverCACert":          string(rootCertPEM),
				"agentCert":             string(agentCertPEM),
				"serverCertFingerprint": storkutil.BytesToHex(serverCertFingerprint[:]),
			}
			json.NewEncoder(w).Encode(resp)

			// Initialize the GRPC API.
			agentAPI = agentcomm.NewConnectedAgents(
				&agentcomm.AgentsSettings{}, fec,
				rootCertPEM, serverCertPEM, serverKeyPEM,
			)
		}

		if strings.HasSuffix(r.URL.Path, "/ping") {
			// The create machine endpoint must be called before ping.
			require.NotNil(t, agentAPI)

			serverTokenReceived := req["serverToken"].(string)
			agentToken := req["agentToken"].(string)

			require.NotEmpty(t, agentToken)
			if serverToken != "" {
				require.EqualValues(t, serverToken, serverTokenReceived)

				// Send ping request to the agent. There should be a temporary
				// Ping handler running.
				err = agentAPI.Ping(context.Background(), &dbmodel.Machine{
					Address:   agentAddr,
					AgentPort: int64(agentPort),
				})
				require.NoError(t, err)
			}

			w.WriteHeader(http.StatusOK)
			resp := map[string]interface{}{
				"id": 10,
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer ts.Close()

	serverURL := ts.URL

	// register with server token
	err = Register(serverURL, serverToken, agentAddr, agentPort, regenKey, retry, newHTTPClientWithDefaults())
	require.ErrorContains(t,
		err,
		fmt.Sprintf("Stork agent detected a program bound to port %d", agentPort),
	)
}

// Check if registration works when server returns bad response.
func TestRegisterBadServer(t *testing.T) {
	// prepare temp dir for cert files
	tmpDir, err := os.MkdirTemp("", "reg")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	os.Mkdir(path.Join(tmpDir, "certs"), 0o755)
	os.Mkdir(path.Join(tmpDir, "tokens"), 0o755)

	// redefined consts with paths to cert files
	restoreCerts := RememberPaths()
	defer restoreCerts()

	KeyPEMFile = path.Join(tmpDir, "certs/key.pem")
	CertPEMFile = path.Join(tmpDir, "certs/cert.pem")
	RootCAFile = path.Join(tmpDir, "certs/ca.pem")
	AgentTokenFile = path.Join(tmpDir, "tokens/agent-token.txt")
	ServerCertFingerprintFile = path.Join(tmpDir, "tokens/server-cer.sha256")

	// register arguments
	serverToken := "serverToken"
	agentAddr := "127.0.0.1"
	agentPort, err := testutil.GetFreeLocalTCPPort()
	require.NoError(t, err)
	regenKey := false
	retry := false

	withID := true
	withServerCert := true
	withAgentCert := true
	withServerCertFingerprint := true

	var idValue interface{}
	var serverCertValue interface{}
	var agentCertValue interface{}
	var serverCertFingerprint interface{}
	idValue = 10
	serverCertValue = nil
	agentCertValue = nil

	// internal http server for testing
	require.NoError(t, err)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("URL: %v\n", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		fmt.Printf("BODY: %v\n", string(body))
		var req map[string]interface{}
		err = json.Unmarshal(body, &req)
		require.NoError(t, err)

		// response to register machine
		if r.URL.Path == "/api/machines" {
			// missing data in response
			w.WriteHeader(http.StatusOK)

			agentCSR := []byte(req["agentCSR"].(string))
			require.NotEmpty(t, agentCSR)

			_, rootKeyPEM, _, rootCertPEM, err := pki.GenCAKeyCert(1)
			require.NoError(t, err)
			agentCertPEM, _, paramsErr, innerErr := pki.SignCert(agentCSR, 2, rootCertPEM, rootKeyPEM)
			require.NoError(t, paramsErr)
			require.NoError(t, innerErr)
			initialFingerprint := [32]byte{42}

			w.WriteHeader(http.StatusOK)
			resp := map[string]interface{}{
				"id":                    idValue,
				"serverCACert":          string(rootCertPEM),
				"agentCert":             string(agentCertPEM),
				"serverCertFingerprint": storkutil.BytesToHex(initialFingerprint[:]),
			}

			if serverCertValue != nil {
				resp["serverCACert"] = serverCertValue
			}

			if agentCertValue != nil {
				resp["agentCert"] = agentCertValue
			}

			if serverCertFingerprint != nil {
				resp["serverCertFingerprint"] = serverCertFingerprint
			}

			if !withID {
				delete(resp, "id")
			}
			if !withServerCert {
				delete(resp, "serverCACert")
			}
			if !withAgentCert {
				delete(resp, "agentCert")
			}
			if !withServerCertFingerprint {
				delete(resp, "serverCertFingerprint")
			}

			json.NewEncoder(w).Encode(resp)
		}

		// response to ping machine
		if strings.HasSuffix(r.URL.Path, "/ping") {
			serverTokenReceived := req["serverToken"].(string)
			agentToken := req["agentToken"].(string)
			require.NotEmpty(t, agentToken)
			if serverToken != "" {
				require.EqualValues(t, serverToken, serverTokenReceived)
			}

			w.WriteHeader(http.StatusOK)
			resp := map[string]interface{}{
				"id": 10,
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer ts.Close()

	serverURL := ts.URL

	// initially all is OK
	err = Register(serverURL, serverToken, agentAddr, agentPort, regenKey, retry, newHTTPClientWithDefaults())
	require.NoError(t, err)

	// missing ID in response
	withID = false
	err = Register(serverURL, serverToken, agentAddr, agentPort, regenKey, retry, newHTTPClientWithDefaults())
	require.Error(t, err)
	withID = true

	// bad ID in response
	idValue = "bad-value"
	err = Register(serverURL, serverToken, agentAddr, agentPort, regenKey, retry, newHTTPClientWithDefaults())
	require.Error(t, err)
	idValue = 10 // restore proper value

	// missing serverCACert in response
	withServerCert = false
	err = Register(serverURL, serverToken, agentAddr, agentPort, regenKey, retry, newHTTPClientWithDefaults())
	require.Error(t, err)
	withServerCert = true // restore proper value

	// bad serverCACert in response
	serverCertValue = 5
	err = Register(serverURL, serverToken, agentAddr, agentPort, regenKey, retry, newHTTPClientWithDefaults())
	require.Error(t, err)
	serverCertValue = nil // restore proper value

	// missing agentCert in response
	withAgentCert = false
	err = Register(serverURL, serverToken, agentAddr, agentPort, regenKey, retry, newHTTPClientWithDefaults())
	require.Error(t, err)
	withAgentCert = true // restore proper value

	// bad serverCACert in response
	agentCertValue = 5
	err = Register(serverURL, serverToken, agentAddr, agentPort, regenKey, retry, newHTTPClientWithDefaults())
	require.Error(t, err)
	agentCertValue = nil // restore proper value

	// missing serverCertFingerprint in response
	withServerCertFingerprint = false
	err = Register(serverURL, serverToken, agentAddr, agentPort, regenKey, retry, newHTTPClientWithDefaults())
	require.Error(t, err)
	withServerCertFingerprint = true

	// bad serverCertFingerprint in response
	serverCertFingerprint = "bad-fingerprint"
	err = Register(serverURL, serverToken, agentAddr, agentPort, regenKey, retry, newHTTPClientWithDefaults())
	require.Error(t, err)
	serverCertFingerprint = nil // restore proper value

	// finally all is OK
	err = Register(serverURL, serverToken, agentAddr, agentPort, regenKey, retry, newHTTPClientWithDefaults())
	require.NoError(t, err)
}

// Check Register response to bad arguments or how it behaves in bad environment.
func TestRegisterNegative(t *testing.T) {
	// prepare temp dir for cert files
	sb := testutil.NewSandbox()
	defer sb.Close()

	// redefined consts with paths to cert files
	restoreCerts := RememberPaths()
	defer restoreCerts()

	KeyPEMFile = path.Join(sb.BasePath, "certs/key.pem")
	CertPEMFile = path.Join(sb.BasePath, "certs/cert.pem")
	RootCAFile = path.Join(sb.BasePath, "certs/ca.pem")
	AgentTokenFile = path.Join(sb.BasePath, "tokens/agent-token.txt")
	ServerCertFingerprintFile = path.Join(sb.BasePath, "tokens/server-cert.sha256")

	// bad server URL
	err := Register("12:3", "serverToken", "1.2.3.4", 8080, false, false, newHTTPClientWithDefaults())
	require.Error(t, err)

	// empty server URL
	err = Register("", "serverToken", "1.2.3.4", 8080, false, false, newHTTPClientWithDefaults())
	require.Error(t, err)

	// cannot prompt for server token (regenKey is true)
	err = Register("http:://localhost:54333", "", "1.2.3.4", 8080, true, false, newHTTPClientWithDefaults())
	require.Error(t, err)

	// bad agent port
	err = Register("http:://localhost:54333", "", "1.2.3.4", 20240704, false, false, newHTTPClientWithDefaults())
	require.Error(t, err)

	// bad folder for certs
	KeyPEMFile = path.Join(sb.BasePath, "non-existing-dir/key.pem")
	err = Register("http:://localhost:54333", "", "1.2.3.4", 8080, false, false, newHTTPClientWithDefaults())
	require.Error(t, err)
	KeyPEMFile = path.Join(sb.BasePath, "certs/key.pem") // Restore proper value.

	// bad folder for agent token
	AgentTokenFile = path.Join(sb.BasePath, "non-existing-dir/agent-token.txt")
	err = Register("http:://localhost:54333", "", "1.2.3.4", 8080, false, false, newHTTPClientWithDefaults())
	require.Error(t, err)
	AgentTokenFile = path.Join(sb.BasePath, "tokens/agent-token.txt") // restore proper value

	// bad folder for server cert fingerprint
	ServerCertFingerprintFile = path.Join(sb.BasePath, "non-existing-dir/server-cert.sha256")
	err = Register("http:://localhost:54333", "", "1.2.3.4", 8080, false, false, newHTTPClientWithDefaults())
	require.Error(t, err)
	ServerCertFingerprintFile = path.Join(sb.BasePath, "server-cert.sha256")

	// not running agent on 54444 port
	err = Register("http://localhost:54333", "serverToken", "localhost", 54444, false, false, newHTTPClientWithDefaults())
	require.Error(t, err)
}

// Check if generating and regenerating of key and cert by
// generateCerts works depending on existence/non-existence of files
// and value of regenKey flag.
func TestGenerateCSRHelper(t *testing.T) {
	// prepare temp dir for cert files
	tmpDir, err := os.MkdirTemp("", "reg")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	os.Mkdir(path.Join(tmpDir, "certs"), 0o755)
	os.Mkdir(path.Join(tmpDir, "tokens"), 0o755)

	// redefined consts with paths to cert files
	restoreCerts := RememberPaths()
	defer restoreCerts()

	KeyPEMFile = path.Join(tmpDir, "certs/key.pem")
	CertPEMFile = path.Join(tmpDir, "certs/cert.pem")
	RootCAFile = path.Join(tmpDir, "certs/ca.pem")
	AgentTokenFile = path.Join(tmpDir, "tokens/agent-token.txt")
	ServerCertFingerprintFile = path.Join(tmpDir, "tokens/server-cert.sha256")

	certStore := NewCertStoreDefault()
	// By-pass evaluating CSR.
	evaluateCSR := func(csr []byte) {
		_, parentPrivateKeyPEM, _, rootCAPEM, err := pki.GenCAKeyCert(42)
		require.NoError(t, err)
		childCertPEM, _, paramErr, execErr := pki.SignCert(csr, 42, rootCAPEM, parentPrivateKeyPEM)
		require.NoError(t, paramErr)
		require.NoError(t, execErr)

		certStore.WriteRootCAPEM(rootCAPEM)
		certStore.WriteCertPEM(childCertPEM)
		certStore.WriteServerCertFingerprint([32]byte{42})
	}

	// 1) just generate
	agentAddr := "addr"
	regenKey := false
	csrPEM1, err := generateCSR(certStore, agentAddr, regenKey)
	require.NoError(t, err)
	require.NotEmpty(t, csrPEM1)
	agentToken1, err := certStore.ReadToken()
	require.NoError(t, err)
	require.NotEmpty(t, agentToken1)
	privKeyPEM1, err := os.ReadFile(KeyPEMFile)
	require.NoError(t, err)
	require.NotEmpty(t, privKeyPEM1)
	evaluateCSR(csrPEM1)

	// 2) generate again, no changes to args, result key should be the same
	csrPEM2, err := generateCSR(certStore, agentAddr, regenKey)
	require.NoError(t, err)
	require.NotEmpty(t, csrPEM2)
	agentToken2, err := certStore.ReadToken()
	require.NoError(t, err)
	require.NotEmpty(t, agentToken2)
	// CSR is regenerated but no agent token
	require.NotEqualValues(t, csrPEM1, csrPEM2)
	require.EqualValues(t, agentToken1, agentToken2)
	// but key in the file is the same
	privKeyPEM2, err := os.ReadFile(KeyPEMFile)
	require.NoError(t, err)
	require.NotEmpty(t, privKeyPEM2)
	require.EqualValues(t, privKeyPEM1, privKeyPEM2)
	evaluateCSR(csrPEM2)

	// 3) generate again but now regenKey is true, result should be be different
	regenKey = true
	csrPEM3, err := generateCSR(certStore, agentAddr, regenKey)
	require.NoError(t, err)
	require.NotEmpty(t, csrPEM3)
	agentToken3, err := certStore.ReadToken()
	require.NoError(t, err)
	require.NotEmpty(t, agentToken3)
	// CSR is regenerated and its agent token too
	require.NotEqualValues(t, csrPEM2, csrPEM3)
	require.NotEqualValues(t, agentToken1, agentToken3)
	// but this time key in the file is different (regenerated)
	privKeyPEM3, err := os.ReadFile(KeyPEMFile)
	require.NoError(t, err)
	require.NotEmpty(t, privKeyPEM3)
	require.NotEqualValues(t, privKeyPEM2, privKeyPEM3)
	evaluateCSR(csrPEM3)

	// 4) generate again but the server cert fingerprint is missing
	regenKey = false
	_ = os.Remove(ServerCertFingerprintFile)
	csrPEM4, err := generateCSR(certStore, agentAddr, regenKey)
	require.NoError(t, err)
	require.NotEmpty(t, csrPEM2)
	agentToken4, err := certStore.ReadToken()
	require.NoError(t, err)
	require.NotEmpty(t, agentToken4)
	// CSR is regenerated but no agent token
	require.NotEqualValues(t, csrPEM3, csrPEM4)
	require.EqualValues(t, agentToken3, agentToken4)
	// but key in the file is the same
	privKeyPEM4, err := os.ReadFile(KeyPEMFile)
	require.NoError(t, err)
	require.NotEmpty(t, privKeyPEM4)
	require.EqualValues(t, privKeyPEM3, privKeyPEM4)
	evaluateCSR(csrPEM4)
}

// Check if generating agent token file works and a value in the file
// matches a value received by server.
func TestWriteAgentTokenFileDuringRegistration(t *testing.T) {
	// prepare temp dir for cert files
	tmpDir, err := os.MkdirTemp("", "reg")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// redefined consts with paths to cert files
	restoreCerts := RememberPaths()
	defer restoreCerts()

	KeyPEMFile = path.Join(tmpDir, "certs/key.pem")
	CertPEMFile = path.Join(tmpDir, "certs/cert.pem")
	RootCAFile = path.Join(tmpDir, "certs/ca.pem")
	AgentTokenFile = path.Join(tmpDir, "tokens/agent-token.txt")
	ServerCertFingerprintFile = path.Join(tmpDir, "tokens/server-cert.sha256")

	// register arguments
	serverToken := "serverToken"
	agentAddr := "127.0.0.1"
	agentPort, err := testutil.GetFreeLocalTCPPort()
	require.NoError(t, err)
	regenKey := false
	retry := false

	// Received agent tokens
	var lastPingAgentToken string
	var lastRegisterAgentToken string

	// internal http server for testing
	require.NoError(t, err)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var req map[string]interface{}
		err = json.Unmarshal(body, &req)
		require.NoError(t, err)

		// response to register machine
		if r.URL.Path == "/api/machines" {
			// missing data in response
			w.WriteHeader(http.StatusOK)

			agentToken := req["agentToken"].(string)
			require.NotEmpty(t, agentToken)
			lastRegisterAgentToken = agentToken
			require.NotEmpty(t, req["caCertFingerprint"].(string))

			agentCSR := []byte(req["agentCSR"].(string))
			require.NotEmpty(t, agentCSR)

			_, rootKeyPEM, _, rootCertPEM, err := pki.GenCAKeyCert(1)
			require.NoError(t, err)
			agentCertPEM, _, paramsErr, innerErr := pki.SignCert(agentCSR, 2, rootCertPEM, rootKeyPEM)
			require.NoError(t, paramsErr)
			require.NoError(t, innerErr)

			w.WriteHeader(http.StatusOK)
			fingerprint := [32]byte{42}
			resp := map[string]interface{}{
				"id":                    10,
				"serverCACert":          string(rootCertPEM),
				"agentCert":             string(agentCertPEM),
				"serverCertFingerprint": storkutil.BytesToHex(fingerprint[:]),
			}
			json.NewEncoder(w).Encode(resp)
		}

		// response to ping machine
		if strings.HasSuffix(r.URL.Path, "/ping") {
			agentToken := req["agentToken"].(string)
			require.NotEmpty(t, agentToken)

			w.WriteHeader(http.StatusOK)
			resp := map[string]interface{}{
				"id": 10,
			}
			json.NewEncoder(w).Encode(resp)

			lastPingAgentToken = agentToken
		}
	}))
	defer ts.Close()

	serverURL := ts.URL

	err = Register(serverURL, serverToken, agentAddr, agentPort, regenKey, retry, newHTTPClientWithDefaults())
	require.NoError(t, err)
	require.NotEmpty(t, lastRegisterAgentToken)
	require.NotEmpty(t, lastPingAgentToken)
	require.Equal(t, lastPingAgentToken, lastRegisterAgentToken)

	agentTokenFromFileRaw, err := os.ReadFile(AgentTokenFile)
	require.NoError(t, err)
	agentTokenFromFile := string(agentTokenFromFileRaw)
	require.NotEmpty(t, agentTokenFromFile)
	require.Equal(t, agentTokenFromFile, lastPingAgentToken)
}

// Check if registration doesn't change the agent token and certs
// for an already registered agent until the agent doesn't force regenerating certs.
func TestRepeatRegister(t *testing.T) {
	// prepare temp dir for cert files
	tmpDir, err := os.MkdirTemp("", "reg")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	os.Mkdir(path.Join(tmpDir, "certs"), 0o755)
	os.Mkdir(path.Join(tmpDir, "tokens"), 0o755)

	// redefined consts with paths to cert files
	restoreCerts := RememberPaths()
	defer restoreCerts()

	KeyPEMFile = path.Join(tmpDir, "certs/key.pem")
	CertPEMFile = path.Join(tmpDir, "certs/cert.pem")
	RootCAFile = path.Join(tmpDir, "certs/ca.pem")
	AgentTokenFile = path.Join(tmpDir, "tokens/agent-token.txt")
	ServerCertFingerprintFile = path.Join(tmpDir, "tokens/server-cert.sha256")

	// register arguments
	serverToken := "serverToken"
	agentAddr := "127.0.0.1"
	agentPort, err := testutil.GetFreeLocalTCPPort()
	require.NoError(t, err)
	regenKey := false
	retry := false

	lastAgentToken := ""
	locationHeaderValue := "/api/machines/10"

	// internal http server for testing
	require.NoError(t, err)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("URL: %v\n", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		fmt.Printf("BODY: %v\n", string(body))
		var req map[string]interface{}
		err = json.Unmarshal(body, &req)
		require.NoError(t, err)

		if r.URL.Path == "/api/machines" {
			require.EqualValues(t, req["address"].(string), agentAddr)
			require.EqualValues(t, int(req["agentPort"].(float64)), agentPort)
			serverTokenReceived := req["serverToken"].(string)
			agentToken := req["agentToken"].(string)

			require.NotEmpty(t, agentToken)
			if serverToken != "" {
				require.EqualValues(t, serverToken, serverTokenReceived)
			}
			require.NotEmpty(t, req["caCertFingerprint"])

			if agentToken == lastAgentToken {
				w.Header().Add("Location", locationHeaderValue)
				w.WriteHeader(409)
				return
			}

			lastAgentToken = agentToken

			agentCSR := []byte(req["agentCSR"].(string))
			require.NotEmpty(t, agentCSR)

			_, rootKeyPEM, _, rootCertPEM, err := pki.GenCAKeyCert(1)
			require.NoError(t, err)
			agentCertPEM, _, paramsErr, innerErr := pki.SignCert(agentCSR, 2, rootCertPEM, rootKeyPEM)
			require.NoError(t, paramsErr)
			require.NoError(t, innerErr)

			w.WriteHeader(http.StatusOK)
			fingerprint := [32]byte{42}
			resp := map[string]interface{}{
				"id":                    10,
				"serverCACert":          string(rootCertPEM),
				"agentCert":             string(agentCertPEM),
				"serverCertFingerprint": storkutil.BytesToHex(fingerprint[:]),
			}
			json.NewEncoder(w).Encode(resp)
		}

		if strings.HasSuffix(r.URL.Path, "/ping") {
			serverTokenReceived := req["serverToken"].(string)
			agentToken := req["agentToken"].(string)

			require.NotEmpty(t, agentToken)
			if serverToken != "" {
				require.EqualValues(t, serverToken, serverTokenReceived)
			}

			w.WriteHeader(http.StatusOK)
			resp := map[string]interface{}{
				"id": 10,
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer ts.Close()

	serverURL := ts.URL

	// register with server token
	err = Register(serverURL, serverToken, agentAddr, agentPort, regenKey, retry, newHTTPClientWithDefaults())
	require.NoError(t, err)

	privKeyPEM1, err := os.ReadFile(KeyPEMFile)
	require.NoError(t, err)
	agentToken1, err := os.ReadFile(AgentTokenFile)
	require.NoError(t, err)
	certPEM1, err := os.ReadFile(CertPEMFile)
	require.NoError(t, err)
	rootCA1, err := os.ReadFile(RootCAFile)
	require.NoError(t, err)
	serverCertFingerprint1, err := os.ReadFile(ServerCertFingerprintFile)
	require.NoError(t, err)

	// re-register with the same agent token
	serverToken = ""
	err = Register(serverURL, serverToken, agentAddr, agentPort, regenKey, retry, newHTTPClientWithDefaults())
	require.NoError(t, err)

	privKeyPEM2, err := os.ReadFile(KeyPEMFile)
	require.NoError(t, err)
	agentToken2, err := os.ReadFile(AgentTokenFile)
	require.NoError(t, err)
	certPEM2, err := os.ReadFile(CertPEMFile)
	require.NoError(t, err)
	rootCA2, err := os.ReadFile(RootCAFile)
	require.NoError(t, err)
	serverCertFingerprint2, err := os.ReadFile(ServerCertFingerprintFile)
	require.NoError(t, err)

	require.Equal(t, privKeyPEM1, privKeyPEM2)
	require.Equal(t, agentToken1, agentToken2)
	require.Equal(t, certPEM1, certPEM2)
	require.Equal(t, rootCA1, rootCA2)
	require.Equal(t, serverCertFingerprint1, serverCertFingerprint2)

	// Regenerate certs
	regenKey = true
	serverToken = "serverToken"
	err = Register(serverURL, serverToken, agentAddr, agentPort, regenKey, retry, newHTTPClientWithDefaults())
	require.NoError(t, err)

	privKeyPEM3, err := os.ReadFile(KeyPEMFile)
	require.NoError(t, err)
	agentToken3, err := os.ReadFile(AgentTokenFile)
	require.NoError(t, err)
	certPEM3, err := os.ReadFile(CertPEMFile)
	require.NoError(t, err)
	rootCA3, err := os.ReadFile(RootCAFile)
	require.NoError(t, err)
	serverCertFingerprint3, err := os.ReadFile(ServerCertFingerprintFile)
	require.NoError(t, err)

	require.NotEqual(t, privKeyPEM1, privKeyPEM3)
	require.NotEqual(t, agentToken1, agentToken3)
	require.NotEqual(t, certPEM1, certPEM3)
	require.NotEqual(t, rootCA1, rootCA3)
	// The server cert has not been changed.
	require.Equal(t, serverCertFingerprint1, serverCertFingerprint3)

	// Re-registration, but invalid header is returned
	regenKey = false
	invalidHeaderValues := []string{"", "/machines/", "/machines", "/machines/abc", "/machines/1a", "/machines/2a2"}
	for _, value := range invalidHeaderValues {
		locationHeaderValue = value
		err = Register(serverURL, serverToken, agentAddr, agentPort, regenKey, retry, newHTTPClientWithDefaults())
		require.Error(t, err)
	}
}
