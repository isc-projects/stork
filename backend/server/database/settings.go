package dbops

import (
	"fmt"
	"path"
	"strings"

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
	DBName      string
	User        string
	Password    string
	Host        string
	Port        int
	SSLMode     string
	SSLCert     string
	SSLKey      string
	SSLRootCert string
	TraceSQL    LoggingQueryPreset
}

// Returns generic connection parameters as a list of space separated name/value pairs.
// All string values are enclosed in quotes. The quotes and double quotes within the
// string values are escaped. Empty or zero values are not included in the returned
// connection string.
// The parameter names must correspond to the respective libpq parameters.
// See https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-PARAMKEYWORDS.
func (s *DatabaseSettings) ConvertToConnectionString() string {
	escapeQuotes := func(paramValue string) string {
		// Escape quotes and double quotes.
		paramValue = strings.ReplaceAll(paramValue, "'", `\'`)
		paramValue = strings.ReplaceAll(paramValue, `"`, `\"`)
		// Enclose all strings in quotes in case they contain spaces.
		paramValue = fmt.Sprintf("'%s'", paramValue)
		return paramValue
	}

	params := [][]string{}

	if len(s.DBName) != 0 {
		params = append(params, []string{
			"dbname", escapeQuotes(s.DBName),
		})
	}

	if len(s.User) != 0 {
		params = append(params, []string{
			"user", escapeQuotes(s.User),
		})
	}

	if len(s.Password) != 0 {
		params = append(params, []string{
			"password", escapeQuotes(s.Password),
		})
	}

	if len(s.Host) != 0 {
		params = append(params, []string{
			"host", escapeQuotes(s.Host),
		})
	}

	if s.Port != 0 {
		params = append(params, []string{
			"port", fmt.Sprint(s.Port),
		})
	}

	if len(s.SSLMode) != 0 {
		params = append(params, []string{
			"sslmode", escapeQuotes(s.SSLMode),
		})
	} else {
		params = append(params, []string{
			"sslmode", escapeQuotes("disable"),
		})
	}

	if len(s.SSLCert) != 0 {
		params = append(params, []string{
			"sslcert", escapeQuotes(s.SSLCert),
		})
	}

	if len(s.SSLKey) != 0 {
		params = append(params, []string{
			"sslkey", escapeQuotes(s.SSLKey),
		})
	}

	if len(s.SSLRootCert) != 0 {
		params = append(params, []string{
			"sslrootcert", escapeQuotes(s.SSLRootCert),
		})
	}

	paramsStr := make([]string, len(params))
	idx := 0
	for _, param := range params {
		key, value := param[0], param[1]
		paramsStr[idx] = fmt.Sprintf("%s=%s", key, value)
		idx++
	}

	return strings.Join(paramsStr, " ")
}

// Converts generic connection parameters to go-pg specific parameters.
func (s *DatabaseSettings) convertToPgOptions() (*PgOptions, error) {
	pgopts := &PgOptions{Database: s.DBName, User: s.User, Password: s.Password}
	socketPath := path.Join(s.Host, fmt.Sprintf(".s.PGSQL.%d", s.Port))

	if s.Host == "" {
		pgopts.Network = "unix"
	} else if storkutil.IsSocket(socketPath) {
		pgopts.Addr = socketPath
		pgopts.Network = "unix"
	} else {
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
