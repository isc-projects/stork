package agentcomm

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	agentapi "isc.org/stork/api"
	keactrl "isc.org/stork/appctrl/kea"
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

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
	Address                  string
	AgentVersion             string
	Cpus                     int64
	CpusLoad                 string
	Memory                   int64
	Hostname                 string
	Uptime                   int64
	UsedMemory               int64
	Os                       string
	Platform                 string
	PlatformFamily           string
	PlatformVersion          string
	KernelVersion            string
	KernelArch               string
	VirtualizationSystem     string
	VirtualizationRole       string
	HostID                   string
	AgentUsesHTTPCredentials bool
	LastVisitedAt            time.Time
	Error                    string
	Apps                     []*App
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

// An enum type indicating a state of the communication with an agent
// or a daemon behind an agent.
type commState int

const (
	/// Indicates that there was no communication issue.
	noCommIssue commState = iota
	/// Indicates that there is a new communication issue.
	newCommIssue
	/// Indicates that the previous communication issue still occurs.
	existingCommIssue
	/// Indicates that the communication issue has gone away.
	correctedCommIssue
)

// Based on the gRPC request, response and an error it checks the state of
// the communication with an agent and returns this state. The status message
// is returned as a second parameter.
func (agents *connectedAgentsData) checkAgentCommState(agentAddr string, reqData any, reqErr error, resp any) (commState, string) {
	agent, err := agents.GetConnectedAgent(agentAddr)
	if err != nil {
		return noCommIssue, ""
	}
	dataType := reflect.TypeOf(reqData).Elem().Name()

	var (
		status  *agentapi.Status
		details string
	)
	if statusResp, ok := resp.(agentResponse); ok {
		status = statusResp.GetStatus()
		details = statusResp.GetStatus().Message
	}

	agent.Stats.mutex.Lock()
	defer agent.Stats.mutex.Unlock()
	commErrors := agent.Stats.AgentCommErrors[dataType]

	switch {
	case commErrors == 0 && (reqErr != nil || (status != nil && status.GetCode() != agentapi.Status_OK)):
		// New communication issue.
		agent.Stats.AgentCommErrors[dataType]++
		return newCommIssue, details
	case commErrors > 0 && (reqErr != nil || (status != nil && status.GetCode() != agentapi.Status_OK)):
		// Old communication issue
		agent.Stats.AgentCommErrors[dataType]++
		return existingCommIssue, details
	case commErrors == 0 && (reqErr == nil && (status == nil || status.GetCode() == agentapi.Status_OK)):
		// Everything still ok.
		return noCommIssue, details
	case commErrors > 0 && (reqErr == nil && (status == nil || status.GetCode() == agentapi.Status_OK)):
		// Communication resumed.
		if status != nil && status.GetCode() != agentapi.Status_OK {
			agent.Stats.AgentCommErrors[dataType]++
			return existingCommIssue, details
		}
		delete(agent.Stats.AgentCommErrors, dataType)
		return correctedCommIssue, details
	}
	return noCommIssue, details
}

