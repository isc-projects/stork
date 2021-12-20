package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/security/advancedtls"

	"isc.org/stork"
	agentapi "isc.org/stork/api"
)

// Global Stork Agent state.
type StorkAgent struct {
	Settings   *cli.Context
	AppMonitor AppMonitor

	HTTPClient     *HTTPClient // to communicate with Kea Control Agent and named statistics-channel
	server         *grpc.Server
	logTailer      *logTailer
	keaInterceptor *keaInterceptor
}

// API exposed to Stork Server.
func NewStorkAgent(settings *cli.Context, appMonitor AppMonitor) *StorkAgent {
	logTailer := newLogTailer()

	sa := &StorkAgent{
		Settings:       settings,
		AppMonitor:     appMonitor,
		HTTPClient:     NewHTTPClient(settings.Bool("skip-tls-cert-verification")),
		logTailer:      logTailer,
		keaInterceptor: newKeaInterceptor(),
	}

	registerKeaInterceptFns(sa)

	return sa
}

// Read the latest root CA cert from file for Stork Server's cert verification.
func getRootCertificates(params *advancedtls.GetRootCAsParams) (*advancedtls.GetRootCAsResults, error) {
	certPool := x509.NewCertPool()
	ca, err := os.ReadFile(RootCAFile)
	if err != nil {
		err = errors.Wrapf(err, "could not read CA certificate: %s", RootCAFile)
		log.Errorf("%+v", err)
		return nil, err
	}
	// append the client certificates from the CA
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		err = errors.New("failed to append client certs")
		log.Errorf("%+v", err)
		return nil, err
	}
	log.Printf("loaded CA cert: %s\n", RootCAFile)
	return &advancedtls.GetRootCAsResults{
		TrustCerts: certPool,
	}, nil
}

// Read the latest Stork Agent's cert from file for presenting its identity to the Stork server.
func getIdentityCertificatesForServer(info *tls.ClientHelloInfo) ([]*tls.Certificate, error) {
	keyPEM, err := os.ReadFile(KeyPEMFile)
	if err != nil {
		err = errors.Wrapf(err, "could not load key PEM file: %s", KeyPEMFile)
		log.Errorf("%+v", err)
		return nil, err
	}
	certPEM, err := os.ReadFile(CertPEMFile)
	if err != nil {
		err = errors.Wrapf(err, "could not load cert PEM file: %s", CertPEMFile)
		log.Errorf("%+v", err)
		return nil, err
	}
	certificate, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		err = errors.Wrapf(err, "could not setup TLS key pair")
		log.Errorf("%+v", err)
		return nil, err
	}
	log.Printf("loaded server cert: %s and key: %s\n", CertPEMFile, KeyPEMFile)
	return []*tls.Certificate{&certificate}, nil
}

// Prepare gRPC server with configured TLS.
func newGRPCServerWithTLS() (*grpc.Server, error) {
	// Prepare structure for advanced TLS. It defines hook functions
	// that dynamically load key and cert from files just before establishing
	// connection. Thanks to this if these files changed in meantime then
	// always latest version for new connections is used.
	// Beside that there is enabled client authentication and forced
	// cert and host verification.
	options := &advancedtls.ServerOptions{
		// pull latest root CA cert for stork server cert verification
		RootOptions: advancedtls.RootCertificateOptions{
			GetRootCertificates: getRootCertificates,
		},
		// pull latest stork agent cert for presenting its identity to stork server
		IdentityOptions: advancedtls.IdentityCertificateOptions{
			GetIdentityCertificatesForServer: getIdentityCertificatesForServer,
		},
		// force stork server cert verification
		RequireClientCert: true,
		// check cert and if it matches host IP
		VType: advancedtls.CertAndHostVerification,
	}
	creds, err := advancedtls.NewServerCreds(options)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot create server credentials for TLS")
	}

	srv := grpc.NewServer(grpc.Creds(creds))
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
		log.WithFields(log.Fields{
			"Address": in.Address,
			"Port":    in.Port,
		}).Errorf("Failed to forward commands to rndc: %+v", err)
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
	namedRsp, err := sa.HTTPClient.Call(reqURL, bytes.NewBuffer([]byte(req.Request)))
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
		keaRsp, err := sa.HTTPClient.Call(reqURL, bytes.NewBuffer([]byte(req.Request)))
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

		// Push Kea response for async processing. One of the use cases is to
		// extract log files used by Kea and to allow the log viewer to access
		// them.
		go sa.keaInterceptor.asyncHandle(sa, req, body)

		// gzip json response received from Kea
		var gzippedBuf bytes.Buffer
		zw := gzip.NewWriter(&gzippedBuf)
		_, err = zw.Write(body)
		if err != nil {
			log.WithFields(log.Fields{
				"URL": reqURL,
			}).Errorf("Failed to compress the Kea response: %+v", err)
			rsp.Status.Code = agentapi.Status_ERROR
			rsp.Status.Message = fmt.Sprintf("Failed to compress the Kea response: %s", err.Error())
			response.KeaResponses = append(response.KeaResponses, rsp)
			if err2 := zw.Close(); err2 != nil {
				log.Errorf("error while closing gzip writer: %s", err2)
			}
			continue
		}
		if err := zw.Close(); err != nil {
			log.WithFields(log.Fields{
				"URL": reqURL,
			}).Errorf("Failed to finish compressing the Kea response: %+v", err)
			rsp.Status.Code = agentapi.Status_ERROR
			rsp.Status.Message = fmt.Sprintf("Failed to finish compressing the Kea response: %s", err.Error())
			response.KeaResponses = append(response.KeaResponses, rsp)
			continue
		}
		if len(body) > 0 {
			log.Printf("Compressing response from %d B to %d B, ratio %d%%", len(body), gzippedBuf.Len(), 100*gzippedBuf.Len()/len(body))
		}

		// Everything looks good, so include the gzipped body in the response.
		rsp.Response = gzippedBuf.Bytes()
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

func (sa *StorkAgent) Serve() {
	// Install gRPC API handlers.
	agentapi.RegisterAgentServer(sa.server, sa)

	// Prepare listener on configured address.
	addr := net.JoinHostPort(sa.Settings.String("host"), strconv.Itoa(sa.Settings.Int("port")))
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on port: %+v", err)
	}

	// Start serving gRPC
	log.WithFields(log.Fields{
		"address": lis.Addr(),
	}).Infof("started serving Stork Agent")
	if err := sa.server.Serve(lis); err != nil {
		log.Fatalf("Failed to listen on port: %+v", err)
	}
}

func (sa *StorkAgent) Shutdown() {
	log.Infof("stopping StorkAgent")
	if sa.server != nil {
		sa.server.GracefulStop()
	}
}
