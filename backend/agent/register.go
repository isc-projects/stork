package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Showmax/go-fqdn"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/term"
	storkutil "isc.org/stork/util"
)

// Prompt user for server token. If user hits enter key then empty
// string is returned.
func promptUserForServerToken() (string, error) {
	fmt.Printf(">>>> Server access token (optional): ")
	serverToken, err := term.ReadPassword(0)
	fmt.Print("\n")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(serverToken)), nil
}

// Get agent's address and port from user if not provided via command line options.
func getAgentAddrAndPortFromUser(agentAddr, agentPort string) (string, int, error) {
	if agentAddr == "" {
		agentAddrTip, err := fqdn.FqdnHostname()
		msg := ">>>> IP address or FQDN of the host with Stork Agent (for the Stork Server connection)"
		if err != nil {
			agentAddrTip = ""
			msg += ": "
		} else {
			msg += fmt.Sprintf(" [%s]: ", agentAddrTip)
		}
		fmt.Print(msg)
		fmt.Scanln(&agentAddr)
		agentAddr = strings.TrimSpace(agentAddr)
		if agentAddr == "" {
			agentAddr = agentAddrTip
		}
	}

	if agentPort == "" {
		fmt.Printf(">>>> Port number that Stork Agent will listen on [8080]: ")
		fmt.Scanln(&agentPort)
		agentPort = strings.TrimSpace(agentPort)
		if agentPort == "" {
			agentPort = "8080"
		}
	}

	agentPortInt, err := strconv.Atoi(agentPort)
	if err != nil {
		log.Errorf("%s is not a valid agent port number: %s", agentPort, err)
		return "", 0, err
	}
	return agentAddr, agentPortInt, nil
}

// Generate or regenerate agent key and CSR (Certificate Signing
// Request). They are generated when they do not exist. They are
// regenerated only if regenCerts is true. If they exist and
// regenCerts is false then they are used.
func generateCSR(certStore *CertStore, agentAddr string, regenKey bool) ([]byte, error) {
	empty, err := certStore.IsEmpty()
	if err != nil {
		// Problem with access to the files.
		return nil, err
	}
	switch {
	case empty:
		log.Info("There are no agent certificates - they will be generated.")
		regenKey = true
	case regenKey:
		log.Info("Forced agent certificates regeneration.")
	default:
		// Special case for the agent that was authorized before 1.15.1.
		// Create an server cert fingerprint file with zero value.
		if ok, err := certStore.IsServerCertFingerprintFileExist(); !ok && err == nil {
			err := certStore.WriteServerCertFingerprint([32]byte{})
			if err != nil {
				return nil, errors.WithMessage(err, "cannot write zero server cert fingerprint")
			}
		}

		if err := certStore.IsValid(); err != nil {
			log.WithError(err).Warn("The agent certificates are invalid - they will be regenerated.")
			regenKey = true
		}
	}

	if regenKey {
		err := certStore.CreateKey()
		if err != nil {
			return nil, errors.WithMessage(err, "cannot generate private key")
		}
	}

	csrPEM, fingerprint, err := certStore.GenerateCSR(agentAddr)
	if err != nil {
		return nil, errors.WithMessage(err, "cannot generate CSR")
	}

	if regenKey {
		// Generate new agent token.
		err := certStore.WriteFingerprintAsToken(fingerprint)
		if err != nil {
			return nil, errors.WithMessage(err, "cannot write agent token")
		}
	}

	return csrPEM, nil
}

// Prepare agent registration request payload to Stork Server in JSON format.
func prepareRegistrationRequestPayload(csrPEM []byte, serverToken, agentToken, agentAddr string, agentPort int, caCertFingerprint [32]byte) (*bytes.Buffer, error) {
	values := map[string]interface{}{
		"address":           agentAddr,
		"agentPort":         agentPort,
		"agentCSR":          string(csrPEM),
		"serverToken":       serverToken,
		"agentToken":        agentToken,
		"caCertFingerprint": storkutil.BytesToHex(caCertFingerprint[:]),
	}
	jsonValue, err := json.Marshal(values)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot marshal registration request")
	}
	return bytes.NewBuffer(jsonValue), nil
}

