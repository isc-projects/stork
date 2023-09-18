package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/pki"
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

	// register arguments
	serverToken := "serverToken"
	agentAddr := "1.2.3.4"
	agentPort := 8080
	regenKey := false
	retry := false

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

			agentCSR := []byte(req["agentCSR"].(string))
			require.NotEmpty(t, agentCSR)

			_, rootKeyPEM, _, rootCertPEM, err := pki.GenCAKeyCert(1)
			require.NoError(t, err)
			agentCertPEM, _, paramsErr, innerErr := pki.SignCert(agentCSR, 2, rootCertPEM, rootKeyPEM)
			require.NoError(t, paramsErr)
			require.NoError(t, innerErr)

			w.WriteHeader(http.StatusOK)
			resp := map[string]interface{}{
				"id":           10,
				"serverCACert": string(rootCertPEM),
				"agentCert":    string(agentCertPEM),
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
	res := Register(serverURL, serverToken, agentAddr, fmt.Sprintf("%d", agentPort), regenKey, retry, NewHTTPClient())
	require.True(t, res)

	// register with agent token
	serverToken = ""
	res = Register(serverURL, serverToken, agentAddr, fmt.Sprintf("%d", agentPort), regenKey, retry, NewHTTPClient())
	require.True(t, res)
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

	// register arguments
	serverToken := "serverToken"
	agentAddr := "1.2.3.4"
	agentPort := 8080
	regenKey := false
	retry := false

	withID := true
	withServerCert := true
	withAgentCert := true

	var idValue interface{}
	var serverCertValue interface{}
	var agentCertValue interface{}
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

			w.WriteHeader(http.StatusOK)
			resp := map[string]interface{}{
				"id":           idValue,
				"serverCACert": string(rootCertPEM),
				"agentCert":    string(agentCertPEM),
			}

			if serverCertValue != nil {
				resp["serverCACert"] = serverCertValue
			}

			if agentCertValue != nil {
				resp["agentCert"] = agentCertValue
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

	// missing ID in response
	withID = false
	res := Register(serverURL, serverToken, agentAddr, fmt.Sprintf("%d", agentPort), regenKey, retry, NewHTTPClient())
	require.False(t, res)
	withID = true

	// bad ID in response
	idValue = "bad-value"
	res = Register(serverURL, serverToken, agentAddr, fmt.Sprintf("%d", agentPort), regenKey, retry, NewHTTPClient())
	require.False(t, res)
	idValue = 10 // restore proper value

	// missing serverCACert in response
	withServerCert = false
	res = Register(serverURL, serverToken, agentAddr, fmt.Sprintf("%d", agentPort), regenKey, retry, NewHTTPClient())
	require.False(t, res)
	withServerCert = true // restore proper value

	// bad serverCACert in response
	serverCertValue = 5
	res = Register(serverURL, serverToken, agentAddr, fmt.Sprintf("%d", agentPort), regenKey, retry, NewHTTPClient())
	require.False(t, res)
	serverCertValue = nil // restore proper value

	// missing agentCert in response
	withAgentCert = false
	res = Register(serverURL, serverToken, agentAddr, fmt.Sprintf("%d", agentPort), regenKey, retry, NewHTTPClient())
	require.False(t, res)
	withAgentCert = true // restore proper value

	// bad serverCACert in response
	agentCertValue = 5
	res = Register(serverURL, serverToken, agentAddr, fmt.Sprintf("%d", agentPort), regenKey, retry, NewHTTPClient())
	require.False(t, res)
	agentCertValue = nil // restore proper value
}

// Check Register response to bad arguments or how it behaves in bad environment.
func TestRegisterNegative(t *testing.T) {
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

	// bad server URL
	res := Register("12:3", "serverToken", "1.2.3.4", "8080", false, false, NewHTTPClient())
	require.False(t, res)

	// empty server URL
	res = Register("", "serverToken", "1.2.3.4", "8080", false, false, NewHTTPClient())
	require.False(t, res)

	// cannot prompt for server token (regenKey is true)
	res = Register("http:://localhost:54333", "", "1.2.3.4", "8080", true, false, NewHTTPClient())
	require.False(t, res)

	// bad agent port
	res = Register("http:://localhost:54333", "", "1.2.3.4", "port", false, false, NewHTTPClient())
	require.False(t, res)

	// bad folder for certs
	KeyPEMFile = "/non/existing/dir/key.pem"
	httpClient := NewHTTPClient()
	require.NoError(t, err)
	res = Register("http:://localhost:54333", "", "1.2.3.4", "8080", false, false, httpClient)
	require.False(t, res)
	KeyPEMFile = path.Join(tmpDir, "certs/key.pem") // Restore proper value.

	// bad folder for agent token
	AgentTokenFile = "/non/existing/dir/agent-token.txt"
	require.NoError(t, err)
	res = Register("http:://localhost:54333", "", "1.2.3.4", "8080", false, false, httpClient)
	require.False(t, res)
	AgentTokenFile = path.Join(tmpDir, "tokens/agent-token.txt") // restore proper value

	// not running agent on 54444 port
	res = Register("http://localhost:54333", "serverToken", "localhost", "54444", false, false, NewHTTPClient())
	require.False(t, res)
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

	// register arguments
	serverToken := "serverToken"
	agentAddr := "1.2.3.4"
	agentPort := 8080
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

			agentCSR := []byte(req["agentCSR"].(string))
			require.NotEmpty(t, agentCSR)

			_, rootKeyPEM, _, rootCertPEM, err := pki.GenCAKeyCert(1)
			require.NoError(t, err)
			agentCertPEM, _, paramsErr, innerErr := pki.SignCert(agentCSR, 2, rootCertPEM, rootKeyPEM)
			require.NoError(t, paramsErr)
			require.NoError(t, innerErr)

			w.WriteHeader(http.StatusOK)
			resp := map[string]interface{}{
				"id":           10,
				"serverCACert": string(rootCertPEM),
				"agentCert":    string(agentCertPEM),
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

	res := Register(serverURL, serverToken, agentAddr, fmt.Sprintf("%d", agentPort), regenKey, retry, NewHTTPClient())
	require.True(t, res)
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

	// register arguments
	serverToken := "serverToken"
	agentAddr := "1.2.3.4"
	agentPort := 8080
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
			resp := map[string]interface{}{
				"id":           10,
				"serverCACert": string(rootCertPEM),
				"agentCert":    string(agentCertPEM),
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
	agentPortStr := fmt.Sprintf("%d", agentPort)

	// register with server token
	res := Register(serverURL, serverToken, agentAddr, agentPortStr, regenKey, retry, NewHTTPClient())
	require.True(t, res)

	privKeyPEM1, err := os.ReadFile(KeyPEMFile)
	require.NoError(t, err)
	agentToken1, err := os.ReadFile(AgentTokenFile)
	require.NoError(t, err)
	certPEM1, err := os.ReadFile(CertPEMFile)
	require.NoError(t, err)
	rootCA1, err := os.ReadFile(RootCAFile)
	require.NoError(t, err)

	// re-register with the same agent token
	serverToken = ""
	res = Register(serverURL, serverToken, agentAddr, agentPortStr, regenKey, retry, NewHTTPClient())
	require.True(t, res)

	privKeyPEM2, err := os.ReadFile(KeyPEMFile)
	require.NoError(t, err)
	agentToken2, err := os.ReadFile(AgentTokenFile)
	require.NoError(t, err)
	certPEM2, err := os.ReadFile(CertPEMFile)
	require.NoError(t, err)
	rootCA2, err := os.ReadFile(RootCAFile)
	require.NoError(t, err)

	require.Equal(t, privKeyPEM1, privKeyPEM2)
	require.Equal(t, agentToken1, agentToken2)
	require.Equal(t, certPEM1, certPEM2)
	require.Equal(t, rootCA1, rootCA2)

	// Regenerate certs
	regenKey = true
	serverToken = "serverToken"
	res = Register(serverURL, serverToken, agentAddr, agentPortStr, regenKey, retry, NewHTTPClient())
	require.True(t, res)

	privKeyPEM3, err := os.ReadFile(KeyPEMFile)
	require.NoError(t, err)
	agentToken3, err := os.ReadFile(AgentTokenFile)
	require.NoError(t, err)
	certPEM3, err := os.ReadFile(CertPEMFile)
	require.NoError(t, err)
	rootCA3, err := os.ReadFile(RootCAFile)
	require.NoError(t, err)

	require.NotEqual(t, privKeyPEM1, privKeyPEM3)
	require.NotEqual(t, agentToken1, agentToken3)
	require.NotEqual(t, certPEM1, certPEM3)
	require.NotEqual(t, rootCA1, rootCA3)

	// Re-registration, but invalid header is returned
	regenKey = false
	invalidHeaderValues := []string{"", "/machines/", "/machines", "/machines/abc", "/machines/1a", "/machines/2a2"}
	for _, value := range invalidHeaderValues {
		locationHeaderValue = value
		res = Register(serverURL, serverToken, agentAddr, agentPortStr, regenKey, retry, NewHTTPClient())
		require.False(t, res)
	}
}
