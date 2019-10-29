package dbops

import (
	"fmt"
	"log"
	"os"
	"strings"
	"github.com/go-pg/pg/v9"
	"github.com/go-pg/pg/v9/orm"
	"golang.org/x/crypto/ssh/terminal"
)

type DatabaseSettings struct {
	DbName    string `short:"d" long:"db-name" description:"the name of the database to connect to" env:"STORK_DATABASE_NAME" default:"stork"`
	User      string `short:"u" long:"db-user" description:"the user name to be used for database connections" env:"STORK_DATABASE_USER_NAME" default:"stork"`
	Password  string `description:"the database password to be used for database connections" env:"STORK_DATABASE_PASSWORD"`
	Host      string `short:"h" long:"db-host" description:"the name of the host where database is available" env:"STORK_DATABASE_HOST" default:"localhost"`
	Port      int `short:"p" long:"db-port" description:"the port on which the database is available" env:"STORK_DATABASE_PORT" default:"5432"`
}

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
	conn := &DatabaseSettings{Port: 5432}
	return conn
}

// Returns generic connection parameters as a list of space separated name/value pairs.
func (c *DatabaseSettings) ConnectionParams() string {
	s := fmt.Sprintf("%+v", *c)
	s = strings.ReplaceAll(s, ":", "=")
	s = strings.Trim(s, "{}")
	s = strings.ToLower(s)
	return s
}

// Converts generic connection parameters to go-pg specific parameters.
func (c *DatabaseSettings) PgParams() *PgOptions {
	pgopts := &PgOptions{Database: c.DbName, User: c.User, Password: c.Password}
	pgopts.Addr = fmt.Sprintf("%s:%d", c.Host, c.Port)
	return pgopts
}

// Fetches database password from the environment variable or prompts the user
// for the password.
func Password(settings *DatabaseSettings) {
	settings.Password = os.Getenv("STORK_DATABASE_PASSWORD")
	if len(settings.Password) == 0 {
		// Prompt the user for database password.
		fmt.Printf("database password: ")
		pass, err := terminal.ReadPassword(0)
		fmt.Print("\n")

		if err != nil {
			log.Fatal(err.Error())
		}

		settings.Password = string(pass)
	}
}
