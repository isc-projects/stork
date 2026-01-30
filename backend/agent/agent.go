package agent

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	log "github.com/sirupsen/logrus"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/security/advancedtls"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	// Registers the gzip compression in the internal init() call.
	// The server chooses the compression algorithm based on the client's request.
	_ "google.golang.org/grpc/encoding/gzip"

	"isc.org/stork"
	agentapi "isc.org/stork/api"
	bind9config "isc.org/stork/daemoncfg/bind9"
	keactrl "isc.org/stork/daemonctrl/kea"
	"isc.org/stork/daemondata/kea"
	"isc.org/stork/datamodel/daemonname"
	dnsmodel "isc.org/stork/datamodel/dns"
	"isc.org/stork/pki"
	storkutil "isc.org/stork/util"
)

// Global Stork Agent state.
type StorkAgent struct {
	Host    string
	Port    int
	Monitor Monitor
	// BIND9 HTTP stats client.
	bind9StatsClient *bind9StatsClient
	// PowerDNS webserver client.
	pdnsClient *pdnsClient
	// An HTTP client configuration used to create the HTTP clients for
	// particular Kea daemons.
	// If the agent is registered, it will use the GRPC credentials obtained
	// from the server as TLS client certificate.
	KeaHTTPClientConfig HTTPClientConfig
	server              *grpc.Server
	logTailer           *logTailer
	keaInterceptor      *keaInterceptor
	shutdownOnce        sync.Once
	hookManager         *HookManager

	agentapi.UnimplementedAgentServer
}

// API exposed to Stork Server.
func NewStorkAgent(host string, port int, monitor Monitor, bind9StatsClient *bind9StatsClient, hookManager *HookManager) *StorkAgent {
	logTailer := newLogTailer()

	sa := &StorkAgent{
		Host:             host,
		Port:             port,
		Monitor:          monitor,
		bind9StatsClient: bind9StatsClient,
		pdnsClient:       newPDNSClient(),
		logTailer:        logTailer,
		keaInterceptor:   newKeaInterceptor(),
		hookManager:      hookManager,
	}

	registerKeaInterceptFns(sa)

	return sa
}

// Creates the GRPC server callback using the provided cert store. The callback
// returns the root CA on demand.
func createGetRootCertificatesHandler(certStore *CertStore) func(*advancedtls.ConnectionInfo) (*advancedtls.RootCertificates, error) {
	// Read the latest root CA cert from file for Stork Server's cert verification.
	return func(params *advancedtls.ConnectionInfo) (*advancedtls.RootCertificates, error) {
		certPool, err := certStore.ReadRootCA()
		if err != nil {
			log.WithError(err).Error("Cannot extract root CA")
			return nil, err
		}
		log.Info("Loaded CA cert")
		return &advancedtls.RootCertificates{
			TrustCerts: certPool,
		}, nil
	}
}

// Creates the GRPC server callback using the provided cert store. The callback
// returns the TLS certificate.
func createGetIdentityCertificatesForServerHandler(certStore *CertStore) func(chi *tls.ClientHelloInfo) ([]*tls.Certificate, error) {
	// Read the latest Stork Agent's cert from file for presenting its identity to the Stork server.
	return func(chi *tls.ClientHelloInfo) ([]*tls.Certificate, error) {
		certificate, err := certStore.ReadTLSCert()
		if err != nil {
			log.WithError(err).Error("Could not setup TLS key pair")
			return nil, err
		}
		log.Info("Loaded server cert")
		return []*tls.Certificate{certificate}, nil
	}
}

// Creates the GRPC server callback to perform extra verification of the peer
// certificate.
// The callback is running at the end of client certificate verification.
// Accepts the fingerprint of the GRPC client allowed to establish the
// connection.
//
// Warning: The GRPC library doesn't perform the verification of the
// client IP addresses/DNS names fields.
// The client (Stork server) host is unknown for the agent if the
// standalone register command is used. Other problematic case is
// when the Stork server is running behind the reverse proxy or inside
// the container - the hostname from the agent side (the side that
// verifies the cert) may not match the hostname from the server side
// (certificate issuer and client).
func createVerifyPeer(allowedCertFingerprint [32]byte) advancedtls.PostHandshakeVerificationFunc {
	return func(params *advancedtls.HandshakeVerificationInfo) (*advancedtls.PostHandshakeVerificationResults, error) {
		// The peer must have the extended key usage set.
		if len(params.Leaf.ExtKeyUsage) == 0 {
			return nil, errors.New("peer certificate does not have the extended key usage set")
		}

		// Verify the peer certificate fingerprint. Only single Stork
		// server is allowed to connect.
		actualFingerprint := pki.CalculateFingerprint(params.Leaf)
		if actualFingerprint != allowedCertFingerprint {
			return nil, errors.New("peer certificate fingerprint does not match the allowed one")
		}

		return &advancedtls.PostHandshakeVerificationResults{}, nil
	}
}

