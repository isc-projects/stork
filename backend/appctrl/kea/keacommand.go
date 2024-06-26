package keactrl

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/pkg/errors"
	keaconfig "isc.org/stork/appcfg/kea"
)

// Kea command name type.
type CommandName string

// Kea daemon name.
type DaemonName = string

const (
	DHCPv4 DaemonName = "dhcp4"
	DHCPv6 DaemonName = "dhcp6"
	D2     DaemonName = "d2"
	CA     DaemonName = "ca"
)

// See "src/lib/cc/command_interpreter.h" in the Kea repository for details.
const (
	// Status code indicating a successful operation.
	ResponseSuccess = 0
	// Status code indicating a general failure.
	ResponseError = 1
	// Status code indicating that the specified command is not supported.
	ResponseCommandUnsupported = 2
	// Status code indicating that the specified command was completed
	// correctly, but failed to produce any results. For example, get
	// completed the search, but couldn't find the object it was looking for.
	ResponseEmpty = 3
	// Status code indicating that the command was unsuccessful due to a
	// conflict between the command arguments and the server state. For example,
	// a lease4-add fails when the subnet identifier in the command does not
	// match the subnet identifier in the server configuration.
	ResponseConflict = 4
)

// Interface returning a list of daemons in the command.
type DaemonsLister interface {
	GetDaemonsList() []DaemonName
}

// Interface to a Kea command that can be marshalled and sent.
type SerializableCommand interface {
	DaemonsLister
	GetCommand() CommandName
	Marshal() string
}

// Represents a command sent to Kea including command name, daemons list
// (service list in Kea terms) and arguments.
type Command struct {
	Command   CommandName  `json:"command"`
	Daemons   []DaemonName `json:"service,omitempty"`
	Arguments interface{}  `json:"arguments,omitempty"`
}

// Common fields in each received Kea response.
type ResponseHeader struct {
	Result int    `json:"result"`
	Text   string `json:"text"`
	Daemon string `json:"-"`
}

// Represents unmarshaled response from Kea daemon.
type Response struct {
	ResponseHeader
	Arguments *map[string]interface{} `json:"arguments,omitempty"`
}

// A list of responses from multiple Kea daemons by the Kea Control Agent.
type ResponseList []Response