// It checks the communication state with the Kea daemons behind an agent. This
// function is called if there was no communication problem with an agent itself.
// If checks the status codes returned by the Kea Control Agent and returns the
// communication states for each of the daemons.
func (agents *connectedAgentsData) checkKeaCommState(agentAddr string, commands []keactrl.SerializableCommand, app ControlledApp, resp *agentapi.ForwardToKeaOverHTTPRsp) (controlAgentIssue commState, controlAgentErrMessages []error, dhcp4Issue commState, dhcp4ErrMessage string, dhcp6Issue commState, dhcp6ErrMessage string, d2Issue commState, d2ErrMessage string) {
	agent, err := agents.GetConnectedAgent(agentAddr)
	if err != nil {
		return controlAgentIssue, controlAgentErrMessages, dhcp4Issue, dhcp4ErrMessage, dhcp6Issue, dhcp6ErrMessage, d2Issue, d2ErrMessage
	}

	var (
		controlAgentErr int64 = -1
		dhcp4Err        int64 = -1
		dhcp6Err        int64 = -1
		d2Err           int64 = -1
	)
	agent.Stats.mutex.Lock()
	defer agent.Stats.mutex.Unlock()
	keaCommErrors := agent.Stats.KeaCommErrors[app.GetID()]

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
		err = keactrl.UnmarshalResponseList(commands[idx], daemonResp.Response, &parsedResp)
		if err != nil {
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
					dhcp4ErrMessage = daemonResp.GetText()
				}
			case dbmodel.DaemonNameDHCPv6:
				if dhcp6Err < 0 {
					dhcp6Err = 0
				}
				if daemonResp.GetResult() == keactrl.ResponseError {
					dhcp6Err++
					dhcp6ErrMessage = daemonResp.GetText()
				}
			case dbmodel.DaemonNameD2:
				if d2Err < 0 {
					d2Err = 0
				}
				if daemonResp.GetResult() == keactrl.ResponseError {
					d2Err++
					d2ErrMessage = daemonResp.GetText()
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
		if controlAgentErrMessageBuilder.Len() == 0 {
			controlAgentErrMessages = append(controlAgentErrMessages, nil)
		} else {
			controlAgentErrMessages = append(controlAgentErrMessages, errors.New(controlAgentErrMessageBuilder.String()))
		}
	}
	controlAgentIssue = updateKeaCommErrors(controlAgentErr, &keaCommErrors.ControlAgent)
	dhcp4Issue = updateKeaCommErrors(dhcp4Err, &keaCommErrors.DHCPv4)
	dhcp6Issue = updateKeaCommErrors(dhcp6Err, &keaCommErrors.DHCPv6)
	d2Issue = updateKeaCommErrors(d2Err, &keaCommErrors.D2)
	agent.Stats.KeaCommErrors[app.GetID()] = keaCommErrors
	return controlAgentIssue, controlAgentErrMessages, dhcp4Issue, dhcp4ErrMessage, dhcp6Issue, dhcp6ErrMessage, d2Issue, d2ErrMessage
}

// A function called internally by the checkKeaCommState function. It checks
// the communication status with a Kea daemon and updates the error count.
func updateKeaCommErrors(newErrCount int64, updatedErrCount *int64) commState {
	if newErrCount < 0 {
		return noCommIssue
	}
	switch {
	case newErrCount == 0 && *updatedErrCount == 0:
		return noCommIssue
	case newErrCount == 0 && *updatedErrCount > 0:
		*updatedErrCount = 0
		return correctedCommIssue
	case newErrCount > 0 && *updatedErrCount == 0:
		*updatedErrCount = newErrCount
		return newCommIssue
	case newErrCount > 0 && *updatedErrCount > 0:
		*updatedErrCount += newErrCount
		return existingCommIssue
	}
	return noCommIssue
}

// Check connectivity with machine.
func (agents *connectedAgentsData) Ping(ctx context.Context, machine dbmodel.MachineTag) error {
	addrPort := net.JoinHostPort(machine.GetAddress(), strconv.FormatInt(machine.GetAgentPort(), 10))

	// Call agent for version.
	req := &agentapi.PingReq{}
	resp, reqErr := agents.sendAndRecvViaQueue(addrPort, req)

	commIssue, details := agents.checkAgentCommState(addrPort, req, reqErr, resp)
	switch commIssue {
	case newCommIssue:
		log.WithFields(log.Fields{
			"agent": addrPort,
		}).Warn("Failed to ping the agent")
		agents.EventCenter.AddErrorEvent("Pinging {machine} failed", machine, dbmodel.SSEConnectivity, details)

	case correctedCommIssue:
		agents.EventCenter.AddWarningEvent("Pinging {machine} no longer fails", machine, dbmodel.SSEConnectivity, details)

	case existingCommIssue:
		log.WithFields(log.Fields{
			"agent": addrPort,
		}).Warn("Failed to ping the agent; the agent is still not responding")
	default:
	}

	if reqErr != nil {
		return errors.Wrapf(reqErr, "failed to ping agent %s", addrPort)
	}
	if resp.(*agentapi.PingRsp) != nil {
		return errors.Wrapf(reqErr, "wrong response for ping agent %s", addrPort)
	}
	return nil
}

