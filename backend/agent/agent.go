package agent

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"runtime"

	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"isc.org/stork"
	agentapi "isc.org/stork/api"
)

// Stork Agent settings.
type Settings struct {
	Host string `long:"host" description:"the IP to listen on" env:"STORK_AGENT_ADDRESS"`
	Port int    `long:"port" description:"the port to listen on for connections" default:"8080" env:"STORK_AGENT_PORT"`
}

// Global Stork Agent state
type StorkAgent struct {
	Settings   Settings
	AppMonitor AppMonitor

	CAClient *CAClient
}

// API exposed to Stork Server

func NewStorkAgent() *StorkAgent {
	caClient := NewCAClient()
	sa := &StorkAgent{
		AppMonitor: NewAppMonitor(caClient),
		CAClient:   caClient,
	}
	return sa
}

// Get state of machine.
func (sa *StorkAgent) GetState(ctx context.Context, in *agentapi.GetStateReq) (*agentapi.GetStateRsp, error) {
	vm, _ := mem.VirtualMemory()
	hostInfo, _ := host.Info()
	load, _ := load.Avg()
	loadStr := fmt.Sprintf("%.2f %.2f %.2f", load.Load1, load.Load5, load.Load15)

	var apps []*agentapi.App
	for _, srv := range sa.AppMonitor.GetApps() {
		switch s := srv.(type) {
		case AppKea:
			var daemons []*agentapi.KeaDaemon
			for _, d := range s.Daemons {
				daemons = append(daemons, &agentapi.KeaDaemon{
					Pid:             d.Pid,
					Name:            d.Name,
					Active:          d.Active,
					Version:         d.Version,
					ExtendedVersion: d.ExtendedVersion,
				})
			}
			apps = append(apps, &agentapi.App{
				Version:     s.Version,
				CtrlAddress: s.CtrlAddress,
				CtrlPort:    s.CtrlPort,
				Active:      s.Active,
				App: &agentapi.App_Kea{
					Kea: &agentapi.AppKea{
						ExtendedVersion: s.ExtendedVersion,
						Daemons:         daemons,
					},
				},
			})
		case AppBind9:
			var daemon = &agentapi.Bind9Daemon{
				Pid:     s.Daemon.Pid,
				Name:    s.Daemon.Name,
				Active:  s.Daemon.Active,
				Version: s.Daemon.Version,
			}
			apps = append(apps, &agentapi.App{
				Version:  s.Version,
				CtrlPort: s.CtrlPort,
				Active:   s.Active,
				App: &agentapi.App_Bind9{
					Bind9: &agentapi.AppBind9{
						Daemon: daemon,
					},
				},
			})
		default:
			panic(fmt.Sprint("Unknown app type"))
		}
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

// Restart Kea app.
func (sa *StorkAgent) RestartKea(ctx context.Context, in *agentapi.RestartKeaReq) (*agentapi.RestartKeaRsp, error) {
	log.Printf("Received: RestartKea %v", in)
	return &agentapi.RestartKeaRsp{Xyz: "321"}, nil
}

// Forwards Kea command sent by the Stork server to the appropriate Kea instance over
// HTTP (via Control Agent).
func (sa *StorkAgent) ForwardToKeaOverHTTP(ctx context.Context, in *agentapi.ForwardToKeaOverHTTPReq) (*agentapi.ForwardToKeaOverHTTPRsp, error) {
	reqURL := in.GetUrl()

	payload := in.GetKeaRequest()

	rsp := &agentapi.ForwardToKeaOverHTTPRsp{}

	// Try to forward the command to Kea Control Agent.
	keaRsp, err := sa.CAClient.Call(reqURL, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		log.WithFields(log.Fields{
			"URL": reqURL,
		}).Errorf("Failed to forward command to Kea: %+v", err)
		return rsp, err
	}
	defer keaRsp.Body.Close()

	// Read the response body.
	body, err := ioutil.ReadAll(keaRsp.Body)
	if err != nil {
		log.WithFields(log.Fields{
			"URL": reqURL,
		}).Errorf("Failed to read the body of the Kea response to forwarded command: %+v", err)
		return rsp, err
	}

	// Everything looks good, so include the body in the response.
	rsp.KeaResponse = string(body)

	return rsp, err
}

func (sa *StorkAgent) Serve() {
	// Install gRPC API handlers.
	server := grpc.NewServer()
	agentapi.RegisterAgentServer(server, sa)

	// Prepare listener on configured address.
	addr := fmt.Sprintf("%s:%d", sa.Settings.Host, sa.Settings.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on port: %+v", err)
	}

	// Start serving gRPC
	log.WithFields(log.Fields{
		"address": lis.Addr(),
	}).Infof("started serving Stork Agent")
	if err := server.Serve(lis); err != nil {
		log.Fatalf("Failed to listen on port: %+v", err)
	}
}