// Prepare gRPC server with configured TLS.
func newGRPCServerWithTLS() (*grpc.Server, error) {
	// Prepare structure for advanced TLS. It defines hook functions
	// that dynamically load key and cert from files just before establishing
	// connection. Thanks to this if these files changed in meantime then
	// always latest version for new connections is used.
	// Beside that there is enabled client authentication and forced
	// cert and host verification.
	certStore := NewCertStoreDefault()

	if ok, _ := certStore.IsEmpty(); ok {
		return nil, errors.New("the agent cannot start due to missing " +
			"certificates; consider running 'stork-agent register' to obtain " +
			"certificates or specify the --server-url to do it automatically")
	}
	if err := certStore.IsValid(); err != nil {
		return nil, errors.WithMessage(err,
			"the agent cannot start due to invalid certificates; consider "+
				"running 'stork-agent register' to obtain valid certificates",
		)
	}

	// Only Stork server is allowed to connect to Stork agent over GRPC. This
	// function accepts the fingerprint of the certificate of the allowed GRPC
	// client. This fingerprint is obtained during the registration process.
	allowedCertFingerprint, err := certStore.ReadServerCertFingerprint()
	if err != nil {
		// It should never happen as the cert store is validated a few lines
		// above.
		return nil, err
	}

	options := &advancedtls.Options{
		// Pull latest root CA cert for stork server cert verification.
		RootOptions: advancedtls.RootCertificateOptions{
			// Read the latest root CA cert from file for Stork Server's cert verification.
			GetRootCertificates: createGetRootCertificatesHandler(certStore),
		},
		// Pull latest stork agent cert for presenting its identity to stork server.
		IdentityOptions: advancedtls.IdentityCertificateOptions{
			// Read the latest Stork Agent's cert from file for presenting its identity to the Stork server.
			GetIdentityCertificatesForServer: createGetIdentityCertificatesForServerHandler(certStore),
		},
		// Force stork server cert verification.
		RequireClientCert: true,
		// Check cert and if it matches host IP.
		VerificationType: advancedtls.CertAndHostVerification,
		// Only Stork server is allowed to connect to Stork agent over GRPC
		// and it always uses TLS 1.3.
		MinTLSVersion:              tls.VersionTLS13,
		MaxTLSVersion:              tls.VersionTLS13,
		AdditionalPeerVerification: createVerifyPeer(allowedCertFingerprint),
	}
	creds, err := advancedtls.NewServerCreds(options)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot create server credentials for TLS")
	}

	timeoutOption := grpc.ConnectionTimeout(30 * time.Second)
	srv := grpc.NewServer(grpc.Creds(creds), timeoutOption)
	return srv, nil
}

// Setup the agent as gRPC server endpoint.
func (sa *StorkAgent) SetupGRPCServer() error {
	server, err := newGRPCServerWithTLS()
	if err != nil {
		return err
	}
	sa.server = server
	return nil
}

// Respond to ping request from the server. It assures the server that
// the connection from the server to client is established.
func (sa *StorkAgent) Ping(ctx context.Context, in *agentapi.PingReq) (*agentapi.PingRsp, error) {
	rsp := agentapi.PingRsp{}
	return &rsp, nil
}

// Get state of machine.
func (sa *StorkAgent) GetState(ctx context.Context, in *agentapi.GetStateReq) (*agentapi.GetStateRsp, error) {
	vm, _ := mem.VirtualMemory()
	hostInfo, _ := host.Info()
	load, _ := load.Avg()
	loadStr := fmt.Sprintf("%.2f %.2f %.2f", load.Load1, load.Load5, load.Load15)

	var daemons []*agentapi.Daemon

	for _, daemon := range sa.Monitor.GetDaemons() {
		var accessPoints []*agentapi.AccessPoint
		accessPointTypes := map[string]struct{}{}
		// TODO: Handle multiple access points on the server side.
		// The server expects only one access point with a given type per daemon.
		// This rule is enforced by the database constraints, so handling it
		// requires deep changes in the server code. For now, we share only the
		// first access point for the daemon and use other quietly.
		// See #2237.
		for _, point := range daemon.GetAccessPoints() {
			if _, ok := accessPointTypes[point.Type]; ok {
				continue
			}
			accessPointTypes[point.Type] = struct{}{}

			accessPoints = append(accessPoints, &agentapi.AccessPoint{
				Type:              point.Type,
				Address:           point.Address,
				Port:              point.Port,
				Key:               point.Key,
				Protocol:          string(point.Protocol),
				UseSecureProtocol: point.Protocol.IsSecure(),
			})
		}

		daemons = append(daemons, &agentapi.Daemon{
			Name:         string(daemon.GetName()),
			AccessPoints: accessPoints,
		})
	}

	state := agentapi.GetStateRsp{
		AgentVersion:         stork.Version,
		Daemons:              daemons,
		Hostname:             hostInfo.Hostname,
		Cpus:                 int64(runtime.NumCPU()),
		CpusLoad:             loadStr,
		Memory:               int64(vm.Total / (1024 * 1024 * 1024)), // in GiB
		UsedMemory:           int64(vm.UsedPercent),
		Uptime:               int64(hostInfo.Uptime / (60 * 60 * 24)), // in days
		Os:                   hostInfo.OS,
		Platform:             hostInfo.Platform,
		PlatformFamily:       hostInfo.PlatformFamily,
		PlatformVersion:      hostInfo.PlatformVersion,
		KernelVersion:        hostInfo.KernelVersion,
		KernelArch:           hostInfo.KernelArch,
		VirtualizationSystem: hostInfo.VirtualizationSystem,
		VirtualizationRole:   hostInfo.VirtualizationRole,
		HostID:               hostInfo.HostID,
		Error:                "",
		// This field is not used by the agent. It is here to keep the
		// API backward compatibility. The Stork Server should not rely
		// on this field.
		AgentUsesHTTPCredentials: false,
	}

	return &state, nil
}

