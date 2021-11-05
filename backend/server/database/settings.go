package dbops

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/go-pg/pg/v9"
	"github.com/go-pg/pg/v9/orm"

	storkutil "isc.org/stork/util"
)

// Represents database connection settings. The field names and values must correspond to the
// respective libpq parameters.
// See https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-PARAMKEYWORDS.
type BaseDatabaseSettings struct {
	DBName      string `short:"d" long:"db-name" description:"the name of the database to connect to" env:"STORK_DATABASE_NAME" default:"stork"`
	User        string `short:"u" long:"db-user" description:"the user name to be used for database connections" env:"STORK_DATABASE_USER_NAME" default:"stork"`
	Password    string `description:"the database password to be used for database connections" env:"STORK_DATABASE_PASSWORD"`
	Host        string `long:"db-host" description:"the name of the host where database is available" env:"STORK_DATABASE_HOST" default:"localhost"`
	Port        int    `short:"p" long:"db-port" description:"the port on which the database is available" env:"STORK_DATABASE_PORT" default:"5432"`
	SSLMode     string `long:"db-sslmode" description:"the SSL mode for connecing to the database" choice:"disable" choice:"require" choice:"verify-ca" choice:"verify-full" env:"STORK_DATABASE_SSLMODE" default:"disable"` //nolint:staticcheck
	SSLCert     string `long:"db-sslcert" description:"the location of the SSL certificate used by the server to connect to the database" env:"STORK_DATABASE_SSLCERT"`
	SSLKey      string `long:"db-sslkey" description:"the location of the SSL key used by the server to connect to the database" env:"STORK_DATABASE_SSLKEY"`
	SSLRootCert string `long:"db-sslrootcert" description:"the location of the root certificate file used to verify the database server's certificate" env:"STORK_DATABASE_SSLROOTCERT"`
}

type DatabaseSettings struct {
	BaseDatabaseSettings
	TraceSQL string `long:"db-trace-queries" description:"enable tracing SQL queries: run (only run-time, without migrations), all (migrations and run-time), all is the default and covers both migrations and run-time." env:"STORK_DATABASE_TRACE" optional:"true" optional-value:"all"`
}

// Alias to pg.DB.
type PgDB = pg.DB

// Alias to pg.Conn.
type PgConn = pg.Conn

// Alias to pg.Options.
type PgOptions = pg.Options

// Enables singular SQL table names for go-pg ORM.
func init() {
	orm.SetTableNameInflector(func(s string) string {
		return s
	})
}

// Creates new generic connection structure and sets the port to the default
// port number used by PostgreSQL.
func NewDatabaseSettings() *DatabaseSettings {
	conn := &DatabaseSettings{BaseDatabaseSettings: BaseDatabaseSettings{Port: 5432}}
	return conn
}

// Returns generic connection parameters as a list of space separated name/value pairs.
// All string values are enclosed in quotes. The quotes and double quotes within the
// string values are escaped. Empty or zero values are not included in the returned
// connection string.
func (c *BaseDatabaseSettings) ConnectionParams() string {
	// Copy the structure as we don't want to modify the original.
	settingsCopy := *c

	// Get the reflect representation of the structure.
	v := reflect.ValueOf(&settingsCopy).Elem()

	// Get the types of the fields in the structure.
	vType := v.Type()

	// Iterate over the fields and append them to the connection string if needed.
	var s string
	for i := 0; i < v.NumField(); i++ {
		// Check the type of the current field.
		switch vType.Field(i).Type.Kind() {
		case reflect.String:
			// Only append the parameter if it is non-empty.
			fieldValue := v.Field(i).String()
			if len(fieldValue) == 0 {
				continue
			}
			// Escape quotes and double quotes.
			fieldValue = strings.ReplaceAll(fieldValue, "'", `\'`)
			fieldValue = strings.ReplaceAll(fieldValue, `"`, `\"`)
			// Enclose all strings in quotes in case they contain spaces.
			fieldValue = fmt.Sprintf("'%s'", fieldValue)
			v.Field(i).SetString(fieldValue)
		case reflect.Int:
			// If the int value is zero, do not include it.
			fieldValue := v.Field(i).Int()
			if fieldValue == 0 {
				continue
			}
		default:
		}
		// If we are not on the first field, add a space after previous field.
		if i > 0 {
			s += " "
		}
		// Append the parameter in the name=value format.
		s += fmt.Sprintf("%s=%v", strings.ToLower(vType.Field(i).Name), v.Field(i).Interface())
	}
	if len(c.SSLMode) == 0 {
		s += " sslmode='disable'"
	}
	return s
}

// Converts generic connection parameters to go-pg specific parameters.
func (c *DatabaseSettings) PgParams() (*PgOptions, error) {
	pgopts := &PgOptions{Database: c.DBName, User: c.User, Password: c.Password}
	pgopts.Addr = fmt.Sprintf("%s:%d", c.Host, c.Port)
	tlsConfig, err := GetTLSConfig(c.SSLMode, c.Host, c.SSLCert, c.SSLKey, c.SSLRootCert)
	if err != nil {
		return nil, err
	}
	pgopts.TLSConfig = tlsConfig
	return pgopts, nil
}

// Fetches database password from the environment variable or prompts the user
// for the password.
func Password(settings *DatabaseSettings) {
	if passwd, ok := os.LookupEnv("STORK_DATABASE_PASSWORD"); ok {
		settings.Password = passwd
	} else {
		// Prompt the user for database password.
		pass := storkutil.GetSecretInTerminal("database password: ")
		settings.Password = pass
	}
}

// Parse DB URL to Pg options.
func ParseURL(url string) (*pg.Options, error) {
	return pg.ParseURL(url)
}
