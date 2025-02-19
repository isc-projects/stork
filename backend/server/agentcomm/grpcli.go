package agentcomm

import (
	"context"
	"fmt"
	"io"
	"iter"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	agentapi "isc.org/stork/api"
	keactrl "isc.org/stork/appctrl/kea"
	"isc.org/stork/appdata/bind9stats"
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

var _ ConnectedAgents = (*connectedAgentsImpl)(nil)

// An access point for an application to retrieve information such
// as status or metrics.
type AccessPoint struct {
	Type              string
	Address           string
	Port              int64
	Key               string
	UseSecureProtocol bool
}

// Currently supported types are: "control" and "statistics".
const (
	AccessPointControl    = "control"
	AccessPointStatistics = "statistics"
)

// The application entry detected by an agent. It unambiguously indicates the
// application location.
type App struct {
	Type         string
	AccessPoints []AccessPoint
}

// Currently supported types are: "kea" and "bind9".
const (
	AppTypeKea   = "kea"
	AppTypeBind9 = "bind9"
)

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
	LastVisitedAt        time.Time
	Error                string
	Apps                 []*App
}

// An interface to an app that can receive commands from Stork.
// Kea app receiving control commands is an example.
type ControlledApp interface {
	dbmodel.AppTag
	GetControlAccessPoint() (string, int64, string, bool, error)
	GetMachineTag() dbmodel.MachineTag
	GetDaemonTags() []dbmodel.DaemonTag
	GetDaemonTag(daemonName string) dbmodel.DaemonTag
}

// An interface to a machine that can receive commands from Stork.
type ControlledMachine interface {
	GetAddress() string
	GetAgentPort() int64
}

// MakeAccessPoint is an utility to make an array of one access point.
func MakeAccessPoint(tp, address, key string, port int64) []AccessPoint {
	return []AccessPoint{{
		Type:    tp,
		Address: address,
		Port:    port,
		Key:     key,
	}}
}

// An interface to the response from gRPC including a command status.
type agentResponse interface {
	GetStatus() *agentapi.Status
}

// Based on the gRPC request, response and an error it checks the state of
// the communication with an agent and returns this state. The status message
// is returned as a second parameter.
func (agents *connectedAgentsImpl) checkAgentCommState(stats *AgentCommStats, reqData any, reqErr error) (CommErrorTransition, string) {
	commErrors := stats.GetErrorCount(reqData)

	switch {
	case commErrors == 0 && reqErr != nil:
		// New communication issue.
		stats.IncreaseErrorCount(reqData)
		return CommErrorNew, reqErr.Error()
	case commErrors > 0 && reqErr != nil:
		// Old communication issue
		stats.IncreaseErrorCount(reqData)
		return CommErrorContinued, reqErr.Error()
	case commErrors == 0 && reqErr == nil:
		// Everything still ok.
		return CommErrorNone, ""
	case commErrors > 0 && reqErr == nil:
		// Communication resumed.
		stats.ResetErrorCount(reqData)
		return CommErrorReset, ""
	}
	return CommErrorNone, ""
}

func (agents *connectedAgentsImpl) checkBind9CommState(stats *Bind9AppCommErrorStats, channelType Bind9ChannelType, resp any) (CommErrorTransition, string) {
	var (
		status  *agentapi.Status
		details string
	)
	if statusResp, ok := resp.(agentResponse); ok {
		status = statusResp.GetStatus()
		details = statusResp.GetStatus().Message
	}
	if status == nil {
		return CommErrorNone, details
	}

	commErrors := stats.GetErrorCount(channelType)

	switch {
	case commErrors == 0 && status.GetCode() != agentapi.Status_OK:
		stats.IncreaseErrorCount(channelType)
		return CommErrorNew, details
	case commErrors > 0 && status.GetCode() != agentapi.Status_OK:
		stats.IncreaseErrorCount(channelType)
		return CommErrorContinued, details
	case commErrors == 0 && status.GetCode() == agentapi.Status_OK:
		return CommErrorNone, details
	case commErrors > 0 && status.GetCode() == agentapi.Status_OK:
		stats.ResetErrorCount(channelType)
		return CommErrorReset, details
	}
	return CommErrorNone, details
}

// Holds the communication states of the Kea daemons returned
// by the checkKeaCommState function.
type keaCommState struct {
	controlAgentState CommErrorTransition
	// Contains an item for each command. If the command was successful, the
	// item is nil.
	controlAgentErrors []error
	dhcp4State         CommErrorTransition
	dhcp4ErrMessage    string
	dhcp6State         CommErrorTransition
	dhcp6ErrMessage    string
	d2State            CommErrorTransition
	d2ErrMessage       string
}

