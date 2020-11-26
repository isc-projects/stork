package keactrl

import (
	"encoding/json"
	"testing"

	require "github.com/stretchr/testify/require"
)

// Test that empty map of daemons can be created.
func TestNewKeaDaemonsEmpty(t *testing.T) {
	daemons, err := NewDaemons()
	require.NoError(t, err)
	require.NotNil(t, daemons)
	require.Len(t, *daemons, 0)
}

// Test that multiple unique daemons can be specified.
func TestNewDaemonsMultiple(t *testing.T) {
	daemons, err := NewDaemons("dhcp4", "dhcp6", "dhcp-ddns")
	require.NoError(t, err)
	require.NotNil(t, daemons)
	require.Len(t, *daemons, 3)
	require.True(t, daemons.Contains("dhcp4"))
	require.True(t, daemons.Contains("dhcp6"))
	require.True(t, daemons.Contains("dhcp-ddns"))
	require.False(t, daemons.Contains("ctrl-agent"))
}

// Test that duplicated daemons are rejected.
func TestNewDaemonsDuplicate(t *testing.T) {
	daemons, err := NewDaemons("dhcp4", "dhcp6", "dhcp4")
	require.Error(t, err)
	require.Nil(t, daemons)
}

// Test that daemon name must be non-empty.
func TestNewDaemonsEmptyName(t *testing.T) {
	daemons, err := NewDaemons("dhcp4", "", "dhcp6")
	require.Error(t, err)
	require.Nil(t, daemons)
}

// Test successful creation of the Kea command with daemons and arguments.
func TestNewCommand(t *testing.T) {
	daemons, err := NewDaemons("dhcp4", "dhcp6")
	require.NotNil(t, daemons)
	require.NoError(t, err)

	cmd, err := NewCommand("values-set", daemons,
		&map[string]interface{}{"value-a": 1, "value-b": 2, "value-c": []int{1, 2, 3}})

	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.NotNil(t, cmd.Daemons)
	require.NotNil(t, cmd.Arguments)

	require.Equal(t, "values-set", cmd.Command)
	require.Len(t, *cmd.Daemons, 2)
	require.True(t, cmd.Daemons.Contains("dhcp4"))
	require.True(t, cmd.Daemons.Contains("dhcp6"))
	require.Contains(t, *cmd.Arguments, "value-a")
	require.Contains(t, *cmd.Arguments, "value-b")
	require.Contains(t, *cmd.Arguments, "value-c")
	require.NotContains(t, *cmd.Arguments, "value-d")
}

// Test that command name must be non-empty.
func TestNewCommandEmptyName(t *testing.T) {
	daemons, err := NewDaemons("dhcp4")
	require.NoError(t, err)
	require.NotNil(t, daemons)

	cmd, err := NewCommand("", daemons, &map[string]interface{}{"value-a": 1})
	require.Error(t, err)
	require.Nil(t, cmd)
}

func TestNewCommandFromJSON(t *testing.T) {
	jsonCommand := `{
        "command": "subnet4-get",
        "service": [ "dhcp4", "dhcp6" ],
        "arguments": {
            "subnet-id": 10
        }
    }`
	command, err := NewCommandFromJSON(jsonCommand)
	require.NoError(t, err)
	require.Equal(t, "subnet4-get", command.Command)
	require.NotNil(t, command.Arguments)
	require.Contains(t, *command.Arguments, "subnet-id")
	require.EqualValues(t, 10, (*command.Arguments)["subnet-id"])
	require.NotNil(t, command.Daemons)
	require.Contains(t, *command.Daemons, "dhcp4")
	require.Contains(t, *command.Daemons, "dhcp6")
}

func TestNewCommandFromJSONNoService(t *testing.T) {
	jsonCommand := `{
        "command": "subnet4-get",
        "arguments": {
            "subnet-id": 11
        }
    }`
	command, err := NewCommandFromJSON(jsonCommand)
	require.NoError(t, err)
	require.Equal(t, "subnet4-get", command.Command)
	require.NotNil(t, command.Arguments)
	require.Contains(t, *command.Arguments, "subnet-id")
	require.EqualValues(t, 11, (*command.Arguments)["subnet-id"])
	require.Nil(t, command.Daemons)
}

