package agent

import (
	"testing"

	"github.com/stretchr/testify/require"
	agentapi "isc.org/stork/api"
	keactrl "isc.org/stork/appctrl/kea"
)

// Test that new instance of the Kea interceptor is created successfully.
func TestNewKeaInterceptor(t *testing.T) {
	interceptor := newKeaInterceptor()
	require.NotNil(t, interceptor)
	require.NotNil(t, interceptor.asyncTargets)
	require.Empty(t, interceptor.asyncTargets)
}

// Test that it is possible to register callbacks to intercept selected
// commands and that these callbacks are invoked when the commands are
// received.
func TestKeaInterceptorHandle(t *testing.T) {
	interceptor := newKeaInterceptor()
	require.NotNil(t, interceptor)

	// Record which command was invoked.
	var commandInvoked string
	// Record responses.
	var capturedResponses []*keactrl.Response

	// Register callback to be invoked for config-get commands.
	interceptor.register(func(agent *StorkAgent, resp *keactrl.Response) error {
		commandInvoked = "config-get"
		capturedResponses = append(capturedResponses, resp)
		return nil
	}, "config-get")

	// Register callback to be invoked for the subnet4-get.
	interceptor.register(func(agent *StorkAgent, resp *keactrl.Response) error {
		commandInvoked = "subnet4-get"
		capturedResponses = append(capturedResponses, resp)
		return nil
	}, "subnet4-get")

	// Simulate sending config-get command to the DHCPv4 and DHCPv6
	// server.
	command := keactrl.NewCommand("config-get", []string{"dhcp4", "dhcp6"}, nil)
	request := &agentapi.KeaRequest{
		Request: command.Marshal(),
	}
	response := []byte(`[
            {
                "result": 0,
                "text": "invoked successfully"
            },
            {
                "result": 1,
                "text": "invoked unsuccessfully"
            }
        ]`)

	// Invoke the registered callbacks for config-get.
	interceptor.asyncHandle(nil, request, response)
	require.Equal(t, "config-get", commandInvoked)
	// There should be two responses recorded, one for the DHCPv4 and
	// one for DHCPv6.
	require.Len(t, capturedResponses, 2)
	// Check that the callback received the response correctly.
	require.Zero(t, capturedResponses[0].Result)
	require.Equal(t, "invoked successfully", capturedResponses[0].Text)
	require.Equal(t, "dhcp4", capturedResponses[0].Daemon)
	require.EqualValues(t, 1, capturedResponses[1].Result)
	require.Equal(t, "invoked unsuccessfully", capturedResponses[1].Text)
	require.Equal(t, "dhcp6", capturedResponses[1].Daemon)

	// Make sure that we can invoke different callback when using different
	// command.
	command = keactrl.NewCommand("subnet4-get", []string{"dhcp4"}, nil)
	request = &agentapi.KeaRequest{
		Request: command.Marshal(),
	}
	interceptor.asyncHandle(nil, request, response)
	require.Equal(t, "subnet4-get", commandInvoked)
}

// Test that intercepting commands sent to Kea Control Agent works.
func TestKeaInterceptorHandleControlAgent(t *testing.T) {
	interceptor := newKeaInterceptor()
	require.NotNil(t, interceptor)

	var capturedResponses []*keactrl.Response
	interceptor.register(func(agent *StorkAgent, resp *keactrl.Response) error {
		capturedResponses = append(capturedResponses, resp)
		return nil
	}, "config-get")

	// Simulate sending command to the Control Agent.
	command := keactrl.NewCommand("config-get", nil, nil)
	request := &agentapi.KeaRequest{
		Request: command.Marshal(),
	}
	response := []byte(`[
            {
                "result": 1,
                "text": "invocation error"
            }
        ]`)

	// Invoke the callbacks and validate the data recorded by this
	// callback.
	interceptor.asyncHandle(nil, request, response)
	require.Len(t, capturedResponses, 1)
	require.EqualValues(t, 1, capturedResponses[0].Result)
	require.Equal(t, "invocation error", capturedResponses[0].Text)
	require.Empty(t, capturedResponses[0].Daemon)
}

// Test that it is possible to register multiple handlers for a single
// command.
func TestKeaInterceptorMultipleHandlers(t *testing.T) {
	interceptor := newKeaInterceptor()
	require.NotNil(t, interceptor)

	// Register first handler
	func1Invoked := false
	interceptor.register(func(agent *StorkAgent, resp *keactrl.Response) error {
		func1Invoked = true
		return nil
	}, "config-get")

	// Register second handler.
	func2Invoked := false
	interceptor.register(func(agent *StorkAgent, resp *keactrl.Response) error {
		func2Invoked = true
		return nil
	}, "config-get")

	// Send the command matching the handlers.
	command := keactrl.NewCommand("config-get", nil, nil)
	request := &agentapi.KeaRequest{
		Request: command.Marshal(),
	}
	response := []byte(`[
            {
                "result": 0,
                "text": "fine"
            }
        ]`)

	// Make sure that both handlers have been invoked.
	interceptor.asyncHandle(nil, request, response)
	require.True(t, func1Invoked)
	require.True(t, func2Invoked)
}
