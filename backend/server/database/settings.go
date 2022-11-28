package dbops

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"github.com/pkg/errors"

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

// Enables singular SQL table names for go-pg ORM.
func init() {
	orm.SetTableNameInflector(func(s string) string {
		return s
	})
}

// Represents database connection settings. The "pq" tag names and their values
// must correspond to the respective libpq parameters.
// See https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-PARAMKEYWORDS.
type DatabaseSettings struct {
	DBName      string `pq:"dbname"`
	User        string `pq:"user"`
	Password    string `pq:"password"`
	Host        string `pq:"host"`
	Port        int    `pq:"port"`
	SSLMode     string `pq:"sslmode"`
	SSLCert     string `pq:"sslcert"`
	SSLKey      string `pq:"sslkey"`
	SSLRootCert string `pq:"sslrootcert"`
	TraceSQL    LoggingQueryPreset
}

// Returns generic connection parameters as a list of space separated name/value pairs.
// All string values are enclosed in quotes. The quotes and double quotes within the
// string values are escaped. Empty or zero values are not included in the returned
// connection string.
func (s *DatabaseSettings) ConvertToConnectionString() string {
	// Get the reflect representation of the structure.
	v := reflect.ValueOf(s).Elem()

	// Get the types of the fields in the structure.
	vType := v.Type()

	// Iterate over the fields and append them to the connection string if needed.
	var params [][]string

	for i := 0; i < v.NumField(); i++ {
		field := vType.Field(i)

		// Parameter name
		paramName, ok := field.Tag.Lookup("pq")
		if !ok {
			continue
		}

		// Parameter value
		var paramValue string

		// Check the type of the current field.
		switch field.Type.Kind() {
		case reflect.String:
			// Only append the parameter if it is non-empty.
			paramValue = v.Field(i).String()
			if len(paramValue) == 0 {
				continue
			}
			// Escape quotes and double quotes.
			paramValue = strings.ReplaceAll(paramValue, "'", `\'`)
			paramValue = strings.ReplaceAll(paramValue, `"`, `\"`)
			// Enclose all strings in quotes in case they contain spaces.
			paramValue = fmt.Sprintf("'%s'", paramValue)

		case reflect.Int:
			// If the int value is zero, do not include it.
			paramValueInt := v.Field(i).Int()
			if paramValueInt == 0 {
				continue
			}
			paramValue = fmt.Sprint(paramValueInt)
		default:
			// Unsupported type.
			continue
		}

		params = append(params, []string{paramName, paramValue})
	}
	if len(s.SSLMode) == 0 {
		params = append(params, []string{"sslmode", "'disable'"})
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

	if storkutil.IsSocket(socketPath) {
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

// Iterates over the struct members including the nested ones.
func iterateOverFields(obj any, f func(field reflect.StructField, valueField reflect.Value)) {
	type fieldValuePair struct {
		field reflect.StructField
		value reflect.Value
	}

	// Get the reflect representation of the structure.
	v := reflect.ValueOf(obj).Elem()

	// Get the types of the fields in the structure.
	vType := reflect.TypeOf(obj).Elem()

	// It'll iterate over nested structs without recursion.
	var fieldQueue []fieldValuePair

	// Iterate over the top-level members and load them into queue.
	for i := 0; i < vType.NumField(); i++ {
		var valueField reflect.Value
		// If the object is nil then the value fields are invalid. Accessing
		// them causes panic.
		if v.IsValid() {
			valueField = v.Field(i)
		}

		fieldQueue = append(fieldQueue, fieldValuePair{
			field: vType.Field(i),
			value: valueField,
		})
	}

	// Perform until exhaust the queue.
	for len(fieldQueue) != 0 {
		// Pop first element.
		pair := fieldQueue[0]
		// Remove the first element.
		fieldQueue = fieldQueue[1:]

		// Extract field type.
		fieldType := pair.field.Type

		// Iterate over the nested fields. Only structs are supported.
		if fieldType.Kind() == reflect.Struct {
			for i := 0; i < fieldType.NumField(); i++ {
				var valueField reflect.Value
				// If the object is nil then the value fields are invalid. Accessing
				// them causes panic.
				if v.IsValid() {
					valueField = pair.value.Field(i)
				}

				// Push the nested field to a queue.
				fieldQueue = append(fieldQueue, fieldValuePair{
					field: fieldType.Field(i),
					value: valueField,
				})
			}

			// Not call the callback for the nested field.
			continue
		}

		// Call the callback for the leaf fields.
		f(pair.field, pair.value)
	}
}

// Sets the member fields of a given object using the structure tags and value
// lookup object. The function searches for the provided tag in the member tags.
// If found, the tag value is passed to the value lookup. The output string is
// set as a member value.
//
// The output string is converted to a number if the member type is an integer.
// The function supports only string and integer (int) data types.
//
// If the member doesn't have a specific tag, value lookup returns no value,
// the member has an unsupported type, or integer conversion fails, then the
// member is skipped. It has a default value.
func setFieldsBasedOnTags(obj any, tagName string, valueLookup func(string) (string, bool)) {
	iterateOverFields(obj, func(field reflect.StructField, valueField reflect.Value) {
		key, ok := field.Tag.Lookup(tagName)
		if !ok {
			return
		}

		value, ok := valueLookup(key)
		if !ok {
			return
		}

		switch field.Type.Kind() {
		case reflect.String:
			valueField.SetString(value)
		case reflect.Int:
			envValueInt, err := strconv.ParseInt(value, 10, 0)
			if err != nil {
				return
			}
			valueField.SetInt(envValueInt)
		default:
			// Skip unsupported fields.
		}
	})
}

// Sets the member fields of a given object using the structure tags and the
// environment variables. The function searches for the 'env' tag in the member
// tags. If found, the tag value is . The output string is set as a member value.
func readFromEnvironment(obj any) {
	setFieldsBasedOnTags(obj, "env", os.LookupEnv)
}

// Defines the interface to perform the lookup value of the CLI flags.
type CLILookup interface {
	// Check if the CLI flag with a given name exists.
	IsSet(key string) bool
	// Returns the value of CLI flag with a given name.
	String(key string) string
}

// Sets the member fields of a given object using the structure tags and CLI
// lookup object. The function searches for the 'long' tag in the member tags
// for recognize a related CLI flag. If found, the flag value is set as a
// member value.
func readFromCLI(obj any, lookup CLILookup) {
	setFieldsBasedOnTags(obj, "long", func(key string) (string, bool) {
		value := lookup.String(key)
		if value != "" || lookup.IsSet(key) {
			return value, true
		}
		return "", false
	})
}

// The definition of the CLI flag compatible with the struct tags
// used by the 'github.com/jessevdk/go-flags' library.
type CLIFlagDefinition struct {
	Short               string
	Long                string
	Description         string
	EnvironmentVariable string
	Default             string
	Kind                reflect.Kind
}

// Reads the CLI flags metadata from the struct tags. It must be safe for nil
// pointers.
func convertToCLIFlagDefinitions(obj any) []*CLIFlagDefinition {
	var flags []*CLIFlagDefinition

	iterateOverFields(obj, func(field reflect.StructField, _ reflect.Value) {
		var flag CLIFlagDefinition

		flag.Kind = field.Type.Kind()

		value, ok := field.Tag.Lookup("short")
		if ok {
			flag.Short = value
		}

		value, ok = field.Tag.Lookup("long")
		if ok {
			flag.Long = value
		}

		value, ok = field.Tag.Lookup("description")
		if ok {
			flag.Description = value
		}

		value, ok = field.Tag.Lookup("env")
		if ok {
			flag.EnvironmentVariable = value
		}

		value, ok = field.Tag.Lookup("default")
		if ok {
			flag.Default = value
		}

		flags = append(flags, &flag)
	})

	return flags
}

// General definition of the CLI flags used to connect to the database.
type DatabaseCLIFlags struct {
	URL         string `long:"db-url" description:"The URL to locate the Stork PostgreSQL database" env:"STORK_DATABASE_URL"`
	DBName      string `short:"d" long:"db-name" description:"The name of the database to connect to" env:"STORK_DATABASE_NAME" default:"stork"`
	User        string `short:"u" long:"db-user" description:"The user name to be used for database connections" env:"STORK_DATABASE_USER_NAME" default:"stork"`
	Password    string `long:"db-password" description:"The database password to be used for database connections" env:"STORK_DATABASE_PASSWORD"`
	Host        string `long:"db-host" description:"The host name, IP address or socket where database is available" env:"STORK_DATABASE_HOST" default:"/var/run/postgresql"`
	Port        int    `short:"p" long:"db-port" description:"The port on which the database is available" env:"STORK_DATABASE_PORT" default:"5432"`
	SSLMode     string `long:"db-sslmode" description:"The SSL mode for connecting to the database" choice:"disable" choice:"require" choice:"verify-ca" choice:"verify-full" env:"STORK_DATABASE_SSLMODE" default:"disable"` //nolint:staticcheck
	SSLCert     string `long:"db-sslcert" description:"The location of the SSL certificate used by the server to connect to the database" env:"STORK_DATABASE_SSLCERT"`
	SSLKey      string `long:"db-sslkey" description:"The location of the SSL key used by the server to connect to the database" env:"STORK_DATABASE_SSLKEY"`
	SSLRootCert string `long:"db-sslrootcert" description:"The location of the root certificate file used to verify the database server's certificate" env:"STORK_DATABASE_SSLROOTCERT"`
	TraceSQL    string `long:"db-trace-queries" description:"Enable tracing SQL queries: run (only run-time, without migrations), all (migrations and run-time), or none (no query logging)." env:"STORK_DATABASE_TRACE" choice:"run" choice:"all" choice:"none" default:"none"` //nolint:staticcheck
}

// Converts the values of CLI flags to the database settings. They don't
// use the maintenance parameters. The standard user will connect to the
// standard database.
// It may parse the access options from the URL but returns an error if it's
// provided simultaneously with the standard parameters.
func (s *DatabaseCLIFlags) ConvertToDatabaseSettings() (*DatabaseSettings, error) {
	settings := &DatabaseSettings{
		DBName:      s.DBName,
		User:        s.User,
		Password:    s.Password,
		Host:        s.Host,
		Port:        s.Port,
		SSLMode:     s.SSLMode,
		SSLCert:     s.SSLCert,
		SSLKey:      s.SSLKey,
		SSLRootCert: s.SSLRootCert,
		TraceSQL:    LoggingQueryPreset(s.TraceSQL),
	}

	if s.URL != "" {
		// URL is mutually exclusive with some other parameters.
		var nonEmptyParam string
		switch {
		case s.DBName != "":
			nonEmptyParam = "database name"
		case s.User != "":
			nonEmptyParam = "user"
		case s.Password != "":
			nonEmptyParam = "password"
		case s.Host != "":
			nonEmptyParam = "host"
		case s.Port != 0:
			nonEmptyParam = "port"
		}

		if nonEmptyParam != "" {
			return nil, errors.Errorf("URL is mutually exclusive with the %s", nonEmptyParam)
		}

		// Parse URL.
		opts, err := pg.ParseURL(s.URL)
		if err != nil {
			return nil, errors.Wrap(err, "invalid database URL")
		}

		// Parse host and port.
		host, portRaw, ok := strings.Cut(opts.Addr, ":")
		if !ok {
			// The pg.ParseURL always appends the port if it's missing.
			return nil, errors.Errorf("Unknown address format: '%s'", opts.Addr)
		}
		port, err := strconv.ParseInt(portRaw, 10, 0)
		if err != nil {
			return nil, errors.Wrapf(err, "Invalid port: '%s'", portRaw)
		}

		// Set parameters.
		settings.DBName = opts.Database
		settings.Host = host
		settings.Port = int(port)
		settings.Password = opts.Password
		settings.User = opts.User

		// The sslmode parameter is supported by the pg library but it's not permitted.
		// The sslmode must be provided using the dedicated flags because
		// the created TLS object is incomplete, and the exact SSL mode
		// value is lost.
	}

	return settings, nil
}

// Returns the CLI flag definitions as objects. This function is dedicated to
// avoiding parsing the struct tags outside the module.
func (s *DatabaseCLIFlags) ConvertToCLIFlagDefinitions() []*CLIFlagDefinition {
	return convertToCLIFlagDefinitions(s)
}

// Reads the database settings (without maintenance) from the environment variables.
func (s *DatabaseCLIFlags) ReadFromEnvironment() {
	readFromEnvironment(s)
}

// Reads the database settings (without maintenance) from the CLI lookup.
func (s *DatabaseCLIFlags) ReadFromCLI(lookup CLILookup) {
	readFromCLI(s, lookup)
}

// The database CLI flags are extended with the maintenance credentials.
// The maintenance access should be used to perform operations outside the
// standard database as creating or removing databases and users, creating
// extensions, or granting privileges.
type DatabaseCLIFlagsWithMaintenance struct {
	DatabaseCLIFlags
	MaintenanceDBName   string `short:"m" long:"db-maintenance-name" description:"The existing maintenance database name" env:"STORK_DATABASE_MAINTENANCE_NAME" default:"postgres"`
	MaintenanceUser     string `short:"a" long:"db-maintenance-user" description:"The Postgres database administrator user name" env:"STORK_DATABASE_MAINTENANCE_USER_NAME" default:"postgres"`
	MaintenancePassword string `long:"db-maintenance-password" description:"The Postgres database administrator password; if not specified, the user will be prompted for the password" env:"STORK_DATABASE_MAINTENANCE_PASSWORD"`
}

// Converts the values of CLI flags to the database settings. They use the
// maintenance parameters. The maintenance user will connect to the maintenance
// database.
func (s *DatabaseCLIFlagsWithMaintenance) ConvertToMaintenanceDatabaseSettings() (*DatabaseSettings, error) {
	settings, err := s.ConvertToDatabaseSettings()
	if err != nil {
		return nil, err
	}

	settings.DBName = s.MaintenanceDBName
	settings.User = s.MaintenanceUser
	settings.Password = s.MaintenancePassword
	return settings, nil
}

// Converts the values of CLI flags to the database settings. They use the
// maintenance credentials. The maintenance user will connect to the standard
// database.
func (s *DatabaseCLIFlagsWithMaintenance) ConvertToDatabaseSettingsAsMaintenance() (*DatabaseSettings, error) {
	settings, err := s.ConvertToMaintenanceDatabaseSettings()
	if err != nil {
		return nil, err
	}
	settings.DBName = s.DBName
	return settings, nil
}

// Returns the CLI flag definitions as objects. This function is dedicated to
// avoiding parsing the struct tags outside the module.
func (s *DatabaseCLIFlagsWithMaintenance) ConvertToCLIFlagDefinitions() []*CLIFlagDefinition {
	return convertToCLIFlagDefinitions(s)
}

// Reads the database settings (with maintenance) from the environment variables.
func (s *DatabaseCLIFlagsWithMaintenance) ReadFromEnvironment() {
	readFromEnvironment(s)
}

// Reads the database settings (with maintenance) from the CLI lookup.
func (s *DatabaseCLIFlagsWithMaintenance) ReadFromCLI(lookup CLILookup) {
	readFromCLI(s, lookup)
}
