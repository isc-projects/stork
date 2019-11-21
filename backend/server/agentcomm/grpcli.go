package agentcomm

import (
	"time"
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/pkg/errors"

	"isc.org/stork"
	"isc.org/stork/api"
)

// State of the machine. It describes multiple properties of the machine like number of CPUs
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
func (agents *connectedAgentsData) GetState(ctx context.Context, address string) (*State, error) {
	// Find agent in map.
	agent, err := agents.GetConnectedAgent(address)
	if err != nil {
		return nil, err
	}

	// Call agent for version.
	grpcState, err := agent.Client.GetState(ctx, &agentapi.GetStateReq{})
	if err != nil {
		return nil, errors.Wrap(err, "problem with connection to agent")
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
		LastVisited: stork.UTCNow(),
		Error: grpcState.Error,
	}

	return &state, nil
}
