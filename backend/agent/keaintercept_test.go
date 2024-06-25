package agent

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	agentapi "isc.org/stork/api"
	keactrl "isc.org/stork/appctrl/kea"
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
	command := keactrl.NewCommandBase(keactrl.ConfigGet, keactrl.DHCPv4, keactrl.DHCPv6)
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
	command = keactrl.NewCommandBase(keactrl.Subnet4Get, keactrl.DHCPv4)
	request = &agentapi.KeaRequest{
		Request: command.Marshal(),
	}
	interceptor.asyncHandle(nil, request, response)
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
	command := keactrl.NewCommandBase(keactrl.ConfigGet)
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
	command := keactrl.NewCommandBase(keactrl.ConfigGet)
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

	command := keactrl.NewCommandBase(keactrl.CommandName("foobar"), keactrl.DHCPv4)
	request := &agentapi.KeaRequest{
		Request: command.Marshal(),
	}
	inResponse := []byte(`[
		{
			"result": 0,
			"text": "fine"
		}
	]`)
	var buffer bytes.Buffer
	_ = json.Compact(&buffer, inResponse)
	expectedOutResponse := buffer.Bytes()

	// Act
	outResponse, err := interceptor.syncHandle(nil, request, inResponse)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, expectedOutResponse, outResponse)
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

	command := keactrl.NewCommandBase(keactrl.CommandName("foobar"), keactrl.DHCPv4)
	request := &agentapi.KeaRequest{
		Request: command.Marshal(),
	}
	inResponse := []byte(`[
		{
			"result": 0,
			"text": "fine"
		}
	]`)
	var buffer bytes.Buffer
	_ = json.Compact(&buffer, inResponse)
	expectedOutResponse := buffer.Bytes()

	// Act
	outResponse, err := interceptor.syncHandle(nil, request, inResponse)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, expectedOutResponse, outResponse)
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

	command := keactrl.NewCommandBase(keactrl.CommandName("foobar"), keactrl.DHCPv4)
	request := &agentapi.KeaRequest{
		Request: command.Marshal(),
	}

	inResponse := []byte(`[
		{
			"result": 0,
			"text": "fine"
		}
	]`)

	expectedOutResponse := []byte(`[
		{
			"result": 42,
			"text": "barfoo"
		}
	]`)
	var buffer bytes.Buffer
	_ = json.Compact(&buffer, expectedOutResponse)
	expectedOutResponse = buffer.Bytes()

	// Act
	outResponse, _ := interceptor.syncHandle(nil, request, inResponse)

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

	command := keactrl.NewCommandBase(keactrl.CommandName("foobar"), keactrl.DHCPv4)
	request := &agentapi.KeaRequest{
		Request: command.Marshal(),
	}

	inResponse := []byte(`[
		{
			"result": 0,
			"text": "fine"
		}
	]`)

	// Act
	outResponse, err := interceptor.syncHandle(nil, request, inResponse)

	// Assert
	require.Nil(t, outResponse)
	require.Error(t, err)
	require.Zero(t, callCount)
}