// Get version from agent.
func (agents *connectedAgentsData) GetState(ctx context.Context, machine dbmodel.MachineTag) (*State, error) {
	addrPort := net.JoinHostPort(machine.GetAddress(), strconv.FormatInt(machine.GetAgentPort(), 10))

	// Call agent for version.
	req := &agentapi.GetStateReq{}
	resp, err := agents.sendAndRecvViaQueue(addrPort, req)

	commIssue, details := agents.checkAgentCommState(addrPort, req, err, resp)
	switch commIssue {
	case newCommIssue:
		log.WithFields(log.Fields{
			"agent": addrPort,
		}).Warn("Failed to get state from the agent")
		agents.EventCenter.AddErrorEvent("Communication with {machine} to get its state failed", machine, dbmodel.SSEConnectivity, details)

	case correctedCommIssue:
		agents.EventCenter.AddWarningEvent("Communication with {machine} to get its state resumed", machine, dbmodel.SSEConnectivity, details)

	case existingCommIssue:
		log.WithFields(log.Fields{
			"agent": addrPort,
		}).Warn("Failed to get state from the agent; the agent is still not responding")
	default:
	}

	if err != nil {
		return nil, errors.Wrapf(err, "failed to get state from agent %s", addrPort)
	}
	grpcState := resp.(*agentapi.GetStateRsp)
	if grpcState == nil {
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
		Address:                  machine.GetAddress(),
		AgentVersion:             grpcState.AgentVersion,
		Cpus:                     grpcState.Cpus,
		CpusLoad:                 grpcState.CpusLoad,
		Memory:                   grpcState.Memory,
		Hostname:                 grpcState.Hostname,
		Uptime:                   grpcState.Uptime,
		UsedMemory:               grpcState.UsedMemory,
		Os:                       grpcState.Os,
		Platform:                 grpcState.Platform,
		PlatformFamily:           grpcState.PlatformFamily,
		PlatformVersion:          grpcState.PlatformVersion,
		KernelVersion:            grpcState.KernelVersion,
		KernelArch:               grpcState.KernelArch,
		VirtualizationSystem:     grpcState.VirtualizationSystem,
		VirtualizationRole:       grpcState.VirtualizationRole,
		HostID:                   grpcState.HostID,
		AgentUsesHTTPCredentials: grpcState.AgentUsesHTTPCredentials,
		LastVisitedAt:            storkutil.UTCNow(),
		Error:                    grpcState.Error,
		Apps:                     apps,
	}

	return &state, nil
}

// The extracted output of the RNDC command.
type RndcOutput struct {
	Output string
	Error  error
}

func (agents *connectedAgentsData) ForwardRndcCommand(ctx context.Context, app ControlledApp, command string) (*RndcOutput, error) {
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

	commIssue, details := agents.checkAgentCommState(addrPort, req, err, resp)
	switch commIssue {
	case newCommIssue:
		log.WithFields(log.Fields{
			"agent": addrPort,
			"rndc":  net.JoinHostPort(ctrlAddress, strconv.FormatInt(ctrlPort, 10)),
		}).Warnf("Failed to send the rndc command: %s", command)
		agents.EventCenter.AddErrorEvent("communication with {app} on {machine} over rndc failed", app, app.GetMachineTag(), dbmodel.SSEConnectivity, details)

	case existingCommIssue:
		log.WithFields(log.Fields{
			"agent": addrPort,
			"rndc":  net.JoinHostPort(ctrlAddress, strconv.FormatInt(ctrlPort, 10)),
		}).Warnf("Failed to send the rndc command: %s; agent is still not responding", command)

	case correctedCommIssue:
		agents.EventCenter.AddWarningEvent("communication with {app} on {machine} over rndc resumed", app, app.GetMachineTag(), dbmodel.SSEConnectivity, details)
	default:
	}

	if err != nil {
		return nil, errors.Wrapf(err, "failed to send the rndc command to agent %s", addrPort)
	}
	response := resp.(*agentapi.ForwardRndcCommandRsp)
	if response == nil {
		return nil, errors.Errorf("wrong response to the rndc command from agent %s", addrPort)
	}

	result := &RndcOutput{
		Output: "",
		Error:  err,
	}

	if response.Status.Code != agentapi.Status_OK {
		err = errors.New(response.Status.Message)
	}

	if err == nil {
		rndcResponse := response.GetRndcResponse()
		if rndcResponse.Status.Code != agentapi.Status_OK {
			result.Error = errors.New(response.Status.Message)
		} else {
			result.Output = rndcResponse.Response
		}
	}

	// Start updating error statistics for this agent and the BIND9 app we've
	// been communicating with.
	agent, err := agents.GetConnectedAgent(addrPort)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get statistics for the non-existing agent %s", agent.Address)
	}
	agent.Stats.mutex.Lock()
	defer agent.Stats.mutex.Unlock()

	bind9CommErrors := agent.Stats.Bind9CommErrors[app.GetID()]
	if result.Error != nil {
		bind9CommErrors := agent.Stats.Bind9CommErrors[app.GetID()]
		bind9CommErrors.RNDC++
		if bind9CommErrors.RNDC > 1 {
			result.Error = errors.Errorf("failed to send rndc command via the agent %s; BIND 9 is still failing",
				agent.Address)
		}
		log.WithFields(log.Fields{
			"agent": addrPort,
			"rndc":  net.JoinHostPort(ctrlAddress, strconv.FormatInt(ctrlPort, 10)),
		}).Warnf("Failed to send the rndc command: %s", command)
	} else {
		bind9CommErrors.RNDC = 0
	}
	return result, err
}