// Register agent in Stork Server under provided URL using reqPayload in request.
// If retry is true then registration is repeated until it connection to server
// is established. This case is used when agent automatically tries to register
// during startup.
// If the agent is already registered then only ID is returned, the certificates are empty.
func registerAgentInServer(client *HTTPClient, baseSrvURL *url.URL, reqPayload *bytes.Buffer, retry bool) (machineID int64, serverCACert []byte, agentCert []byte, serverCertFingerprint [32]byte, err error) {
	url, _ := baseSrvURL.Parse("api/machines")
	var resp *http.Response
	for {
		resp, err = client.Call(url.String(), reqPayload)
		if err == nil {
			break
		}

		// If connection is refused and retries are enabled than wait for 10 seconds
		// and try again. This method is used in case of agent token based registration
		// to allow smooth automated registration even if server is down for some time.
		// In case of server token based registration this method is invoked manually so
		// it should fail immediately if there is no connection to the server.
		if retry && strings.Contains(err.Error(), "connection refused") {
			log.Println("Sleeping for 10 seconds before next registration attempt")
			time.Sleep(10 * time.Second)
		} else {
			return 0, nil, nil, [32]byte{}, errors.Wrapf(err, "problem registering machine")
		}
	}
	data, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return 0, nil, nil, [32]byte{}, errors.Wrapf(err, "problem reading server's response while registering the machine")
	}

	// Special case - the agent is already registered
	if resp.StatusCode == http.StatusConflict {
		location := resp.Header.Get("Location")
		lastSeparatorIdx := strings.LastIndex(location, "/")
		if lastSeparatorIdx < 0 || lastSeparatorIdx+1 >= len(location) {
			return 0, nil, nil, [32]byte{}, errors.New("missing machine ID in response from server for registration request")
		}
		machineID, err := strconv.Atoi(location[lastSeparatorIdx+1:])
		if err != nil {
			return 0, nil, nil, [32]byte{}, errors.New("bad machine ID in response from server for registration request")
		}
		return int64(machineID), nil, nil, [32]byte{}, nil
	}

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		return 0, nil, nil, [32]byte{}, errors.Wrapf(err, "problem parsing server's response while registering the machine: %v", result)
	}
	errTxt := result["error"]
	if errTxt != nil {
		msg := "Problem registering machine"
		errTxtStr, ok := errTxt.(string)
		if ok {
			msg = fmt.Sprintf("problem registering machine: %s", errTxtStr)
		}
		return 0, nil, nil, [32]byte{}, errors.New(msg)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		errTxt = result["message"]
		var msg string
		if errTxt != nil {
			errTxtStr, ok := errTxt.(string)
			if ok {
				msg = fmt.Sprintf("problem registering machine: %s", errTxtStr)
			} else {
				msg = "problem registering machine"
			}
		} else {
			msg = fmt.Sprintf("problem registering machine: http status code %d", resp.StatusCode)
		}
		return 0, nil, nil, [32]byte{}, errors.New(msg)
	}

	// check received machine ID
	if result["id"] == nil {
		return 0, nil, nil, [32]byte{}, errors.New("missing ID in response from server for registration request")
	}
	machineIDFloat64, ok := result["id"].(float64)
	if !ok {
		return 0, nil, nil, [32]byte{}, errors.New("bad ID in response from server for registration request")
	}
	machineID = int64(machineIDFloat64)

	// check received serverCACert
	if result["serverCACert"] == nil {
		return 0, nil, nil, [32]byte{}, errors.New("missing serverCACert in response from server for registration request")
	}
	serverCACertStr, ok := result["serverCACert"].(string)
	if !ok {
		return 0, nil, nil, [32]byte{}, errors.New("bad serverCACert in response from server for registration request")
	}
	serverCACert = []byte(serverCACertStr)

	// check received agentCert
	if result["agentCert"] == nil {
		return 0, nil, nil, [32]byte{}, errors.New("missing agentCert in response from server for registration request")
	}
	agentCertStr, ok := result["agentCert"].(string)
	if !ok {
		return 0, nil, nil, [32]byte{}, errors.New("bad agentCert in response from server for registration request")
	}
	agentCert = []byte(agentCertStr)

	// Check received Stork server cert fingerprint.
	if result["serverCertFingerprint"] == nil {
		return 0, nil, nil, [32]byte{}, errors.New("missing serverCertFingerprint in response from server for registration request")
	}
	serverCertFingerprintRaw, ok := result["serverCertFingerprint"].(string)
	if !ok {
		return 0, nil, nil, [32]byte{}, errors.New("bad serverCertFingerprint in response from server for registration request")
	}
	serverCertFingerprintBytes := storkutil.HexToBytes(serverCertFingerprintRaw)
	if len(serverCertFingerprintBytes) != 32 {
		return 0, nil, nil, [32]byte{}, errors.New("invalid length of serverCertFingerprint in response from server for registration request")
	}
	serverCertFingerprint = [32]byte(serverCertFingerprintBytes)

	// all ok
	log.Printf("Machine registered")
	return machineID, serverCACert, agentCert, serverCertFingerprint, nil
}

