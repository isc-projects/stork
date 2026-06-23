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
	// Path is the path to the legal log file. It is only used by the
	// legal log hook's configuration. It does not contain the path to
	// the lease file.
	Path string `json:"path"`
	// Type is a string constant indicating which type of lease
	// backend to use. Valid values are:
	// - `memfile`: Store leases on disk in a CSV file
	// - `postgresql`: Store leases in a PostgreSQL database
	// - `mysql`: Store leases in a MySQL/MariaDB database
	// It may be set to something else if someone runs Kea with a hook which
	// replaces the database backend.
	Type string `json:"type"`
	// Name contains the path to the lease memfile on disk, or the name
	// of the database to use on an SQL database server, depending on
	// the value of [Database.Type].
	Name string `json:"name"`
	// Host contains the hostname or IP address of the SQL database
	// backend server.
	Host string `json:"host"`
	// Port contains the port used to talk to the SQL database backend
	// server.
	Port int `json:"port,omitempty"`
	// User contains the username to use when authenticating with an SQL
	// database server.
	User string `json:"user"`
	// TrustAnchor contains the path to the full certificate chain used
	// for talking to an SQL database server.
	TrustAnchor string `json:"trust-anchor"`
	// CertFile contains the path to the client certificate file used
	// for talking to an SQL database server.
	CertFile string `json:"cert-file"`
	// KeyFile contains the path to the client private key used for
	// talking to an SQL database server.
	KeyFile string `json:"key-file"`
	// Persist is true when in memfile mode and the lease database
	// should be saved to disk. It is false when in memfile mode and the
	// lease database *should not* be saved to disk. When it is nil, the
	// default value is true, but that is only meaningful when Kea is
	// configured in `memfile` mode.
	Persist *bool `json:"persist,omitempty"`
}

// Indicates whether a full TLS client certificate setup is configured.
func (d Database) IsTLSClientCertConfigured() bool {
	return len(d.TrustAnchor) > 0 && len(d.CertFile) > 0 && len(d.KeyFile) > 0
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
