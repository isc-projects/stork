// The package provides a set of utilities to parse and convert the CLI flags
// and the environment variables into the database settings. Stork is composed
// of a few binaries that use different CLI libraries. It's essential to
// process the database parameters consistently regardless of how they are
// provided.
package dbops

import (
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
)

// Iterates over the struct fields. It non-recursively iterates over nested
// structures using a breadth-first approach. The provided function is called
// for members that aren't structs. It accepts an object describing the field
// of the struct (it may be used to retrieve the field name or tags) and
// another object representing the corresponding value of the iterated object
// (it may be used to read or modify the field value). If the field value is
// nil, the value object passed to the callback function will be marked invalid.
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

	// Perform until the queue is exhausted.
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

			// Not call the callback for the nested structs.
			continue
		}

		// Call the callback for the leaf fields.
		f(pair.field, pair.value)
	}
}

// Helper function that generalizes setting the value of the members based on
// the keys in the member tags and an external value lookup.
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
		case reflect.Int64:
			// Is it time.Duration?
			if field.Type.AssignableTo(reflect.TypeOf(time.Duration(0))) {
				duration, err := time.ParseDuration(value)
				if err != nil {
					return
				}
				valueField.SetInt(int64(duration))
			}
			// If it's not time.Duration, it's a regular int64. Skip as
			// it's not supported.
		case reflect.String:
			valueField.SetString(value)
		case reflect.Int:
			envValueInt, err := strconv.ParseInt(value, 10, 0)
			if err != nil {
				return
			}
			valueField.SetInt(envValueInt)
		default:
			// Skip an unsupported field.
		}
	})
}

// Reads the member values from the environment variables. The related
// environment variable name is read from the 'env' struct tag.
func readFromEnvironment(obj any) {
	setFieldsBasedOnTags(obj, "env", os.LookupEnv)
}

// The lookup object to read the CLI values from the external source.
type CLILookup interface {
	IsSet(key string) bool
	String(key string) string
}

// Reads the member value from the CLI flags using the external CLI lookup.
// The flag names are read from the 'long' struct tag.
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

		if value, ok := field.Tag.Lookup("short"); ok {
			flag.Short = value
		}

		if value, ok := field.Tag.Lookup("long"); ok {
			flag.Long = value
		}

		if value, ok := field.Tag.Lookup("description"); ok {
			flag.Description = value
		}

		if value, ok := field.Tag.Lookup("env"); ok {
			flag.EnvironmentVariable = value
		}

		if value, ok := field.Tag.Lookup("default"); ok {
			flag.Default = value
		}

		flags = append(flags, &flag)
	})

	return flags
}

// General definition of the CLI flags used to connect to the database.
type DatabaseCLIFlags struct {
	URL          string        `long:"db-url" description:"The URL to locate the Stork PostgreSQL database" env:"STORK_DATABASE_URL"`
	DBName       string        `short:"d" long:"db-name" description:"The name of the database to connect to" env:"STORK_DATABASE_NAME" default:"stork"`
	User         string        `short:"u" long:"db-user" description:"The user name to be used for database connections" env:"STORK_DATABASE_USER_NAME" default:"stork"`
	Password     string        `long:"db-password" description:"The database password to be used for database connections; it is recommended to provide this value using an environment variable or leave it empty to type it in the safe prompt." env:"STORK_DATABASE_PASSWORD"`
	Host         string        `long:"db-host" description:"The host name, IP address or socket where database is available" env:"STORK_DATABASE_HOST" default:""`
	Port         int           `short:"p" long:"db-port" description:"The port on which the database is available" env:"STORK_DATABASE_PORT" default:"5432"`
	SSLMode      string        `long:"db-sslmode" description:"The SSL mode for connecting to the database" choice:"disable" choice:"require" choice:"verify-ca" choice:"verify-full" env:"STORK_DATABASE_SSLMODE" default:"disable"` //nolint:staticcheck
	SSLCert      string        `long:"db-sslcert" description:"The location of the SSL certificate used by the server to connect to the database" env:"STORK_DATABASE_SSLCERT"`
	SSLKey       string        `long:"db-sslkey" description:"The location of the SSL key used by the server to connect to the database" env:"STORK_DATABASE_SSLKEY"`
	SSLRootCert  string        `long:"db-sslrootcert" description:"The location of the root certificate file used to verify the database server's certificate" env:"STORK_DATABASE_SSLROOTCERT"`
	TraceSQL     string        `long:"db-trace-queries" description:"Enable tracing SQL queries: run (only run-time, without migrations), all (migrations and run-time), or none (no query logging)" env:"STORK_DATABASE_TRACE" choice:"run" choice:"all" choice:"none" default:"none"` //nolint:staticcheck
	ReadTimeout  time.Duration `long:"db-read-timeout" description:"Timeout for socket reads. If reached, commands will fail instead of blocking, zero disables the timeout; requires unit: ms (milliseconds), s (seconds), m (minutes), e.g.: 42s" env:"STORK_DATABASE_READ_TIMEOUT" default:"0s"`
	WriteTimeout time.Duration `long:"db-write-timeout" description:"Timeout for socket writes. If reached, commands will fail instead of blocking, zero disables the timeout; requires unit: ms (milliseconds), s (seconds), m (minutes), e.g.: 42s" env:"STORK_DATABASE_WRITE_TIMEOUT" default:"0s"`
}

