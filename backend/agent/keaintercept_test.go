package agent

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	keactrl "isc.org/stork/daemonctrl/kea"
	"isc.org/stork/datamodel/daemonname"
)

// Test that new instance of the Kea interceptor is created successfully.
func TestNewKeaInterceptor(t *testing.T) {
	interceptor := newKeaInterceptor()
	require.NotNil(t, interceptor)
	require.NotNil(t, interceptor.asyncTargets)
	require.NotNil(t, interceptor.syncTargets)
	require.Empty(t, interceptor.asyncTargets)
}

// Test that it is possible to register async callbacks to intercept selected
// commands and that these callbacks are invoked when the commands are
// received.
func TestKeaInterceptorAsyncHandle(t *testing.T) {
	interceptor := newKeaInterceptor()
	require.NotNil(t, interceptor)

	// Record which command was invoked.
	var commandInvoked string
	// Record responses.
	var capturedResponses []*keactrl.Response

	// Register callback to be invoked for config-get commands.
	interceptor.registerAsync(func(agent *StorkAgent, resp *keactrl.Response) error {
		commandInvoked = "config-get"
		capturedResponses = append(capturedResponses, resp)
		return nil
	}, "config-get")

	// Register callback to be invoked for the subnet4-get.
	interceptor.registerAsync(func(agent *StorkAgent, resp *keactrl.Response) error {
		commandInvoked = "subnet4-get"
		capturedResponses = append(capturedResponses, resp)
		return nil
	}, "subnet4-get")

	// Simulate sending config-get command to the DHCPv4 and DHCPv6
	// server.
	command := *keactrl.NewCommandBase(keactrl.ConfigGet, daemonname.DHCPv4)
	response := keactrl.Response{
		ResponseHeader: keactrl.ResponseHeader{
			Result: 0,
			Text:   "invoked successfully",
		},
	}

	// Invoke the registered callbacks for config-get.
	interceptor.asyncHandle(nil, command, response)
	require.Equal(t, "config-get", commandInvoked)

	command = *keactrl.NewCommandBase(keactrl.ConfigGet, daemonname.DHCPv6)
	response = keactrl.Response{
		ResponseHeader: keactrl.ResponseHeader{
			Result: 1,
			Text:   "invoked unsuccessfully",
		},
	}

	// Invoke the registered callbacks for config-get.
	interceptor.asyncHandle(nil, command, response)
	require.Equal(t, "config-get", commandInvoked)

	// There should be two responses recorded, one for the DHCPv4 and
	// one for DHCPv6.
	require.Len(t, capturedResponses, 2)
	// Check that the callback received the response correctly.
	require.Zero(t, capturedResponses[0].Result)
	require.Equal(t, "invoked successfully", capturedResponses[0].Text)
	require.EqualValues(t, 1, capturedResponses[1].Result)
	require.Equal(t, "invoked unsuccessfully", capturedResponses[1].Text)

	// Make sure that we can invoke different callback when using different
	// command.
	command = *keactrl.NewCommandBase(keactrl.Subnet4Get, daemonname.DHCPv4)
	interceptor.asyncHandle(nil, command, response)
	require.Equal(t, "subnet4-get", commandInvoked)
}

// Test that async intercepting commands sent to Kea Control Agent works.
func TestKeaInterceptorAsyncHandleControlAgent(t *testing.T) {
	interceptor := newKeaInterceptor()
	require.NotNil(t, interceptor)

	var capturedResponses []*keactrl.Response
	interceptor.registerAsync(func(agent *StorkAgent, resp *keactrl.Response) error {
		capturedResponses = append(capturedResponses, resp)
		return nil
	}, "config-get")

	// Simulate sending command to the Control Agent.
	command := *keactrl.NewCommandBase(keactrl.ConfigGet, daemonname.DHCPv4)
	response := keactrl.Response{
		ResponseHeader: keactrl.ResponseHeader{
			Result: 1,
			Text:   "invocation error",
		},
	}
	// Invoke the callbacks and validate the data recorded by this
	// callback.
	interceptor.asyncHandle(nil, command, response)
	require.Len(t, capturedResponses, 1)
	require.EqualValues(t, 1, capturedResponses[0].Result)
	require.Equal(t, "invocation error", capturedResponses[0].Text)
}

// Test that it is possible to register multiple async handlers for a single
// command.
func TestKeaInterceptorMultipleAsyncHandlers(t *testing.T) {
	interceptor := newKeaInterceptor()
	require.NotNil(t, interceptor)

	// Register first handler
	func1Invoked := false
	interceptor.registerAsync(func(agent *StorkAgent, resp *keactrl.Response) error {
		func1Invoked = true
		return nil
	}, "config-get")

	// Register second handler.
	func2Invoked := false
	interceptor.registerAsync(func(agent *StorkAgent, resp *keactrl.Response) error {
		func2Invoked = true
		return nil
	}, "config-get")

	// Send the command matching the handlers.
	command := *keactrl.NewCommandBase(keactrl.ConfigGet, daemonname.DHCPv4)
	response := keactrl.Response{
		ResponseHeader: keactrl.ResponseHeader{
			Result: 0,
			Text:   "fine",
		},
	}

	// Make sure that both handlers have been invoked.
	interceptor.asyncHandle(nil, command, response)
	require.True(t, func1Invoked)
	require.True(t, func2Invoked)
}