// It checks the communication state with the Kea daemons behind an agent. This
// function is called if there was no communication problem with an agent itself.
// If checks the status codes returned by the Kea Control Agent and returns the
// communication states for each of the daemons.
func (agents *connectedAgentsImpl) checkKeaCommState(stats *KeaAppCommErrorStats, commands []keactrl.SerializableCommand, resp *agentapi.ForwardToKeaOverHTTPRsp) keaCommState {
	var (
		state           keaCommState
		controlAgentErr int64 = -1
		dhcp4Err        int64 = -1
		dhcp6Err        int64 = -1
		d2Err           int64 = -1
	)

	// Get all responses from the Kea server.
	for idx, daemonResp := range resp.GetKeaResponses() {
		if controlAgentErr < 0 {
			controlAgentErr = 0
		}
		var controlAgentErrMessageBuilder strings.Builder
		if daemonResp.Status.Code != agentapi.Status_OK {
			controlAgentErr++
			if daemonResp.Status.Message != "" {
				fmt.Fprintln(&controlAgentErrMessageBuilder, daemonResp.Status.Message)
			}
			continue
		}
		var parsedResp []keactrl.ResponseHeader
		err := keactrl.UnmarshalResponseList(commands[idx], daemonResp.Response, &parsedResp)
		if err != nil {
			controlAgentErr++
			fmt.Fprintf(&controlAgentErrMessageBuilder, "received invalid response from Kea ControlAgent: %s\n", err)
		}

		for _, daemonResp := range parsedResp {
			switch daemonResp.GetDaemon() {
			case dbmodel.DaemonNameDHCPv4:
				if dhcp4Err < 0 {
					dhcp4Err = 0
				}
				if daemonResp.GetResult() == keactrl.ResponseError {
					dhcp4Err++
					state.dhcp4ErrMessage = daemonResp.GetText()
				}
			case dbmodel.DaemonNameDHCPv6:
				if dhcp6Err < 0 {
					dhcp6Err = 0
				}
				if daemonResp.GetResult() == keactrl.ResponseError {
					dhcp6Err++
					state.dhcp6ErrMessage = daemonResp.GetText()
				}
			case dbmodel.DaemonNameD2:
				if d2Err < 0 {
					d2Err = 0
				}
				if daemonResp.GetResult() == keactrl.ResponseError {
					d2Err++
					state.d2ErrMessage = daemonResp.GetText()
				}
			case dbmodel.DaemonNameCA, "":
				if daemonResp.GetResult() == keactrl.ResponseError {
					controlAgentErr++
					fmt.Fprintln(&controlAgentErrMessageBuilder, daemonResp.GetText())
				}
			default:
				continue
			}
		}

		var commandErr error
		if controlAgentErrMessageBuilder.Len() != 0 {
			commandErr = errors.New(controlAgentErrMessageBuilder.String())
		}
		state.controlAgentErrors = append(state.controlAgentErrors, commandErr)
	}
	state.controlAgentState = stats.UpdateErrorCount(KeaDaemonCA, controlAgentErr)
	state.dhcp4State = stats.UpdateErrorCount(KeaDaemonDHCPv4, dhcp4Err)
	state.dhcp6State = stats.UpdateErrorCount(KeaDaemonDHCPv6, dhcp6Err)
	state.d2State = stats.UpdateErrorCount(KeaDaemonD2, d2Err)
	return state
}

// Check connectivity with a machine.
func (agents *connectedAgentsImpl) Ping(ctx context.Context, machine dbmodel.MachineTag) error {
	addrPort := net.JoinHostPort(machine.GetAddress(), strconv.FormatInt(machine.GetAgentPort(), 10))

	req := &agentapi.PingReq{}
	resp, err := agents.sendAndRecvViaQueue(addrPort, req)

	stats := agents.getConnectedAgentStats(machine.GetAddress(), machine.GetAgentPort())
	if stats == nil {
		return errors.Errorf("failed to get statistics for the non-existing agent %s", addrPort)
	}

	stats.mutex.Lock()
	defer stats.mutex.Unlock()

	// Check connectivity with the Stork agent by examining the returned error.
	commState, details := agents.checkAgentCommState(stats, req, err)
	switch commState {
	case CommErrorNew:
		log.WithField("agent", addrPort).Warn("Failed to ping the agent")
		agents.eventCenter.AddErrorEvent("pinging Stork agent on {machine} failed", machine, dbmodel.SSEConnectivity, details)

	case CommErrorReset:
		agents.eventCenter.AddWarningEvent("pinging Stork agent on {machine} succeeded", machine, dbmodel.SSEConnectivity, details)

	case CommErrorContinued:
		log.WithField("agent", addrPort).Warn("Failed to ping the Stork agent; the agent is still not responding")

	default:
		// Communication with the agent was ok and is still ok.
	}

	// If there was an error in communication with the agent, there is no need
	// to check the response because it is probably nil anyway. Return an derror.
	if err != nil {
		return errors.Wrapf(err, "failed to ping the Stork agent %s", addrPort)
	}
	if _, ok := resp.(*agentapi.PingRsp); !ok {
		return errors.Wrapf(err, "wrong response for ping from the Stork agent %s", addrPort)
	}
	return nil
}

