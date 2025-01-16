package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	agentapi "isc.org/stork/api"
	storkutil "isc.org/stork/util"
)

// The GRPC handler used in the registration with the server token to respond
// to the ping request. It doesn't support other GRPC methods.
type grpcPingHandler struct {
	agentapi.UnimplementedAgentServer
}

// Respond to ping request from the server. It assures the server that the
// connection from the server to client is established. It is used in server
// token registration procedure.
func (grpcPingHandler) Ping(ctx context.Context, in *agentapi.PingReq) (*agentapi.PingRsp, error) {
	rsp := agentapi.PingRsp{}
	return &rsp, nil
}

// Starts the GRPC server to handle the ping request from the Stork server.
// It doesn't support other operations.
// Returns the function that must be called to stop the server or an error.
func runPingGRPCServer(host string, port int) (func(), error) {
	server, err := newGRPCServerWithTLS()
	if err != nil {
		err = errors.WithMessage(err, "cannot setup the GRPC server")
		return nil, err
	}

	handler := &grpcPingHandler{}

	// Install gRPC API handlers.
	agentapi.RegisterAgentServer(server, handler)

	// Prepare listener on configured address.
	addr := net.JoinHostPort(host, fmt.Sprint(port))
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		if errors.Is(err, unix.EADDRINUSE) {
			err = errors.Wrapf(err,
				"agent registration using the server token requires that the "+
					"existing Stork agent instances are stopped; Stork agent "+
					"detected a program bound to port %d; if it is a Stork agent "+
					"instance, please shut it down before attempting the "+
					"re-registration; if it is a different program, please "+
					"configure the Stork agent to use an available port", port)
		} else {
			err = errors.Wrap(err, "cannot setup the GRPC listener")
		}

		return nil, err
	}

	go func() {
		if err := server.Serve(listener); err != nil {
			log.WithError(err).Errorf("failed to serve on: %s", addr)
		}
	}()

	return server.GracefulStop, nil
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
		// Create a server cert fingerprint file with zero value.
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
func registerAgentInServer(client *httpClient, baseSrvURL *url.URL, reqPayload *bytes.Buffer, retry bool) (machineID int64, serverCACert []byte, agentCert []byte, serverCertFingerprint [32]byte, err error) {
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
func pingAgentViaServer(client *httpClient, baseSrvURL *url.URL, machineID int64, serverToken, agentToken string) error {
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
// registration is automatic during agent service startup.
// If the server token is provided, the agent will be immediately authorized
// in the server. If server token is empty (in automatic registration or
// when it is not provided in manual registration) then agent is added to
// server but requires manual authorization in web UI.
func Register(serverURL, serverToken, agentHost string, agentPort int, regenCerts bool, retry bool, httpClient *httpClient) error {
	// parse URL to server
	baseSrvURL, err := url.Parse(serverURL)
	if err != nil {
		return errors.Wrapf(err, "cannot parse server URL: %s", serverURL)
	} else if baseSrvURL.String() == "" {
		return errors.Errorf("server URL is empty")
	}

	certStore := NewCertStoreDefault()

	// Generate agent private key and cert. If they already exist then regenerate them if forced.
	csrPEM, err := generateCSR(certStore, agentHost, regenCerts)
	if err != nil {
		return errors.WithMessage(err, "problem generating certs")
	}

	agentToken, err := certStore.ReadToken()
	if err != nil {
		return errors.WithMessage(err, "cannot load the agent token")
	}

	caCertFingerprint, err := certStore.ReadRootCAFingerprint()
	if err != nil {
		return errors.WithMessage(err, "cannot load the CA cert fingerprint")
	}

	// Use cert fingerprint as agent token.
	// Agent token is another mode for checking identity of an agent.
	log.Println("=============================================================================")
	log.Printf("AGENT TOKEN: %s", agentToken)
	log.Println("=============================================================================")

	if serverToken == "" {
		log.Println("Authorize the machine in the Stork web UI")
	} else {
		log.Println("Machine will be automatically registered using the server token")
		log.Println("Agent token is printed above for informational purposes only")
		log.Println("User does not need to copy or verify the agent token during registration via the server token")
		log.Println("It will be sent to the server but it is not directly used in this type of machine registration")
	}

	// register new machine i.e. current agent
	reqPayload, err := prepareRegistrationRequestPayload(csrPEM, serverToken, agentToken, agentHost, agentPort, caCertFingerprint)
	if err != nil {
		return errors.WithMessage(err, "cannot prepare the registration request")
	}
	log.Println("Try to register agent in Stork Server")
	// If the machine is already registered then the ID is returned. If the
	// agent certificate was signed by the current server CA cert then the
	// server CA cert and agent cert are empty. Otherwise they are returned
	// and should be stored.
	machineID, serverCACert, agentCert, serverCertFingerprint, err := registerAgentInServer(httpClient, baseSrvURL, reqPayload, retry)
	if err != nil {
		return errors.WithMessage(err, "problem registering machine")
	}

	// store certs
	// if server and agent CA certs are empty then the agent should use existing ones
	if serverCACert != nil && agentCert != nil {
		err = checkAndStoreCerts(certStore, serverCACert, agentCert, serverCertFingerprint)
		if err != nil {
			return errors.WithMessage(err, "problem with certs")
		}
	}

	if serverToken != "" {
		// Start the listener to handle the ping request.
		teardown, err := runPingGRPCServer(agentHost, agentPort)
		if err != nil {
			return errors.WithMessage(err, "cannot run the GRPC server to handle Ping")
		}
		defer teardown()

		// Invoke getting machine state via server.
		for i := 1; i < 4; i++ {
			err = pingAgentViaServer(httpClient, baseSrvURL, machineID, serverToken, agentToken)
			if err == nil {
				break
			}
			if i < 3 {
				log.WithError(err).Errorf("Retrying ping %d/3 due to error", i)
				time.Sleep(2 * time.Duration(i) * time.Second)
			}
		}
		if err != nil {
			return errors.WithMessage(err, "cannot ping machine")
		}
	}

	return nil
}
