package agent

import (
	"net"
	"fmt"
	"runtime"
	"context"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/load"

	"isc.org/stork/api"
	"isc.org/stork"
)


// Stork Agent settings.
type AgentSettings struct {
	Host         string        `long:"host" description:"the IP to listen on" env:"STORK_AGENT_ADDRESS"`
	Port         int           `long:"port" description:"the port to listen on for connections" default:"8080" env:"STORK_AGENT_PORT"`
}

// Global Stork Agent state
type StorkAgent struct {
	Settings AgentSettings
	AppMonitor AppMonitor
}


// API exposed to Stork Server

// Get state of machine.
func (s *StorkAgent) GetState(ctx context.Context, in *agentapi.GetStateReq) (*agentapi.GetStateRsp, error) {
	vm, _ := mem.VirtualMemory()
	hostInfo, _ := host.Info()
	load, _ := load.Avg()
	loadStr := fmt.Sprintf("%.2f %.2f %.2f", load.Load1, load.Load5, load.Load15)

	var apps []*agentapi.App
	for _, srv := range s.AppMonitor.GetApps() {
		switch s := srv.(type) {
		case AppKea:
			var daemons []*agentapi.KeaDaemon
			for _, d := range s.Daemons {
				daemons = append(daemons, &agentapi.KeaDaemon{
					Pid: d.Pid,
					Name: d.Name,
					Active: d.Active,
					Version: d.Version,
					ExtendedVersion: d.ExtendedVersion,
				})
			}
			apps = append(apps, &agentapi.App{
				Version: s.Version,
				CtrlPort: s.CtrlPort,
				Active: s.Active,
				App: &agentapi.App_Kea{
					Kea: &agentapi.AppKea{
						ExtendedVersion: s.ExtendedVersion,
						Daemons: daemons,
					},
				},
			})
		default:
			panic(fmt.Sprint("Unknown app type"))
		}
	}

	state := agentapi.GetStateRsp{
		AgentVersion: stork.Version,
		Apps: apps,
		Hostname: hostInfo.Hostname,
		Cpus: int64(runtime.NumCPU()),
		CpusLoad: loadStr,
		Memory: int64(vm.Total / (1024 * 1024 * 1024)), // in GiB
		UsedMemory: int64(vm.UsedPercent),
		Uptime: int64(hostInfo.Uptime / (60 * 60 * 24)), // in days
		Os: hostInfo.OS,
		Platform: hostInfo.Platform,
		PlatformFamily: hostInfo.PlatformFamily,
		PlatformVersion: hostInfo.PlatformVersion,
		KernelVersion: hostInfo.KernelVersion,
		KernelArch: hostInfo.KernelArch,
		VirtualizationSystem: hostInfo.VirtualizationSystem,
		VirtualizationRole: hostInfo.VirtualizationRole,
		HostID: hostInfo.HostID,
		Error: "",
	}

	return &state, nil
}

// Restart Kea app.
func (s *StorkAgent) RestartKea(ctx context.Context, in *agentapi.RestartKeaReq) (*agentapi.RestartKeaRsp, error) {
	log.Printf("Received: RestartKea %v", in)
	return &agentapi.RestartKeaRsp{Xyz: "321"}, nil
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
