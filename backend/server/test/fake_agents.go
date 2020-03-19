package storktest

import (
	"context"

	"isc.org/stork/server/agentcomm"
)

// Helper struct to mock Agents behavior.
type FakeAgents struct {
	RecordedURL      string
	RecordedCommands []string
	mockKeaFunc      func(int, []interface{})
	callNo           int

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

// FakeAgents specific implementation of the function to forward a command
// to the Kea servers. It records some arguments used in the call to this
// function so as they can be later validated. It also returns a custom
// response to the command by calling the function specified in the
// call to NewFakeAgents.
func (fa *FakeAgents) ForwardToKeaOverHTTP(ctx context.Context, agentAddress string, agentPort int64, caURL string, commands []*agentcomm.KeaCommand, cmdResponses ...interface{}) (*agentcomm.KeaCmdsResult, error) {
	fa.RecordedURL = caURL
	result := &agentcomm.KeaCmdsResult{}
	for _, cmd := range commands {
		fa.RecordedCommands = append(fa.RecordedCommands, cmd.Command)
		result.CmdsErrors = append(result.CmdsErrors, nil)
	}
	// Generate response.
	if fa.mockKeaFunc != nil {
		fa.mockKeaFunc(fa.callNo, cmdResponses)
	}
	fa.callNo++
	return result, nil
}

// FakeAgents specific implementation of the function to forward a command
// to the named statistics-channel. It records some arguments used in the call
// to this function so as they can be later validated. It also returns a custom
// response to the command by calling the function specified in the
// call to NewFakeAgents.
func (fa *FakeAgents) ForwardToNamedStats(ctx context.Context, agentAddress string, agentPort int64, statsURL string, statsOutput interface{}) error {
	fa.RecordedStatsURL = statsURL

	// Generate response.
	if fa.mockNamedFunc != nil {
		fa.mockNamedFunc(fa.callNo, statsOutput)
	}
	fa.callNo++
	return nil
}

// FakeAgents specific implementation of the function to forward a command
// to rndc. It records some arguments used in the call to this function
// so as they can be later validated. It also returns a custom response
// to the command by calling the function specified in the call to
// NewFakeAgents.
func (fa *FakeAgents) ForwardRndcCommand(ctx context.Context, agentAddress string, agentPort int64, rndcSettings agentcomm.Bind9Control, command string) (*agentcomm.RndcOutput, error) {
	fa.RecordedAddress = rndcSettings.Address
	fa.RecordedPort = rndcSettings.Port
	fa.RecordedKey = rndcSettings.Key
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
