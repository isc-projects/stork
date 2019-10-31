package agentcomm

import (
	"time"
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/pkg/errors"

	"isc.org/stork/api"
)

// State of machine. It describe multiple aspect of machine like its number of CPUs
// or operating system name and version.
type State struct {
	Address string
	AgentVersion string
	Cpus int64
	CpusLoad string
	Memory int64
	Hostname string
	Uptime int64
	UsedMemory int64
	Os string
	Platform string
	PlatformFamily string
	PlatformVersion string
	KernelVersion string
	KernelArch string
	VirtualizationSystem string
	VirtualizationRole string
	HostID string
	LastVisited time.Time
	Error string
}

// Get version from agent.
func (agents *connectedAgentsData) GetState(address string) (*State, error) {
	// Find agent in map.
	agent, err := agents.GetConnectedAgent(address)
	if err != nil {
		return nil, err
	}

	// Call agent for version.
	ctx, _ := context.WithTimeout(context.Background(), 10 * time.Second)
	grpcState, err := agent.Client.GetState(ctx, &agentapi.GetStateReq{})
	if err != nil {
		// Problem with connection, try to reconnect and retry the call
		log.Infof("problem with connection to agent %v, reconnecting", err)
		err2 := agent.MakeGrpcConnection()
		if err2 != nil {
			return nil, errors.Wrap(err2, "problem with connection to agent")
		}
		ctx, _ = context.WithTimeout(context.Background(), 10 * time.Second)
		grpcState, err = agent.Client.GetState(ctx, &agentapi.GetStateReq{})
		if err != nil {
			return nil, errors.Wrap(err, "problem with connection to agent")
		}
	}

	log.Printf("state returned is %+v", grpcState)

	state := State{
		Address: address,
		AgentVersion: grpcState.AgentVersion,
		Cpus: grpcState.Cpus,
		CpusLoad: grpcState.CpusLoad,
		Memory: grpcState.Memory,
		Hostname: grpcState.Hostname,
		Uptime: grpcState.Uptime,
		UsedMemory: grpcState.UsedMemory,
		Os: grpcState.Os,
		Platform: grpcState.Platform,
		PlatformFamily: grpcState.PlatformFamily,
		PlatformVersion: grpcState.PlatformVersion,
		KernelVersion: grpcState.KernelVersion,
		KernelArch: grpcState.KernelArch,
		VirtualizationSystem: grpcState.VirtualizationSystem,
		VirtualizationRole: grpcState.VirtualizationRole,
		HostID: grpcState.HostID,
		LastVisited: time.Now().UTC(),
		Error: grpcState.Error,
	}

	return &state, nil
}
