package agent

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	tmpDir, err := ioutil.TempDir("", "reg")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	os.Mkdir(path.Join(tmpDir, "certs"), 0755)
	os.Mkdir(path.Join(tmpDir, "tokens"), 0755)

	// redefined consts with paths to cert files
	KeyPEMFile = path.Join(tmpDir, "certs/key.pem")
	CertPEMFile = path.Join(tmpDir, "certs/cert.pem")
	RootCAFile = path.Join(tmpDir, "certs/ca.pem")
	AgentTokenFile = path.Join(tmpDir, "tokens/agent-token.txt")

	// register arguments
	serverToken := "serverToken"
	agentAddr := "1.2.3.4"
	agentPort := 8080
	regenCerts := false
	retry := false

	// internal http server for testing
	require.NoError(t, err)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("URL: %v\n", r.URL.Path)

		body, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)
		fmt.Printf("BODY: %v\n", string(body))
		var req map[string]interface{}
		err = json.Unmarshal(body, &req)
		require.NoError(t, err)

		if r.URL.Path == "/api/machines" {
			require.EqualValues(t, req["address"].(string), agentAddr)
			require.EqualValues(t, int(req["agentPort"].(float64)), agentPort)
			serverTokenRcvd := req["serverToken"].(string)
			agentToken := req["agentToken"].(string)
			if serverToken == "" {
				require.NotEmpty(t, agentToken)
			} else {
				require.EqualValues(t, serverToken, serverTokenRcvd)
				require.Empty(t, agentToken)
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
			serverTokenRcvd := req["serverToken"].(string)
			agentToken := req["agentToken"].(string)
			if serverToken == "" {
				require.NotEmpty(t, agentToken)
			} else {
				require.EqualValues(t, serverToken, serverTokenRcvd)
				require.Empty(t, agentToken)
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
	res := Register(serverURL, serverToken, agentAddr, fmt.Sprintf("%d", agentPort), regenCerts, retry)
	require.True(t, res)

	// register with agent token
	serverToken = ""
	res = Register(serverURL, serverToken, agentAddr, fmt.Sprintf("%d", agentPort), regenCerts, retry)
	require.True(t, res)
}

// Check if registration works when server returns bad response.
func TestRegisterBadServer(t *testing.T) {
	// prepare temp dir for cert files
	tmpDir, err := ioutil.TempDir("", "reg")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	os.Mkdir(path.Join(tmpDir, "certs"), 0755)
	os.Mkdir(path.Join(tmpDir, "tokens"), 0755)

	// redefined consts with paths to cert files
	KeyPEMFile = path.Join(tmpDir, "certs/key.pem")
	CertPEMFile = path.Join(tmpDir, "certs/cert.pem")
	RootCAFile = path.Join(tmpDir, "certs/ca.pem")
	AgentTokenFile = path.Join(tmpDir, "tokens/agent-token.txt")

	// register arguments
	serverToken := "serverToken"
	agentAddr := "1.2.3.4"
	agentPort := 8080
	regenCerts := false
	retry := false

	machineRegResp := map[string]interface{}{
		"id":           10,
		"serverCACert": "rootCertPEM",
		"agentCert":    "agentCertPEM",
	}
	machinRegRespPtr := &machineRegResp

	// internal http server for testing
	require.NoError(t, err)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("URL: %v\n", r.URL.Path)

		body, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)
		fmt.Printf("BODY: %v\n", string(body))
		var req map[string]interface{}
		err = json.Unmarshal(body, &req)
		require.NoError(t, err)

		// response to register machine
		if r.URL.Path == "/api/machines" {
			// missing data in response
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(*machinRegRespPtr)
		}

		// response to ping machine
		if strings.HasSuffix(r.URL.Path, "/ping") {
			serverTokenRcvd := req["serverToken"].(string)
			agentToken := req["agentToken"].(string)
			if serverToken == "" {
				require.NotEmpty(t, agentToken)
			} else {
				require.EqualValues(t, serverToken, serverTokenRcvd)
				require.Empty(t, agentToken)
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
	delete(machineRegResp, "id")
	res := Register(serverURL, serverToken, agentAddr, fmt.Sprintf("%d", agentPort), regenCerts, retry)
	require.False(t, res)
	machineRegResp["id"] = 10 // restore proper value

	// bad ID in response
	machineRegResp["id"] = "agerw"
	res = Register(serverURL, serverToken, agentAddr, fmt.Sprintf("%d", agentPort), regenCerts, retry)
	require.False(t, res)
	machineRegResp["id"] = 10 // restore proper value

	// missing serverCACert in response
	delete(machineRegResp, "serverCACert")
	res = Register(serverURL, serverToken, agentAddr, fmt.Sprintf("%d", agentPort), regenCerts, retry)
	require.False(t, res)
	machineRegResp["serverCACert"] = "serverCACert" // restore proper value

	// bad serverCACert in response
	machineRegResp["serverCACert"] = 5
	res = Register(serverURL, serverToken, agentAddr, fmt.Sprintf("%d", agentPort), regenCerts, retry)
	require.False(t, res)
	machineRegResp["serverCACert"] = "serverCACert" // restore proper value

	// missing agentCert in response
	delete(machineRegResp, "agentCert")
	res = Register(serverURL, serverToken, agentAddr, fmt.Sprintf("%d", agentPort), regenCerts, retry)
	require.False(t, res)
	machineRegResp["agentCert"] = "agentCert" // restore proper value

	// bad serverCACert in response
	machineRegResp["agentCert"] = 5
	res = Register(serverURL, serverToken, agentAddr, fmt.Sprintf("%d", agentPort), regenCerts, retry)
	require.False(t, res)
	machineRegResp["agentCert"] = "agentCert" // restore proper value
}

// Check Register response to bad arguments or how it behaves in bad environment.
func TestRegisterNegative(t *testing.T) {
	// prepare temp dir for cert files
	tmpDir, err := ioutil.TempDir("", "reg")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	os.Mkdir(path.Join(tmpDir, "certs"), 0755)
	os.Mkdir(path.Join(tmpDir, "tokens"), 0755)

	// redefined consts with paths to cert files
	KeyPEMFile = path.Join(tmpDir, "certs/key.pem")
	CertPEMFile = path.Join(tmpDir, "certs/cert.pem")
	RootCAFile = path.Join(tmpDir, "certs/ca.pem")
	AgentTokenFile = path.Join(tmpDir, "tokens/agent-token.txt")

	// bad server URL
	res := Register("12:3", "serverToken", "1.2.3.4", "8080", false, false)
	require.False(t, res)

	// empty server URL
	res = Register("", "serverToken", "1.2.3.4", "8080", false, false)
	require.False(t, res)

	// cannot prompt for server token (regenCerts is true)
	res = Register("http:://localhost:54333", "", "1.2.3.4", "8080", true, false)
	require.False(t, res)

	// bad agent port
	res = Register("http:://localhost:54333", "", "1.2.3.4", "port", false, false)
	require.False(t, res)

	// bad folder for certs
	KeyPEMFile = "/root/key.pem"
	res = Register("http:://localhost:54333", "", "1.2.3.4", "8080", false, false)
	require.False(t, res)
	KeyPEMFile = path.Join(tmpDir, "certs/key.pem") // restore proper value

	// bad folder for agent token
	AgentTokenFile = "/root/agent-token.txt"
	res = Register("http:://localhost:54333", "", "1.2.3.4", "8080", false, false)
	require.False(t, res)
	AgentTokenFile = path.Join(tmpDir, "tokens/agent-token.txt") // restore proper value

	// not running agent on 54444 port
	res = Register("http://localhost:54333", "serverToken", "localhost", "54444", false, false)
	require.False(t, res)
}

// Check if generating and regenerating of key and cert by
// generateCerts works depending on existence/non-existence of files
// and value of regenCerts flag.
func TestGenerateCerts(t *testing.T) {
	// prepare temp dir for cert files
	tmpDir, err := ioutil.TempDir("", "reg")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	os.Mkdir(path.Join(tmpDir, "certs"), 0755)
	os.Mkdir(path.Join(tmpDir, "tokens"), 0755)

	// redefined consts with paths to cert files
	KeyPEMFile = path.Join(tmpDir, "certs/key.pem")
	CertPEMFile = path.Join(tmpDir, "certs/cert.pem")
	RootCAFile = path.Join(tmpDir, "certs/ca.pem")
	AgentTokenFile = path.Join(tmpDir, "tokens/agent-token.txt")

	// 1) just generate
	agentAddr := "addr"
	regenCerts := false
	csrPEM1, fingerprintStr1, err := generateCerts(agentAddr, regenCerts)
	require.NoError(t, err)
	require.NotEmpty(t, csrPEM1)
	require.NotEmpty(t, fingerprintStr1)
	privKeyPEM1, err := ioutil.ReadFile(KeyPEMFile)
	require.NoError(t, err)
	require.NotEmpty(t, privKeyPEM1)

	// 2) generate again, no changes to args, result key should be the same
	csrPEM2, fingerprintStr2, err := generateCerts(agentAddr, regenCerts)
	require.NoError(t, err)
	require.NotEmpty(t, csrPEM2)
	require.NotEmpty(t, fingerprintStr2)
	// CSR is regenerated and its fingerprint too
	require.NotEqualValues(t, csrPEM1, csrPEM2)
	require.NotEqualValues(t, fingerprintStr1, fingerprintStr2)
	// but key in the file is the same
	privKeyPEM2, err := ioutil.ReadFile(KeyPEMFile)
	require.NoError(t, err)
	require.NotEmpty(t, privKeyPEM2)
	require.EqualValues(t, privKeyPEM1, privKeyPEM2)

	// 3) generate again but now regenCerts is true, result should be be different
	regenCerts = true
	csrPEM3, fingerprintStr3, err := generateCerts(agentAddr, regenCerts)
	require.NoError(t, err)
	require.NotEmpty(t, csrPEM3)
	require.NotEmpty(t, fingerprintStr3)
	// CSR is regenerated and its fingerprint too
	require.NotEqualValues(t, csrPEM2, csrPEM3)
	require.NotEqualValues(t, fingerprintStr2, fingerprintStr3)
	// but this time key in the file is different (regenerated)
	privKeyPEM3, err := ioutil.ReadFile(KeyPEMFile)
	require.NoError(t, err)
	require.NotEmpty(t, privKeyPEM3)
	require.NotEqualValues(t, privKeyPEM2, privKeyPEM3)
}
