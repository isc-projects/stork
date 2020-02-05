package dbmodel

import (
	"encoding/json"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"

	"strings"
)

// Kea daemon configuration map. It comprises a set of functions
// which retrieve complex data structures from the configuration.
type KeaConfig map[string]interface{}

// Structure representing a configuration of the single hooks library.
type KeaConfigHooksLibrary struct {
	Library    string
	Parameters map[string]interface{}
}

// Structure representing a configuration of a HA peer.
type Peer struct {
	Name         *string
	URL          *string
	Role         *string
	AutoFailover *bool `mapstructure:"auto-failover"`
}

// Structure representing a configuration of the HA hooks library
type KeaConfigHA struct {
	ThisServerName    *string `mapstructure:"this-server-name"`
	Mode              *string
	HeartbeatDelay    *int `mapstructure:"heartbeat-delay"`
	MaxResponseDelay  *int `mapstructure:"max-response-delay"`
	MaxAckDelay       *int `mapstructure:"max-ack-delay"`
	MaxUnackedClients *int `mapstructure:"max-unacked-clients"`
	Peers             []Peer
}

// Creates new instance from the pointer to the map of interfaces.
func NewKeaConfig(rawCfg *map[string]interface{}) *KeaConfig {
	newCfg := KeaConfig(*rawCfg)
	return &newCfg
}

// Create new instance from the configuration provided as JSON text.
func NewKeaConfigFromJSON(rawCfg string) (*KeaConfig, error) {
	var cfg KeaConfig
	err := json.Unmarshal([]byte(rawCfg), &cfg)
	if err != nil {
		err := errors.Wrapf(err, "problem with parsing JSON text: %s", rawCfg)
		return nil, err
	}
	return &cfg, nil
}

// Returns name of the root configuration node, e.g. Dhcp4.
// The second returned value designates whether the root node
// name was successfully found or not.
func (c *KeaConfig) GetRootName() (string, bool) {
	// This map will typically hold just a single element, but
	// in the past Kea supported Logging parameter aside of the
	// DHCP server configuration so we need to eliminate this one.
	for key := range *c {
		if key != "Logging" {
			return key, true
		}
	}
	return "", false
}

// Returns a list found at the top level of the configuration under
// a given name. If the given parameter does not exist or it is
// not a list, the ok value returned is set to false.
func (c *KeaConfig) GetTopLevelList(name string) (list []interface{}, ok bool) {
	root, ok := c.GetRootName()
	if !ok {
		return list, ok
	}

	if cfg, ok := (*c)[root]; ok {
		if rootNode, ok := cfg.(map[string]interface{}); ok {
			if listNode, ok := rootNode[name].([]interface{}); ok {
				return listNode, ok
			}
		}
	}

	return list, false
}

// Returns a list of all hooks libraries found in the configuration.
func (c *KeaConfig) GetHooksLibraries() (parsedLibraries []KeaConfigHooksLibrary) {
	if hooksLibrariesList, ok := c.GetTopLevelList("hooks-libraries"); ok {
		_ = mapstructure.Decode(hooksLibrariesList, &parsedLibraries)
	}
	return parsedLibraries
}

// Returns the information about a hooks library having a specified name
// if it exists in the configuration. The name parameter designates the
// name of the library, e.g. libdhcp_ha. The returned values include the
// path to the library, library configuration and the flag indicating
// whether the library exists or not.
func (c *KeaConfig) GetHooksLibrary(name string) (path string, params map[string]interface{}, ok bool) {
	libraries := c.GetHooksLibraries()
	for _, lib := range libraries {
		if strings.Contains(lib.Library, name) {
			path = lib.Library
			params = lib.Parameters
			ok = true
		}
	}
	return path, params, ok
}

// Returns configuration of the HA hooks library in a parsed form.
func (c *KeaConfig) GetHAHooksLibrary() (path string, params KeaConfigHA, ok bool) {
	path, paramsMap, ok := c.GetHooksLibrary("libdhcp_ha")
	if !ok {
		return path, params, ok
	}

	// HA hooks library should contain high-availability parameter being a
	// single element list. If it doesn't exist, it is an error.
	if haParamsList, ok := paramsMap["high-availability"].([]interface{}); !ok {
		path = ""
	} else {
		// Parse the list of HA configurations into a list of structures.
		var paramsList []KeaConfigHA
		err := mapstructure.Decode(haParamsList, &paramsList)
		if err != nil || len(paramsList) == 0 {
			path = ""
		} else {
			// HA configuration found, return it.
			params = paramsList[0]
		}
	}

	return path, params, ok
}