// Check certs received from server.
func checkAndStoreCerts(certStore *CertStore, serverCACert, agentCert []byte, serverCertFingerprint [32]byte) error {
	err := certStore.WriteRootCAPEM(serverCACert)
	if err != nil {
		return errors.WithMessage(err, "cannot write agent cert")
	}

	err = certStore.WriteCertPEM(agentCert)
	if err != nil {
		return errors.WithMessage(err, "cannot write server CA cert")
	}

	err = certStore.WriteServerCertFingerprint(serverCertFingerprint)
	if err != nil {
		return errors.WithMessage(err, "cannot write server cert fingerprint")
	}

	log.Info("Stored agent-signed cert and CA cert")
	return nil
}

// Ping Stork Agent service via Stork Server. It is used during manual registration
// to confirm that TLS connection between agent and server can be established.
func pingAgentViaServer(client *HTTPClient, baseSrvURL *url.URL, machineID int64, serverToken, agentToken string) error {
	urlSuffix := fmt.Sprintf("api/machines/%d/ping", machineID)
	url, err := baseSrvURL.Parse(urlSuffix)
	if err != nil {
		return errors.Wrapf(err, "problem preparing url %s + %s", baseSrvURL.String(), urlSuffix)
	}
	req := map[string]interface{}{
		"serverToken": serverToken,
		"agentToken":  agentToken,
	}
	jsonReq, _ := json.Marshal(req)

	resp, err := client.Call(url.String(), bytes.NewBuffer(jsonReq))
	if err != nil {
		return errors.Wrapf(err, "problem pinging machine")
	}
	data, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return errors.Wrapf(err, "problem reading server's response while pinging machine")
	}
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	// Normally the response is empty so unmarshalling is failing, if it didn't
	// fail it means that there could be some error information.
	if err == nil {
		errTxt := result["error"]
		if errTxt != nil {
			msg := "Problem pinging machine"
			errTxtStr, ok := errTxt.(string)
			if ok {
				msg = fmt.Sprintf("problem pinging machine: %s", errTxtStr)
			}
			return errors.New(msg)
		}
	}
	if resp.StatusCode >= http.StatusBadRequest {
		var msg string
		if result != nil {
			errTxt := result["message"]
			if errTxt != nil {
				errTxtStr, ok := errTxt.(string)
				if ok {
					msg = fmt.Sprintf("problem pinging machine: %s", errTxtStr)
				}
			}
		}
		if msg == "" {
			msg = fmt.Sprintf("problem pinging machine: http status code %d", resp.StatusCode)
		}
		return errors.New(msg)
	}

	log.Printf("Machine ping over TLS: OK")

	return nil
}