// Test that it is possible to register sync callbacks to intercept selected
// commands.
func TestKeaInterceptorSyncHandleRegister(t *testing.T) {
	// Arrange
	interceptor := newKeaInterceptor()
	callback := func(agent *StorkAgent, resp *keactrl.Response) error {
		return nil
	}

	// Act
	interceptor.registerSync(callback, "foobar")

	// Assert
	require.Len(t, interceptor.syncTargets["foobar"].handlers, 1)
}

// Test that the registered sync callbacks are invoked when the commands are
// received.
func TestKeaInterceptorSyncHandleExecute(t *testing.T) {
	// Arrange
	interceptor := newKeaInterceptor()
	callCount := 0
	interceptor.registerSync(func(sa *StorkAgent, r *keactrl.Response) error {
		callCount++
		return nil
	}, "foobar")

	command := *keactrl.NewCommandBase(keactrl.CommandName("foobar"), daemonname.DHCPv4)
	response := keactrl.Response{
		ResponseHeader: keactrl.ResponseHeader{
			Result: 0,
			Text:   "fine",
		},
	}

	// Act
	outResponse, err := interceptor.syncHandle(nil, command, response)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, response, outResponse)
	require.EqualValues(t, 1, callCount)
}

// Test that the multiple registered sync callbacks are invoked sequentially
// when the commands are received.
func TestKeaInterceptorMultipleSyncHandlesExecute(t *testing.T) {
	// Arrange
	interceptor := newKeaInterceptor()
	callCount := map[string]int64{
		"foo": 0,
		"bar": 0,
	}
	interceptor.registerSync(func(sa *StorkAgent, r *keactrl.Response) error {
		callCount["foo"]++
		return nil
	}, "foobar")
	interceptor.registerSync(func(sa *StorkAgent, r *keactrl.Response) error {
		callCount["bar"]++
		return nil
	}, "foobar")

	command := *keactrl.NewCommandBase(keactrl.CommandName("foobar"), daemonname.DHCPv4)
	response := keactrl.Response{
		ResponseHeader: keactrl.ResponseHeader{
			Result: 0,
			Text:   "fine",
		},
	}

	// Act
	outResponse, err := interceptor.syncHandle(nil, command, response)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, response, outResponse)
	require.EqualValues(t, 1, callCount["foo"])
	require.EqualValues(t, 1, callCount["bar"])
}

// Test that the sync callback can rewrite the response.
func TestKeaInterceptorSyncHandleRewriteResponse(t *testing.T) {
	// Arrange
	interceptor := newKeaInterceptor()
	interceptor.registerSync(func(sa *StorkAgent, r *keactrl.Response) error {
		r.Text = "barfoo"
		r.Result = 42
		return nil
	}, "foobar")

	command := *keactrl.NewCommandBase(keactrl.CommandName("foobar"), daemonname.DHCPv4)
	inResponse := keactrl.Response{
		ResponseHeader: keactrl.ResponseHeader{
			Result: 0,
			Text:   "fine",
		},
	}

	expectedOutResponse := keactrl.Response{
		ResponseHeader: keactrl.ResponseHeader{
			Result: 42,
			Text:   "barfoo",
		},
	}

	// Act
	outResponse, _ := interceptor.syncHandle(nil, command, inResponse)

	// Assert
	require.EqualValues(t, expectedOutResponse, outResponse)
}

// Test that the error throwing in the sync handler breaks execution and
// returns.
func TestKeaInterceptorSyncHandleReturnError(t *testing.T) {
	// Arrange
	interceptor := newKeaInterceptor()
	interceptor.registerSync(func(sa *StorkAgent, r *keactrl.Response) error {
		return errors.New("Expected error")
	}, "foobar")
	callCount := 0
	interceptor.registerSync(func(sa *StorkAgent, r *keactrl.Response) error {
		callCount++
		return nil
	}, "foobar")

	command := *keactrl.NewCommandBase(keactrl.CommandName("foobar"), daemonname.DHCPv4)
	inResponse := keactrl.Response{
		ResponseHeader: keactrl.ResponseHeader{
			Result: 0,
			Text:   "fine",
		},
	}

	// Act
	outResponse, err := interceptor.syncHandle(nil, command, inResponse)

	// Assert
	require.Empty(t, outResponse)
	require.Error(t, err)
	require.Zero(t, callCount)
}
