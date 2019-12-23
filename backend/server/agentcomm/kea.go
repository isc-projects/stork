package agentcomm

import (
	"encoding/json"
	"github.com/pkg/errors"
	"sort"
)

// Map holding names of services to which the command is sent. This is
// stored in the map rather than a list to guarantee uniqueness of the
// service names.
type KeaServices map[string]bool

// Represents a command sent to Kea including command, service list and
// arguments.
type KeaCommand struct {
	Command   string                  `json:"command"`
	Services  *KeaServices            `json:"service,omitempty"`
	Arguments *map[string]interface{} `json:"arguments,omitempty"`
}

// Represents unmarshaled response from Kea deamon.
type KeaResponse struct {
	Result    int                     `json:"result"`
	Text      string                  `json:"text"`
	Service   string                  `json:",ignore"`
	Arguments *map[string]interface{} `json:"arguments,omitempty"`
}

// A list of responses from multiple Kea deamons by the Kea Control Agent.
type KeaResponseList []KeaResponse

// Creates a map holding specified service names. The service names
// must be unique and non-empty.
func NewKeaServices(serviceNames ...string) (*KeaServices, error) {
	keaServices := make(KeaServices)
	for _, name := range serviceNames {
		if len(name) == 0 {
			// Service name must be non-empty.
			return nil, errors.Errorf("service name must not be empty")

		} else if _, exists := keaServices[name]; exists {
			// Duplicates are not allowed.
			return nil, errors.Errorf("duplicate service name %s", name)
		}
		keaServices[name] = true
	}
	return &keaServices, nil
}

// Checks if the service with the given name has been specified.
func (s *KeaServices) Contains(serviceName string) bool {
	_, ok := (*s)[serviceName]
	return ok
}

// Returns the services as a sorted list of strings.
func (s *KeaServices) List() []string {
	var keys []string
	for name := range *s {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	return keys
}

// Implementation of the MarshalJSON which convert map of services into
// a list of map keys.
func (s *KeaServices) MarshalJSON() ([]byte, error) {
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

// Creates new Kea command from specified command name, service list and arguments.
func NewKeaCommand(command string, service *KeaServices, arguments *map[string]interface{}) (*KeaCommand, error) {
	if len(command) == 0 {
		return nil, errors.Errorf("command name must not be empty")
	}

	cmd := &KeaCommand{
		Command:   command,
		Services:  service,
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
func UnmarshalKeaResponseList(request *KeaCommand, response string) (*KeaResponseList, error) {
	responses := KeaResponseList{}
	err := json.Unmarshal([]byte(response), &responses)
	if err != nil {
		err = errors.Wrapf(err, "failed to parse responses from Kea: %s", response)
		return nil, err
	}

	// Try to match the responses with the services in the request and tag them with
	// the service names.
	if (request.Services != nil) && (len(*request.Services) > 0) && (len(responses) > 0) {
		serviceNames := request.Services.List()
		for i, service := range serviceNames {
			if i+1 > len(responses) {
				break
			}
			responses[i].Service = service
		}
	}

	return &responses, err
}
