package agentcomm

import (
	"context"
	"net"
	"strconv"
	"time"

	"github.com/pkg/errors"

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
	addrPort := net.JoinHostPort(address, strconv.FormatInt(agentPort, 10))

	// Call agent for version.
	resp, err := agents.sendAndRecvViaQueue(addrPort, &agentapi.GetStateReq{})
	if err != nil {
		return nil, errors.Wrap(err, "problem with connection to agent")
	}
	grpcState := resp.(*agentapi.GetStateRsp)

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

type RndcOutput struct {
	Output string
	Error  error
}

func (agents *connectedAgentsData) ForwardRndcCommand(ctx context.Context, agentAddress string, agentPort int64, rndcSettings Bind9Control, command string) (*RndcOutput, error) {
	addrPort := net.JoinHostPort(agentAddress, strconv.FormatInt(agentPort, 10))

	// Prepare the on-wire representation of the commands.
	req := &agentapi.ForwardRndcCommandReq{
		CtrlAddress: rndcSettings.CtrlAddress,
		CtrlPort:    rndcSettings.CtrlPort,
		CtrlKey:     rndcSettings.CtrlKey,
		RndcRequest: &agentapi.RndcRequest{
			Request: command,
		},
	}

	// Send the command to the Stork agent.
	resp, err := agents.sendAndRecvViaQueue(addrPort, req)
	if err != nil {
		err = errors.Wrapf(err, "failed to forward rndc command %s", command)
		return nil, err
	}
	response := resp.(*agentapi.ForwardRndcCommandRsp)

	if response.Status.Code != agentapi.Status_OK {
		err = errors.New(response.Status.Message)
		return nil, err
	}

	result := &RndcOutput{
		Output: "",
		Error:  nil,
	}

	rndcResponse := response.GetRndcResponse()
	if rndcResponse.Status.Code != agentapi.Status_OK {
		result.Error = errors.New(response.Status.Message)
	} else {
		result.Output = rndcResponse.Response
	}
	return result, nil
}

type KeaCmdsResult struct {
	Error      error
	CmdsErrors []error
}

// Forwards a Kea command via the Stork Agent and Kea Control Agent and then
// parses the response. caURL is URL to Kea Control Agent.
func (agents *connectedAgentsData) ForwardToKeaOverHTTP(ctx context.Context, agentAddress string, agentPort int64, caURL string, commands []*KeaCommand, cmdResponses ...interface{}) (*KeaCmdsResult, error) {
	addrPort := net.JoinHostPort(agentAddress, strconv.FormatInt(agentPort, 10))

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
	resp, err := agents.sendAndRecvViaQueue(addrPort, fdReq)
	if err != nil {
		err = errors.Wrapf(err, "failed to forward Kea commands to %s, commands were: %+v", caURL, fdReq.KeaRequests)
		return nil, err
	}
	fdRsp := resp.(*agentapi.ForwardToKeaOverHTTPRsp)

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
