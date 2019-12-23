package agentcomm

import (
	"testing"
	"github.com/stretchr/testify/require"
)

// Test that empty map of services can be created.
func TestNewKeaServicesEmpty(t *testing.T) {
	services, err := NewKeaServices()
	require.NoError(t, err)
	require.NotNil(t, services)
	require.Equal(t, 0, len(*services))
}

// Test that multiple unique services can be specified.
func TestNewKeaServicesMultiple(t *testing.T) {
	services, err := NewKeaServices("dhcp4", "dhcp6", "dhcp-ddns")
	require.NoError(t, err)
	require.NotNil(t, services)
	require.Equal(t, 3, len(*services))
	require.True(t, services.Contains("dhcp4"))
	require.True(t, services.Contains("dhcp6"))
	require.True(t, services.Contains("dhcp-ddns"))
	require.False(t, services.Contains("ctrl-agent"))
}

// Test that duplicated services are rejected.
func TestNewKeaServicesDuplicate(t *testing.T) {
	services, err := NewKeaServices("dhcp4", "dhcp6", "dhcp4")
	require.Error(t, err)
	require.Nil(t, services)
}

// Test that service name must be non-empty.
func TestNewKeaServicesEmptyName(t *testing.T) {
	services, err := NewKeaServices("dhcp4", "", "dhcp6")
	require.Error(t, err)
	require.Nil(t, services)
}

// Test successful creation of the Kea command with services and arguments.
func TestNewKeaCommand(t *testing.T) {
	services, err := NewKeaServices("dhcp4", "dhcp6")
	require.NotNil(t, services)
	require.NoError(t, err)

	cmd, err := NewKeaCommand("values-set", services,
		&map[string]interface{}{"value-a": 1, "value-b": 2, "value-c": []int{1, 2, 3}})

	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.NotNil(t, cmd.Services)
	require.NotNil(t, cmd.Arguments)

	require.Equal(t, "values-set", cmd.Command)
	require.Equal(t, 2, len(*cmd.Services))
	require.True(t, cmd.Services.Contains("dhcp4"))
	require.True(t, cmd.Services.Contains("dhcp6"))
	require.Contains(t, *cmd.Arguments, "value-a")
	require.Contains(t, *cmd.Arguments, "value-b")
	require.Contains(t, *cmd.Arguments, "value-c")
	require.NotContains(t, *cmd.Arguments, "value-d")
}

// Test that command name must be non-empty.
func TestNewKeaCommandEmptyName(t *testing.T) {
	services, err := NewKeaServices("dhcp4")
	require.NoError(t, err)
	require.NotNil(t, services)

	cmd, err := NewKeaCommand("", services, &map[string]interface{}{"value-a": 1})
	require.Error(t, err)
	require.Nil(t, cmd)
}

// Test that JSON representation of the command is created correctly when
// both service and arguments are present.
func TestKeaCommandMarshal(t *testing.T) {
	services, err := NewKeaServices("dhcp4")
	require.NotNil(t, services)
	require.NoError(t, err)

	cmd, err := NewKeaCommand("values-set", services,
		&map[string]interface{}{"value-a": 1, "value-b": 2, "value-c": []int{1, 2, 3}})
	require.NoError(t, err)
	require.NotNil(t, cmd)

	marshaled := cmd.Marshal()
	require.JSONEq(t,
		`{
             "command":"values-set",
             "service":["dhcp4"],
             "arguments": {
                 "value-a":1,
                 "value-b":2,
                 "value-c": [1,2,3]
             }
         }`,
    marshaled)
}

// Test that empty service list is generated when service list is empty.
func TestKeaCommandMarshalEmptyServicesArguments(t *testing.T) {
	services, err := NewKeaServices()
	require.NotNil(t, services)
	require.NoError(t, err)

	cmd, err := NewKeaCommand("values-set", services, &map[string]interface{}{})
	require.NoError(t, err)
	require.NotNil(t, cmd)

	marshaled := cmd.Marshal()
	require.JSONEq(t,
		`{
             "command":"values-set",
             "service": [ ],
             "arguments": { }
         }`,
    marshaled)
}

// Test that it is possible to send a command without arguments and without
// service list.
func TestKeaCommandMarshalCommandOnly(t *testing.T) {
	services, err := NewKeaServices()
	require.NotNil(t, services)
	require.NoError(t, err)

	cmd, err := NewKeaCommand("list-commands", nil, nil)
	require.NoError(t, err)
	require.NotNil(t, cmd)

	marshaled := cmd.Marshal()
	require.JSONEq(t,
		`{
             "command":"list-commands"
         }`,
    marshaled)
}