// Main function used to register an agent (with a given address and
// port) in a server indicated by given URL. If regenCerts is true
// then agent key and cert are regenerated, otherwise the ones stored
// in files are used. RegenCerts is used when registration is run
// manually. If retry is true then registration is retried if
// connection to server cannot be established. This case is used when
// registration is automatic during agent service startup. Server
// token can be provided in manual registration via command line
// switch. This way the agent will be immediately authorized in the
// server. If server token is empty (in automatic registration or
// when it is not provided in manual registration) then agent is added
// to server but requires manual authorization in web UI.
func Register(serverURL, serverToken, agentAddr, agentPort string, regenCerts bool, retry bool, httpClient *HTTPClient) bool {
	// parse URL to server
	baseSrvURL, err := url.Parse(serverURL)
	if err != nil || baseSrvURL.String() == "" {
		log.WithError(err).Errorf("Cannot parse server URL: %s", serverURL)
		return false
	}

	// Get server token from user (if not provided in cmd line) to authenticate in the server.
	// Do not ask if regenCerts is true (ie. Register is called from agent).
	serverToken2 := serverToken
	if serverToken == "" && regenCerts {
		serverToken2, err = promptUserForServerToken()
		if err != nil {
			log.WithError(err).Error("Problem getting server token")
			return false
		}
	}

	agentAddr, agentPortInt, err := getAgentAddrAndPortFromUser(agentAddr, agentPort)
	if err != nil {
		return false
	}

	certStore := NewCertStoreDefault()

	// Generate agent private key and cert. If they already exist then regenerate them if forced.
	csrPEM, err := generateCSR(certStore, agentAddr, regenCerts)
	if err != nil {
		log.WithError(err).Error("Problem generating certs")
		return false
	}

	agentToken, err := certStore.ReadToken()
	if err != nil {
		log.WithError(err).Error("cannot load the agent token")
		return false
	}

	caCertFingerprint, err := certStore.ReadRootCAFingerprint()
	if err != nil {
		log.WithError(err).Error("cannot load the CA cert fingerprint")
		return false
	}

	// Use cert fingerprint as agent token.
	// Agent token is another mode for checking identity of an agent.
	log.Println("=============================================================================")
	log.Printf("AGENT TOKEN: %s", agentToken)
	log.Println("=============================================================================")

	if serverToken2 == "" {
		log.Println("Authorize the machine in the Stork web UI")
	} else {
		log.Println("Machine will be automatically registered using the server token")
		log.Println("Agent token is printed above for informational purposes only")
		log.Println("User does not need to copy or verify the agent token during registration via the server token")
		log.Println("It will be sent to the server but it is not directly used in this type of machine registration")
	}

	// register new machine i.e. current agent
	reqPayload, err := prepareRegistrationRequestPayload(csrPEM, serverToken2, agentToken, agentAddr, agentPortInt, caCertFingerprint)
	if err != nil {
		log.Errorln(err.Error())
		return false
	}
	log.Println("Try to register agent in Stork Server")
	// If the machine is already registered then the ID is returned. If the
	// agent certificate was signed by the current server CA cert then the
	// server CA cert and agent cert are empty. Otherwise they are returned
	// and should be stored.
	machineID, serverCACert, agentCert, serverCertFingerprint, err := registerAgentInServer(httpClient, baseSrvURL, reqPayload, retry)
	if err != nil {
		log.WithError(err).Error("Problem registering machine")
		return false
	}

	// store certs
	// if server and agent CA certs are empty then the agent should use existing ones
	if serverCACert != nil && agentCert != nil {
		err = checkAndStoreCerts(certStore, serverCACert, agentCert, serverCertFingerprint)
		if err != nil {
			log.WithError(err).Errorf("Problem with certs")
			return false
		}
	}

	if serverToken2 != "" {
		// invoke getting machine state via server
		for i := 1; i < 4; i++ {
			err = pingAgentViaServer(httpClient, baseSrvURL, machineID, serverToken2, agentToken)
			if err == nil {
				break
			}
			if i < 3 {
				log.WithError(err).Errorf("Retrying ping %d/3 due to error", i)
				time.Sleep(2 * time.Duration(i) * time.Second)
			}
		}
		if err != nil {
			log.WithError(err).Errorf("Cannot ping machine")
			return false
		}
	}

	return true
}