// Converts the CLI flag values to the database settings object.
// It may parse the access options from the URL but returns an error if it's
// provided simultaneously with the standard parameters.
func (s *DatabaseCLIFlags) ConvertToDatabaseSettings() (*DatabaseSettings, error) {
	settings := &DatabaseSettings{
		DBName:       s.DBName,
		User:         s.User,
		Password:     s.Password,
		Host:         s.Host,
		Port:         s.Port,
		SSLMode:      s.SSLMode,
		SSLCert:      s.SSLCert,
		SSLKey:       s.SSLKey,
		SSLRootCert:  s.SSLRootCert,
		TraceSQL:     newLoggingQueryPreset(s.TraceSQL),
		ReadTimeout:  s.ReadTimeout,
		WriteTimeout: s.WriteTimeout,
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
			return nil, errors.Wrapf(err, "invalid database URL: %s", s.URL)
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

		// The sslmode parameter is supported by the go-pg library but it's not permitted by Stork.
		// The sslmode must be provided using the dedicated flags because
		// the TLS object created by go-pg is incomplete, and the exact SSL mode
		// value is lost.
	}

	return settings, nil
}

// Returns the CLI flag definitions as objects. This function is dedicated to
// avoiding parsing the struct tags outside the module.
func (s *DatabaseCLIFlags) ConvertToCLIFlagDefinitions() []*CLIFlagDefinition {
	return convertToCLIFlagDefinitions(s)
}

// Reads the member values from the environment variables.
func (s *DatabaseCLIFlags) ReadFromEnvironment() {
	readFromEnvironment(s)
}

// Reads the member values from the CLI flags using the external CLI lookup.
func (s *DatabaseCLIFlags) ReadFromCLI(lookup CLILookup) {
	readFromCLI(s, lookup)
}

// The database CLI flags extended with the maintenance credentials.
// The maintenance access should be used to perform operations outside the
// standard database as creating or removing databases and users, creating
// extensions, or granting privileges.
type DatabaseCLIFlagsWithMaintenance struct {
	DatabaseCLIFlags
	MaintenanceDBName   string `short:"m" long:"db-maintenance-name" description:"The existing maintenance database name" env:"STORK_DATABASE_MAINTENANCE_NAME" default:"postgres"`
	MaintenanceUser     string `short:"a" long:"db-maintenance-user" description:"The Postgres database administrator user name" env:"STORK_DATABASE_MAINTENANCE_USER_NAME" default:"postgres"`
	MaintenancePassword string `long:"db-maintenance-password" description:"The Postgres database administrator password; if not specified, the user will be prompted for the password if necessary" env:"STORK_DATABASE_MAINTENANCE_PASSWORD"`
}

// Returns the database settings needed to connect to the maintenance database
// using the maintenance credentials.
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

// Returns the CLI flag definitions as objects. This function is dedicated to
// avoiding parsing the struct tags outside the module.
func (s *DatabaseCLIFlagsWithMaintenance) ConvertToCLIFlagDefinitions() []*CLIFlagDefinition {
	return convertToCLIFlagDefinitions(s)
}

// Reads the member values from the environment variables.
func (s *DatabaseCLIFlagsWithMaintenance) ReadFromEnvironment() {
	readFromEnvironment(s)
}

// Reads the member values from the CLI flags using the external CLI lookup.
func (s *DatabaseCLIFlagsWithMaintenance) ReadFromCLI(lookup CLILookup) {
	readFromCLI(s, lookup)
}
