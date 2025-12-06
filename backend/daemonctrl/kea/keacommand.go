package keactrl

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"isc.org/stork/datamodel/daemonname"
)

// Kea command name type.
type CommandName string

// Kea response result codes.
type ResponseResult int

// See "src/lib/cc/command_interpreter.h" in the Kea repository for details.
const (
	// Status code indicating a successful operation.
	ResponseSuccess ResponseResult = 0
	// Status code indicating a general failure.
	ResponseError ResponseResult = 1
	// Status code indicating that the specified command is not supported.
	ResponseCommandUnsupported ResponseResult = 2
	// Status code indicating that the specified command was completed
	// correctly, but failed to produce any results. For example, get
	// completed the search, but couldn't find the object it was looking for.
	ResponseEmpty ResponseResult = 3
	// Status code indicating that the command was unsuccessful due to a
	// conflict between the command arguments and the server state. For example,
	// a lease4-add fails when the subnet identifier in the command does not
	// match the subnet identifier in the server configuration.
	ResponseConflict ResponseResult = 4
)

// Interface to a Kea command that can be marshalled and sent.
type SerializableCommand interface {
	GetDaemonsList() []daemonname.Name
	GetCommand() CommandName
	Marshal() ([]byte, error)
}



// Represents a command sent to Kea including command name, daemons list
// (service list in Kea terms) and arguments.
type Command struct {
	Command   CommandName       `json:"command"`
	Daemons   []daemonname.Name `json:"service,omitempty"`
	Arguments any               `json:"arguments,omitempty"`
}

var _ SerializableCommand = (*Command)(nil)

// Creates new Kea command from specified command name, daemons list and arguments.
// The arguments are required to be a map or struct.
func newCommand(command CommandName, daemon daemonname.Name, arguments any) *Command {
	if len(command) == 0 {
		return nil
	}

	if arguments != nil {
		if _, ok := arguments.(json.RawMessage); !ok {
			argsType := reflect.TypeOf(arguments)
			switch argsType.Kind() {
			case reflect.Map, reflect.Struct:
				break
			case reflect.Ptr:
				if argsType.Elem().Kind() != reflect.Struct {
					return nil
				}
			default:
				return nil
			}
		}
	}

	cmd := &Command{
		Command:   command,
		Daemons:   []daemonname.Name{daemon},
		Arguments: arguments,
	}
	return cmd
}

// Constructs new command with no arguments.
func NewCommandBase(command CommandName, daemon daemonname.Name) *Command {
	return newCommand(command, daemon, nil)
}

// Returns command name.
func (c Command) GetCommand() CommandName {
	return c.Command
}

// Returns daemon names specified within the command.
func (c Command) GetDaemonsList() []daemonname.Name {
	return c.Daemons
}

// Sets daemon names for the command.
func (c *Command) SetDaemonsList(daemons []daemonname.Name) {
	c.Daemons = daemons
}

// Marshals the command to JSON.
func (c Command) Marshal() ([]byte, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal Kea command: %s", c.Command)
	}

	return data, nil
}

// Represents a command with non-serialized arguments.
type CommandWithRawArguments struct {
	Command   CommandName       `json:"command"`
	Daemons   []daemonname.Name `json:"service,omitempty"`
	Arguments json.RawMessage   `json:"arguments,omitempty"`
}

var _ SerializableCommand = (*CommandWithRawArguments)(nil)

// Returns command name.
func (c CommandWithRawArguments) GetCommand() CommandName {
	return c.Command
}

// Returns daemon names specified within the command.
func (c CommandWithRawArguments) GetDaemonsList() []daemonname.Name {
	return c.Daemons
}

// Sets daemon names for the command.
func (c *CommandWithRawArguments) SetDaemonsList(daemons []daemonname.Name) {
	c.Daemons = daemons
}

// Marshals the command to JSON.
func (c CommandWithRawArguments) Marshal() ([]byte, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal Kea command: %s", c.Command)
	}

	return data, nil
}

// Common fields in each received Kea response.
type ResponseHeader struct {
	Result ResponseResult `json:"result"`
	Text   string         `json:"text"`
}

