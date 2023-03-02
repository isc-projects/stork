package keaconfig

import "encoding/json"

// An interface exposing a function to fetch all database connection
// configurations for a Kea server. It is implemented by the
// keaconfig.Config.
type DatabaseConfig interface {
	GetAllDatabases() Databases
}

// Configuration of the Kea configuration backend connections.
type ConfigControl struct {
	ConfigDatabases     []Database `json:"config-databases"`
	ConfigFetchWaitTime int64      `json:"config-wait-fetch-time"`
}

// A structure holding all possible database configurations in the Kea
// configuration structure (i.e., lease database, hosts databases,
// config backend and forensic logging database). If some of the
// configurations are not present, nil values or empty slices are
// returned for them. This structure is returned by functions parsing
// Kea configurations to find present database configurations.
type Databases struct {
	Lease    *Database
	Hosts    []Database
	Config   []Database
	Forensic *Database
}

// A structure representing the database connection parameters. It is common
// for all supported backend types.
type Database struct {
	Path string `json:"path"`
	Type string `json:"type"`
	Name string `json:"name"`
	Host string `json:"host"`
}

// Parses database connection configuration setting the default
// host value 'localhost', if it hasn't been specified.
func (d *Database) UnmarshalJSON(data []byte) error {
	type t Database
	if err := json.Unmarshal(data, (*t)(d)); err != nil {
		return err
	}
	if len(d.Host) == 0 && len(d.Path) == 0 {
		d.Host = "localhost"
	}
	return nil
}
