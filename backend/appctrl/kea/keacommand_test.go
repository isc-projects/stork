package keactrl

import (
	"encoding/json"
	"testing"

	require "github.com/stretchr/testify/require"
)

// Test successful creation of the Kea command with daemons and arguments.
func TestNewCommand(t *testing.T) {
	cmd := NewCommand("values-set", []string{"dhcp4", "dhcp6"},
		map[string]interface{}{"value-a": 1, "value-b": 2, "value-c": []int{1, 2, 3}})

	require.NotNil(t, cmd)
	require.NotNil(t, cmd.Daemons)
	require.NotNil(t, cmd.Arguments)

	require.Equal(t, "values-set", cmd.Command)
	require.Len(t, cmd.Daemons, 2)
	require.Contains(t, cmd.Daemons, "dhcp4")
	require.Contains(t, cmd.Daemons, "dhcp6")
	require.Contains(t, cmd.Arguments.(map[string]interface{}), "value-a")
	require.Contains(t, cmd.Arguments.(map[string]interface{}), "value-b")
	require.Contains(t, cmd.Arguments.(map[string]interface{}), "value-c")
	require.NotContains(t, cmd.Arguments.(map[string]interface{}), "value-d")
}

// Test successful creation of the Kea command with arguments specified as a structure.
func TestNewCommandWithStructArgs(t *testing.T) {
	type argsType struct {
		ValueA int
		ValueB int
		ValueC []int
	}
	args := argsType{
		ValueA: 2,
		ValueB: 3,
		ValueC: []int{5, 6, 7},
	}
	cmd := NewCommand("values-set", []string{"dhcp4"}, args)
	require.NotNil(t, cmd)
	require.NotNil(t, cmd.Daemons)
	require.NotNil(t, cmd.Arguments)
	require.Equal(t, "values-set", cmd.Command)
	require.Len(t, cmd.Daemons, 1)
	require.Contains(t, cmd.Daemons, "dhcp4")
	require.Equal(t, args, cmd.Arguments)
}

// Test successful creation of the Kea command with arguments specified as a pointer
// to a structure.
func TestNewCommandWithStructPtrArgs(t *testing.T) {
	type argsType struct {
		ValueA int
	}
	args := argsType{
		ValueA: 2,
	}
	cmd := NewCommand("values-set", []string{"dhcp4"}, &args)
	require.NotNil(t, cmd)
	require.NotNil(t, cmd.Daemons)
	require.NotNil(t, cmd.Arguments)
	require.Equal(t, "values-set", cmd.Command)
	require.Len(t, cmd.Daemons, 1)
	require.Contains(t, cmd.Daemons, "dhcp4")
	require.Equal(t, &args, cmd.Arguments)
}

// Test that the command is not created when the arguments have an invalid type.
func TestNewCommandWithInvalidArgTypes(t *testing.T) {
	require.Nil(t, NewCommand("values-set", []string{"dhcp4"}, 123))
	require.Nil(t, NewCommand("values-set", []string{"dhcp4"}, []int{123, 345}))
	m := make(map[string]interface{})
	require.Nil(t, NewCommand("values-set", []string{"dhcp4"}, &m))
}

// Test that command name must be non-empty.
func TestNewCommandEmptyName(t *testing.T) {
	cmd := NewCommand("", []string{"dhcp4"}, map[string]interface{}{"value-a": 1})
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
	require.Contains(t, command.Arguments, "subnet-id")
	require.EqualValues(t, 10, (command.Arguments.(map[string]interface{}))["subnet-id"])
	require.NotNil(t, command.Daemons)
	require.Contains(t, command.Daemons, "dhcp4")
	require.Contains(t, command.Daemons, "dhcp6")
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
	require.Contains(t, command.Arguments, "subnet-id")
	require.EqualValues(t, 11, (command.Arguments.(map[string]interface{}))["subnet-id"])
	require.Nil(t, command.Daemons)
}

