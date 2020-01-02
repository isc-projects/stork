package storktest

import (
	"context"
	"isc.org/stork/server/agentcomm"
)

// Helper struct to mock Agents behavior.
type FakeAgents struct {
	RecordedURL     string
	RecordedCommand string
	mockFunc        func(interface{})
}

func (fa *FakeAgents) Shutdown() {}
func (fa *FakeAgents) GetConnectedAgent(address string) (*agentcomm.Agent, error) {
	return nil, nil
}

// FakeAgents specific implementation of the GetState.
func (fa *FakeAgents) GetState(ctx context.Context, address string, agentPort int64) (*agentcomm.State, error) {
	state := agentcomm.State{
		Cpus:   1,
		Memory: 4,
	}
	return &state, nil
}

// Creates new instance of the FakeAgents structure with the function returning
// a custom response set.
func NewFakeAgents(fn func(interface{})) *FakeAgents {
	fa := &FakeAgents{
		mockFunc: fn,
	}
	return fa
}

// FakeAgents specific Implementation of the function to forward a command
// to the Kea servers. It records some arguments used in the call to this
// function so as they can be later validated. It also returns a custom
// response to the command by calling the function specified in the
// call to NewFakeAgents.
func (fa *FakeAgents) ForwardToKeaOverHttp(ctx context.Context, caURL string, agentAddress string, agentPort int64, command *agentcomm.KeaCommand, response interface{}) error {
	fa.RecordedURL = caURL
	fa.RecordedCommand = command.Command
	// Generate response.
	fa.mockFunc(response)
	return nil
}

