package agentcommtest

import (
	"context"

	"isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

// Helper struct to mock Agents behavior.
type FakeAgents struct {
	RecordedURL      string
	RecordedCommands []agentcomm.KeaCommand
	mockKeaFunc      func(int, []interface{})
	CallNo           int

	RecordedAddress string
	RecordedPort    int64
	RecordedKey     string
	RecordedCommand string
	mockRndcOutput  string

	RecordedStatsURL string
	mockNamedFunc    func(int, interface{})

	MachineState *agentcomm.State
}

// mockRndcOutput returns some mocked named response.
func mockRndcOutput() string {
	return `version: 9.9.9
running on agent-bind9: Linux x86_64 4.15.0-72-generic
boot time: Mon, 03 Feb 2020 13:39:36 GMT
last configured: Mon, 03 Feb 2020 14:39:36 GMT
configuration file: /etc/bind/named.conf
CPUs found: 4
worker threads: 4
UDP listeners per interface: 3
number of zones: 101 (96 automatic)
debug level: 0
xfers running: 0
xfers deferred: 0
soa queries in progress: 0
query logging is OFF
recursive clients: 0/900/1000
tcp clients: 3/150
server is up and running`
}

// Creates new instance of the FakeAgents structure with the function returning
// a custom response set.
func NewFakeAgents(fnKea func(int, []interface{}), fnNamed func(int, interface{})) *FakeAgents {
	fa := &FakeAgents{
		mockKeaFunc:    fnKea,
		mockNamedFunc:  fnNamed,
		mockRndcOutput: mockRndcOutput(),
	}
	return fa
}

func (fa *FakeAgents) Shutdown() {}
func (fa *FakeAgents) GetConnectedAgent(address string) (*agentcomm.Agent, error) {
	return nil, nil
}

// Returns fake statistics for the selected connected agent.
func (fa *FakeAgents) GetConnectedAgentStats(address string, port int64) *agentcomm.AgentStats {
	stats := agentcomm.AgentStats{
		CurrentErrors: 1,
		AppCommStats: map[agentcomm.AppCommStatsKey]interface{}{
			{Address: "localhost", Port: 1234}: &agentcomm.AgentKeaCommStats{
				CurrentErrorsCA: 2,
				CurrentErrorsDaemons: map[string]int64{
					"dhcp4": 5,
				},
			},
			{Address: "localhost", Port: 4321}: &agentcomm.AgentBind9CommStats{
				CurrentErrorsRNDC:  2,
				CurrentErrorsStats: 3,
			},
		},
	}
	return &stats
}

// FakeAgents specific implementation of the GetState.
func (fa *FakeAgents) GetState(ctx context.Context, address string, agentPort int64) (*agentcomm.State, error) {
	if fa.MachineState != nil {
		return fa.MachineState, nil
	}

	state := agentcomm.State{
		Cpus:   1,
		Memory: 4,
	}
	return &state, nil
}

// Returns last received command by FakeAgents or nil if no command
// has been received yet.
func (fa *FakeAgents) GetLastCommand() *agentcomm.KeaCommand {
	if len(fa.RecordedCommands) == 0 {
		return nil
	}
	return &fa.RecordedCommands[len(fa.RecordedCommands)-1]
}

// FakeAgents specific implementation of the function to forward a command
// to the Kea servers. It records some arguments used in the call to this
// function so as they can be later validated. It also returns a custom
// response to the command by calling the function specified in the
// call to NewFakeAgents.
func (fa *FakeAgents) ForwardToKeaOverHTTP(ctx context.Context, dbApp *dbmodel.App, commands []*agentcomm.KeaCommand, cmdResponses ...interface{}) (*agentcomm.KeaCmdsResult, error) {
	ctrlPoint, _ := dbApp.GetAccessPoint(dbmodel.AccessPointControl)
	caAddress := ctrlPoint.Address
	caPort := ctrlPoint.Port

	caURL := storkutil.HostWithPortURL(caAddress, caPort)

	fa.RecordedURL = caURL
	result := &agentcomm.KeaCmdsResult{}
	for _, cmd := range commands {
		fa.RecordedCommands = append(fa.RecordedCommands, *cmd)
		result.CmdsErrors = append(result.CmdsErrors, nil)
	}
	// Generate response.
	if fa.mockKeaFunc != nil {
		fa.mockKeaFunc(fa.CallNo, cmdResponses)
	}
	fa.CallNo++
	return result, nil
}

// FakeAgents specific implementation of the function to forward a command
// to the named statistics-channel. It records some arguments used in the call
// to this function so as they can be later validated. It also returns a custom
// response to the command by calling the function specified in the
// call to NewFakeAgents.
func (fa *FakeAgents) ForwardToNamedStats(ctx context.Context, agentAddress string, agentPort int64, statsAddress string, statsPort int64, path string, statsOutput interface{}) error {
	fa.RecordedStatsURL = storkutil.HostWithPortURL(statsAddress, statsPort) + path

	// Generate response.
	if fa.mockNamedFunc != nil {
		fa.mockNamedFunc(fa.CallNo, statsOutput)
	}
	fa.CallNo++
	return nil
}

// FakeAgents specific implementation of the function to forward a command
// to rndc. It records some arguments used in the call to this function
// so as they can be later validated. It also returns a custom response
// to the command by calling the function specified in the call to
// NewFakeAgents.
func (fa *FakeAgents) ForwardRndcCommand(ctx context.Context, dbApp *dbmodel.App, command string) (*agentcomm.RndcOutput, error) {
	ctrlPoint, _ := dbApp.GetAccessPoint(dbmodel.AccessPointControl)
	fa.RecordedAddress = ctrlPoint.Address
	fa.RecordedPort = ctrlPoint.Port
	fa.RecordedKey = ctrlPoint.Key
	fa.RecordedCommand = command

	if fa.mockRndcOutput != "" {
		output := &agentcomm.RndcOutput{
			Output: fa.mockRndcOutput,
			Error:  nil,
		}
		return output, nil
	}

	return nil, nil
}

// Mimics tailing text file.
func (fa *FakeAgents) TailTextFile(ctx context.Context, agentAddress string, agentPort int64, path string, offset int64) ([]string, error) {
	return []string{"lorem ipsum"}, nil
}
