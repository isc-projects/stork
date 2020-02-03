package storktest

import (
	"context"
	"fmt"

	"isc.org/stork/server/agentcomm"
)

// Helper struct to mock Agents behavior.
type FakeAgents struct {
	RecordedURL      string
	RecordedCommands []string
	mockFunc         func(int, []interface{})
	callNo           int

	RecordedCtrlAddress string
	RecordedCtrlPort    int64
	RecordedCtrlKey     string
	RecordedCommand     string
	mockRndcOutput      string

	MachineState *agentcomm.State
}

// mockRndcOutput returns some mocked named response.
func mockRndcOutput() string {
	version := "9.9.9"
	bootTime := "Tue, 21 Jan 2020 08:25:43 GMT"
	lastConfigured := "Tue, 21 Jan 2020 17:04:40 GMT"
	return fmt.Sprintf("version: %s\nboot time: %s\nlast configured: %s\nserver is up and running", version, bootTime, lastConfigured)
}

// Creates new instance of the FakeAgents structure with the function returning
// a custom response set.
func NewFakeAgents(fn func(int, []interface{})) *FakeAgents {
	fa := &FakeAgents{
		mockFunc:       fn,
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
	if fa.mockFunc != nil {
		fa.mockFunc(fa.callNo, cmdResponses)
	}
	fa.callNo++
	return result, nil
}

// FakeAgents specific implementation of the function to forward a command
// to rndc. It records some arguments used in the call to this function
// so as they can be later validated. It also returns a custom response
// to the command by calling the function specified in the call to
// NewFakeAgents.
func (fa *FakeAgents) ForwardRndcCommand(ctx context.Context, agentAddress string, agentPort int64, rndcSettings agentcomm.Bind9Control, command string) (*agentcomm.RndcOutput, error) {
	fa.RecordedCtrlAddress = rndcSettings.CtrlAddress
	fa.RecordedCtrlPort = rndcSettings.CtrlPort
	fa.RecordedCtrlKey = rndcSettings.CtrlKey
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
