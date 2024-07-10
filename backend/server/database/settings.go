package dbops

import (
	"fmt"
	"path"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"

	storkutil "isc.org/stork/util"
)

// Alias to pg.DB.
type PgDB = pg.DB

// Alias to pg.Conn.
type PgConn = pg.Conn

// Alias to pg.Options.
type PgOptions = pg.Options

// A type for constants defining the supported presets of SQL query logging.
type LoggingQueryPreset string

const (
	// Log all SQL queries. Includes runtime and migration queries.
	LoggingQueryPresetAll LoggingQueryPreset = "all"
	// Log the runtime SQL queries. Skips the migration queries.
	LoggingQueryPresetRuntime LoggingQueryPreset = "run"
	// Disable SQL query logging.
	LoggingQueryPresetNone LoggingQueryPreset = "none"
)

// Converts a raw string into the logging query preset enum.
func newLoggingQueryPreset(raw string) LoggingQueryPreset {
	switch raw {
	case string(LoggingQueryPresetRuntime), string(LoggingQueryPresetAll):
		return LoggingQueryPreset(raw)
	default:
		return LoggingQueryPresetNone
	}
}

// Enables singular SQL table names for go-pg ORM.
func init() {
	orm.SetTableNameInflector(func(s string) string {
		return s
	})
}

// Represents database connection settings.
type DatabaseSettings struct {
	DBName       string
	User         string
	Password     string
	Host         string
	Port         int
	SSLMode      string
	SSLCert      string
	SSLKey       string
	SSLRootCert  string
	TraceSQL     LoggingQueryPreset
	WriteTimeout time.Duration
	ReadTimeout  time.Duration
}

// Converts generic connection parameters to go-pg specific parameters.
func (s *DatabaseSettings) convertToPgOptions() (*PgOptions, error) {
	pgopts := &PgOptions{
		Database:        s.DBName,
		User:            s.User,
		Password:        s.Password,
		ApplicationName: "stork-server",
		ReadTimeout:     s.ReadTimeout,
		WriteTimeout:    s.WriteTimeout,
	}
	socketPath := path.Join(s.Host, fmt.Sprintf(".s.PGSQL.%d", s.Port))

	switch {
	case s.Host == "":
		pgopts.Network = "unix"
	case storkutil.IsSocket(socketPath):
		pgopts.Addr = socketPath
		pgopts.Network = "unix"
	default:
		pgopts.Addr = fmt.Sprintf("%s:%d", s.Host, s.Port)
		pgopts.Network = "tcp"
		tlsConfig, err := GetTLSConfig(s.SSLMode, s.Host, s.SSLCert, s.SSLKey, s.SSLRootCert)
		if err != nil {
			return nil, err
		}
		pgopts.TLSConfig = tlsConfig
	}

	return pgopts, nil
}