// Get machine statistics and version number.
func (agents *connectedAgentsImpl) GetState(ctx context.Context, machine dbmodel.MachineTag) (*State, error) {
	addrPort := net.JoinHostPort(machine.GetAddress(), strconv.FormatInt(machine.GetAgentPort(), 10))

	req := &agentapi.GetStateReq{}
	resp, err := agents.sendAndRecvViaQueue(addrPort, req)

	stats := agents.getConnectedAgentStats(machine.GetAddress(), machine.GetAgentPort())
	if stats == nil {
		return nil, errors.Errorf("failed to get statistics for the non-existing agent %s", addrPort)
	}

	stats.mutex.Lock()
	defer stats.mutex.Unlock()

	// Check connectivity with the Stork agent by examining the returned error.
	commState, details := agents.checkAgentCommState(stats, req, err)
	switch commState {
	case CommErrorNew:
		log.WithField("agent", addrPort).Warn("Failed to get state from the Stork agent")
		agents.eventCenter.AddErrorEvent("communication with Stork agent on {machine} to get state failed", machine, dbmodel.SSEConnectivity, details)

	case CommErrorReset:
		agents.eventCenter.AddWarningEvent("communication with Stork agent on {machine} to get state succeeded", machine, dbmodel.SSEConnectivity, details)

	case CommErrorContinued:
		log.WithField("agent", addrPort).Warn("Failed to get state from the Stork agent; the agent is still not responding")

	default:
		// Communication with the agent was ok and is still ok.
	}

	// If there was an error in communication with the agent, there is no need
	// to check the response because it is probably nil anyway. Return an derror.
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get state from agent %s", addrPort)
	}

	// Communication successful. Let's decode the response.
	grpcState, ok := resp.(*agentapi.GetStateRsp)
	if !ok || grpcState == nil {
		return nil, errors.Errorf("wrong response to get state from agent %s", addrPort)
	}

	var apps []*App
	for _, app := range grpcState.Apps {
		var accessPoints []AccessPoint

		for _, point := range app.AccessPoints {
			accessPoints = append(accessPoints, AccessPoint{
				Type:              point.Type,
				Address:           point.Address,
				Port:              point.Port,
				Key:               point.Key,
				UseSecureProtocol: point.UseSecureProtocol,
			})
		}

		apps = append(apps, &App{
			Type:         app.Type,
			AccessPoints: accessPoints,
		})
	}

	state := State{
		Address:              machine.GetAddress(),
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
		LastVisitedAt:        storkutil.UTCNow(),
		Error:                grpcState.Error,
		Apps:                 apps,
	}

	return &state, nil
}

// The extracted output of the RNDC command.
type RndcOutput struct {
	Output string
}

