package agentcomm

import (
	"context"
	"net"
	"strconv"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	agentapi "isc.org/stork/api"
	storkutil "isc.org/stork/util"
)

type App struct {
	Type        string // currently supported types are: "kea" and "bind9"
	CtrlAddress string
	CtrlPort    int64
	CtrlKey     string
}

// State of the machine. It describes multiple properties of the machine like number of CPUs
// or operating system name and version.
type State struct {
	Address              string
	AgentVersion         string
	Cpus                 int64
	CpusLoad             string
	Memory               int64
	Hostname             string
	Uptime               int64
	UsedMemory           int64
	Os                   string
	Platform             string
	PlatformFamily       string
	PlatformVersion      string
	KernelVersion        string
	KernelArch           string
	VirtualizationSystem string
	VirtualizationRole   string
	HostID               string
	LastVisited          time.Time
	Error                string
	Apps                 []*App
}

// Get version from agent.
func (agents *connectedAgentsData) GetState(ctx context.Context, address string, agentPort int64) (*State, error) {
	// Find agent in map.
	addrPort := net.JoinHostPort(address, strconv.FormatInt(agentPort, 10))
	agent, err := agents.GetConnectedAgent(addrPort)
	if err != nil {
		return nil, err
	}

	// Call agent for version.
	grpcState, err := agent.Client.GetState(ctx, &agentapi.GetStateReq{})
	if err != nil {
		// reconnect and try again
		err2 := agent.MakeGrpcConnection()
		if err2 != nil {
			log.Warn(err)
			return nil, errors.Wrap(err2, "problem with connection to agent")
		}
		grpcState, err = agent.Client.GetState(ctx, &agentapi.GetStateReq{})
		if err != nil {
			return nil, errors.Wrap(err, "problem with connection to agent")
		}
	}

	var apps []*App
	for _, app := range grpcState.Apps {
		apps = append(apps, &App{
			Type:        app.Type,
			CtrlAddress: app.CtrlAddress,
			CtrlPort:    app.CtrlPort,
			CtrlKey:     app.CtrlKey,
		})
	}

	state := State{
		Address:              address,
		AgentVersion:         grpcState.AgentVersion,
		Cpus:                 grpcState.Cpus,
		CpusLoad:             grpcState.CpusLoad,
		Memory:               grpcState.Memory,
		Hostname:             grpcState.Hostname,
		Uptime:               grpcState.Uptime,
		UsedMemory:           grpcState.UsedMemory,
		Os:                   grpcState.Os,
		Platform:             grpcState.Platform,
		PlatformFamily:       grpcState.PlatformFamily,
		PlatformVersion:      grpcState.PlatformVersion,
		KernelVersion:        grpcState.KernelVersion,
		KernelArch:           grpcState.KernelArch,
		VirtualizationSystem: grpcState.VirtualizationSystem,
		VirtualizationRole:   grpcState.VirtualizationRole,
		HostID:               grpcState.HostID,
		LastVisited:          storkutil.UTCNow(),
		Error:                grpcState.Error,
		Apps:                 apps,
	}

	return &state, nil
}

type Bind9Daemon struct {
	Pid     int32
	Name    string
	Active  bool
	Version string
}

type Bind9State struct {
	Version string
	Active  bool
	Daemon  Bind9Daemon
}

func (agents *connectedAgentsData) GetBind9State(ctx context.Context, agentAddress string, agentPort int64) (*Bind9State, error) {
	// Find the agent by address and port.
	addrPort := net.JoinHostPort(agentAddress, strconv.FormatInt(agentPort, 10))
	agent, err := agents.GetConnectedAgent(addrPort)
	if err != nil {
		err = errors.Wrapf(err, "there is no agent available at address %s:%d", agentAddress, agentPort)
		return nil, err
	}

	req := &agentapi.GetBind9StateReq{
		CtrlAddress: agentAddress,
		CtrlPort:    agentPort,
	}

	// call agent to get BIND 9 state
	rsp, err := agent.Client.GetBind9State(ctx, req)
	if err != nil {
		err = errors.Wrapf(err, "failed to get BIND 9 state: %+v", req)
		return nil, err
	}
	if rsp.Status.Code != agentapi.Status_OK {
		err = errors.New(rsp.Status.Message)
		return nil, err
	}

	state := &Bind9State{
		Version: rsp.Version,
		Active:  rsp.Active,
		Daemon: Bind9Daemon{
			Pid:     rsp.Daemon.Pid,
			Name:    rsp.Daemon.Name,
			Active:  rsp.Daemon.Active,
			Version: rsp.Daemon.Version,
		},
	}

	return state, nil
}

type KeaCmdsResult struct {
	Error      error
	CmdsErrors []error
}

// Forwards a Kea command via the Stork Agent and Kea Control Agent and then
// parses the response. caURL is URL to Kea Control Agent.
func (agents *connectedAgentsData) ForwardToKeaOverHTTP(ctx context.Context, agentAddress string, agentPort int64, caURL string, commands []*KeaCommand, cmdResponses ...interface{}) (*KeaCmdsResult, error) {
	// Find the agent by address and port.
	addrPort := net.JoinHostPort(agentAddress, strconv.FormatInt(agentPort, 10))
	agent, err := agents.GetConnectedAgent(addrPort)
	if err != nil {
		err = errors.Wrapf(err, "there is no agent available at address %s:%d", agentAddress, agentPort)
		return nil, err
	}

	// Prepare the on-wire representation of the commands.
	fdReq := &agentapi.ForwardToKeaOverHTTPReq{
		Url: caURL,
	}
	for _, cmd := range commands {
		fdReq.KeaRequests = append(fdReq.KeaRequests, &agentapi.KeaRequest{
			Request: cmd.Marshal(),
		})
	}

	// Send the commands to the Stork agent.
	fdRsp, err := agent.Client.ForwardToKeaOverHTTP(ctx, fdReq)
	if err != nil {
		err = errors.Wrapf(err, "failed to forward Kea commands to %s, commands were: %+v", caURL, fdReq.KeaRequests)
		return nil, err
	}

	result := &KeaCmdsResult{}
	result.Error = nil
	if fdRsp.Status.Code != agentapi.Status_OK {
		result.Error = errors.New(fdRsp.Status.Message)
	}

	for idx, rsp := range fdRsp.GetKeaResponses() {
		cmdResp := cmdResponses[idx]
		if rsp.Status.Code != agentapi.Status_OK {
			result.CmdsErrors = append(result.CmdsErrors, errors.New(rsp.Status.Message))
			continue
		}

		// Try to parse the response from the on-wire format.
		err = UnmarshalKeaResponseList(commands[idx], rsp.Response, cmdResp)
		if err != nil {
			err = errors.Wrapf(err, "failed to parse Kea response from %s, response was: %s", caURL, rsp)
			result.CmdsErrors = append(result.CmdsErrors, err)
			continue
		}

		result.CmdsErrors = append(result.CmdsErrors, nil)
	}

	// Everything was fine, so return no error.
	return result, nil
}
