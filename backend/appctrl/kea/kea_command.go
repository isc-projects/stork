package keactrl

import (
	"encoding/json"
	"reflect"
	"sort"

	"github.com/pkg/errors"
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

// Parses response received from the Kea Control Agent.
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

	return nil
}