// Forwards an RNDC command to named.
func (agents *connectedAgentsImpl) ForwardRndcCommand(ctx context.Context, app ControlledApp, command string) (*RndcOutput, error) {
	agentAddress := app.GetMachineTag().GetAddress()
	agentPort := app.GetMachineTag().GetAgentPort()

	// Get rndc control settings
	ctrlAddress, ctrlPort, _, _, err := app.GetControlAccessPoint()
	if err != nil {
		return nil, err
	}

	addrPort := net.JoinHostPort(agentAddress, strconv.FormatInt(agentPort, 10))

	// Prepare the on-wire representation of the commands.
	req := &agentapi.ForwardRndcCommandReq{
		Address: ctrlAddress,
		Port:    ctrlPort,
		RndcRequest: &agentapi.RndcRequest{
			Request: command,
		},
	}

	// Send the command to the Stork Agent.
	resp, err := agents.sendAndRecvViaQueue(addrPort, req)

	stats := agents.getConnectedAgentStats(agentAddress, agentPort)
	if stats == nil {
		return nil, errors.Errorf("failed to get statistics for the non-existing agent %s", addrPort)
	}

	stats.mutex.Lock()
	defer stats.mutex.Unlock()

	// Check connectivity with the Stork agent by examining the returned error.
	commState, details := agents.checkAgentCommState(stats, req, err)
	switch commState {
	case CommErrorNew:
		log.WithFields(log.Fields{
			"agent": addrPort,
			"rndc":  net.JoinHostPort(ctrlAddress, strconv.FormatInt(ctrlPort, 10)),
		}).Warnf("Failed to send the rndc command: %s", command)
		agents.eventCenter.AddErrorEvent("communication with Stork agent on {machine} to forward rndc command failed", app.GetMachineTag(), dbmodel.SSEConnectivity, details)

	case CommErrorReset:
		agents.eventCenter.AddWarningEvent("communication with Stork agent on {machine} to forward rndc command succeeded", app.GetMachineTag(), dbmodel.SSEConnectivity, details)

	case CommErrorContinued:
		log.WithFields(log.Fields{
			"agent": addrPort,
			"rndc":  net.JoinHostPort(ctrlAddress, strconv.FormatInt(ctrlPort, 10)),
		}).Warnf("Failed to send the rndc command to the Stork agent: %s; agent is still not responding", command)

	default:
		// Communication with the agent was ok and is still ok.
	}

	// Check the result of the communication between the Stork agent and named by
	// examining the returned status code.
	commState, details = agents.checkBind9CommState(stats.GetBind9CommErrorStats(app.GetID()), Bind9ChannelRNDC, resp)
	switch commState {
	case CommErrorNew:
		log.WithFields(log.Fields{
			"agent": addrPort,
			"rndc":  net.JoinHostPort(ctrlAddress, strconv.FormatInt(ctrlPort, 10)),
		}).Warnf("Failed to send the rndc command: %s", command)
		agents.eventCenter.AddErrorEvent("communication between the Stork agent on {machine} and {app} to forward rndc command failed", app.GetMachineTag(), app, dbmodel.SSEConnectivity, details)

	case CommErrorReset:
		agents.eventCenter.AddWarningEvent("communication between the Stork agent on {machine} and {app} to forward rndc command succeeded", app.GetMachineTag(), app, dbmodel.SSEConnectivity, details)

	case CommErrorContinued:
		log.WithFields(log.Fields{
			"agent": addrPort,
			"rndc":  net.JoinHostPort(ctrlAddress, strconv.FormatInt(ctrlPort, 10)),
		}).Warnf("Failed to send the rndc command from Stork agent to named: %s; named is still returning an error", command)

	default:
		// Communication between the Stork agent and named was ok and is still ok.
	}

	// Stork agent returned an error.
	if err != nil {
		return nil, errors.Wrapf(err, "failed to send the rndc command to Stork agent %s", addrPort)
	}

	// Communication with the Stork agent was ok, but named returned an error.
	if commState != CommErrorReset && commState != CommErrorNone {
		err = errors.Errorf("error communicating between Stork agent %s and named to send rndc command: %s", addrPort, details)
		return nil, err
	}

	response, ok := resp.(*agentapi.ForwardRndcCommandRsp)
	if !ok || response == nil {
		return nil, errors.Errorf("wrong response to the rndc command from the Stork agent %s", addrPort)
	}

	result := &RndcOutput{
		Output: "",
	}

	// named has responded but the response may also contain an error status.
	rndcResponse := response.GetRndcResponse()
	bind9CommErrors := stats.GetBind9CommErrorStats(app.GetID())

	// If the status is ok, let's just return the result.
	if rndcResponse.Status.Code == agentapi.Status_OK {
		result.Output = rndcResponse.Response
		bind9CommErrors.ResetErrorCount(Bind9ChannelRNDC)
		return result, nil
	}

	// Status code was not ok, so let's record an error message.
	err = errors.New(response.Status.Message)

	// Bump up error statistics. If this is a consecutive error let's
	// just return it and not log it again and again.
	if bind9CommErrors.IncreaseErrorCount(Bind9ChannelRNDC) > 1 {
		err = errors.Errorf("failed to send rndc command via the agent %s; BIND 9 is still failing",
			agentAddress)
		return nil, err
	}
	// This is apparently the first error like this. Let's log it.
	log.WithFields(log.Fields{
		"agent": addrPort,
		"rndc":  net.JoinHostPort(ctrlAddress, strconv.FormatInt(ctrlPort, 10)),
	}).Warnf("named returned an error status to the RNDC command: %s", command)

	return result, err
}

