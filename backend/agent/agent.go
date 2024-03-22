package agent

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/security/advancedtls"

	// Registers the gzip compression in the internal init() call.
	// The server chooses the compression algorithm based on the client's request.
	_ "google.golang.org/grpc/encoding/gzip"

	"isc.org/stork"
	agentapi "isc.org/stork/api"
	"isc.org/stork/pki"
)

// Global Stork Agent state.
type StorkAgent struct {
	Host       string
	Port       int
	AppMonitor AppMonitor
	// General-purpose HTTP client. It doesn't use any app-specific features.
	GeneralHTTPClient *HTTPClient
	// To communicate with Kea Control Agent.
	// It may contain the HTTP credentials.
	// If the agent is registered, it will use the GRPC credentials obtained
	// from the server as TLS client certificate.
	KeaHTTPClient  *HTTPClient
	server         *grpc.Server
	logTailer      *logTailer
	keaInterceptor *keaInterceptor
	shutdownOnce   sync.Once
	hookManager    *HookManager

	agentapi.UnimplementedAgentServer
}

// API exposed to Stork Server.
func NewStorkAgent(host string, port int, appMonitor AppMonitor, httpClient, keaHTTPClient *HTTPClient, hookManager *HookManager) *StorkAgent {
	logTailer := newLogTailer()

	sa := &StorkAgent{
		Host:              host,
		Port:              port,
		AppMonitor:        appMonitor,
		GeneralHTTPClient: httpClient,
		KeaHTTPClient:     keaHTTPClient,
		logTailer:         logTailer,
		keaInterceptor:    newKeaInterceptor(),
		hookManager:       hookManager,
	}

	registerKeaInterceptFns(sa)

	return sa
}