// Represents unmarshaled response from Kea daemon.
type Response struct {
	ResponseHeader
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// Represents an error returned by Kea CA.
type KeaError struct {
	result ResponseResult
	text   string
}

// Returns the error message.
func (e KeaError) Error() string {
	if e.text != "" {
		return fmt.Sprintf(
			"non-success response result from Kea: %d, text: %s",
			e.result, e.text,
		)
	}
	return fmt.Sprintf("non-success response result from Kea: %d", e.result)
}

// Represents an error returned by Kea CA when the number overflow occurs.
type NumberOverflowKeaError struct {
	KeaError
}

// Represents an error returned by Kea CA when the DHCP daemon is likely to be
// offline.
type ConnectivityIssueKeaError struct {
	KeaError
}

// Represents an error returned by Kea CA when the operation is not supported
// (e.g., the specific hook is not loaded).
type UnsupportedOperationKeaError struct {
	KeaError
}

// The factory function to create a new KeaError instance based on the result
// and text received from Kea CA.
// It returns a most specific error type based on the text.
func newKeaError(result ResponseResult, text string) error {
	// Kea returns a proper response if the status is ResponseEmpty, so there
	// is no need to treat it as an error.
	if result == ResponseSuccess || result == ResponseEmpty {
		return nil
	}
	if result == ResponseCommandUnsupported {
		return UnsupportedOperationKeaError{KeaError{result, text}}
	}
	if strings.Contains(text, "Number overflow") {
		return NumberOverflowKeaError{KeaError{result, text}}
	}
	if strings.Contains(text, "server is likely to be offline") {
		return ConnectivityIssueKeaError{KeaError{result, text}}
	}
	return errors.WithStack(KeaError{result, text})
}

// An interface exposing properties of the response allowing for
// error checking.
type ExaminableResponse interface {
	GetResult() ResponseResult
	GetText() string
	GetArguments() json.RawMessage
}

// Given the command pointer it returns existing arguments map or creates
// a new arguments map, if it doesn't exist yet. It panics when the existing
// arguments are not a map.
func createOrGetArguments(command *Command) (mapArgs map[string]any) {
	if command.Arguments == nil {
		mapArgs = make(map[string]any)
		command.Arguments = mapArgs
		return
	}
	var ok bool
	mapArgs, ok = command.Arguments.(map[string]any)
	if !ok {
		panic("command arguments are not a map")
	}
	return
}

// Appends argument to the command. If the arguments are nil, the
// map of arguments is instantiated by this function. If the arguments
// are not a map, this function panics. Otherwise, the specified argument
// is set in the arguments map under the specified name.
func (c Command) WithArgument(name string, value any) *Command {
	command := c
	mapValue := createOrGetArguments(&command)
	mapValue[name] = value
	return &command
}

// Appends an array of arguments to the command. If the arguments are nil,
// the map of arguments is instantiated by this function. If the arguments
// are not a map, this function panics. Otherwise, an array of values is
// set in the arguments map under the specified name.
func (c Command) WithArrayArgument(name string, value ...any) *Command {
	command := c
	mapValue := createOrGetArguments(&command)
	mapValue[name] = value
	return &command
}

// Sets arguments for a command and returns a command copy.
func (c Command) WithArguments(arguments any) *Command {
	command := c
	command.Arguments = arguments
	return &command
}

// Returns status code.
func (r ResponseHeader) GetResult() ResponseResult {
	return r.Result
}

// Returns status text.
func (r ResponseHeader) GetText() string {
	return r.Text
}

// Returns status code.
func (r Response) GetResult() ResponseResult {
	return r.ResponseHeader.GetResult()
}

// Returns status text.
func (r Response) GetText() string {
	return r.ResponseHeader.GetText()
}

// Returns response arguments.
func (r Response) GetArguments() json.RawMessage {
	return r.Arguments
}

// Error returns the error returned by Kea.
func (r ResponseHeader) GetError() error {
	return newKeaError(r.Result, r.Text)
}

// Check response status code and returns appropriate error or nil if the
// response was successful.
func GetResponseError(response ExaminableResponse) (err error) {
	if response.GetResult() == ResponseError || response.GetResult() == ResponseCommandUnsupported {
		statusName := "error"
		if response.GetResult() == ResponseCommandUnsupported {
			statusName = "unsupported command"
		}
		err = errors.Errorf("%s status (%d) returned by Kea with text: '%s'",
			statusName, response.GetResult(), response.GetText())
	}
	return
}

// Unwraps Kea response from an array. Returns an error if the response is an
// array but its length is not equal to 1.
// If the response is not an array, it is returned as is.
func UnwrapKeaResponseArray(responseData json.RawMessage) (json.RawMessage, error) {
	var arrayBody []json.RawMessage
	err := json.Unmarshal(responseData, &arrayBody)
	if err == nil {
		if len(arrayBody) != 1 {
			return nil, errors.Errorf("invalid number of responses received, got: %d, expected: 1", len(arrayBody))
		}
		responseData = arrayBody[0]
	}
	return responseData, nil
}
