package agentcomm

import (
	"encoding/json"
	"reflect"
	"github.com/pkg/errors"
)

// Map holding names of services to which the command is sent. This is
// stored in the map rather than a list to guarantee uniqueness of the
// service names.
type KeaServices map[string]bool

// Represents a command sent to Kea including command, service list and
// arguments.
type KeaCommand struct {
	Command   string `json:"command"`
	Services  *KeaServices `json:"service,omitempty"`
	Arguments *map[string]interface{} `json:"arguments,omitempty"`
}

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

// Implementation of the MarshalJSON which convert map of services into
// a list of map keys.
func (s *KeaServices) MarshalJSON() ([]byte, error) {
	o := "["
	for i, name := range reflect.ValueOf(*s).MapKeys() {
		if i > 0 {
			o += ","
		}
		o += "\"" + name.String() + "\""
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
		Command: command,
		Services: service,
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
