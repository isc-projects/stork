package agentcomm

import (
	"github.com/stretchr/testify/require"
	"testing"
)

// Test that empty map of daemons can be created.
func TestNewKeaDaemonsEmpty(t *testing.T) {
	daemons, err := NewKeaDaemons()
	require.NoError(t, err)
	require.NotNil(t, daemons)
	require.Equal(t, 0, len(*daemons))
}

// Test that multiple unique daemons can be specified.
func TestNewKeaDaemonsMultiple(t *testing.T) {
	daemons, err := NewKeaDaemons("dhcp4", "dhcp6", "dhcp-ddns")
	require.NoError(t, err)
	require.NotNil(t, daemons)
	require.Equal(t, 3, len(*daemons))
	require.True(t, daemons.Contains("dhcp4"))
	require.True(t, daemons.Contains("dhcp6"))
	require.True(t, daemons.Contains("dhcp-ddns"))
	require.False(t, daemons.Contains("ctrl-agent"))
}

// Test that duplicated daemons are rejected.
func TestNewKeaDaemonsDuplicate(t *testing.T) {
	daemons, err := NewKeaDaemons("dhcp4", "dhcp6", "dhcp4")
	require.Error(t, err)
	require.Nil(t, daemons)
}

// Test that daemon name must be non-empty.
func TestNewKeaDaemonsEmptyName(t *testing.T) {
	daemons, err := NewKeaDaemons("dhcp4", "", "dhcp6")
	require.Error(t, err)
	require.Nil(t, daemons)
}

// Test successful creation of the Kea command with daemons and arguments.
func TestNewKeaCommand(t *testing.T) {
	daemons, err := NewKeaDaemons("dhcp4", "dhcp6")
	require.NotNil(t, daemons)
	require.NoError(t, err)

	cmd, err := NewKeaCommand("values-set", daemons,
		&map[string]interface{}{"value-a": 1, "value-b": 2, "value-c": []int{1, 2, 3}})

	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.NotNil(t, cmd.Daemons)
	require.NotNil(t, cmd.Arguments)

	require.Equal(t, "values-set", cmd.Command)
	require.Equal(t, 2, len(*cmd.Daemons))
	require.True(t, cmd.Daemons.Contains("dhcp4"))
	require.True(t, cmd.Daemons.Contains("dhcp6"))
	require.Contains(t, *cmd.Arguments, "value-a")
	require.Contains(t, *cmd.Arguments, "value-b")
	require.Contains(t, *cmd.Arguments, "value-c")
	require.NotContains(t, *cmd.Arguments, "value-d")
}

// Test that command name must be non-empty.
func TestNewKeaCommandEmptyName(t *testing.T) {
	daemons, err := NewKeaDaemons("dhcp4")
	require.NoError(t, err)
	require.NotNil(t, daemons)

	cmd, err := NewKeaCommand("", daemons, &map[string]interface{}{"value-a": 1})
	require.Error(t, err)
	require.Nil(t, cmd)
}

// Test that JSON representation of the command is created correctly when
// both daemon name (service in Kea terms) and arguments are present.
func TestKeaCommandMarshal(t *testing.T) {
	daemons, err := NewKeaDaemons("dhcp4")
	require.NotNil(t, daemons)
	require.NoError(t, err)

	cmd, err := NewKeaCommand("values-set", daemons,
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

// Test that empty service list is generated when daemons list is empty.
func TestKeaCommandMarshalEmptyDaemonsArguments(t *testing.T) {
	daemons, err := NewKeaDaemons()
	require.NotNil(t, daemons)
	require.NoError(t, err)

	cmd, err := NewKeaCommand("values-set", daemons, &map[string]interface{}{})
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
// daemons list.
func TestKeaCommandMarshalCommandOnly(t *testing.T) {
	daemons, err := NewKeaDaemons()
	require.NotNil(t, daemons)
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

// Test that well formed list of Kea responses can be parsed.
func TestUnmarshalKeaResponseList(t *testing.T) {
	daemons, _ := NewKeaDaemons("dhcp4", "dhcp6")
	request, _ := NewKeaCommand("list-commands", daemons, nil)

	response := `[
        {
            "result": 0,
            "text": "command successful",
            "arguments": {
                "subnet-id": 1,
                "prefix": "192.0.2.0/24"
            }
        },
        {
            "result": 1,
            "text": "command unsuccessful"
        }
    ]`

	list, err := UnmarshalKeaResponseList(request, response)
	require.NoError(t, err)
	require.NotNil(t, list)

	// There should be two responses encapsulated.
	require.Equal(t, 2, len(*list))

	// The first result value is 0.
	require.Equal(t, 0, (*list)[0].Result)
	require.Equal(t, "command successful", (*list)[0].Text)

	// The arguments should be non-nil and contain two parameters.
	require.NotNil(t, (*list)[0].Arguments)
	require.Contains(t, *((*list)[0]).Arguments, "subnet-id")
	require.Contains(t, *((*list)[0]).Arguments, "prefix")

	// The daemon should be set based on the command instance provided.
	require.Equal(t, "dhcp4", ((*list)[0]).Daemon)

	// Validate the arguments.
	require.EqualValues(t, map[string]interface{}{"subnet-id": float64(1), "prefix": "192.0.2.0/24"},
		*((*list)[0]).Arguments)

	// The second response should contain different result and text. The
	// arguments are not present, so should be nil.
	require.Equal(t, 1, (*list)[1].Result)
	require.Equal(t, "command unsuccessful", (*list)[1].Text)
	require.Nil(t, (*list)[1].Arguments)
	require.Equal(t, "dhcp6", ((*list)[1]).Daemon)
}

// Test that the Kea response containing invalid result value is rejected.
func TestUnmarshalKeaResponseListMalformedResult(t *testing.T) {
	daemons, _ := NewKeaDaemons("dhcp4")
	request, _ := NewKeaCommand("list-commands", daemons, nil)

	response := `[
        {
            "result": "1"
        }
    ]`
	list, err := UnmarshalKeaResponseList(request, response)
	require.Error(t, err)
	require.Nil(t, list)
}

// Test that the Kea response containing invalid text value is rejected.
func TestUnmarshalKeaResponseListMalformedText(t *testing.T) {
	daemons, _ := NewKeaDaemons("dhcp4")
	request, _ := NewKeaCommand("list-commands", daemons, nil)

	response := `[
        {
            "result": 1,
            "text": 123
        }
    ]`
	list, err := UnmarshalKeaResponseList(request, response)
	require.Error(t, err)
	require.Nil(t, list)
}

// Test that the Kea response containing invalid arguments (being a list
// rather than a map) is rejected.
func TestUnmarshalKeaResponseListMalformedArguments(t *testing.T) {
	daemons, _ := NewKeaDaemons("dhcp4")
	request, _ := NewKeaCommand("list-commands", daemons, nil)

	response := `[
        {
            "result": 0,
            "arguments": [ 1, 2, 3 ]
        }
    ]`
	list, err := UnmarshalKeaResponseList(request, response)
	require.Error(t, err)
	require.Nil(t, list)
}

// Test that the Kea response not being a list is rejected.
func TestUnmarshalKeaResponseNotList(t *testing.T) {
	daemons, _ := NewKeaDaemons("dhcp4")
	request, _ := NewKeaCommand("list-commands", daemons, nil)

	response := `
        {
            "result": 0
        }
    `
	list, err := UnmarshalKeaResponseList(request, response)
	require.Error(t, err)
	require.Nil(t, list)
}
