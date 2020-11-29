package keactrl

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	"github.com/pkg/errors"
	storkutil "isc.org/stork/util"
)

const (
	ResponseSuccess            = 0
	ResponseError              = 1
	ResponseCommandUnsupported = 2
	ResponseEmpty              = 3
)

// Map holding names of deamons to which the command is sent. This is
// stored in the map rather than a list to guarantee uniqueness of the
// deamons' names. Not that deamons are called services in the Kea terms,
// however we use Stork specific terminology here. A service means
// something different in Stork, i.e. it is an aggregation of multiple
// cooperating applications.
type Daemons map[string]bool

// Represents a command sent to Kea including command name, daemons list
// (service list in Kea terms) and arguments.
type Command struct {
	Command   string                  `json:"command"`
	Daemons   *Daemons                `json:"service,omitempty"`
	Arguments *map[string]interface{} `json:"arguments,omitempty"`
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

// Represents unmarshaled response from Kea daemon with hash value computed
// from the arguments.
type HashedResponse struct {
	ResponseHeader
	Arguments     *map[string]interface{} `json:"arguments,omitempty"`
	ArgumentsHash string                  `json:"-"`
}

// A list of responses including hash value computed from the arguments.
type HashedResponseList []HashedResponse

// In some cases we need to compute a hash from the arguments received
// in a response. The arguments are passed as a string to a hashing
// function. Capturing the arguments as string requires hooking up to
// the JSON unmarshaller with a custom unmarshalling function. The
// hasherValue and hasher types serve this purpose.
type hasherValue string

type hasher struct {
	Value *hasherValue `json:"arguments,omitempty"`
}

// Creates a map holding specified daemons names. The daemons names
// must be unique and non-empty.
func NewDaemons(daemonNames ...string) (*Daemons, error) {
	keaDaemons := make(Daemons)
	for _, name := range daemonNames {
		if len(name) == 0 {
			// A name must be non-empty.
			return nil, errors.Errorf("daemon name must not be empty")
		} else if _, exists := keaDaemons[name]; exists {
			// Duplicates are not allowed.
			return nil, errors.Errorf("duplicate daemon name %s", name)
		}
		keaDaemons[name] = true
	}
	return &keaDaemons, nil
}

// Checks if the daemon with the given name has been specified.
func (s *Daemons) Contains(daemonName string) bool {
	_, ok := (*s)[daemonName]
	return ok
}

// Returns the daemons as a sorted list of strings.
func (s *Daemons) List() []string {
	var keys []string
	for name := range *s {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	return keys
}

// Implementation of the MarshalJSON which converts map of daemons into
// a list of map keys.
func (s *Daemons) MarshalJSON() ([]byte, error) {
	keys := s.List()

	o := "["
	for i, key := range keys {
		if i > 0 {
			o += ","
		}
		o += "\"" + key + "\""
	}
	o += "]"
	return []byte(o), nil
}

// Implementation of the UnmarshalJSON which converts daemons into a list
// of map keys.
func (s *Daemons) UnmarshalJSON(b []byte) error {
	var keyList []string
	err := json.Unmarshal(b, &keyList)
	if err != nil {
		return errors.WithStack(err)
	}
	*s = make(Daemons)
	for i := range keyList {
		if _, ok := (*s)[keyList[i]]; ok {
			return errors.Errorf("duplicate daemon name %s", keyList[i])
		}
		(*s)[keyList[i]] = true
	}
	return nil
}

// Custom unmarshaller hashing arguments string with FNV128 hashing function.
func (v *hasherValue) UnmarshalJSON(b []byte) error {
	*v = hasherValue(storkutil.Fnv128(fmt.Sprintf("%s", b)))
	return nil
}

// Creates new Kea command from specified command name, damons list and arguments.
func NewCommand(command string, daemons *Daemons, arguments *map[string]interface{}) (*Command, error) {
	if len(command) == 0 {
		return nil, errors.Errorf("command name must not be empty")
	}

	cmd := &Command{
		Command:   command,
		Daemons:   daemons,
		Arguments: arguments,
	}
	return cmd, nil
}

func NewCommandFromJSON(jsonCommand string) (*Command, error) {
	cmd := Command{}
	err := json.Unmarshal([]byte(jsonCommand), &cmd)
	if err != nil {
		err = errors.Wrapf(err, "failed to parse Kea command: %s", jsonCommand)
		return nil, err
	}
	return &cmd, nil
}

// Returns JSON representation of the Kea command, which can be sent to
// the Kea servers over GRPC.
func (c *Command) Marshal() string {
	bytes, _ := json.Marshal(c)
	return string(bytes)
}

// Parses response received from the Kea Control Agent. The "parsed" argument
// should be a slice of Response, HashedResponse or similar structures.
func UnmarshalResponseList(request *Command, response []byte, parsed interface{}) error {
	err := json.Unmarshal(response, parsed)
	if err != nil {
		err = errors.Wrapf(err, "failed to parse responses from Kea: %s", response)
		return err
	}

	// Try to match the responses with the services in the request and tag them with
	// the service names.
	parsedList := reflect.ValueOf(parsed).Elem()

	if (request.Daemons != nil) && (len(*request.Daemons) > 0) && (parsedList.Len() > 0) {
		daemonNames := request.Daemons.List()
		for i, daemon := range daemonNames {
			if i+1 > parsedList.Len() {
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
	// We may consider optimizing it to hash while unmarshaling the response. This,
	// however, would require having a dedicated structure for arguments and custom
	// unmarshaller to be implemented for it. While this makes sense, it gives
	// significantly less flexibility on the caller side to use different structures
	// into which the responses are unmarshalled. Hopefully, several milliseconds more
	// for hashing the response doesn't matter for user experience, especially that
	// it is conducted in background.
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
		if i > len(hashers) {
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