// ForwardRndcCommand forwards one rndc command sent by the Stork Server to
// the named daemon.
func (sa *StorkAgent) ForwardRndcCommand(ctx context.Context, in *agentapi.ForwardRndcCommandReq) (*agentapi.ForwardRndcCommandRsp, error) {
	rndcRsp := &agentapi.RndcResponse{
		Status: &agentapi.Status{},
	}
	response := &agentapi.ForwardRndcCommandRsp{
		Status: &agentapi.Status{
			Code: agentapi.Status_OK, // all ok
		},
		RndcResponse: rndcRsp,
	}

	daemon := sa.Monitor.GetDaemonByAccessPoint(AccessPointControl, in.Address, in.Port)
	if daemon == nil {
		rndcRsp.Status.Code = agentapi.Status_ERROR
		rndcRsp.Status.Message = "cannot find BIND 9 daemon"
		response.Status = rndcRsp.Status
		return response, nil
	}

	bind9Daemon := daemon.(*Bind9Daemon)
	if bind9Daemon == nil {
		rndcRsp.Status.Code = agentapi.Status_ERROR
		rndcRsp.Status.Message = fmt.Sprintf("incorrect daemon found: %s instead of BIND 9", daemon.GetName())
		response.Status = rndcRsp.Status
		return response, nil
	}

	request := in.GetRndcRequest()

	// Try to forward the command to rndc.
	output, err := bind9Daemon.sendRNDCCommand(strings.Fields(request.Request))
	if err != nil {
		log.WithError(err).
			WithFields(log.Fields{
				"Address": in.Address,
				"Port":    in.Port,
			}).Errorf("Failed to forward commands to rndc")
		rndcRsp.Status.Code = agentapi.Status_ERROR
		rndcRsp.Status.Message = fmt.Sprintf("failed to forward commands to rndc: %s", err.Error())
	} else {
		rndcRsp.Status.Code = agentapi.Status_OK
		rndcRsp.Response = string(output)
	}

	response.Status = rndcRsp.Status
	return response, nil
}