// Test that JSON representation of the command is created correctly when
// both daemon name (service in Kea terms) and arguments are present.
func TestKeaCommandMarshal(t *testing.T) {
	cmd := NewCommand("values-set", []string{"dhcp4"},
		map[string]interface{}{"value-a": 1, "value-b": 2, "value-c": []int{1, 2, 3}})
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

// Test that JSON representation of the command is created correctly when
// arguments are specified in a structure.
func TestKeaCommandMarshalWithStructArgs(t *testing.T) {
	type argsType struct {
		ValueA int   `json:"value-a"`
		ValueB int   `json:"value-b"`
		ValueC []int `json:"value-c"`
	}
	args := argsType{
		ValueA: 222,
		ValueB: 333,
		ValueC: []int{123, 234},
	}
	cmd := NewCommand("values-set", []string{"dhcp4"}, &args)
	require.NotNil(t, cmd)

	marshaled := cmd.Marshal()
	require.JSONEq(t,
		`{
             "command":"values-set",
             "service":["dhcp4"],
             "arguments": {
                 "value-a":222,
                 "value-b":333,
                 "value-c": [123,234]
             }
         }`,
		marshaled)
}

// Test that no service list is included when daemons list is empty.
func TestKeaCommandMarshalEmptyDaemonsArguments(t *testing.T) {
	cmd := NewCommand("values-set", []string{}, map[string]interface{}{})
	require.NotNil(t, cmd)

	marshaled := cmd.Marshal()
	require.JSONEq(t,
		`{
             "command":"values-set",
             "arguments": { }
         }`,
		marshaled)
}

// Test that it is possible to send a command without arguments and without
// daemons list.
func TestKeaCommandMarshalCommandOnly(t *testing.T) {
	cmd := NewCommand("list-commands", nil, nil)
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
	request := NewCommand("list-subnets", []string{"dhcp4", "dhcp6"}, nil)

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
	request := NewCommand("list-subnets", []string{"dhcp4", "dhcp6"}, nil)

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
	require.Len(t, list[0].ArgumentsHash, 32)

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
	request := NewCommand("list-subnets", []string{"dhcp4"}, nil)

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
	request := NewCommand("list-subnets", []string{"dhcp4"}, nil)

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
	request := NewCommand("list-commands", []string{"dhcp4"}, nil)

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
	request := NewCommand("list-commands", []string{"dhcp4"}, nil)

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
	request := NewCommand("list-commands", []string{"dhcp4"}, nil)

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
	request := NewCommand("list-commands", []string{"dhcp4"}, nil)

	response := []byte(`
        {
            "result": 0
        }
    `)
	list := ResponseList{}
	err := UnmarshalResponseList(request, response, &list)
	require.Error(t, err)
}

// Test that the Kea response is serialized properly.
func TestMarshalStandardResponseList(t *testing.T) {
	// Arrange
	responses := ResponseList{
		{
			ResponseHeader: ResponseHeader{
				Result: 42,
				Text:   "foo",
				Daemon: "bar",
			},
			Arguments: &map[string]interface{}{
				"baz": 24,
			},
		},
	}

	// Act
	serialized, err := MarshalResponseList(responses)

	// Assert
	require.NoError(t, err)

	var data any
	_ = json.Unmarshal(serialized, &data)

	dataList := data.([]any)
	require.Len(t, dataList, 1)

	dataItem := dataList[0].(map[string]any)
	require.EqualValues(t, 42, dataItem["result"])
	require.EqualValues(t, "foo", dataItem["text"])
	require.NotContains(t, "daemon", dataItem)

	dataArguments := dataItem["arguments"].(map[string]any)
	require.EqualValues(t, 24, dataArguments["baz"])
}

// Test that the hashed Kea response is serialized properly.
func TestMarshalHashedResponseList(t *testing.T) {
	// Arrange
	responses := HashedResponseList{
		{
			ResponseHeader: ResponseHeader{
				Result: 42,
				Text:   "foo",
				Daemon: "bar",
			},
			Arguments: &map[string]interface{}{
				"baz": 24,
			},
			ArgumentsHash: "foobar",
		},
	}

	// Act
	serialized, err := MarshalResponseList(responses)

	// Assert
	require.NoError(t, err)

	var data any
	_ = json.Unmarshal(serialized, &data)
	dataList := data.([]any)
	dataItem := dataList[0].(map[string]any)

	require.EqualValues(t, 42, dataItem["result"])
	require.NotContains(t, dataItem, "argumentHash")
}

// Test that GetCommand() function returns the command name.
func TestGetCommand(t *testing.T) {
	command := NewCommand("list-commands", nil, nil)
	require.NotNil(t, command)
	require.Equal(t, "list-commands", command.GetCommand())
}

// Test that Response properly implements the ExaminableResponse interface.
func TestExaminableResponse(t *testing.T) {
	arguments := make(map[string]interface{})
	response := Response{
		ResponseHeader: ResponseHeader{
			Result: ResponseError,
			Text:   "a response text",
			Daemon: "dhcp4",
		},
		Arguments: &arguments,
	}
	require.EqualValues(t, 1, response.GetResult())
	require.Equal(t, "a response text", response.GetText())
	require.Equal(t, "dhcp4", response.GetDaemon())
	require.Equal(t, &arguments, response.GetArguments())
}

// Test that HashedResponse properly implements the ExaminableResponse interface.
func TestHashedExaminableResponse(t *testing.T) {
	arguments := make(map[string]interface{})
	response := HashedResponse{
		ResponseHeader: ResponseHeader{
			Result: 0,
			Text:   "another response text",
			Daemon: "dhcp6",
		},
		Arguments: &arguments,
	}
	require.Zero(t, response.GetResult())
	require.Equal(t, "another response text", response.GetText())
	require.Equal(t, "dhcp6", response.GetDaemon())
	require.Equal(t, &arguments, response.GetArguments())
}

// Test returning an error for a response with error status.
func TestGetResponseError(t *testing.T) {
	response := Response{
		ResponseHeader: ResponseHeader{
			Result: ResponseError,
			Text:   "another response text",
			Daemon: "dhcp6",
		},
	}
	err := GetResponseError(response)
	require.ErrorContains(t, err, "error status (1) returned by Kea dhcp6 daemon with text: 'another response text'")
}

// Test returning an error for a response with unsupported command status.
func TestGetResponseUnsupportedCommand(t *testing.T) {
	response := Response{
		ResponseHeader: ResponseHeader{
			Result: ResponseCommandUnsupported,
			Text:   "it is unsupported",
		},
	}
	err := GetResponseError(response)
	require.ErrorContains(t, err, "unsupported command status (2) returned by Kea with text: 'it is unsupported'")
}

// Test that no error is returned for a response with empty status.
func TestGetResponseEmpty(t *testing.T) {
	response := Response{
		ResponseHeader: ResponseHeader{
			Result: ResponseEmpty,
		},
	}
	require.Nil(t, GetResponseError(response))
}

// Test that no error is returned for a response with success status.
func TestGetResponseSuccess(t *testing.T) {
	response := Response{
		ResponseHeader: ResponseHeader{
			Result: ResponseSuccess,
		},
	}
	require.Nil(t, GetResponseError(response))
}

// Runs two benchmarks checking performance of Kea response unmarshalling
// with and without hashing the response arguments.
func BenchmarkUnmarshalHashedResponseList(b *testing.B) {
	request := NewCommand("list-subnets", []string{"dhcp4"}, nil)

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
			"result":    0,
			"text":      "success",
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
	b.Run("NoHashConfig", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			list := ResponseList{}
			err = UnmarshalResponseList(request, response, &list)
		}
	})
}
