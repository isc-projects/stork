package agentcomm

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"strconv"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	agentapi "isc.org/stork/api"
	storkutil "isc.org/stork/util"
)

// An access point for an application to retrieve information such
// as status or metrics.
type AccessPoint struct {
	Type    string
	Address string
	Port    int64
	Key     string
}

// Currently supported types are: "control" and "statistics"
const AccessPointControl = "control"
const AccessPointStatistics = "statistics"

type App struct {
	Type         string
	AccessPoints []AccessPoint
}

// Currently supported types are: "kea" and "bind9"
const AppTypeKea = "kea"
const AppTypeBind9 = "bind9"

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
				Type:    point.Type,
				Address: point.Address,
				Port:    point.Port,
				Key:     point.Key,
			})
		}

		apps = append(apps, &App{
			Type:         app.Type,
			AccessPoints: accessPoints,
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
		LastVisitedAt:        storkutil.UTCNow(),
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
		Address: rndcSettings.Address,
		Port:    rndcSettings.Port,
		Key:     rndcSettings.Key,
		RndcRequest: &agentapi.RndcRequest{
			Request: command,
		},
	}

	// Send the command to the Stork agent.
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
				err = fmt.Errorf("failed to send rndc command via the agent %s, the agent is still not responding",
					agent.Address)
			}
			// Log the commands that failed.
			log.WithFields(log.Fields{
				"agent": addrPort,
				"rndc":  net.JoinHostPort(rndcSettings.Address, strconv.FormatInt(rndcSettings.Port, 10)),
			}).Warnf("failed to send the following rndc command: %s", command)
		}
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
		bind9CommStats, bind9CommStatsExist := agent.Stats.AppCommStats[AppCommStatsKey{rndcSettings.Address, rndcSettings.Port}].(*AgentBind9CommStats)
		if !bind9CommStatsExist {
			bind9CommStats = &AgentBind9CommStats{}
			agent.Stats.AppCommStats[AppCommStatsKey{rndcSettings.Address, rndcSettings.Port}] = bind9CommStats
		}
		if err != nil || result.Error != nil {
			bind9CommStats.CurrentErrorsRNDC++
			// This is not the first tie the BIND9 RNDC is not responding, so let's
			// print the brief message.
			if bind9CommStats.CurrentErrorsRNDC > 1 {
				result.Error = fmt.Errorf("failed to send rndc command via the agent %s, BIND9 is still not responding",
					agent.Address)
				err = result.Error
				// Log the commands that failed.
				log.WithFields(log.Fields{
					"agent": addrPort,
					"rndc":  net.JoinHostPort(rndcSettings.Address, strconv.FormatInt(rndcSettings.Port, 10)),
				}).Warnf("failed to send the following rndc command: %s", command)
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
	statsURL := storkutil.HostWithPortURL(statsAddress, statsPort)
	statsURL += statsPath

	// Prepare the on-wire representation of the commands.
	storkReq := &agentapi.ForwardToNamedStatsReq{
		Url: statsURL,
	}
	storkReq.NamedStatsRequest = &agentapi.NamedStatsRequest{
		Request: "",
	}

	// Send the commands to the Stork agent.
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
				err = fmt.Errorf("failed to send named statistics command via the agent %s, the agent is still not responding",
					agent.Address)
			}
			// Log the commands that failed.
			log.WithFields(log.Fields{
				"agent":     addrPort,
				"stats URL": statsURL,
			}).Warnf("failed to send the following named statistics command: %+v", storkReq.NamedStatsRequest)
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
				err = fmt.Errorf("failed to send named stats command via the agent %s, BIND9 is still not responding",
					agent.Address)
				// Log the commands that failed.
				log.WithFields(log.Fields{
					"agent":     addrPort,
					"stats URL": statsURL,
				}).Warnf("failed to send the following named stats command: %s", storkReq.NamedStatsRequest)
			}
		} else {
			bind9CommStats.CurrentErrorsStats = 0
		}
	}

	return err
}

type KeaCmdsResult struct {
	Error      error
	CmdsErrors []error
}