// ForwardToNamedStats forwards a statistics request to the named daemon.
func (sa *StorkAgent) ForwardToNamedStats(ctx context.Context, in *agentapi.ForwardToNamedStatsReq) (*agentapi.ForwardToNamedStatsRsp, error) {
	innerGrpcResponse := &agentapi.NamedStatsResponse{
		Status: &agentapi.Status{},
	}
	grpcResponse := &agentapi.ForwardToNamedStatsRsp{
		Status: &agentapi.Status{
			Code: agentapi.Status_OK, // all ok
		},
		NamedStatsResponse: innerGrpcResponse,
	}

	var (
		// This parameter is deprecated and was replaced by a set of new parameters that
		// specify address, port and request type. However, we still support this parameter
		// for backward compatibility with older Stork servers.
		reqURL   string
		response httpResponse
		payload  []byte
		err      error
	)
	// The new parameters take precedence over the request URL.
	if in.GetStatsAddress() != "" {
		request := sa.bind9StatsClient.createRequest(in.GetStatsAddress(), in.GetStatsPort())
		var requestType string
	SWITCH_BLOCK:
		switch in.GetRequestType() {
		case agentapi.ForwardToNamedStatsReq_DEFAULT:
			requestType = "default"
			response, payload, err = request.getRawJSON("")
		case agentapi.ForwardToNamedStatsReq_STATUS:
			requestType = "status"
			response, payload, err = request.getRawJSON("status")
		case agentapi.ForwardToNamedStatsReq_SERVER:
			requestType = "server"
			response, payload, err = request.getRawJSON("server")
		case agentapi.ForwardToNamedStatsReq_ZONES:
			requestType = "zones"
			response, payload, err = request.getRawJSON("zones")
		case agentapi.ForwardToNamedStatsReq_NETWORK:
			requestType = "net"
			response, payload, err = request.getRawJSON("net")
		case agentapi.ForwardToNamedStatsReq_MEMORY:
			requestType = "mem"
			response, payload, err = request.getRawJSON("mem")
		case agentapi.ForwardToNamedStatsReq_TRAFFIC:
			requestType = "traffic"
			response, payload, err = request.getRawJSON("traffic")
		case agentapi.ForwardToNamedStatsReq_SERVER_AND_TRAFFIC:
			// This is a special case that requires sending two requests to named
			// and combining the responses into one result. BIND can only return
			// all the required statistics in a single response when the request
			// is sent to the root URL. However, it also returns zones yielding
			// a potentially very large response. To avoid this problem, we send
			// two requests and combine the responses.
			requestType = "server and traffic"
			result := make(map[string]any)
			for resp, e := range request.getCombinedJSON(&result, "server", "traffic") {
				response = resp
				// If there was an error, record it and break.
				if e != nil {
					err = e
					break SWITCH_BLOCK
				}
				// If there was an HTTP error, record the response and break.
				if resp.IsError() {
					break SWITCH_BLOCK
				}
			}
			// Both responses were OK. Need to convert it back to the binary form
			// and return to the Stork server.
			payload, err = json.Marshal(result)
			if err != nil {
				log.WithFields(log.Fields{
					"statsAddress": in.GetStatsAddress(),
					"statsPort":    in.GetStatsPort(),
				}).WithError(err).Error("Failed to marshal the server and traffic stats to return to the Stork server")
				innerGrpcResponse.Status.Code = agentapi.Status_ERROR
				innerGrpcResponse.Status.Message = fmt.Sprintf("failed to marshal the server and traffic stats to return to the Stork server: %s", err.Error())
				return grpcResponse, nil
			}
		}
		// Log depending on whether we got an error or an HTTP error. These
		// messages are logged with stats address, port and request type.
		if err != nil {
			log.WithFields(log.Fields{
				"statsAddress": in.GetStatsAddress(),
				"statsPort":    in.GetStatsPort(),
				"requestType":  requestType,
			}).WithError(err).Error("Failed to forward commands to named over the stats channel")
		} else if response.IsError() {
			log.WithFields(log.Fields{
				"statsAddress": in.GetStatsAddress(),
				"statsPort":    in.GetStatsPort(),
				"requestType":  requestType,
				"status":       response.StatusCode(),
			}).Errorf("named stats channel returned error status code with message: %s", response.String())
		}
	} else {
		// The request uses deprecated URL parameter and lacks the new parameters.
		//nolint:staticcheck
		reqURL = in.GetUrl()
		response, payload, err = sa.bind9StatsClient.createRequestFromURL(reqURL).getRawJSON("/")

		// Log depending on whether we got an error or an HTTP error. These
		// messages are logged with URL.
		if err != nil {
			log.WithFields(log.Fields{
				"url": reqURL,
			}).WithError(err).Error("Failed to forward commands to named over the stats channel")
		} else if response.IsError() {
			log.WithFields(log.Fields{
				"url": reqURL,
			}).Errorf("named stats channel returned error status code with message: %s", response.String())
		}
	}

	// Set the status code and message based on the error or response.
	switch {
	case err != nil:
		innerGrpcResponse.Status.Code = agentapi.Status_ERROR
		innerGrpcResponse.Status.Message = fmt.Sprintf("failed to forward commands to named over the stats channel: %s", err.Error())
	case response.IsError():
		innerGrpcResponse.Status.Code = agentapi.Status_ERROR
		innerGrpcResponse.Status.Message = fmt.Sprintf("named stats channel returned error status code with message: %s", response.String())
	default:
		// Everything looks good, so include the body in the response.
		innerGrpcResponse.Status.Code = agentapi.Status_OK
	}
	innerGrpcResponse.Response = string(payload)
	return grpcResponse, nil
}

// Returns general information about the PowerDNS server. It uses the
// PowerDNS REST API to retrieve this information from the /api/v1/servers/localhost
// endpoint.
func (sa *StorkAgent) GetPowerDNSServerInfo(ctx context.Context, req *agentapi.GetPowerDNSServerInfoReq) (*agentapi.GetPowerDNSServerInfoRsp, error) {
	daemon := sa.Monitor.GetDaemonByAccessPoint(AccessPointControl, req.WebserverAddress, req.WebserverPort)
	if daemon == nil {
		st := status.Newf(codes.FailedPrecondition, "PowerDNS server %s:%d not found", req.WebserverAddress, req.WebserverPort)
		ds, err := st.WithDetails(&errdetails.ErrorInfo{
			Reason: "DAEMON_NOT_FOUND",
		})
		if err != nil {
			// If this unlikely error occurs, it is better to return the original
			// error.
			return nil, st.Err()
		}
		return nil, ds.Err()
	}

	// The API key is required to access the PowerDNS REST API.
	accessPoint := daemon.GetAccessPoint(AccessPointControl)
	if accessPoint == nil || accessPoint.Key == "" {
		st := status.Newf(codes.FailedPrecondition, "API key not configured for PowerDNS server %s:%d", req.WebserverAddress, req.WebserverPort)
		ds, err := st.WithDetails(&errdetails.ErrorInfo{
			Reason: "API_KEY_NOT_CONFIGURED",
		})
		if err != nil {
			return nil, st.Err()
		}
		return nil, ds.Err()
	}

	// Use the PowerDNS REST API to retrieve the server information.
	response, serverInfo, err := sa.pdnsClient.getCombinedServerInfo(accessPoint.Key, req.WebserverAddress, req.WebserverPort)
	if err != nil {
		// If there is an error, the PowerDNS server is most likely unavailable.
		st := status.New(codes.Unavailable, err.Error())
		return nil, st.Err()
	}

	if response.IsError() {
		// Communication successful but HTTP error code returned. There are many
		// different kinds of errors that the server can return. The response
		// text contains the details. It doesn't make much sense to check for the
		// exact error codes. The client can detect that it is the REST API error
		// by checking the gRPC unknown status code.
		st := status.New(codes.Unknown, response.String())
		return nil, st.Err()
	}

	// Response is OK. Return the server information.
	rsp := &agentapi.GetPowerDNSServerInfoRsp{
		Type:             serverInfo.Type,
		Id:               serverInfo.ID,
		DaemonType:       serverInfo.DaemonType,
		Version:          serverInfo.Version,
		Url:              serverInfo.URL,
		ConfigURL:        serverInfo.ConfigURL,
		ZonesURL:         serverInfo.ZonesURL,
		AutoprimariesURL: serverInfo.AutoprimariesURL,
		Uptime:           serverInfo.Uptime,
	}
	return rsp, nil
}

