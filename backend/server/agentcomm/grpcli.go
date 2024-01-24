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

// Check connectivity with machine.
func (agents *connectedAgentsData) Ping(ctx context.Context, address string, agentPort int64) error {
	addrPort := net.JoinHostPort(address, strconv.FormatInt(agentPort, 10))

	// Call agent for version.
	resp, err := agents.sendAndRecvViaQueue(addrPort, &agentapi.PingReq{})
	if err != nil {
		return errors.Wrapf(err, "failed to ping agent %s", addrPort)
	}
	if resp.(*agentapi.PingRsp) != nil {
		return errors.Wrapf(err, "wrong response for ping agent %s", addrPort)
	}
	return nil
}

// Get version from agent.
func (agents *connectedAgentsData) GetState(ctx context.Context, address string, agentPort int64) (*State, error) {
	addrPort := net.JoinHostPort(address, strconv.FormatInt(agentPort, 10))

	// Call agent for version.
	resp, err := agents.sendAndRecvViaQueue(addrPort, &agentapi.GetStateReq{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get state from agent %s", addrPort)
	}
	grpcState := resp.(*agentapi.GetStateRsp)

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
		Address:                  address,
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
	if err != nil {
		if agent, agentExists := agents.AgentsMap[addrPort]; agentExists {
			agent.Stats.mutex.Lock()
			defer agent.Stats.mutex.Unlock()
			if agent.Stats.CurrentErrors == 1 {
				// If this is the first time we failed to communicate with the
				// agent, let's print the stack trace for debugging purposes.
				err = errors.WithStack(err)
			} else {
				// This is not the first time we can't communicate with the
				// agent. Let's be brief and say that the communication is
				// still broken.
				err = errors.Errorf("failed to send rndc command via the agent %s, the agent is still not responding",
					agent.Address)
			}
			// Log the commands that failed.
			log.WithFields(log.Fields{
				"agent": addrPort,
				"rndc":  net.JoinHostPort(ctrlAddress, strconv.FormatInt(ctrlPort, 10)),
			}).Warnf("Failed to send the following rndc command: %s", command)
		}
		agents.EventCenter.AddErrorEvent("cannot connect to agent on {machine}", app.GetMachineTag(), dbmodel.SSEConnectivity)
		return nil, err
	}
	response := resp.(*agentapi.ForwardRndcCommandRsp)

	if response.Status.Code != agentapi.Status_OK {
		err = errors.New(response.Status.Message)
	}

	result := &RndcOutput{
		Output: "",
		Error:  nil,
	}
	if err == nil {
		rndcResponse := response.GetRndcResponse()
		if rndcResponse.Status.Code != agentapi.Status_OK {
			result.Error = errors.New(response.Status.Message)
		} else {
			result.Output = rndcResponse.Response
		}
	}

	if err != nil || result.Error != nil {
		var errStr string
		if err != nil {
			errStr = fmt.Sprintf("%s", err)
		} else {
			errStr = fmt.Sprintf("%s", result.Error)
		}
		agents.EventCenter.AddErrorEvent("Communication with {app} failed", errStr, app, dbmodel.SSEConnectivity)
	}

	// Start updating error statistics for this agent and the BIND9 app we've
	// been communicating with.
	agent, agentExists := agents.AgentsMap[addrPort]
	if agentExists {
		// This function may be called by multiple goroutines, so we need to make
		// sure that the statistics update is safe in terms of concurrent access.
		agent.Stats.mutex.Lock()
		defer agent.Stats.mutex.Unlock()

		// For this address and port we may already have BIND9 statistics stored.
		// If not, this is first time we communicate with this endpoint.
		bind9CommStats, bind9CommStatsExist := agent.Stats.AppCommStats[AppCommStatsKey{ctrlAddress, ctrlPort}].(*AgentBind9CommStats)
		if !bind9CommStatsExist {
			bind9CommStats = &AgentBind9CommStats{}
			agent.Stats.AppCommStats[AppCommStatsKey{ctrlAddress, ctrlPort}] = bind9CommStats
		}
		if err != nil || result.Error != nil {
			bind9CommStats.CurrentErrorsRNDC++
			// This is not the first tie the BIND9 RNDC is not responding, so let's
			// print the brief message.
			if bind9CommStats.CurrentErrorsRNDC > 1 {
				result.Error = errors.Errorf("failed to send rndc command via the agent %s, BIND 9 is still not responding",
					agent.Address)
				err = result.Error
				// Log the commands that failed.
				log.WithFields(log.Fields{
					"agent": addrPort,
					"rndc":  net.JoinHostPort(ctrlAddress, strconv.FormatInt(ctrlPort, 10)),
				}).Warnf("Failed to send the following rndc command: %s", command)
			}
		} else {
			bind9CommStats.CurrentErrorsRNDC = 0
		}
	}

	return result, err
}

// Forwards a statistics request via the Stork Agent to the named daemon and
// then parses the response. statsAddress, statsPort and statsPath are used to
// construct the URL to the statistics-channel of the named daemon.
func (agents *connectedAgentsData) ForwardToNamedStats(ctx context.Context, agentAddress string, agentPort int64, statsAddress string, statsPort int64, statsPath string, statsOutput interface{}) error {
	addrPort := net.JoinHostPort(agentAddress, strconv.FormatInt(agentPort, 10))
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
	if err != nil {
		if agent, agentExists := agents.AgentsMap[addrPort]; agentExists {
			agent.Stats.mutex.Lock()
			defer agent.Stats.mutex.Unlock()
			if agent.Stats.CurrentErrors == 1 {
				// If this is the first time we failed to communicate with the
				// agent, let's print the stack trace for debugging purposes.
				err = errors.WithStack(err)
			} else {
				// This is not the first time we can't communicate with the
				// agent. Let's be brief and say that the communication is
				// still broken.
				err = errors.Errorf("failed to send named statistics command via the agent %s, the agent is still not responding",
					agent.Address)
			}
			// Log the commands that failed.
			log.WithFields(log.Fields{
				"agent":     addrPort,
				"stats URL": statsURL,
			}).Warnf("Failed to send the following named statistics command: %+v", storkReq.NamedStatsRequest)
		}
		return err
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
	agent, agentExists := agents.AgentsMap[addrPort]
	if agentExists {
		// This function may be called by multiple goroutines, so we need to make
		// sure that the statistics update is safe in terms of concurrent access.
		agent.Stats.mutex.Lock()
		defer agent.Stats.mutex.Unlock()

		// For this address and port we may already have BIND9 statistics stored.
		// If not, this is first time we communicate with this endpoint.
		bind9CommStats, bind9CommStatsExist := agent.Stats.AppCommStats[AppCommStatsKey{statsAddress, statsPort}].(*AgentBind9CommStats)
		if !bind9CommStatsExist {
			bind9CommStats = &AgentBind9CommStats{}
			agent.Stats.AppCommStats[AppCommStatsKey{statsAddress, statsPort}] = bind9CommStats
		}
		if err != nil {
			bind9CommStats.CurrentErrorsStats++
			// This is not the first tie the BIND9 stats is not responding, so let's
			// print the brief message.
			if bind9CommStats.CurrentErrorsStats > 1 {
				err = errors.Errorf("failed to send named stats command via the agent %s, BIND 9 is still not responding",
					agent.Address)
				// Log the commands that failed.
				log.WithFields(log.Fields{
					"agent":     addrPort,
					"stats URL": statsURL,
				}).Warnf("Failed to send the following named stats command: %s", storkReq.NamedStatsRequest)
			}
		} else {
			bind9CommStats.CurrentErrorsStats = 0
		}
	}

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
// parses the response. caAddress and caPort are used to construct the URL
// of the Kea Control Agent to which the command should be sent.
func (agents *connectedAgentsData) ForwardToKeaOverHTTP(ctx context.Context, app ControlledApp, commands []keactrl.SerializableCommand, cmdResponses ...interface{}) (*KeaCmdsResult, error) {
	agentAddress := app.GetMachineTag().GetAddress()
	agentPort := app.GetMachineTag().GetAgentPort()

	caAddress, caPort, _, caUseSecureProtocol, err := app.GetControlAccessPoint()
	if err != nil {
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

	// Send the commands to the Stork Agent.
	resp, err := agents.sendAndRecvViaQueue(addrPort, fdReq)

	// This should always return an agent but we make this check to be safe
	// and not panic if someone has screwed up something in the code.
	// Concurrent access should be safe assuming that the agent has been
	// already added to the map by the GetConnectedAgent function.
	agent, agentExists := agents.AgentsMap[addrPort]
	if !agentExists {
		err = errors.Errorf("missing agent in agents map: %s", addrPort)
		return nil, err
	}

	// Lock agent comm stats, they may be modified below. This
	// function may be called by multiple goroutines, so we need
	// to make sure that the statistics update is safe in terms of
	// concurrent access.  There should be no time consuming
	// operations below to not block other requests to this agent
	// from other goroutines.
	agent.Stats.mutex.Lock()
	defer agent.Stats.mutex.Unlock()

	prevAgentErrorsCnt := agent.Stats.CurrentErrors

	// now check error from agents.sendAndRecvViaQueue
	if err != nil {
		// agent.Stats.CurrentErrors is incremented in manager.go during agents.sendAndRecvViaQueue
		if agent.Stats.CurrentErrors == 0 {
			// If this is the first time we failed to communicate with the
			// agent, let's print the stack trace for debugging purposes.
			err = errors.WithStack(err)
			agents.EventCenter.AddErrorEvent("Cannot connect to agent on {machine}", err.Error(), app.GetMachineTag(), dbmodel.SSEConnectivity)
		} else {
			// This is not the first time we can't communicate with the
			// agent. Let's be brief and say that the communication is
			// still broken.
			err = errors.Errorf("failed to send Kea commands via the agent %s, the agent is still not responding",
				agent.Address)
		}
		agent.Stats.CurrentErrors++

		// Log the commands that failed.
		log.WithFields(log.Fields{
			"agent": addrPort,
			"kea":   caURL,
		}).Warnf("Failed to send the following commands: %+v", fdReq.KeaRequests)
		return nil, err
	}

	agent.Stats.CurrentErrors = 0
	if prevAgentErrorsCnt > 0 {
		agents.EventCenter.AddWarningEvent("Communication with stork agent on {machine} resumed", app.GetMachineTag(), dbmodel.SSEConnectivity)
	}

	fdRsp := resp.(*agentapi.ForwardToKeaOverHTTPRsp)

	// Gather errors in communication via the Kea Control
	// Agent. It is possible to send multiple commands so there
	// may be multiple errors.
	caErrorsCount := int64(0)
	caErrorStr := ""

	result := &KeaCmdsResult{}
	result.Error = nil
	if fdRsp.Status.Code != agentapi.Status_OK {
		result.Error = errors.New(fdRsp.Status.Message)
		caErrorsCount++
		caErrorStr += "\n" + fdRsp.Status.Message
	}

	// Gather errors from daemons (including CA).
	daemonErrorsCount := make(map[string]int64)

	// Get all responses from the Kea server.
	for idx, rsp := range fdRsp.GetKeaResponses() {
		cmdResp := cmdResponses[idx]
		if rsp.Status.Code != agentapi.Status_OK {
			result.CmdsErrors = append(result.CmdsErrors, errors.New(rsp.Status.Message))
			caErrorsCount++
			caErrorStr += "\n" + rsp.Status.Message
			continue
		}

		// Try to parse the json response from the on-wire format.
		err = keactrl.UnmarshalResponseList(commands[idx], rsp.Response, cmdResp)
		if err != nil {
			err = errors.Wrapf(err, "failed to parse Kea response from %s, response was: %s", caURL, rsp)
			result.CmdsErrors = append(result.CmdsErrors, err)
			// Issues with parsing the response count as issues with communication.
			caErrorsCount++
			caErrorStr += "\n" + fmt.Sprintf("%+v", err)
			continue
		}

		// The response should be a list.
		cmdRespList := reflect.ValueOf(cmdResp).Elem()
		if cmdRespList.Kind() != reflect.Slice {
			err = errors.Wrapf(err, "no well-formatted response from Kea CA %s, response was: %s", caURL, rsp)
			result.CmdsErrors = append(result.CmdsErrors, err)
			// Issues with parsing the response count as issues with communication.
			caErrorsCount++
			caErrorStr += "\n" + fmt.Sprintf("%+v", err)
			continue
		}
		// Iterate over the responses from individual servers behind the CA.
		for i := 0; i < cmdRespList.Len(); i++ {
			// The Daemon field of the response should be present if the
			// caller used the right data structures.
			cmdRespItem := cmdRespList.Index(i)
			daemonField := cmdRespItem.FieldByName("Daemon")
			if !daemonField.IsValid() {
				log.Warnf("Missing Daemon field in response from Kea CA")
				continue
			}
			// The response should contain the result.
			resultField := cmdRespItem.FieldByName("Result")
			if !resultField.IsValid() {
				log.Warnf("Missing Result field in response from Kea CA")
				continue
			}
			daemonName := daemonField.String()
			if daemonName == "" {
				daemonName = "ca"
			}

			// If error was returned, let's bump up the number of errors
			// for this daemon. Otherwise, let's reset the counter.
			if resultField.Int() == keactrl.ResponseError {
				daemonErrorsCount[daemonName]++
			} else {
				daemonErrorsCount[daemonName] = 0
			}
		}
		result.CmdsErrors = append(result.CmdsErrors, nil)
	}

	agents.updateErrorStatsAndRaiseEvents(agent, caAddress, caPort, app, caErrorsCount, addrPort, caURL, fdReq, caErrorStr, daemonErrorsCount)

	// Everything was fine, so return no error.
	return result, nil
}

func (agents *connectedAgentsData) updateErrorStatsAndRaiseEvents(agent *Agent, caAddress string, caPort int64, app ControlledApp, caErrorsCount int64, addrPort, caURL string, fdReq *agentapi.ForwardToKeaOverHTTPReq, caErrorStr string, daemonErrorsCount map[string]int64) {
	// Start updating error statistics for this agent and the Kea app we've been
	// communicating with.
	var (
		keaCommStats      *AgentKeaCommStats
		keaCommStatsExist bool
	)

	// For this address and port we may already have Kea statistics stored.
	// If not, this is first time we communicate with this endpoint.
	keaCommStats, keaCommStatsExist = agent.Stats.AppCommStats[AppCommStatsKey{caAddress, caPort}].(*AgentKeaCommStats)
	// Seems that this is the first request to this Kea server.
	if !keaCommStatsExist {
		keaCommStats = &AgentKeaCommStats{}
		keaCommStats.CurrentErrorsDaemons = make(map[string]int64)
		agent.Stats.AppCommStats[AppCommStatsKey{caAddress, caPort}] = keaCommStats
	}

	// prepare daemons map for quick access
	daemonsMap := make(map[string]dbmodel.DaemonTag)
	for _, dmn := range app.GetDaemonTags() {
		daemonsMap[dmn.GetName()] = dmn
	}

	// If there are any communication errors with CA, let's add them.
	// Otherwise, let's reset the counter.
	prevErrorsCA := keaCommStats.CurrentErrorsCA
	if caErrorsCount > 0 {
		keaCommStats.CurrentErrorsCA += caErrorsCount
		// This is the first time we have a problem in communication with the Kea Control Agent,
		// so let's print a brief message and raise an event.
		if prevErrorsCA == 0 {
			log.WithFields(log.Fields{
				"agent": addrPort,
				"kea":   caURL,
			}).Warnf("communication failed: %+v", fdReq.KeaRequests)
			dmn, ok := daemonsMap["ca"]
			if ok {
				agents.EventCenter.AddErrorEvent("Communication with {daemon} of {app} failed", strings.TrimSpace(caErrorStr), &dmn, app, dbmodel.SSEConnectivity)
			} else {
				agents.EventCenter.AddErrorEvent("Communication with CA daemon of {app} failed", strings.TrimSpace(caErrorStr), app, dbmodel.SSEConnectivity)
			}
		}
	} else {
		keaCommStats.CurrentErrorsCA = 0
		// Now it seems all is ok but there were problems earlier so raise an event
		// that communication resumed.
		if prevErrorsCA > 0 {
			dmn, ok := daemonsMap["ca"]
			if ok {
				agents.EventCenter.AddWarningEvent("Communication with {daemon} of {app} resumed", &dmn, app, dbmodel.SSEConnectivity)
			} else {
				agents.EventCenter.AddWarningEvent("Communication with CA daemon of {app} resumed", app, dbmodel.SSEConnectivity)
			}
		}
	}

	// Set the counters for individual daemons.
	for dmnName, errCnt := range daemonErrorsCount {
		prevErrors, ok := keaCommStats.CurrentErrorsDaemons[dmnName]
		if !ok || errCnt == 0 {
			keaCommStats.CurrentErrorsDaemons[dmnName] = errCnt
		} else {
			keaCommStats.CurrentErrorsDaemons[dmnName] += errCnt
		}

		// if communication with given daemon started or stopped failing then generate an event
		currentErrors := keaCommStats.CurrentErrorsDaemons[dmnName]
		if (prevErrors == 0 && currentErrors > 0) || (prevErrors > 0 && currentErrors == 0) {
			for _, dmn := range app.GetDaemonTags() {
				if dmn.GetName() == dmnName {
					if currentErrors == 0 {
						agents.EventCenter.AddWarningEvent("Communication with {daemon} of {app} resumed", dmn, app, dbmodel.SSEConnectivity)
					} else {
						agents.EventCenter.AddErrorEvent("Communication with {daemon} of {app} failed", dmn, app, dbmodel.SSEConnectivity)
					}
					break
				}
			}
		}
	}
}

// Get the tail of the remote text file.
func (agents *connectedAgentsData) TailTextFile(ctx context.Context, agentAddress string, agentPort int64, path string, offset int64) ([]string, error) {
	addrPort := net.JoinHostPort(agentAddress, strconv.FormatInt(agentPort, 10))

	// Get the path to the file and the (seek) info indicating the location
	// from which the tail should be fetched.
	req := &agentapi.TailTextFileReq{
		Path:   path,
		Offset: offset,
	}

	// Send the request via queue.
	agentResponse, err := agents.sendAndRecvViaQueue(addrPort, req)
	if err != nil {
		log.WithFields(log.Fields{
			"agent": addrPort,
			"file":  path,
		}).Warnf("Failed to fetch text file contents")

		return nil, errors.Wrapf(err, "failed to fetch text file contents: %s", path)
	}

	response := agentResponse.(*agentapi.TailTextFileRsp)

	if response.Status.Code != agentapi.Status_OK {
		return nil, errors.New(response.Status.Message)
	}

	return response.Lines, nil
}
