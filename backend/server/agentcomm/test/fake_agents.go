package agentcommtest

import (
	"context"
	"encoding/json"
	"iter"

	"github.com/pkg/errors"
	agentapi "isc.org/stork/api"
	bind9config "isc.org/stork/daemoncfg/bind9"
	"isc.org/stork/daemoncfg/dnsconfig"
	keactrl "isc.org/stork/daemonctrl/kea"
	pdnsdata "isc.org/stork/daemondata/pdns"
	dnsmodel "isc.org/stork/datamodel/dns"
	"isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

var _ agentcomm.ConnectedAgents = (*FakeAgents)(nil)

// Helper struct to mock Agents behavior.
type FakeAgents struct {
	RecordedURLs     []string
	RecordedCommands []keactrl.SerializableCommand
	mockKeaFunc      []func(int, []any)
	CallNo           int

	RecordedAddress string
	RecordedPort    int64
	RecordedKey     string
	RecordedCommand string
	mockRndcOutput  string

	RecordedStatsURL string
	mockNamedFunc    func(int, interface{})

	MachineState   *agentcomm.State
	GetStateCalled bool
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
func NewFakeAgents(fnKea func(int, []any), fnNamed func(int, interface{})) *FakeAgents {
	fa := &FakeAgents{
		mockNamedFunc:  fnNamed,
		mockRndcOutput: mockRndcOutput(),
		GetStateCalled: false,
	}
	if fnKea != nil {
		fa.mockKeaFunc = append(fa.mockKeaFunc, fnKea)
	}
	return fa
}

// Create new instance of the FakeAgents structure with multiple mock functions
// returning Kea responses. The subsequent mock functions are invoked for each
// new call.
func NewKeaFakeAgents(fnsKea ...func(int, []any)) *FakeAgents {
	fa := &FakeAgents{
		mockKeaFunc: fnsKea,
	}
	return fa
}

// Do nothing. Always returns nil.
func (fa *FakeAgents) Ping(ctx context.Context, machine dbmodel.MachineTag) error {
	return nil
}

// Do nothing.
func (fa *FakeAgents) Shutdown() {}

// Returns fake statistics for the selected connected agent.
func (fa *FakeAgents) GetConnectedAgentStatsWrapper(address string, port int64) *agentcomm.CommStatsWrapper {
	return agentcomm.NewCommStatsWrapper(agentcomm.NewAgentStats())
}

// FakeAgents specific implementation of the GetState.
func (fa *FakeAgents) GetState(ctx context.Context, machine dbmodel.MachineTag) (*agentcomm.State, error) {
	fa.GetStateCalled = true

	if fa.MachineState != nil {
		return fa.MachineState, nil
	}

	state := agentcomm.State{
		Cpus:         1,
		Memory:       4,
		AgentVersion: "2.3.0",
	}
	return &state, nil
}

// Returns the received command by FakeAgents with a specific index or nil if
// no command has been received yet or the index is out of range.
func (fa *FakeAgents) GetCommand(index int) *keactrl.Command {
	if index < 0 || index >= len(fa.RecordedCommands) {
		return nil
	}
	return fa.RecordedCommands[index].(*keactrl.Command)
}

// Returns last received command by FakeAgents or nil if no command
// has been received yet.
func (fa *FakeAgents) GetLastCommand() *keactrl.Command {
	return fa.GetCommand(len(fa.RecordedCommands) - 1)
}

// Returns arguments of the received command by FakeAgents with a specific
// index or nil if no command has been received yet.
func (fa *FakeAgents) GetCommandArguments(index int) map[string]any {
	command := fa.GetCommand(index)
	if command == nil {
		return nil
	}

	arguments, ok := command.Arguments.(map[string]any)
	if !ok {
		return nil
	}
	return arguments
}

// Returns arguments of the last received command by FakeAgents or nil if no
// command has been received yet.
func (fa *FakeAgents) GetLastCommandArguments() map[string]any {
	return fa.GetCommandArguments(len(fa.RecordedCommands) - 1)
}

// FakeAgents specific implementation of the function to forward a command
// to the Kea servers. It records some arguments used in the call to this
// function so as they can be later validated. It also returns a custom
// response to the command by calling the function specified in the
// call to NewFakeAgents or NewKeaFakeAgents.
func (fa *FakeAgents) ForwardToKeaOverHTTP(ctx context.Context, daemon agentcomm.ControlledDaemon, commands []keactrl.SerializableCommand, cmdResponses ...any) (*agentcomm.KeaCmdsResult, error) {
	if len(cmdResponses) != len(commands) {
		return nil, errors.New("number of command responses does not match number of commands")
	}

	accessPoint, _ := daemon.GetAccessPoint(dbmodel.AccessPointControl)
	caURL := storkutil.HostWithPortURL(accessPoint.Address, accessPoint.Port, string(accessPoint.Protocol))

	fa.RecordedURLs = append(fa.RecordedURLs, caURL)

	// Generate response.
	var mock func(int, []any)
	if len(fa.mockKeaFunc) > 0 {
		if fa.CallNo >= len(fa.mockKeaFunc) {
			mock = fa.mockKeaFunc[len(fa.mockKeaFunc)-1]
		} else {
			mock = fa.mockKeaFunc[fa.CallNo]
		}
		mock(fa.CallNo, cmdResponses)
	}
	fa.CallNo++

	result := &agentcomm.KeaCmdsResult{}
	for i, cmd := range commands {
		fa.RecordedCommands = append(fa.RecordedCommands, cmd)

		responseBytes, _ := json.Marshal(cmdResponses[i])
		var response keactrl.Response
		_ = json.Unmarshal(responseBytes, &response)

		err := response.GetError()
		result.CmdsErrors = append(result.CmdsErrors, err)
	}

	return result, nil
}

// FakeAgents specific implementation of the function to forward a command
// to the named statistics-channel. It records some arguments used in the call
// to this function so as they can be later validated. It also returns a custom
// response to the command by calling the function specified in the
// call to NewFakeAgents.
func (fa *FakeAgents) ForwardToNamedStats(ctx context.Context, daemon agentcomm.ControlledDaemon, requestType agentcomm.ForwardToNamedStatsRequestType, statsOutput interface{}) error {
	accessPoint, _ := daemon.GetAccessPoint(dbmodel.AccessPointStatistics)
	fa.RecordedStatsURL = storkutil.HostWithPortURL(accessPoint.Address, accessPoint.Port, string(accessPoint.Protocol))

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
func (fa *FakeAgents) ForwardRndcCommand(ctx context.Context, daemon agentcomm.ControlledDaemon, command string) (*agentcomm.RndcOutput, error) {
	accessPoint, err := daemon.GetAccessPoint(dbmodel.AccessPointControl)
	if err != nil {
		return nil, err
	}

	fa.RecordedAddress = accessPoint.Address
	fa.RecordedPort = accessPoint.Port
	fa.RecordedKey = accessPoint.Key
	fa.RecordedCommand = command

	if fa.mockRndcOutput != "" {
		output := &agentcomm.RndcOutput{
			Output: fa.mockRndcOutput,
		}
		return output, nil
	}

	return nil, nil
}

// FakeAgents specific implementation of the function to get the PowerDNS
// server information. It returns nil.
func (fa *FakeAgents) GetPowerDNSServerInfo(ctx context.Context, daemon agentcomm.ControlledDaemon) (*pdnsdata.ServerInfo, error) {
	return &pdnsdata.ServerInfo{
		Version: "4.7.0",
	}, nil
}

// Mimics tailing text file.
func (fa *FakeAgents) TailTextFile(ctx context.Context, machine dbmodel.MachineTag, path string, offset int64) ([]string, error) {
	return []string{"lorem ipsum"}, nil
}

// FakeAgents specific implementation of the function which gathers the zones from the
// agents one by one.
func (fa *FakeAgents) ReceiveZones(ctx context.Context, daemon agentcomm.ControlledDaemon, filter *dnsmodel.ZoneFilter) iter.Seq2[*dnsmodel.ExtendedZone, error] {
	return nil
}

func (fa *FakeAgents) ReceiveZoneRRs(ctx context.Context, daemon agentcomm.ControlledDaemon, zoneName string, viewName string) iter.Seq2[[]*dnsconfig.RR, error] {
	return nil
}

func (fa *FakeAgents) ReceiveBind9FormattedConfig(ctx context.Context, app agentcomm.ControlledDaemon, fileSelector *bind9config.FileTypeSelector, filter *bind9config.Filter) iter.Seq2[*agentapi.ReceiveBind9ConfigRsp, error] {
	return nil
}