// Forwards a statistics request via the Stork Agent to the named daemon and
// then parses the response. statsAddress, statsPort and statsPath are used to
// construct the URL to the statistics-channel of the named daemon.
func (agents *connectedAgentsImpl) ForwardToNamedStats(ctx context.Context, app ControlledApp, statsAddress string, statsPort int64, statsPath string, statsOutput interface{}) error {
	addrPort := net.JoinHostPort(app.GetMachineTag().GetAddress(), strconv.FormatInt(app.GetMachineTag().GetAgentPort(), 10))
	statsURL := storkutil.HostWithPortURL(statsAddress, statsPort, false)
	statsURL += statsPath

	// Prepare the on-wire representation of the commands.
	req := &agentapi.ForwardToNamedStatsReq{
		Url: statsURL,
	}
	req.NamedStatsRequest = &agentapi.NamedStatsRequest{
		Request: "",
	}

	// Send the commands to the Stork Agent.
	resp, err := agents.sendAndRecvViaQueue(addrPort, req)

	stats := agents.getConnectedAgentStats(app.GetMachineTag().GetAddress(), app.GetMachineTag().GetAgentPort())
	if stats == nil {
		return errors.Errorf("failed to get statistics for the non-existing agent %s", addrPort)
	}

	stats.mutex.Lock()
	defer stats.mutex.Unlock()

	// Check connectivity with the Stork agent by examining the returned error.
	commState, details := agents.checkAgentCommState(stats, req, err)
	switch commState {
	case CommErrorNew:
		log.WithFields(log.Fields{
			"agent":     addrPort,
			"stats URL": statsURL,
		}).Warnf("Failed to send the named stats command: %s", req.NamedStatsRequest)
		agents.eventCenter.AddErrorEvent("communication with Stork agent on {machine} to query for named stats failed", app.GetMachineTag(), dbmodel.SSEConnectivity, details)

	case CommErrorReset:
		agents.eventCenter.AddWarningEvent("communication with Stork agent on {machine} to query for named stats succeeded", app.GetMachineTag(), dbmodel.SSEConnectivity, details)

	case CommErrorContinued:
		log.WithFields(log.Fields{
			"agent":     addrPort,
			"stats URL": statsURL,
		}).Warnf("Failed to send the named stats command to the Stork agent: %s; agent is still not responding", req.NamedStatsRequest)

	default:
		// Communication with the agent was ok and is still ok.
	}

	response, ok := resp.(*agentapi.ForwardToNamedStatsRsp)
	if !ok || response == nil {
		return errors.Errorf("wrong response when querying stats from named via agent %s", addrPort)
	}

	// Check the result of the communication between the Stork agent and named by
	// examining the returned status code.
	commState, details = agents.checkBind9CommState(stats.GetBind9CommErrorStats(app.GetID()), Bind9ChannelStats, response.NamedStatsResponse)
	switch commState {
	case CommErrorNew:
		log.WithFields(log.Fields{
			"agent":     addrPort,
			"stats URL": statsURL,
		}).Warnf("Failed to forward the stats command %s from the Stork agent to named", req.NamedStatsRequest)
		agents.eventCenter.AddErrorEvent("communication between the Stork agent on {machine} and {app} to query named stats failed", app.GetMachineTag(), app, dbmodel.SSEConnectivity, details)

	case CommErrorReset:
		agents.eventCenter.AddWarningEvent("communication between the Stork agent on {machine} and {app} to query named stats succeeded", app.GetMachineTag(), app, dbmodel.SSEConnectivity, details)

	case CommErrorContinued:
		log.WithFields(log.Fields{
			"agent": addrPort,
			"rndc":  statsURL,
		}).Warnf("Failed to forward the stats command %s from the Stork agent to named; it is still returning an error", req.NamedStatsRequest)

	default:
		// Communication between the Stork agent and named was ok and is still ok.
	}

	// Stork agent returned an error.
	if err != nil {
		return errors.Wrapf(err, "failed to query stats from named via agent %s", addrPort)
	}

	// Communication with the Stork agent was ok, but named returned an error.
	if commState != CommErrorReset && commState != CommErrorNone {
		err = errors.Errorf("error communicating between Stork agent %s and named to query named stats: %s", addrPort, details)
		return err
	}

	statsResp := response.NamedStatsResponse

	// named responded but the response may contain an error status.
	if statsResp.Status.Code != agentapi.Status_OK {
		err = errors.New(statsResp.Status.Message)
	}

	bind9CommErrors := stats.GetBind9CommErrorStats(app.GetID())

	// If status was ok, let's try to parse the response from the on-wire format.
	if err == nil {
		err = UnmarshalNamedStatsResponse(statsResp.Response, statsOutput)
		if err != nil {
			err = errors.Wrapf(err, "failed to parse named statistics response from %s, response was: %s", statsURL, statsResp)
		} else {
			// No error parsing the response, so let's return.
			bind9CommErrors.ResetErrorCount(Bind9ChannelStats)
			return nil
		}
	}

	// We've got some errors that have to be recorded in stats.
	if bind9CommErrors.IncreaseErrorCount(Bind9ChannelStats) > 1 {
		err = errors.Errorf("failed to send named stats command via the agent %s, BIND 9 is still failing",
			app.GetMachineTag().GetAddress())
		return err
	}

	// This is apparently the first error like this. Let's log it.
	log.WithFields(log.Fields{
		"agent":     addrPort,
		"stats URL": statsURL,
	}).Warnf("named returned an error status to the stats query command: %s", req.NamedStatsRequest)

	return err
}

// Result of sending Kea commands to Kea.
type KeaCmdsResult struct {
	Error      error
	CmdsErrors []error
}