// Forwards one or more Kea commands sent by the Stork Server to the appropriate Kea instance over
// HTTP (via Control Agent).
func (sa *StorkAgent) ForwardToKeaOverHTTP(ctx context.Context, in *agentapi.ForwardToKeaOverHTTPReq) (*agentapi.ForwardToKeaOverHTTPRsp, error) {
	// Call hook
	if err := sa.hookManager.OnBeforeForwardToKeaOverHTTP(ctx, in); err != nil {
		return nil, err
	}

	// prepare base response
	response := &agentapi.ForwardToKeaOverHTTPRsp{
		Status: &agentapi.Status{
			Code: agentapi.Status_OK, // all ok
		},
	}

	// check URL to CA
	reqURL := in.GetUrl()
	if reqURL == "" {
		response.Status.Code = agentapi.Status_ERROR
		response.Status.Message = "incorrect URL to Kea CA"
		return response, nil
	}

	logFields := log.Fields{
		"url": reqURL,
	}

	host, port, _ := storkutil.ParseURL(reqURL)
	var daemon *keaDaemon
	if unknownDaemon := sa.Monitor.GetDaemonByAccessPoint(AccessPointControl, host, port); unknownDaemon != nil {
		logFields["daemon"] = unknownDaemon.String()
		daemon = unknownDaemon.(*keaDaemon)
		if daemon == nil {
			log.WithFields(logFields).Warn("daemon found is not a Kea daemon")
			response.Status.Code = agentapi.Status_ERROR
			response.Status.Message = "incorrect URL to Kea CA"
			return response, nil
		}
	}

	// forward requests to kea one by one
	for _, keaRequest := range in.GetKeaRequests() {
		grpcResponse := &agentapi.KeaResponse{
			Status: &agentapi.Status{},
		}

		// Handle unknown daemon as a Kea request error, not as a general error
		// because it is a problem on the Kea side or related to the Kea daemon.
		if daemon == nil {
			log.WithFields(logFields).Warn("Cannot find Kea daemon")
			grpcResponse.Status.Code = agentapi.Status_ERROR
			grpcResponse.Status.Message = "cannot find Kea daemon"
			response.KeaResponses = append(response.KeaResponses, grpcResponse)
			continue
		}

		// Unmarshal only the command header to find out the command name.
		// The arguments are kept in raw form.
		var keaCommand keactrl.CommandWithRawArguments
		if err := json.Unmarshal([]byte(keaRequest.Request), &keaCommand); err != nil {
			log.WithFields(logFields).WithError(err).Error("Failed to parse Kea request")
			grpcResponse.Status.Code = agentapi.Status_ERROR
			grpcResponse.Status.Message = fmt.Sprintf("failed to parse Kea request: %s", err.Error())
			response.KeaResponses = append(response.KeaResponses, grpcResponse)
			continue
		}
		var keaResponse keactrl.Response

		// Try to forward the command to Kea Control Agent.
		err := daemon.sendCommand(ctx, &keaCommand, &keaResponse)
		if err != nil {
			log.WithError(err).WithFields(logFields).Errorf("Failed to forward commands to Kea")
			grpcResponse.Status.Code = agentapi.Status_ERROR
			grpcResponse.Status.Message = fmt.Sprintf("failed to forward commands to Kea: %s", err.Error())
			response.KeaResponses = append(response.KeaResponses, grpcResponse)
			continue
		}

		// Push Kea response for synchronous processing. It may modify the
		// response body.
		keaResponse, err = sa.keaInterceptor.syncHandle(sa, keaCommand, keaResponse)
		if err != nil {
			log.WithFields(logFields).WithError(err).Error("Failed to apply synchronous interceptors on Kea response")
			grpcResponse.Status.Code = agentapi.Status_ERROR
			grpcResponse.Status.Message = fmt.Sprintf("failed to postprocess response from Kea: %s", err.Error())
			response.KeaResponses = append(response.KeaResponses, grpcResponse)
			continue
		}

		// Push Kea response for async processing. It is done in background.
		// One of the use cases is to extract log files used by Kea and to
		// allow the log viewer to access them.
		go sa.keaInterceptor.asyncHandle(sa, keaCommand, keaResponse)

		body, err := json.Marshal(keaResponse)
		if err != nil {
			log.WithFields(logFields).WithError(err).Error("Failed to marshal Kea response")
			grpcResponse.Status.Code = agentapi.Status_ERROR
			grpcResponse.Status.Message = fmt.Sprintf("failed to marshal Kea response: %s", err.Error())
			response.KeaResponses = append(response.KeaResponses, grpcResponse)
			continue
		}

		grpcResponse.Response = body
		grpcResponse.Status.Code = agentapi.Status_OK
		response.KeaResponses = append(response.KeaResponses, grpcResponse)
	}

	return response, nil
}

