package dbmodel

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// Kea daemon configuration map. It comprises a set of functions
// which retrieve complex data structures from the configuration.
type KeaConfig map[string]interface{}

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
	for key, _ := range *c {
		if key != "Logging" {
			return key, true
		}
	}
	return "", false
}