// Returns first error found in the KeaCmdsResult structure or nil if no
// error has been found.
func (result *KeaCmdsResult) GetFirstError() error {
	switch {
	case result == nil:
		return nil
	case result.Error != nil:
		return result.Error
	default:
		for _, err := range result.CmdsErrors {
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Forwards a Kea command via the Stork Agent and Kea Control Agent and then
// parses the response. It accepts a slice of commands that are aggregated
// in a single message to the Stork agent. The agent then sends them sequentially
// to the Kea servers via the control agent. This function tracks errors at several
// encapsulation level. First, it tracks the errors in sending the message to
// the Stork agent. Then, it tracks the errors reported by the Stork agent upon
// reception of this message. Next, it tracks the errors in communication between
// the Stork agent and Kea control agent. Finally, it tracks the errors reported
// by the daemons behind the control agent. Any new errors trigger appropriate
// events. If any of the existing errors go away in this communication, the
// warning events are triggered to indicate that the problem has gone away.
// The received responses are unmarshalled into the respective parameters at
// the end of the parameter list. The returned structure holds aggregated errors
// reported at different levels.
func (agents *connectedAgentsImpl) ForwardToKeaOverHTTP(ctx context.Context, app ControlledApp, commands []keactrl.SerializableCommand, cmdResponses ...any) (*KeaCmdsResult, error) {
	agentAddress := app.GetMachineTag().GetAddress()
	agentPort := app.GetMachineTag().GetAgentPort()

	caAddress, caPort, _, caUseSecureProtocol, err := app.GetControlAccessPoint()
	if err != nil {
		log.WithFields(log.Fields{
			"address": caAddress,
			"port":    caPort,
		}).Warnf("No Kea CA access point found matching the specified address and port to send the commands")
		return nil, err
	}

	addrPort := net.JoinHostPort(agentAddress, strconv.FormatInt(agentPort, 10))
	caURL := storkutil.HostWithPortURL(caAddress, caPort, caUseSecureProtocol)

	// Prepare the on-wire representation of the commands.
	req := &agentapi.ForwardToKeaOverHTTPReq{
		Url: caURL,
	}
	for _, cmd := range commands {
		req.KeaRequests = append(req.KeaRequests, &agentapi.KeaRequest{
			Request: cmd.Marshal(),
		})
	}
	// Send the commands to the Stork Agent and get the response.
	resp, err := agents.sendAndRecvViaQueue(addrPort, req)

	// Check the communication issues with the Stork agent.
	stats := agents.getConnectedAgentStats(app.GetMachineTag().GetAddress(), app.GetMachineTag().GetAgentPort())
	if stats == nil {
		return nil, errors.Errorf("failed to get statistics for the non-existing agent %s", addrPort)
	}

	stats.mutex.Lock()
	defer stats.mutex.Unlock()

	// Check connectivity with the Stork agent by examining the returned error.
	commState, details := agents.checkAgentCommState(stats, req, err)
	switch commState {
	case CommErrorNew:
		log.WithField("agent", addrPort).Warnf("Failed to send %d Kea command(s)", len(commands))
		agents.eventCenter.AddErrorEvent("communication with Stork agent on {machine} to forward Kea command failed", app.GetMachineTag(), dbmodel.SSEConnectivity, details)

	case CommErrorReset:
		agents.eventCenter.AddWarningEvent("communication with Stork agent on {machine} to forward Kea command succeeded", app.GetMachineTag(), dbmodel.SSEConnectivity, details)

	case CommErrorContinued:
		log.WithField("agent", addrPort).Warnf("Failed to send %d Kea command(s) to the Stork agent; agent is still not responding", len(commands))

	default:
		// Communication with the agent was ok and is still ok.
	}

	// Stork agent returned an error.
	if err != nil {
		return nil, errors.Wrapf(err, "failed to send Kea commands via Stork agent %s", addrPort)
	}

	// Communication with the Stork agent was ok, but there was an error communicating
	// with a Kea agent. This is rather rare.
	if commState != CommErrorReset && commState != CommErrorNone {
		err = errors.Errorf("error communicating between Stork agent %s and Kea to send commands: %s", addrPort, details)
		return nil, err
	}

	response, ok := resp.(*agentapi.ForwardToKeaOverHTTPRsp)
	if !ok || response == nil {
		return nil, errors.Errorf("wrong response to a Kea command from agent %s", addrPort)
	}

	// We will aggregate the results from various communication levels in this structure.
	result := &KeaCmdsResult{}
	if response.Status.Code != agentapi.Status_OK {
		result.Error = errors.New(response.Status.Message)
	}

	// Check the communication issues with the Kea daemons. For each supported daemon we
	// get the current state of the communication with this daemon and optionally an
	// error message.
	keaCommState := agents.checkKeaCommState(stats.GetKeaCommErrorStats(app.GetID()), commands, response)

	// Save Control Agent Errors.
	result.CmdsErrors = keaCommState.controlAgentErrors

	// Generate events for the Kea Control Agent.
	if keaCommState.controlAgentState == CommErrorNew {
		// The connection was ok but now it is broken.
		log.WithField("agent", addrPort).Warnf("Failed to forward Kea command to Kea Control Agent")
		agents.eventCenter.AddErrorEvent("forwarding Kea command via the Kea Control Agent {daemon} on {machine} failed", app.GetDaemonTag(dbmodel.DaemonNameCA), app.GetMachineTag(), dbmodel.SSEConnectivity, keaCommState.controlAgentErrors)
	} else if keaCommState.controlAgentState == CommErrorReset {
		// The connection was broken but now is ok.
		agents.eventCenter.AddWarningEvent("forwarding Kea command via the Kea Control Agent {daemon} on {machine} succeeded", app.GetDaemonTag(dbmodel.DaemonNameCA), app.GetMachineTag(), dbmodel.SSEConnectivity)
	}
	// Generate events for the DHCPv4 server.
	if keaCommState.dhcp4State == CommErrorNew {
		// The connection was ok but now it is broken.
		log.WithField("agent", addrPort).Warnf("Failed to forward Kea command to Kea DHCPv4 server")
		agents.eventCenter.AddErrorEvent("processing Kea command by {daemon} on {machine} failed", app.GetDaemonTag(dbmodel.DaemonNameDHCPv4), app.GetMachineTag(), dbmodel.SSEConnectivity, keaCommState.dhcp4ErrMessage)
	} else if keaCommState.dhcp4State == CommErrorReset {
		// The connection was broken but now is ok.
		agents.eventCenter.AddWarningEvent("processing Kea command by {daemon} on {machine} succeeded", app.GetDaemonTag(dbmodel.DaemonNameDHCPv4), app.GetMachineTag(), dbmodel.SSEConnectivity)
	}
	// Generate events for the DHCPv6 server.
	if keaCommState.dhcp6State == CommErrorNew {
		// The connection was ok but now it is broken.
		log.WithField("agent", addrPort).Warnf("Failed to forward Kea command to Kea DHCPv6 server")
		agents.eventCenter.AddErrorEvent("processing Kea command by {daemon} on {machine} failed", app.GetDaemonTag(dbmodel.DaemonNameDHCPv6), app.GetMachineTag(), dbmodel.SSEConnectivity, keaCommState.dhcp6ErrMessage)
	} else if keaCommState.dhcp6State == CommErrorReset {
		// The connection was broken but now is ok.
		agents.eventCenter.AddWarningEvent("processing Kea command by {daemon} on {machine} succeeded", app.GetDaemonTag(dbmodel.DaemonNameDHCPv6), app.GetMachineTag(), dbmodel.SSEConnectivity)
	}
	// Generate events for the D2 server.
	if keaCommState.d2State == CommErrorNew {
		// The connection was ok but now it is broken.
		log.WithField("agent", addrPort).Warnf("Failed to forward Kea command to Kea D2 server")
		agents.eventCenter.AddErrorEvent("processing Kea command by {daemon} on {machine} failed", app.GetDaemonTag(dbmodel.DaemonNameD2), app.GetMachineTag(), dbmodel.SSEConnectivity, keaCommState.d2ErrMessage)
	} else if keaCommState.d2State == CommErrorReset {
		// The connection was broken but now is ok.
		agents.eventCenter.AddWarningEvent("processing Kea command by {daemon} on {machine} succeeded", app.GetDaemonTag(dbmodel.DaemonNameD2), app.GetMachineTag(), dbmodel.SSEConnectivity)
	}

	// Get all responses from the Kea server.
	for idx, rsp := range response.GetKeaResponses() {
		cmdResp := cmdResponses[idx]
		// Try to parse the json response from the on-wire format.
		err = keactrl.UnmarshalResponseList(commands[idx], rsp.Response, cmdResp)
		if err != nil {
			err = errors.Wrapf(err, "failed to parse Kea response from %s, response was: %s", caURL, rsp)
			// The sufficient number of elements should have been already allocated but
			// let's make sure.
			if len(result.CmdsErrors) > idx {
				result.CmdsErrors[idx] = err
			} else {
				result.CmdsErrors = append(result.CmdsErrors, err)
			}
			// Failure to parse the response is treated as a Kea Control Agent error.
			if keaCommState.controlAgentState != CommErrorNew && keaCommState.controlAgentState != CommErrorContinued {
				keaCommErrors := stats.GetKeaCommErrorStats(app.GetID())
				keaCommErrors.IncreaseErrorCount(KeaDaemonCA)
			}
		}
	}

	// Everything was fine, so return no error.
	return result, nil
}

// Get the tail of the remote text file.
func (agents *connectedAgentsImpl) TailTextFile(ctx context.Context, machine dbmodel.MachineTag, path string, offset int64) ([]string, error) {
	addrPort := net.JoinHostPort(machine.GetAddress(), strconv.FormatInt(machine.GetAgentPort(), 10))

	// Get the path to the file and the (seek) info indicating the location
	// from which the tail should be fetched.
	req := &agentapi.TailTextFileReq{
		Path:   path,
		Offset: offset,
	}

	// Send the request via queue.
	agentResponse, err := agents.sendAndRecvViaQueue(addrPort, req)

	stats := agents.getConnectedAgentStats(machine.GetAddress(), machine.GetAgentPort())
	if stats == nil {
		return nil, errors.Errorf("failed to get statistics for the non-existing agent %s", addrPort)
	}

	stats.mutex.Lock()
	defer stats.mutex.Unlock()

	// Check connectivity with the Stork agent by examining the returned error.
	commIssue, details := agents.checkAgentCommState(stats, req, err)
	switch commIssue {
	case CommErrorNew:
		log.WithFields(log.Fields{
			"agent": addrPort,
			"file":  path,
		}).Warn("Failed to tail the text file via the Stork agent", path)
		agents.eventCenter.AddErrorEvent("communication with Stork agent on {machine} to tail the text file failed", machine, dbmodel.SSEConnectivity, details)

	case CommErrorReset:
		agents.eventCenter.AddWarningEvent("communication with Stork agent on {machine} to tail the text file succeeded", machine, dbmodel.SSEConnectivity, details)

	case CommErrorContinued:
		log.WithFields(log.Fields{
			"agent": addrPort,
			"file":  path,
		}).Warn("Failed to tail the text file via the Stork agent; the agent is still not responding", path)
	default:
		// Communication with the agent was ok and is still ok.
	}

	// If there was an error in communication with the agent, there is no need
	// to check the response because it is probably nil anyway. Return an error.
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch text file contents: %s", path)
	}

	response, ok := agentResponse.(*agentapi.TailTextFileRsp)
	if !ok || response == nil {
		return nil, errors.Errorf("wrong response to tailing the text file from the Stork agent %s", addrPort)
	}

	// Check the status code.
	if response.Status.Code != agentapi.Status_OK {
		return nil, errors.New(response.Status.Message)
	}

	// All ok.
	return response.Lines, nil
}

// Receive DNS zones over the stream from a selected agent's zone inventory.
// It returns an iterator with a pointer to zone and error. The iterator ends
// when an error occurs. Receiving the zones is not cancellable at the moment.
func (agents *connectedAgentsImpl) ReceiveZones(ctx context.Context, app ControlledApp, filter *bind9stats.ZoneFilter) iter.Seq2[*bind9stats.ExtendedZone, error] {
	return func(yield func(*bind9stats.ExtendedZone, error) bool) {
		// Get control access point for the specified app. It will be sent
		// in the request to the agent, so the agent can identify correct
		// zone inventory.
		ctrlAddress, ctrlPort, _, _, err := app.GetControlAccessPoint()
		if err != nil {
			_ = yield(nil, err)
			return
		}
		// Get the agent's state. It holds the connection with the agent.
		agentAddressPort := net.JoinHostPort(app.GetMachineTag().GetAddress(), strconv.FormatInt(app.GetMachineTag().GetAgentPort(), 10))
		agent, err := agents.getConnectedAgent(agentAddressPort)
		if err != nil {
			_ = yield(nil, err)
			return
		}
		// Start creating the request.
		request := &agentapi.ReceiveZonesReq{
			ControlAddress: ctrlAddress,
			ControlPort:    ctrlPort,
		}
		// Set filtering rules, if required.
		if filter != nil {
			if filter.View != nil {
				request.ViewName = *filter.View
			}
			if filter.Limit != nil {
				request.Limit = int64(*filter.Limit)
			}
		}
		// This is the same pattern we're using in the manager.go. The connection is
		// cached so it is possible that it gets terminated or broken at some point.
		// By trying the actual operation and retrying on failure we should be able
		// to recover. There may be other ways to achieve recovery (e.g., getting
		// the connection state before attempting the call). However, it is hard to
		// say how reliable they are. This approach worked well for several years so
		// it should be fine to continue using it.
		var stream grpc.ServerStreamingClient[agentapi.Zone]
		if stream, err = agent.connector.createClient().ReceiveZones(ctx, request); err != nil {
			if err = agent.connector.connect(); err == nil {
				stream, err = agent.connector.createClient().ReceiveZones(ctx, request)
			}
		}
		if err != nil {
			// The zone inventory may signal errors indicating that it is
			// unable to return the zones because it is in a wrong state.
			// The server should interpret these errors and formulate hints
			// to the user that some administrative actions may be required.
			s := status.Convert(err)
			for _, d := range s.Details() {
				if info, ok := d.(*errdetails.ErrorInfo); ok {
					switch info.Reason {
					case "ZONE_INVENTORY_NOT_INITED":
						// Zone inventory hasn't been initialized.
						_ = yield(nil, NewZoneInventoryNotInitedError(agentAddressPort))
						return
					case "ZONE_INVENTORY_BUSY":
						// Zone inventory is busy. Retrying later may help.
						_ = yield(nil, NewZoneInventoryBusyError(agentAddressPort))
						return
					default:
						_ = yield(nil, err)
						return
					}
				}
			}
			// Other error.
			_ = yield(nil, err)
			return
		}

		for {
			// Start receiving zones.
			receivedZone, err := stream.Recv()
			if err != nil {
				if !errors.Is(err, io.EOF) {
					_ = yield(nil, err)
				}
				return
			}
			zone := &bind9stats.ExtendedZone{
				Zone: bind9stats.Zone{
					ZoneName: receivedZone.GetName(),
					Class:    receivedZone.GetClass(),
					Serial:   receivedZone.GetSerial(),
					Type:     receivedZone.GetType(),
					Loaded:   time.Unix(receivedZone.GetLoaded(), 0).UTC(),
				},
				ViewName:       receivedZone.View,
				TotalZoneCount: receivedZone.TotalZoneCount,
			}
			if !yield(zone, nil) {
				// Stop if the caller no longer iterates over the zones.
				return
			}
		}
	}
}