// Returns the tail of the specified file, typically a log file.
func (sa *StorkAgent) TailTextFile(ctx context.Context, in *agentapi.TailTextFileReq) (*agentapi.TailTextFileRsp, error) {
	response := &agentapi.TailTextFileRsp{
		Status: &agentapi.Status{
			Code: agentapi.Status_OK, // all ok
		},
	}

	lines, err := sa.logTailer.tail(in.Path, in.Offset)
	if err != nil {
		response.Status.Code = agentapi.Status_ERROR
		response.Status.Message = fmt.Sprintf("%s", err)
		return response, nil
	}
	response.Lines = lines

	return response, nil
}

// Generate a streaming response returning DNS zones from a specified
// agent. The response can be filtered or unfiltered, depending on the
// request.
func (sa *StorkAgent) ReceiveZones(req *agentapi.ReceiveZonesReq, server grpc.ServerStreamingServer[agentapi.Zone]) error {
	daemon := sa.Monitor.GetDaemonByAccessPoint(AccessPointControl, req.ControlAddress, req.ControlPort)
	if daemon == nil {
		return status.New(codes.NotFound, fmt.Sprintf("DNS daemon not found at %s:%d", req.ControlAddress, req.ControlPort)).Err()
	}

	dnsDaemon, ok := daemon.(dnsDaemon)
	if !ok {
		// This is rather an exceptional case, so we don't necessarily need to
		// include the detailed error message.
		return status.New(
			codes.InvalidArgument,
			fmt.Sprintf("attempted to receive DNS zones from an unsupported daemon: %s", daemon.GetName()),
		).Err()
	}

	inventory := dnsDaemon.getZoneInventory()
	if inventory == nil {
		// This is also an exceptional case. All DNS servers should have the
		// zone inventory initialized.
		return status.New(codes.FailedPrecondition, "attempted to receive DNS zones from a daemon for which zone inventory was not instantiated").Err()
	}

	if req.ForcePopulate {
		// Refresh the zone inventory if requested by the Stork server. This
		// operation will block until the zone inventory is populated.
		ch, err := inventory.populate(true)
		if err != nil {
			return status.Error(codes.Internal, err.Error())
		}
		if ch != nil {
			s := <-ch
			if s.err != nil {
				return status.Error(codes.Internal, s.err.Error())
			}
		}
	}

	// Set filtering rules based on the request.
	var filter *dnsmodel.ZoneFilter
	if req.ViewName != "" || req.Limit > 0 || req.LoadedAfter > 0 || req.LowerBound != "" {
		filter = dnsmodel.NewZoneFilter()
		if req.ViewName != "" {
			filter.SetView(req.ViewName)
		}
		if req.LowerBound != "" && req.Limit > 0 {
			filter.SetLowerBound(req.LowerBound, int(req.Limit))
		}
		if req.LoadedAfter > 0 {
			filter.SetLoadedAfter(time.Unix(req.LoadedAfter, 0))
		}
	}
	var (
		notInitedError *zoneInventoryNotInitedError
		busyError      *zoneInventoryBusyError
	)
	zones, err := inventory.receiveZones(context.Background(), filter)
	if err != nil {
		// Some of the errors require special handling so the client can
		// interpret them and take specific actions (e.g., try later).
		switch {
		case errors.As(err, &notInitedError):
			st := status.New(codes.FailedPrecondition, err.Error())
			ds, err := st.WithDetails(&errdetails.ErrorInfo{
				Reason: "ZONE_INVENTORY_NOT_INITED",
			})
			if err != nil {
				return st.Err()
			}
			return ds.Err()
		case errors.As(err, &busyError):
			st := status.New(codes.Unavailable, err.Error())
			ds, err := st.WithDetails(&errdetails.ErrorInfo{
				Reason: "ZONE_INVENTORY_BUSY",
			})
			if err != nil {
				return st.Err()
			}
			return ds.Err()
		default:
			return status.Error(codes.Internal, err.Error())
		}
	}
	// Return the zones over the channel.
	for result := range zones {
		if result.err == nil {
			zone := result.zone
			apiZone := &agentapi.Zone{
				Name:           zone.Name(),
				Class:          zone.Class,
				Serial:         zone.Serial,
				Type:           zone.Type,
				Loaded:         zone.Loaded.Unix(),
				View:           zone.ViewName,
				Rpz:            zone.RPZ,
				TotalZoneCount: zone.TotalZoneCount,
			}
			err = server.Send(apiZone)
			if err != nil {
				st := status.New(codes.Aborted, err.Error())
				return st.Err()
			}
		}
	}
	return nil
}