// Creates the GRPC server callback using the provided cert store. The callback
// returns the root CA on demand.
func createGetRootCertificatesHandler(certStore *CertStore) func(*advancedtls.GetRootCAsParams) (*advancedtls.GetRootCAsResults, error) {
	// Read the latest root CA cert from file for Stork Server's cert verification.
	return func(params *advancedtls.GetRootCAsParams) (*advancedtls.GetRootCAsResults, error) {
		certPool, err := certStore.ReadRootCA()
		if err != nil {
			log.WithError(err).Error("Cannot extract root CA")
			return nil, err
		}
		log.Info("Loaded CA cert")
		return &advancedtls.GetRootCAsResults{
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
func createVerifyPeer(allowedCertFingerprint [32]byte) advancedtls.CustomVerificationFunc {
	return func(params *advancedtls.VerificationFuncParams) (*advancedtls.VerificationResults, error) {
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

		return &advancedtls.VerificationResults{}, nil
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

	// Only Stork server is allowed to connect to Stork agent over GRPC. This
	// function accepts the fingerprint of the certificate of the allowed GRPC
	// client. This fingerprint is obtained during the registration process.
	allowedCertFingerprint, err := certStore.ReadServerCertFingerprint()
	if err != nil {
		// TODO: The agent should not start if the certificates are not
		// obtained as nobody is allowed to connect. But the main function and
		// many tests rely on the fact that the agent can be always started.
		log.WithError(err).Warning("Cannot read the cert fingerprint of the " +
			"Stork server; the server will not be able to connect to the agent over GRPC")
		allowedCertFingerprint = [32]byte{}
	}

	options := &advancedtls.ServerOptions{
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
		VType: advancedtls.CertAndHostVerification,
		// Only Stork server is allowed to connect to Stork agent over GRPC
		// and it always uses TLS 1.3.
		MinVersion: tls.VersionTLS13,
		MaxVersion: tls.VersionTLS13,
		VerifyPeer: createVerifyPeer(allowedCertFingerprint),
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
func (sa *StorkAgent) Setup() error {
	server, err := newGRPCServerWithTLS()
	if err != nil {
		return err
	}
	sa.server = server
	return nil
}

// Respond to ping request from the server. It assures the server that
// the connection from the server to client is established. It is used
// in server token registration procedure.
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

	var apps []*agentapi.App
	for _, app := range sa.AppMonitor.GetApps() {
		var accessPoints []*agentapi.AccessPoint
		for _, point := range app.GetBaseApp().AccessPoints {
			accessPoints = append(accessPoints, &agentapi.AccessPoint{
				Type:              point.Type,
				Address:           point.Address,
				Port:              point.Port,
				Key:               point.Key,
				UseSecureProtocol: point.UseSecureProtocol,
			})
		}

		apps = append(apps, &agentapi.App{
			Type:         app.GetBaseApp().Type,
			AccessPoints: accessPoints,
		})
	}

	state := agentapi.GetStateRsp{
		AgentVersion:             stork.Version,
		Apps:                     apps,
		Hostname:                 hostInfo.Hostname,
		Cpus:                     int64(runtime.NumCPU()),
		CpusLoad:                 loadStr,
		Memory:                   int64(vm.Total / (1024 * 1024 * 1024)), // in GiB
		UsedMemory:               int64(vm.UsedPercent),
		Uptime:                   int64(hostInfo.Uptime / (60 * 60 * 24)), // in days
		Os:                       hostInfo.OS,
		Platform:                 hostInfo.Platform,
		PlatformFamily:           hostInfo.PlatformFamily,
		PlatformVersion:          hostInfo.PlatformVersion,
		KernelVersion:            hostInfo.KernelVersion,
		KernelArch:               hostInfo.KernelArch,
		VirtualizationSystem:     hostInfo.VirtualizationSystem,
		VirtualizationRole:       hostInfo.VirtualizationRole,
		HostID:                   hostInfo.HostID,
		Error:                    "",
		AgentUsesHTTPCredentials: sa.KeaHTTPClient.HasAuthenticationCredentials(),
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

	app := sa.AppMonitor.GetApp(AppTypeBind9, AccessPointControl, in.Address, in.Port)
	if app == nil {
		rndcRsp.Status.Code = agentapi.Status_ERROR
		rndcRsp.Status.Message = "Cannot find BIND 9 app"
		response.Status = rndcRsp.Status
		return response, nil
	}
	bind9App := app.(*Bind9App)
	if bind9App == nil {
		rndcRsp.Status.Code = agentapi.Status_ERROR
		rndcRsp.Status.Message = fmt.Sprintf("Incorrect app found: %s instead of BIND 9", app.GetBaseApp().Type)
		response.Status = rndcRsp.Status
		return response, nil
	}

	request := in.GetRndcRequest()

	// Try to forward the command to rndc.
	output, err := bind9App.sendCommand(strings.Fields(request.Request))
	if err != nil {
		log.WithError(err).
			WithFields(log.Fields{
				"Address": in.Address,
				"Port":    in.Port,
			}).Errorf("Failed to forward commands to rndc")
		rndcRsp.Status.Code = agentapi.Status_ERROR
		rndcRsp.Status.Message = fmt.Sprintf("Failed to forward commands to rndc: %s", err.Error())
	} else {
		rndcRsp.Status.Code = agentapi.Status_OK
		rndcRsp.Response = string(output)
	}

	response.Status = rndcRsp.Status
	return response, nil
}

// ForwardToNamedStats forwards a statistics request to the named daemon.
func (sa *StorkAgent) ForwardToNamedStats(ctx context.Context, in *agentapi.ForwardToNamedStatsReq) (*agentapi.ForwardToNamedStatsRsp, error) {
	reqURL := in.GetUrl()
	req := in.GetNamedStatsRequest()

	response := &agentapi.ForwardToNamedStatsRsp{
		Status: &agentapi.Status{
			Code: agentapi.Status_OK, // all ok
		},
	}

	rsp := &agentapi.NamedStatsResponse{
		Status: &agentapi.Status{},
	}

	// Try to forward the command to named daemon.
	namedRsp, err := sa.GeneralHTTPClient.Call(reqURL, bytes.NewBuffer([]byte(req.Request)))
	if err != nil {
		log.WithFields(log.Fields{
			"URL": reqURL,
		}).Errorf("Failed to forward commands to named: %+v", err)
		rsp.Status.Code = agentapi.Status_ERROR
		rsp.Status.Message = fmt.Sprintf("Failed to forward commands to named: %s", err.Error())
		response.NamedStatsResponse = rsp
		return response, nil
	}

	// Read the response body.
	body, err := io.ReadAll(namedRsp.Body)
	namedRsp.Body.Close()
	if err != nil {
		log.WithFields(log.Fields{
			"URL": reqURL,
		}).Errorf("Failed to read the body of the named response: %+v", err)
		rsp.Status.Code = agentapi.Status_ERROR
		rsp.Status.Message = fmt.Sprintf("Failed to read the body of the named response: %s", err.Error())
		response.NamedStatsResponse = rsp
		return response, nil
	}

	// Everything looks good, so include the body in the response.
	rsp.Response = string(body)
	rsp.Status.Code = agentapi.Status_OK
	response.NamedStatsResponse = rsp
	return response, nil
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
		response.Status.Message = "Incorrect URL to Kea CA"
		return response, nil
	}

	requests := in.GetKeaRequests()

	// forward requests to kea one by one
	for _, req := range requests {
		rsp := &agentapi.KeaResponse{
			Status: &agentapi.Status{},
		}
		// Try to forward the command to Kea Control Agent.
		keaRsp, err := sa.KeaHTTPClient.Call(reqURL, bytes.NewBuffer([]byte(req.Request)))
		if err != nil {
			log.WithFields(log.Fields{
				"URL": reqURL,
			}).Errorf("Failed to forward commands to Kea CA: %+v", err)
			rsp.Status.Code = agentapi.Status_ERROR
			rsp.Status.Message = fmt.Sprintf("Failed to forward commands to Kea: %s", err.Error())
			response.KeaResponses = append(response.KeaResponses, rsp)
			continue
		}

		// Read the response body.
		body, err := io.ReadAll(keaRsp.Body)
		keaRsp.Body.Close()
		if err != nil {
			log.WithFields(log.Fields{
				"URL": reqURL,
			}).Errorf("Failed to read the body of the Kea response to forwarded commands: %+v", err)
			rsp.Status.Code = agentapi.Status_ERROR
			rsp.Status.Message = fmt.Sprintf("Failed to read the body of the Kea response: %s", err.Error())
			response.KeaResponses = append(response.KeaResponses, rsp)
			continue
		}

		// Push Kea response for synchronous processing. It may modify the
		// response body.
		body, err = sa.keaInterceptor.syncHandle(sa, req, body)
		if err != nil {
			log.WithFields(log.Fields{
				"URL": reqURL,
			}).Errorf("Failed to apply synchronous interceptors on Kea response: %+v", err)
			continue
		}

		// Push Kea response for async processing. It is done in background.
		// One of the use cases is to extract log files used by Kea and to
		// allow the log viewer to access them.
		go sa.keaInterceptor.asyncHandle(sa, req, body)

		rsp.Response = body
		rsp.Status.Code = agentapi.Status_OK
		response.KeaResponses = append(response.KeaResponses, rsp)
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

// Starts the gRPC and HTTP listeners.
func (sa *StorkAgent) Serve() error {
	// Install gRPC API handlers.
	agentapi.RegisterAgentServer(sa.server, sa)

	// Prepare listener on configured address.
	addr := net.JoinHostPort(sa.Host, strconv.Itoa(sa.Port))
	lis, err := net.Listen("tcp", addr)
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