// Forwards a statistics request via the Stork Agent to the named daemon and
// then parses the response. statsAddress, statsPort and statsPath are used to
// construct the URL to the statistics-channel of the named daemon.
func (agents *connectedAgentsData) ForwardToNamedStats(ctx context.Context, app ControlledApp, statsAddress string, statsPort int64, statsPath string, statsOutput interface{}) error {
	addrPort := net.JoinHostPort(app.GetMachineTag().GetAddress(), strconv.FormatInt(app.GetMachineTag().GetAgentPort(), 10))
	statsURL := storkutil.HostWithPortURL(statsAddress, statsPort, false)
	statsURL += statsPath

	// Prepare the on-wire representation of the commands.
	storkReq := &agentapi.ForwardToNamedStatsReq{
		Url: statsURL,
	}
	storkReq.NamedStatsRequest = &agentapi.NamedStatsRequest{
		Request: "",
	}

	// Send the commands to the Stork Agent.
	storkRsp, err := agents.sendAndRecvViaQueue(addrPort, storkReq)

	commIssue, details := agents.checkAgentCommState(addrPort, storkReq, err, storkRsp)
	switch commIssue {
	case newCommIssue:
		log.WithFields(log.Fields{
			"agent":     addrPort,
			"stats URL": statsURL,
		}).Warnf("Failed to send the following named stats command: %s", storkReq.NamedStatsRequest)
		agents.EventCenter.AddErrorEvent("communication with {app} on {machine} failed when querying for stats", app, app.GetMachineTag(), dbmodel.SSEConnectivity, details)

	case correctedCommIssue:
		agents.EventCenter.AddWarningEvent("communication with {app} on {machine} resumed when querying for stats", app, app.GetMachineTag(), dbmodel.SSEConnectivity, details)

	case existingCommIssue:
		log.WithFields(log.Fields{
			"agent":     addrPort,
			"stats URL": statsURL,
		}).Warnf("Failed to send the following named stats command: %s; agent is still not responding", storkReq.NamedStatsRequest)

	default:
	}

	if err != nil {
		return errors.Wrapf(err, "failed to query stats from named via agent %s", addrPort)
	}
	response := storkRsp.(*agentapi.ForwardToNamedStatsRsp)
	if response == nil {
		return errors.Errorf("wrong response when querying stats from named via agent %s", addrPort)
	}

	fdRsp := storkRsp.(*agentapi.ForwardToNamedStatsRsp)

	statsRsp := fdRsp.NamedStatsResponse
	if statsRsp.Status.Code != agentapi.Status_OK {
		err = errors.New(statsRsp.Status.Message)
	}
	if err == nil {
		// Try to parse the response from the on-wire format.
		err = UnmarshalNamedStatsResponse(statsRsp.Response, statsOutput)
		if err != nil {
			err = errors.Wrapf(err, "failed to parse named statistics response from %s, response was: %s", statsURL, statsRsp)
		}
	}

	// Start updating error statistics for this agent and the BIND9 app we've
	// been communicating with.
	agent, err2 := agents.GetConnectedAgent(addrPort)
	if err2 != nil {
		return errors.Wrapf(err2, "failed to get statistics for the non-existing agent %s", agent.Address)
	}
	agent.Stats.mutex.Lock()
	defer agent.Stats.mutex.Unlock()

	bind9CommErrors := agent.Stats.Bind9CommErrors[app.GetID()]
	if err != nil {
		bind9CommErrors.Stats++
		if bind9CommErrors.Stats > 1 {
			err = errors.Errorf("failed to send named stats command via the agent %s, BIND 9 is still failing",
				agent.Address)
		}
		// Log the commands that failed.
		log.WithFields(log.Fields{
			"agent":     addrPort,
			"stats URL": statsURL,
		}).Warnf("Failed to send the following named stats command: %s", storkReq.NamedStatsRequest)
	} else {
		bind9CommErrors.Stats = 0
	}
	agent.Stats.Bind9CommErrors[app.GetID()] = bind9CommErrors
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
func (agents *connectedAgentsData) ForwardToKeaOverHTTP(ctx context.Context, app ControlledApp, commands []keactrl.SerializableCommand, cmdResponses ...any) (*KeaCmdsResult, error) {
	agentAddress := app.GetMachineTag().GetAddress()
	agentPort := app.GetMachineTag().GetAgentPort()

	caAddress, caPort, _, caUseSecureProtocol, err := app.GetControlAccessPoint()
	if err != nil {
		log.WithFields(log.Fields{
			"address": caAddress,
			"port":    caPort,
		}).Warnf("no Kea CA access point found matching the specified address and port to send the commands")
		return nil, err
	}

	addrPort := net.JoinHostPort(agentAddress, strconv.FormatInt(agentPort, 10))
	caURL := storkutil.HostWithPortURL(caAddress, caPort, caUseSecureProtocol)

	// Prepare the on-wire representation of the commands.
	fdReq := &agentapi.ForwardToKeaOverHTTPReq{
		Url: caURL,
	}
	for _, cmd := range commands {
		fdReq.KeaRequests = append(fdReq.KeaRequests, &agentapi.KeaRequest{
			Request: cmd.Marshal(),
		})
	}
	// Send the commands to the Stork Agent and get the response.
	resp, err := agents.sendAndRecvViaQueue(addrPort, fdReq)

	// Check the communication issues with the Stork agent.
	commState, details := agents.checkAgentCommState(addrPort, fdReq, err, resp)
	switch commState {
	case newCommIssue:
		log.WithFields(log.Fields{
			"agent": addrPort,
		}).Warnf("Failed to send %d Kea command(s)", len(commands))
		agents.EventCenter.AddErrorEvent("communication with {app} on {machine} failed", app, app.GetMachineTag(), dbmodel.SSEConnectivity, details)

	case correctedCommIssue:
		agents.EventCenter.AddWarningEvent("communication with {app} on {machine} resumed", app, app.GetMachineTag(), dbmodel.SSEConnectivity, details)

	case existingCommIssue:
		log.WithFields(log.Fields{
			"agent": addrPort,
		}).Warnf("Failed to send %d Kea command(s); agent is still not responding", len(commands))

	default:
	}

	if err != nil {
		return nil, errors.Wrapf(err, "failed to send Kea commands to agent %s", addrPort)
	}
	response := resp.(*agentapi.ForwardToKeaOverHTTPRsp)
	if response == nil {
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
	controlAgentState, controlAgentErrors, dhcp4State, dhcp4ErrMessage, dhcp6State, dhcp6ErrMessage, d2State, d2ErrMessage := agents.checkKeaCommState(addrPort, commands, app, response)

	// Save Control Agent Errors.
	result.CmdsErrors = controlAgentErrors

	// Generate events for the Kea Control Agent.
	if controlAgentState == newCommIssue {
		// The connection was ok but now it is broken.
		log.WithFields(log.Fields{
			"agent": addrPort,
		}).Warnf("Failed to forward Kea command to Kea Control Agent")
		agents.EventCenter.AddErrorEvent("forwarding Kea command via the Kea Control Agent {daemon} on {machine} failed", app.GetDaemonTag(dbmodel.DaemonNameCA), app.GetMachineTag(), dbmodel.SSEConnectivity, controlAgentErrors)
	} else if controlAgentState == correctedCommIssue {
		// The connection was broken but now is ok.
		agents.EventCenter.AddWarningEvent("forwarding Kea command via the Kea Control Agent {daemon} on {machine} no longer fails", app.GetDaemonTag(dbmodel.DaemonNameCA), app.GetMachineTag(), dbmodel.SSEConnectivity)
	}
	// Generate events for the DHCPv4 server.
	if dhcp4State == newCommIssue {
		// The connection was ok but now it is broken.
		log.WithFields(log.Fields{
			"agent": addrPort,
		}).Warnf("Failed to forward Kea command to Kea DHCPv4 server")
		agents.EventCenter.AddErrorEvent("processing Kea command by {daemon} on {machine} failed", app.GetDaemonTag(dbmodel.DaemonNameDHCPv4), app.GetMachineTag(), dbmodel.SSEConnectivity, dhcp4ErrMessage)
	} else if dhcp4State == correctedCommIssue {
		// The connection was broken but now is ok.
		agents.EventCenter.AddWarningEvent("processing Kea command by {daemon} on {machine} no longer fails", app.GetDaemonTag(dbmodel.DaemonNameDHCPv4), app.GetMachineTag(), dbmodel.SSEConnectivity)
	}
	// Generate events for the DHCPv6 server.
	if dhcp6State == newCommIssue {
		// The connection was ok but now it is broken.
		log.WithFields(log.Fields{
			"agent": addrPort,
		}).Warnf("Failed to forward Kea command to Kea DHCPv6 server")
		agents.EventCenter.AddErrorEvent("processing Kea command by {daemon} on {machine} failed", app.GetDaemonTag(dbmodel.DaemonNameDHCPv6), app.GetMachineTag(), dbmodel.SSEConnectivity, dhcp6ErrMessage)
	} else if dhcp6State == correctedCommIssue {
		// The connection was broken but now is ok.
		agents.EventCenter.AddWarningEvent("processing Kea command by {daemon} on {machine} no longer fails", app.GetDaemonTag(dbmodel.DaemonNameDHCPv6), app.GetMachineTag(), dbmodel.SSEConnectivity)
	}
	// Generate events for the D2 server.
	if d2State == newCommIssue {
		// The connection was ok but now it is broken.
		log.WithFields(log.Fields{
			"agent": addrPort,
		}).Warnf("Failed to forward Kea command to Kea D2 server")
		agents.EventCenter.AddErrorEvent("processing Kea command by {daemon} on {machine} failed", app.GetDaemonTag(dbmodel.DaemonNameD2), app.GetMachineTag(), dbmodel.SSEConnectivity, d2ErrMessage)
	} else if d2State == correctedCommIssue {
		// The connection was broken but now is ok.
		agents.EventCenter.AddWarningEvent("processing Kea command by {daemon} on {machine} no longer fails", app.GetDaemonTag(dbmodel.DaemonNameD2), app.GetMachineTag(), dbmodel.SSEConnectivity)
	}

	// Get all responses from the Kea server.
	for idx, rsp := range response.GetKeaResponses() {
		cmdResp := cmdResponses[idx]
		// Try to parse the json response from the on-wire format.
		err = keactrl.UnmarshalResponseList(commands[idx], rsp.Response, cmdResp)
		if err != nil {
			err = errors.Wrapf(err, "failed to parse Kea response from %s, response was: %s", caURL, rsp)
			// The sufficient number of elements should have been already alocated but
			// let's make sure.
			if len(result.CmdsErrors) > idx {
				result.CmdsErrors[idx] = err
			} else {
				result.CmdsErrors = append(result.CmdsErrors, err)
			}
			// Failure to parse the response is treated as a Kea Control Agent error.
			if agent, err := agents.GetConnectedAgent(addrPort); err == nil {
				agent.Stats.mutex.Lock()
				keaCommErrors := agent.Stats.KeaCommErrors[app.GetID()]
				keaCommErrors.ControlAgent++
				agent.Stats.KeaCommErrors[app.GetID()] = keaCommErrors
				agent.Stats.mutex.Unlock()
			}
		}
	}

	// Everything was fine, so return no error.
	return result, nil
}

// Get the tail of the remote text file.
func (agents *connectedAgentsData) TailTextFile(ctx context.Context, machine dbmodel.MachineTag, path string, offset int64) ([]string, error) {
	addrPort := net.JoinHostPort(machine.GetAddress(), strconv.FormatInt(machine.GetAgentPort(), 10))

	// Get the path to the file and the (seek) info indicating the location
	// from which the tail should be fetched.
	req := &agentapi.TailTextFileReq{
		Path:   path,
		Offset: offset,
	}

	// Send the request via queue.
	agentResponse, err := agents.sendAndRecvViaQueue(addrPort, req)

	commIssue, details := agents.checkAgentCommState(addrPort, req, err, agentResponse)
	switch commIssue {
	case newCommIssue:
		log.WithFields(log.Fields{
			"agent": addrPort,
			"file":  path,
		}).Warn("Failed to tail the text file via the agent", path)
		agents.EventCenter.AddErrorEvent("communication with {machine} failed when tailing a log file", machine, dbmodel.SSEConnectivity, details)

	case correctedCommIssue:
		agents.EventCenter.AddWarningEvent("communication with {machine} resumed when tailing a log file", machine, dbmodel.SSEConnectivity, details)

	case existingCommIssue:
		log.WithFields(log.Fields{
			"agent": addrPort,
			"file":  path,
		}).Warn("Failed to tail the text file via the agent which is still not responding", path)
	default:
	}

	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch text file contents: %s", path)
	}

	response := agentResponse.(*agentapi.TailTextFileRsp)

	if response.Status.Code != agentapi.Status_OK {
		return nil, errors.New(response.Status.Message)
	}

	return response.Lines, nil
}
