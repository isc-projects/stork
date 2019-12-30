package agentcomm

import (
	"encoding/json"
	"github.com/pkg/errors"
	"sort"
)

// Map holding names of deamons to which the command is sent. This is
// stored in the map rather than a list to guarantee uniqueness of the
// deamons' names. Not that deamons are called services in the Kea terms,
// however we use Stork specific terminology here. A service means
// something different in Stork, i.e. it is an aggregation of multiple
// cooperating applications.
type KeaDaemons map[string]bool

// Represents a command sent to Kea including command name, daemons list
// (service list in Kea terms) and arguments.
type KeaCommand struct {
	Command   string                  `json:"command"`
	Daemons   *KeaDaemons             `json:"service,omitempty"`
	Arguments *map[string]interface{} `json:"arguments,omitempty"`
}

// Represents unmarshaled response from Kea daemon.
type KeaResponse struct {
	Result    int                     `json:"result"`
	Text      string                  `json:"text"`
	Daemon    string                  `json:"-"`
	Arguments *map[string]interface{} `json:"arguments,omitempty"`
}

// A list of responses from multiple Kea daemons by the Kea Control Agent.
type KeaResponseList []KeaResponse

// Creates a map holding specified daemons names. The daemons names
// must be unique and non-empty.
func NewKeaDaemons(daemonNames ...string) (*KeaDaemons, error) {
	keaDaemons := make(KeaDaemons)
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
func (s *KeaDaemons) Contains(daemonName string) bool {
	_, ok := (*s)[daemonName]
	return ok
}

// Returns the daemons as a sorted list of strings.
func (s *KeaDaemons) List() []string {
	var keys []string
	for name := range *s {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	return keys
}

// Implementation of the MarshalJSON which converts map of daemons into
// a list of map keys.
func (s *KeaDaemons) MarshalJSON() ([]byte, error) {
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

// Creates new Kea command from specified command name, damons list and arguments.
func NewKeaCommand(command string, daemons *KeaDaemons, arguments *map[string]interface{}) (*KeaCommand, error) {
	if len(command) == 0 {
		return nil, errors.Errorf("command name must not be empty")
	}

	cmd := &KeaCommand{
		Command:   command,
		Daemons:   daemons,
		Arguments: arguments,
	}
	return cmd, nil
}

// Returns JSON representation of the Kea command, which can be sent to
// the Kea servers over GRPC.
func (c *KeaCommand) Marshal() string {
	bytes, _ := json.Marshal(c)
	return string(bytes)
}

// Parses response received from the Kea Control Agent.
func UnmarshalKeaResponseList(request *KeaCommand, response string) (KeaResponseList, error) {
	responses := KeaResponseList{}
	err := json.Unmarshal([]byte(response), &responses)
	if err != nil {
		err = errors.Wrapf(err, "failed to parse responses from Kea: %s", response)
		return nil, err
	}

	// Try to match the responses with the services in the request and tag them with
	// the service names.
	if (request.Daemons != nil) && (len(*request.Daemons) > 0) && (len(responses) > 0) {
		daemonNames := request.Daemons.List()
		for i, daemon := range daemonNames {
			if i+1 > len(responses) {
				break
			}
			responses[i].Daemon = daemon
		}
	}

	return responses, err
}