// Forwards a Kea command via the Stork Agent and Kea Control Agent and then
// parses the response. caAddress and caPort are used to construct the URL
// of the Kea Control Agent to which the command should be sent.
func (agents *connectedAgentsData) ForwardToKeaOverHTTP(ctx context.Context, agentAddress string, agentPort int64, caAddress string, caPort int64, commands []*KeaCommand, cmdResponses ...interface{}) (*KeaCmdsResult, error) {
	addrPort := net.JoinHostPort(agentAddress, strconv.FormatInt(agentPort, 10))
	caURL := storkutil.HostWithPortURL(caAddress, caPort)

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
				err = fmt.Errorf("failed to send Kea commands via the agent %s, the agent is still not responding",
					agent.Address)
			}
			// Log the commands that failed.
			log.WithFields(log.Fields{
				"agent": addrPort,
				"kea":   caURL,
			}).Warnf("failed to send the following commands: %+v", fdReq.KeaRequests)
		}
		return nil, err
	}
	fdRsp := resp.(*agentapi.ForwardToKeaOverHTTPRsp)

	result := &KeaCmdsResult{}
	result.Error = nil
	if fdRsp.Status.Code != agentapi.Status_OK {
		result.Error = errors.New(fdRsp.Status.Message)
	}

	// Gather the total number of errors in communicating with the Kea
	// Control Agent. It is possible to send multiple commands so there
	// may be multiple errors.
	caErrors := int64(0)
	// Gather the per daemon errors in communication via Kea Control Agent.
	daemonErrors := make(map[string]int64)

	// Get all responses from the Kea server.
	for idx, rsp := range fdRsp.GetKeaResponses() {
		cmdResp := cmdResponses[idx]
		if rsp.Status.Code != agentapi.Status_OK {
			result.CmdsErrors = append(result.CmdsErrors, errors.New(rsp.Status.Message))
			caErrors++
			continue
		} else {
			// Communicated successfully, so let's reset the counter of
			// current errors.
			caErrors = 0
		}

		// Try to parse the response from the on-wire format.
		err = UnmarshalKeaResponseList(commands[idx], rsp.Response, cmdResp)
		if err != nil {
			// Issues with parsing the response count as issues with communication.
			caErrors++
			err = errors.Wrapf(err, "failed to parse Kea response from %s, response was: %s", caURL, rsp)
			result.CmdsErrors = append(result.CmdsErrors, err)
			continue
		}

		// The response is a list if we're forwarding the command to the
		// servers behind control agent.
		cmdRespList := reflect.ValueOf(cmdResp).Elem()
		if cmdRespList.Kind() != reflect.Slice {
			continue
		}
		// Iterate over the responses from individual servers behind the CA.
		for i := 0; i < cmdRespList.Len(); i++ {
			// The Daemon field of the response should be present if the
			// caller used the right data structures.
			cmdRespItem := cmdRespList.Index(i)
			daemonField := cmdRespItem.FieldByName("Daemon")
			if !daemonField.IsValid() {
				continue
			}
			// The response should contain the result.
			resultField := cmdRespItem.FieldByName("Result")
			if !resultField.IsValid() {
				continue
			}
			// If error was returned, let's bump up the number of errors
			// for this daemon. Otherwise, let's reset the counter.
			if resultField.Int() == KeaResponseError {
				daemonErrors[daemonField.String()]++
			} else {
				daemonErrors[daemonField.String()] = 0
			}
		}
		result.CmdsErrors = append(result.CmdsErrors, nil)
	}

	// Start updating error statistics for this agent and the Kea app we've been
	// communicating with.
	var (
		agent             *Agent
		agentExists       bool
		keaCommStats      *AgentKeaCommStats
		keaCommStatsExist bool
	)
	// This should always return an agent but we make this check to be safe
	// and not panic if someone has screwed up something in the code.
	// Concurrent access should be safe assuming that the agent has been
	// already added to the map by the GetConnectedAgent function.
	agent, agentExists = agents.AgentsMap[addrPort]
	if !agentExists {
		return result, nil
	}

	// This function may be called by multiple goroutines, so we need to make
	// sure that the statistics update is safe in terms of concurrent access.
	agent.Stats.mutex.Lock()
	defer agent.Stats.mutex.Unlock()

	// An error here indicates some communication problem with the agent.
	if fdRsp.Status.Code != agentapi.Status_OK {
		agent.Stats.CurrentErrors++
	} else {
		agent.Stats.CurrentErrors = 0
	}

	// For this address and port we may already have Kea statistics stored.
	// If not, this is first time we communicate with this endpoint.
	keaCommStats, keaCommStatsExist = agent.Stats.AppCommStats[AppCommStatsKey{caAddress, caPort}].(*AgentKeaCommStats)
	// Seems that this is the first request to this Kea server.
	if !keaCommStatsExist {
		keaCommStats = &AgentKeaCommStats{}
		keaCommStats.CurrentErrorsDaemons = make(map[string]int64)
		agent.Stats.AppCommStats[AppCommStatsKey{caAddress, caPort}] = keaCommStats
	}

	// Let's collect all daemon names to which we have sent any command.
	for _, cmd := range commands {
		if cmd.Daemons == nil {
			continue
		}
		for k := range *cmd.Daemons {
			if !(*cmd.Daemons)[k] {
				continue
			}
			// If the statistics entry does not exist for this daemon, let's
			// create it with the default value of 0.
			if _, ok := keaCommStats.CurrentErrorsDaemons[k]; !ok {
				keaCommStats.CurrentErrorsDaemons[k] = 0
			}
		}
	}

	// If there are any communication errors with CA, let's add them. Otherwise,
	// let's reset the counter.
	if caErrors > 0 {
		keaCommStats.CurrentErrorsCA += caErrors
		// This is not the first tie the Kea Control Agent is not responding, so let's
		// print the brief message.
		if keaCommStats.CurrentErrorsCA > 1 {
			for i := range result.CmdsErrors {
				result.CmdsErrors[i] = fmt.Errorf("failed to send commands to the Kea Control Agent %s, the Kea CA is still not responding", caURL)
			}
			log.WithFields(log.Fields{
				"agent": addrPort,
				"kea":   caURL,
			}).Warnf("failed to send the following commands: %+v", fdReq.KeaRequests)
		}
	} else {
		keaCommStats.CurrentErrorsCA = 0
	}

	// Set the counters for individual daemons.
	for k, d := range daemonErrors {
		if _, ok := keaCommStats.CurrentErrorsDaemons[k]; !ok || d == 0 {
			keaCommStats.CurrentErrorsDaemons[k] = d
		} else {
			keaCommStats.CurrentErrorsDaemons[k] += d
		}
	}

	// Everything was fine, so return no error.
	return result, nil
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
		}).Warnf("failed to fetch text file contents")

		return nil, errors.Wrapf(err, "failed to fetch text file contents: %s", path)
	}

	response := agentResponse.(*agentapi.TailTextFileRsp)

	if response.Status.Code != agentapi.Status_OK {
		return nil, errors.New(response.Status.Message)
	}

	return response.Lines, nil
}