// Generate a streaming response returning DNS zone RRs from a specified agent.
func (sa *StorkAgent) ReceiveZoneRRs(req *agentapi.ReceiveZoneRRsReq, server grpc.ServerStreamingServer[agentapi.ReceiveZoneRRsRsp]) error {
	daemon := sa.Monitor.GetDaemonByAccessPoint(AccessPointControl, req.ControlAddress, req.ControlPort)
	if daemon == nil {
		return status.New(codes.NotFound, fmt.Sprintf("DNS daemon not found at %s:%d", req.ControlAddress, req.ControlPort)).Err()
	}

	dnsDaemon, ok := daemon.(dnsDaemon)
	if !ok {
		// This is rather an exceptional case, so we don't necessarily need to
		// include the detailed error message.
		return status.New(
			codes.InvalidArgument,
			fmt.Sprintf("attempted to receive DNS zones from an unsupported daemon: %s", daemon.GetName()),
		).Err()
	}

	inventory := dnsDaemon.getZoneInventory()
	if inventory == nil {
		// This is also an exceptional case. All DNS servers should have the
		// zone inventory initialized.
		return status.New(codes.FailedPrecondition, "attempted to receive DNS zone RRs from a daemon for which zone inventory was not instantiated").Err()
	}
	respChan, err := inventory.requestAXFR(req.ZoneName, req.ViewName)
	if err != nil {
		// This error most likely indicates that the zone inventory was unable to
		// find credentials in the DNS server configuration. This may be due to a
		// an issue with interpretation of the configuration file or lack of it.
		return status.Error(codes.Internal, err.Error())
	}
	for resp := range respChan {
		if resp.err != nil {
			var (
				notInitedError *zoneInventoryNotInitedError
				busyError      *zoneInventoryAXFRBusyError
			)
			// Some of the errors require special handling so the client can
			// interpret them and take specific actions (e.g., try later).
			switch {
			case errors.As(resp.err, &notInitedError):
				// The zone inventory was not initialized, so it is unaware of the
				// configured zones. It may require explicitly initializing the
				// inventory or restarting the agent.
				st := status.New(codes.FailedPrecondition, resp.err.Error())
				ds, err := st.WithDetails(&errdetails.ErrorInfo{
					Reason: "ZONE_INVENTORY_NOT_INITED",
				})
				if err != nil {
					return st.Err()
				}
				return ds.Err()
			case errors.As(resp.err, &busyError):
				// The zone inventory is busy populating or loading the zones.
				// The client may retry later when this work is finished.
				st := status.New(codes.Unavailable, resp.err.Error())
				ds, err := st.WithDetails(&errdetails.ErrorInfo{
					Reason: "ZONE_INVENTORY_BUSY",
				})
				if err != nil {
					return st.Err()
				}
				return ds.Err()
			default:
				// There was some other error. Most likely something has gone wrong
				// during the zone transfer. Maybe the connection was lost with the
				// DNS server.
				return status.Error(codes.Aborted, resp.err.Error())
			}
		}
		if resp.envelope.Error != nil {
			// An error occurred during the zone transfer - maybe connection was lost.
			return status.Error(codes.Aborted, resp.envelope.Error.Error())
		}
		var rrs []string
		for _, rr := range resp.envelope.RR {
			rrs = append(rrs, rr.String())
		}
		// Everything looks good, so send the response.
		rsp := &agentapi.ReceiveZoneRRsRsp{
			Rrs: rrs,
		}
		err = server.Send(rsp)
		if err != nil {
			return status.Error(codes.Aborted, err.Error())
		}
	}
	return nil
}

// Convenience function receiving BIND 9 configuration from a specified server
// for a specified file type.
func receiveBind9Config(fileType agentapi.Bind9ConfigFileType, bind9Config *bind9config.Config, req *agentapi.ReceiveBind9ConfigReq, server grpc.ServerStreamingServer[agentapi.ReceiveBind9ConfigRsp]) (bool, error) {
	if bind9Config == nil || (req.FileSelector != nil && len(req.FileSelector.FileTypes) > 0 && !slices.Contains(req.FileSelector.FileTypes, fileType)) {
		return false, nil
	}
	file := &agentapi.ReceiveBind9ConfigFile{
		FileType:   fileType,
		SourcePath: bind9Config.GetSourcePath(),
	}
	err := server.Send(&agentapi.ReceiveBind9ConfigRsp{
		Response: &agentapi.ReceiveBind9ConfigRsp_File{
			File: file,
		},
	})
	if err != nil {
		return true, status.Error(codes.Aborted, err.Error())
	}
	for text, err := range bind9Config.GetFormattedTextIterator(0, bind9config.NewFilterFromProto(req.Filter)) {
		if err != nil {
			return true, status.Error(codes.Aborted, err.Error())
		}
		err := server.Send(&agentapi.ReceiveBind9ConfigRsp{
			Response: &agentapi.ReceiveBind9ConfigRsp_Line{
				Line: text,
			},
		})
		if err != nil {
			return true, status.Error(codes.Aborted, err.Error())
		}
	}
	return true, nil
}