// Represents an error returned by Kea CA.
type KeaError struct {
	result int
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
func newKeaError(result int, text string) error {
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

// Represents unmarshaled response from Kea daemon with hash value computed
// from the arguments.
type HashedResponse struct {
	ResponseHeader
	Arguments     *map[string]interface{} `json:"arguments,omitempty"`
	ArgumentsHash string                  `json:"-"`
}

// A list of responses including hash value computed from the arguments.
type HashedResponseList []HashedResponse

// An interface exposing properties of the response allowing for
// error checking.
type ExaminableResponse interface {
	GetResult() int
	GetText() string
	GetDaemon() string
	GetArguments() *map[string]interface{}
}

// In some cases we need to compute a hash from the arguments received
// in a response. The arguments are passed as a string to a hashing
// function. Capturing the arguments as string requires hooking up to
// the JSON unmarshaller with a custom unmarshalling function. The
// hasherValue and hasher types serve this purpose.
type hasherValue string

type hasher struct {
	Value *hasherValue `json:"arguments,omitempty"`
}

// Custom unmarshaller hashing arguments string with FNV128 hashing function.
func (v *hasherValue) UnmarshalJSON(b []byte) error {
	*v = hasherValue(keaconfig.NewHasher().Hash(b))
	return nil
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

// Creates new Kea command from specified command name, daemons list and arguments.
// The arguments are required to be a map or struct.
func newCommand(command CommandName, daemons []DaemonName, arguments any) *Command {
	if len(command) == 0 {
		return nil
	}
	if arguments != nil {
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
	sort.Strings(daemons)
	cmd := &Command{
		Command:   command,
		Daemons:   daemons,
		Arguments: arguments,
	}
	return cmd
}

// Constructs new command from the JSON string.
func NewCommandFromJSON(jsonCommand string) (*Command, error) {
	cmd := Command{}
	err := json.Unmarshal([]byte(jsonCommand), &cmd)
	if err != nil {
		err = errors.Wrapf(err, "failed to parse Kea command: %s", jsonCommand)
		return nil, err
	}
	return &cmd, nil
}

// Constructs new command with no arguments.
func NewCommandBase(command CommandName, daemons ...DaemonName) *Command {
	return newCommand(command, daemons, nil)
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

// Returns JSON representation of the Kea command, which can be sent to
// the Kea servers over GRPC.
func (c Command) Marshal() string {
	bytes, _ := json.Marshal(c)
	return string(bytes)
}

// Returns command name.
func (c Command) GetCommand() CommandName {
	return c.Command
}

// Returns daemon names specified within the command.
func (c Command) GetDaemonsList() []DaemonName {
	return c.Daemons
}

// Converts parsed response from the Kea Control Agent to JSON. The "parsed"
// argument should be a slice of Response, HashedResponse or similar structures.
func MarshalResponseList(parsed interface{}) ([]byte, error) {
	// All computed fields are ignored internally by the json package.
	return json.Marshal(parsed)
}

// Parses response received from the Kea Control Agent. The "parsed" argument
// should be a slice of Response, HashedResponse or similar structures.
func UnmarshalResponseList(request DaemonsLister, response []byte, parsed interface{}) error {
	err := json.Unmarshal(response, parsed)
	if err != nil {
		err = errors.Wrapf(err, "failed to parse responses from Kea: %s", response)
		return err
	}

	// Try to match the responses with the services in the request and tag them with
	// the service names.
	parsedList := reflect.ValueOf(parsed).Elem()

	daemonNames := request.GetDaemonsList()
	if (len(daemonNames) > 0) && (parsedList.Len() > 0) {
		for i, daemon := range daemonNames {
			if i >= parsedList.Len() {
				break
			}
			parsedElem := parsedList.Index(i)
			field := parsedElem.FieldByName("Daemon")
			if field.IsValid() {
				field.SetString(daemon)
			}
		}
	}

	// Start computing hashes from the arguments received in the response.
	// We may consider optimizing it to hash while unmarshalling the response. This,
	// however, would require having a dedicated structure for arguments and custom
	// unmarshaller to be implemented for it. While this makes sense, it gives
	// significantly less flexibility on the caller side to use different structures
	// into which the responses are unmarshalled. Hopefully, several milliseconds more
	// for hashing the response doesn't matter for user experience, especially since
	// it is conducted in the background.
	hashers := []hasher{}
	for i := 0; i < parsedList.Len(); i++ {
		// First, we have to check if the response contains ArgumentsHash field.
		// Existence of this field is an indication that a caller wants us to
		// compute a hash.
		parsedElem := parsedList.Index(i)
		field := parsedElem.FieldByName("ArgumentsHash")
		if !field.IsValid() {
			// Response struct does not contain the ArgumentsHash, so there is
			// nothing to do.
			break
		}
		// If we haven't yet computed the hashes, let's do it now. We use
		// custom unmarshaller which will read the arguments parameter from
		// the response and compute hashes for each daemon from which a
		// response has been received.
		if len(hashers) == 0 {
			err = json.Unmarshal(response, &hashers)
			if err != nil {
				err = errors.Wrapf(err, "failed to compute hashes for Kea responses: %s", response)
				return err
			}
		}
		// This should not happen but let's be safe.
		if i >= len(hashers) {
			break
		}
		// Let's copy the hash value to the response if the hash exists. It may
		// be nil when no arguments were received in the response.
		if hashers[i].Value != nil {
			field.SetString(string(*hashers[i].Value))
		}
	}

	return nil
}

// Returns status code.
func (r ResponseHeader) GetResult() int {
	return r.Result
}

// Returns status text.
func (r ResponseHeader) GetText() string {
	return r.Text
}

// Returns name of the daemon that returned the response.
func (r ResponseHeader) GetDaemon() string {
	return r.Daemon
}

// Returns status code.
func (r Response) GetResult() int {
	return r.ResponseHeader.GetResult()
}

// Returns status text.
func (r Response) GetText() string {
	return r.ResponseHeader.GetText()
}

// Returns name of the daemon that returned the response.
func (r Response) GetDaemon() string {
	return r.ResponseHeader.GetDaemon()
}

// Returns response arguments.
func (r Response) GetArguments() *map[string]interface{} {
	return r.Arguments
}

// Error returns the error returned by Kea.
func (r ResponseHeader) GetError() error {
	return newKeaError(r.Result, r.Text)
}

// Returns status code.
func (r HashedResponse) GetResult() int {
	return r.ResponseHeader.GetResult()
}

// Returns status text.
func (r HashedResponse) GetText() string {
	return r.ResponseHeader.GetText()
}

// Returns name of the daemon that returned the response.
func (r HashedResponse) GetDaemon() string {
	return r.ResponseHeader.GetDaemon()
}

// Returns response arguments.
func (r HashedResponse) GetArguments() *map[string]interface{} {
	return r.Arguments
}

// Check response status code and returns appropriate error or nil if the
// response was successful.
func GetResponseError(response ExaminableResponse) (err error) {
	if response.GetResult() == ResponseError || response.GetResult() == ResponseCommandUnsupported {
		statusName := "error"
		if response.GetResult() == ResponseCommandUnsupported {
			statusName = "unsupported command"
		}
		var daemon string
		if len(response.GetDaemon()) > 0 {
			daemon = "Kea " + response.GetDaemon() + " daemon"
		} else {
			daemon = "Kea"
		}
		err = errors.Errorf("%s status (%d) returned by %s with text: '%s'",
			statusName, response.GetResult(), daemon, response.GetText())
	}
	return
}
