package agentcomm

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// Store BIND 9 access control configuration.
type Bind9Control struct {
	Address string
	Port    int64
	Key     string
}

// Represents unmarshaled response from named statistics-channel.
type NamedStatsGetResponse struct {
	Views *map[string]interface{} `json:"views,omitempty"`
}

// Parses response received from the named statistics-channel.
func UnmarshalNamedStatsResponse(response string, parsed interface{}) error {
	err := json.Unmarshal([]byte(response), parsed)
	if err != nil {
		return errors.Wrapf(err, "failed to parse response from named statistics-channel: %s", response)
	}
	return nil
}

// Represents the type of a BIND 9 configuration file returned by the
// agent when getting the raw configuration.
type Bind9ConfigFileType string

const (
	Bind9ConfigFileTypeConfig  Bind9ConfigFileType = "config"
	Bind9ConfigFileTypeRndcKey Bind9ConfigFileType = "rndc-key"
)