// Test that JSON representation of the command is created correctly when
// both daemon name (service in Kea terms) and arguments are present.
func TestKeaCommandMarshal(t *testing.T) {
	daemons, err := NewDaemons("dhcp4")
	require.NotNil(t, daemons)
	require.NoError(t, err)

	cmd, err := NewCommand("values-set", daemons,
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
	daemons, err := NewDaemons()
	require.NotNil(t, daemons)
	require.NoError(t, err)

	cmd, err := NewCommand("values-set", daemons, &map[string]interface{}{})
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
	daemons, err := NewDaemons()
	require.NotNil(t, daemons)
	require.NoError(t, err)

	cmd, err := NewCommand("list-commands", nil, nil)
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
func TestUnmarshalResponseList(t *testing.T) {
	daemons, _ := NewDaemons("dhcp4", "dhcp6")
	request, _ := NewCommand("list-subnets", daemons, nil)

	response := []byte(`[
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
    ]`)

	list := ResponseList{}
	err := UnmarshalResponseList(request, response, &list)
	require.NoError(t, err)
	require.NotNil(t, list)

	// There should be two responses encapsulated.
	require.Len(t, list, 2)

	// The first result value is 0.
	require.Equal(t, 0, list[0].Result)
	require.Equal(t, "command successful", list[0].Text)

	// The arguments should be non-nil and contain two parameters.
	require.NotNil(t, list[0].Arguments)
	require.Contains(t, *(list[0]).Arguments, "subnet-id")
	require.Contains(t, *(list[0]).Arguments, "prefix")

	// The daemon should be set based on the command instance provided.
	require.Equal(t, "dhcp4", (list[0]).Daemon)

	// Validate the arguments.
	require.EqualValues(t, map[string]interface{}{"subnet-id": float64(1), "prefix": "192.0.2.0/24"},
		*(list[0]).Arguments)

	// The second response should contain different result and text. The
	// arguments are not present, so should be nil.
	require.Equal(t, 1, list[1].Result)
	require.Equal(t, "command unsuccessful", list[1].Text)
	require.Nil(t, list[1].Arguments)
	require.Equal(t, "dhcp6", (list[1]).Daemon)
}

// Test that well formed list of Kea responses can be parsed and that hashes
// are computed from the arguments.
func TestUnmarshalHashedResponseList(t *testing.T) {
	daemons, _ := NewDaemons("dhcp4", "dhcp6")
	request, _ := NewCommand("list-subnets", daemons, nil)

	response := []byte(`[
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
    ]`)

	list := HashedResponseList{}
	err := UnmarshalResponseList(request, response, &list)
	require.NoError(t, err)
	require.NotNil(t, list)

	// There should be two responses encapsulated.
	require.Len(t, list, 2)

	// The first result value is 0.
	require.Equal(t, 0, list[0].Result)
	require.Equal(t, "command successful", list[0].Text)

	// The arguments should be non-nil and contain two parameters.
	require.NotNil(t, list[0].Arguments)
	require.Contains(t, *(list[0]).Arguments, "subnet-id")
	require.Contains(t, *(list[0]).Arguments, "prefix")

	// The daemon should be set based on the command instance provided.
	require.Equal(t, "dhcp4", (list[0]).Daemon)

	// Validate the arguments.
	require.EqualValues(t, map[string]interface{}{"subnet-id": float64(1), "prefix": "192.0.2.0/24"},
		*(list[0]).Arguments)

	// There should be a hash computed from the arguments.
	require.Equal(t, "198f893a3764258b18bef43f33a33f62", list[0].ArgumentsHash)

	// The second response should contain different result and text. The
	// arguments are not present, so should be nil.
	require.Equal(t, 1, list[1].Result)
	require.Equal(t, "command unsuccessful", list[1].Text)
	require.Nil(t, list[1].Arguments)
	require.Equal(t, "dhcp6", (list[1]).Daemon)
	require.Empty(t, list[1].ArgumentsHash)
}

// Test that it is possible to parse Kea response to a custom structure.
func TestUnmarshalCustomResponse(t *testing.T) {
	daemons, _ := NewDaemons("dhcp4")
	request, _ := NewCommand("list-subnets", daemons, nil)

	response := []byte(`[
        {
            "result": 0,
            "text": "command successful",
            "arguments": {
                "subnet": {
                    "subnet-id": 1,
                    "prefix": "192.0.2.0/24"
                }
            }
        }
    ]`)

	type CustomResponse struct {
		ResponseHeader
		Arguments struct {
			Subnet struct {
				SubnetID float64 `json:"subnet-id"`
				Prefix   string
			}
		}
	}

	list := []CustomResponse{}
	err := UnmarshalResponseList(request, response, &list)
	require.NoError(t, err)
	require.NotNil(t, list)

	require.Len(t, list, 1)
	require.Equal(t, 0, list[0].Result)
	require.Equal(t, "command successful", list[0].Text)
	require.EqualValues(t, 1, list[0].Arguments.Subnet.SubnetID)
	require.Equal(t, "192.0.2.0/24", list[0].Arguments.Subnet.Prefix)
}

// Test that custom response without arguments is parsed correctly.
func TestUnmarshalCustomResponseNoArgs(t *testing.T) {
	daemons, _ := NewDaemons("dhcp4")
	request, _ := NewCommand("list-subnets", daemons, nil)

	response := []byte(`[
        {
            "result": 0,
            "text": "command successful",
            "arguments": {
                "param": "value"
            }
        }
    ]`)

	type CustomResponse struct {
		ResponseHeader
	}

	list := []CustomResponse{}
	err := UnmarshalResponseList(request, response, &list)
	require.NoError(t, err)
	require.NotNil(t, list)

	require.Len(t, list, 1)
	require.Equal(t, 0, list[0].Result)
	require.Equal(t, "command successful", list[0].Text)
}

// Test that the Kea response containing invalid result value is rejected.
func TestUnmarshalResponseListMalformedResult(t *testing.T) {
	daemons, _ := NewDaemons("dhcp4")
	request, _ := NewCommand("list-commands", daemons, nil)

	response := []byte(`[
        {
            "result": "1"
        }
    ]`)
	list := ResponseList{}
	err := UnmarshalResponseList(request, response, &list)
	require.Error(t, err)
}

// Test that the Kea response containing invalid text value is rejected.
func TestUnmarshalResponseListMalformedText(t *testing.T) {
	daemons, _ := NewDaemons("dhcp4")
	request, _ := NewCommand("list-commands", daemons, nil)

	response := []byte(`[
        {
            "result": 1,
            "text": 123
        }
    ]`)
	list := ResponseList{}
	err := UnmarshalResponseList(request, response, &list)
	require.Error(t, err)
}

// Test that the Kea response containing invalid arguments (being a list
// rather than a map) is rejected.
func TestUnmarshalResponseListMalformedArguments(t *testing.T) {
	daemons, _ := NewDaemons("dhcp4")
	request, _ := NewCommand("list-commands", daemons, nil)

	response := []byte(`[
        {
            "result": 0,
            "arguments": [ 1, 2, 3 ]
        }
    ]`)
	list := ResponseList{}
	err := UnmarshalResponseList(request, response, &list)
	require.Error(t, err)
}

// Test that the Kea response not being a list is rejected.
func TestUnmarshalResponseNotList(t *testing.T) {
	daemons, _ := NewDaemons("dhcp4")
	request, _ := NewCommand("list-commands", daemons, nil)

	response := []byte(`
        {
            "result": 0
        }
    `)
	list := ResponseList{}
	err := UnmarshalResponseList(request, response, &list)
	require.Error(t, err)
}

// Runs two benchmarks checking performance of Kea response unmarshalling
// with and without hashing the response arguments.
func BenchmarkUnmarshalHashedResponseList(b *testing.B) {
	daemons, _ := NewDaemons("dhcp4")
	request, _ := NewCommand("list-subnets", daemons, nil)

	// Create a large response with 10000 subnet items.
	argumentsMap := map[string]interface{}{
		"subnet4": []map[string]interface{}{},
	}
	for i := 0; i < 10000; i++ {
		argumentsMap["subnet4"] = append(argumentsMap["subnet4"].([]map[string]interface{}),
			map[string]interface{}{
				"id": i,
			})
	}
	// Create the actual response.
	responseMap := []map[string]interface{}{
		{
			"result": 0,
			"text": "success",
			"arguments": argumentsMap,
		},
	}

	// Output the response into json format.
	response, err := json.Marshal(responseMap)
	if err != nil {
		b.Fatalf("unable to marshal responseMap: %+v", err)
	}

	// Run the benchmark with hashing. This should be slower the other case without
	// hashing (more or less 2x).
	b.Run("HashConfig", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			list := HashedResponseList{}
			UnmarshalResponseList(request, response, &list)
		}
	})

	// Run it without hashing. This should be 2x faster.
	b.Run("NoHashConfig",  func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			list := ResponseList{}
			err = UnmarshalResponseList(request, response, &list)
		}
	})
}
