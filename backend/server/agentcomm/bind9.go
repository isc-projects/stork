package agentcomm

import (
	"encoding/json"

	"github.com/pkg/errors"
	agentapi "isc.org/stork/api"
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

// Represents the raw BIND 9 configuration returned by the agent.
// It encapsulates a list of configuration files because BIND 9
// configuration may comprise rndc.key file besides named.conf.
type Bind9RawConfig struct {
	Files []*Bind9ConfigFile
}

// Represents a single BIND 9 configuration file. It contains the
// file type, file source path and the formatted contents.
type Bind9ConfigFile struct {
	FileType   Bind9ConfigFileType
	SourcePath string
	Contents   string
}

// Instantiates a new Bind9ConfigFile from a protobuf message.
func NewBind9ConfigFileFromProto(file *agentapi.Bind9ConfigFile) *Bind9ConfigFile {
	fileType := Bind9ConfigFileTypeConfig
	if file.FileType == agentapi.Bind9ConfigFile_RNDC_KEY {
		fileType = Bind9ConfigFileTypeRndcKey
	}
	return &Bind9ConfigFile{
		FileType:   fileType,
		SourcePath: file.SourcePath,
		Contents:   file.Contents,
	}
}