// Generate a streaming response returning BIND 9 configuration for the specified
// server with filtering. The response may contain multiple files. Each new file
// is marked with a preamble containing the file type and source path. The subsequent
// lines contain the contents of the file.
func (sa *StorkAgent) ReceiveBind9Config(req *agentapi.ReceiveBind9ConfigReq, server grpc.ServerStreamingServer[agentapi.ReceiveBind9ConfigRsp]) (err error) {
	daemon := sa.Monitor.GetDaemonByAccessPoint(AccessPointControl, req.ControlAddress, req.ControlPort)
	if daemon == nil {
		return status.Newf(codes.FailedPrecondition, "BIND 9 server %s:%d not found", req.ControlAddress, req.ControlPort).Err()
	}
	bind9Daemon, ok := daemon.(*Bind9Daemon)
	if !ok {
		return status.Newf(codes.InvalidArgument, "attempted to get BIND 9 configuration from daemon %s instead of BIND 9", daemon.GetName()).Err()
	}
	var (
		configReceived  bool
		rndcKeyReceived bool
	)
	if configReceived, err = receiveBind9Config(agentapi.Bind9ConfigFileType_CONFIG, bind9Daemon.bind9Config, req, server); err != nil {
		return
	}
	if rndcKeyReceived, err = receiveBind9Config(agentapi.Bind9ConfigFileType_RNDC_KEY, bind9Daemon.rndcKeyConfig, req, server); err != nil {
		return
	}
	if !configReceived && !rndcKeyReceived {
		err = status.Errorf(codes.NotFound, "BIND 9 configuration not found for server %s:%d", req.ControlAddress, req.ControlPort)
	}
	return
}

// Serialize a Lease to Protobuf, then write its length as a network-order uint16 followed by the serialized lease to the provided writer.
//
// This function exists because the Protobuf manual recommends against creating
// very large Protobufs.  I think I have correctly understood and implemented
// their recommendation.
// https://protobuf.dev/programming-guides/techniques/#large-data
func writeLease(writer io.Writer, sizeBuf []byte, lease *keadata.Lease) error {
	grpcLease := lease.ToGRPC()
	leaseBin, err := proto.Marshal(&grpcLease)
	if err != nil {
		return errors.New("unable to serialize the provided lease to protobuf")
	}
	leaseBinLen := len(leaseBin)
	if leaseBinLen > (1 << 16) {
		return errors.New("Serialized lease data is too large for the wire format (> 2^16 bytes long)")
	}
	if leaseBinLen <= 0 {
		return errors.New("Serialized lease data is length 0?")
	}
	binary.BigEndian.PutUint16(sizeBuf, uint16(len(leaseBin)))
	writer.Write(sizeBuf)
	writer.Write(leaseBin)
	return nil
}

// Return a point-in-time snapshot of all the active leases that all monitored Kea daemons know about.
func (sa *StorkAgent) GetKeaLeases(ctx context.Context, req *agentapi.GetKeaLeasesReq) (*agentapi.GetKeaLeasesRsp, error) {
	daemons := sa.Monitor.GetDaemons()
	leaseCount := 0
	var encodedLeases bytes.Buffer
	for _, daemon := range daemons {
		name := daemon.GetName()
		switch name {
		case daemonname.DHCPv4:
			fallthrough
		case daemonname.DHCPv6:
			keadaemon, ok := daemon.(*keaDaemon)
			if !ok {
				log.Info("Answering GetKeaLeases; found a daemon with DHCP daemonname, but which could not be cast to keaDaemon")
				continue
			}
			recvSizeBuf := make([]byte, 2)
			leases := keadaemon.GetLeaseSnapshot()
			leaseCount += len(leases)
			for _, lease := range leases {
				err := writeLease(&encodedLeases, recvSizeBuf, lease)
				if err != nil {
					log.WithError(err).Warn("could not write lease to encoded leases buffer")
					leaseCount -= 1
					continue
				}
			}
		default:
			continue
		}
	}
	if leaseCount >= (1 << 32) {
		return nil, errors.New("Too many leases: the lease snapshot contains more than 4 billion leases, which is too many for this API to handle.")
	}
	response := agentapi.GetKeaLeasesRsp{
		LeaseCount:   uint32(leaseCount),
		LeasesPacked: encodedLeases.Bytes(),
	}
	return &response, nil
}

// Starts the gRPC and HTTP listeners.
func (sa *StorkAgent) Serve() error {
	// Install gRPC API handlers.
	agentapi.RegisterAgentServer(sa.server, sa)

	// Prepare listener on configured address.
	addr := net.JoinHostPort(sa.Host, strconv.Itoa(sa.Port))
	listenConfig := &net.ListenConfig{}
	lis, err := listenConfig.Listen(context.Background(), "tcp", addr)
	if err != nil {
		return errors.Wrapf(err, "failed to listen on: %s", addr)
	}

	// Start serving gRPC
	log.WithFields(log.Fields{
		"address": lis.Addr(),
	}).Infof("Started serving Stork Agent")
	if err := sa.server.Serve(lis); err != nil {
		return errors.Wrapf(err, "failed to serve on: %s", addr)
	}
	return nil
}

// Shuts down Stork Agent. The reload flag indicates if the Shutdown is called
// as part of the agent reload (reload=true) or the process is terminating
// (reload=false).
func (sa *StorkAgent) Shutdown(reload bool) {
	sa.shutdownOnce.Do(func() {
		if !reload {
			log.Info("Stopping Stork Agent")
		}
		err := sa.hookManager.Close()
		if err != nil {
			log.
				WithError(err).
				Error("Closing Hook Manager failed")
		}

		if sa.server != nil {
			sa.server.GracefulStop()
		}
	})
}

// Allows access to the specific log file by the log viewer.
func (sa *StorkAgent) allowLog(path string) {
	sa.logTailer.allow(path)
}
