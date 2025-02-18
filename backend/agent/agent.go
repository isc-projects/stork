package agent

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"runtime"
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

	// Registers the gzip compression in the internal init() call.
	// The server chooses the compression algorithm based on the client's request.
	_ "google.golang.org/grpc/encoding/gzip"

	"isc.org/stork"
	agentapi "isc.org/stork/api"
	"isc.org/stork/appdata/bind9stats"
	"isc.org/stork/pki"
	storkutil "isc.org/stork/util"
)

// Global Stork Agent state.
type StorkAgent struct {
	Host string
	Port int
	// Explicitly provided BIND 9 configuration path by user. It will be used
	// to detect the BIND 9 app. It may be empty.
	ExplicitBind9ConfigPath string
	AppMonitor              AppMonitor
	// BIND9 HTTP stats client.
	bind9StatsClient *bind9StatsClient
	// An HTTP client configuration used to create the HTTP clients for
	// particular Kea applications.
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
func NewStorkAgent(host string, port int, appMonitor AppMonitor, bind9StatsClient *bind9StatsClient, keaHTTPClientConfig HTTPClientConfig, hookManager *HookManager, explicitBind9ConfigPath string) *StorkAgent {
	logTailer := newLogTailer()

	sa := &StorkAgent{
		Host:                    host,
		Port:                    port,
		ExplicitBind9ConfigPath: explicitBind9ConfigPath,
		AppMonitor:              appMonitor,
		bind9StatsClient:        bind9StatsClient,
		KeaHTTPClientConfig:     keaHTTPClientConfig,
		logTailer:               logTailer,
		keaInterceptor:          newKeaInterceptor(),
		hookManager:             hookManager,
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
		AgentVersion:         stork.Version,
		Apps:                 apps,
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

	grpcResponse := &agentapi.ForwardToNamedStatsRsp{
		Status: &agentapi.Status{
			Code: agentapi.Status_OK, // all ok
		},
	}

	innerGrpcResponse := &agentapi.NamedStatsResponse{
		Status: &agentapi.Status{},
	}

	// Try to forward the command to named daemon.
	namedResponse, payload, err := sa.bind9StatsClient.createRequestFromURL(reqURL).getRawJSON("/")
	if err != nil {
		log.WithFields(log.Fields{
			"URL": reqURL,
		}).Errorf("Failed to forward commands to named over the stats channel: %+v", err)
		innerGrpcResponse.Status.Code = agentapi.Status_ERROR
		innerGrpcResponse.Status.Message = fmt.Sprintf("Failed to forward commands to named over the stats channel: %s", err.Error())
		grpcResponse.NamedStatsResponse = innerGrpcResponse
		return grpcResponse, nil
	}

	// Communication successful but HTTP error code returned.
	if namedResponse.IsError() {
		log.WithFields(log.Fields{
			"URL":    reqURL,
			"Status": namedResponse.StatusCode(),
		}).Errorf("named stats channel returned error status code with message: %s", namedResponse.String())
		innerGrpcResponse.Status.Code = agentapi.Status_ERROR
		innerGrpcResponse.Status.Message = fmt.Sprintf("named stats channel returned error status code with message: %s", namedResponse.String())
	}

	// Everything looks good, so include the body in the response.
	innerGrpcResponse.Response = string(payload)
	innerGrpcResponse.Status.Code = agentapi.Status_OK
	grpcResponse.NamedStatsResponse = innerGrpcResponse
	return grpcResponse, nil
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

	host, port, _ := storkutil.ParseURL(reqURL)
	app := sa.AppMonitor.GetApp(AppTypeKea, AccessPointControl, host, port)
	if app == nil {
		response.Status.Code = agentapi.Status_ERROR
		response.Status.Message = "Cannot find Kea app"
		return response, nil
	}
	keaApp := app.(*KeaApp)
	if keaApp == nil {
		response.Status.Code = agentapi.Status_ERROR
		response.Status.Message = fmt.Sprintf("Incorrect app found: %s instead of Kea", app.GetBaseApp().Type)
		return response, nil
	}

	requests := in.GetKeaRequests()

	// forward requests to kea one by one
	for _, req := range requests {
		rsp := &agentapi.KeaResponse{
			Status: &agentapi.Status{},
		}
		// Try to forward the command to Kea Control Agent.
		body, err := keaApp.sendCommandRaw([]byte(req.Request))
		if err != nil {
			log.WithFields(log.Fields{
				"URL": reqURL,
			}).Errorf("Failed to forward commands to Kea CA: %+v", err)
			rsp.Status.Code = agentapi.Status_ERROR
			rsp.Status.Message = fmt.Sprintf("Failed to forward commands to Kea: %s", err.Error())
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

// Generate a streaming response returning DNS zones from a specified
// agent. The response can be filtered or unfiltered, depending on the
// request.
func (sa *StorkAgent) ReceiveZones(req *agentapi.ReceiveZonesReq, server grpc.ServerStreamingServer[agentapi.Zone]) error {
	appI := sa.AppMonitor.GetApp(AppTypeBind9, AccessPointControl, req.ControlAddress, req.ControlPort)
	var inventory *zoneInventory
	switch app := appI.(type) {
	case *Bind9App:
		inventory = app.zoneInventory
	default:
		// This is rather an exceptional case, so we don't necessarily need to
		// include the detailed error message.
		return status.New(codes.InvalidArgument, "attempted to receive DNS zones from an unsupported app").Err()
	}
	if inventory == nil {
		// This is also an exceptional case. All DNS servers should have the
		// zone inventory initialized.
		return status.New(codes.FailedPrecondition, "attempted to receive DNS zones from an app for which zone inventory was not instantiated").Err()
	}
	// Set filtering rules based on the request.
	var filter *bind9stats.ZoneFilter
	if req.ViewName != "" || req.Limit > 0 || req.LoadedAfter > 0 || req.LowerBound != "" {
		filter = bind9stats.NewZoneFilter()
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
				Reason: "ZONE_INVENTORY_BUSY_ERROR",
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
