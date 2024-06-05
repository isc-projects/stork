package restservice

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test instantiating EndpointControl.
func TestNewEndpointControl(t *testing.T) {
	control := NewEndpointControl()
	require.False(t, control.IsDisabled(EndpointOpCreateNewMachine))
}

// Test setting the endpoint state to disabled and enabled.
func TestEndpointControlSetEnabled(t *testing.T) {
	control := NewEndpointControl()
	require.False(t, control.IsDisabled(EndpointOpCreateNewMachine))

	control.SetEnabled(EndpointOpCreateNewMachine, true)
	require.False(t, control.IsDisabled(EndpointOpCreateNewMachine))

	control.SetEnabled(EndpointOpCreateNewMachine, false)
	require.True(t, control.IsDisabled(EndpointOpCreateNewMachine))
}
