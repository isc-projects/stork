package dbops

import (
	"fmt"
	"strings"

	"github.com/go-pg/pg/v9"
	"github.com/go-pg/pg/v9/orm"
)

// Structure holding database connection options which can be used by SQL drivers.
// It can be converted to a string of space separated parameters.
type GenericConn struct {
	User string
	Password string
	DbName string
	Host string
	Port int
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
func NewGenericConn() *GenericConn {
	conn := &GenericConn{Port: 5432}
	return conn
}

// Returns generic connection parameters as a list of space separated name/value pairs.
func (c GenericConn) ConnectionParams() string {
	s := fmt.Sprintf("%+v", c)
	s = strings.ReplaceAll(s, ":", "=")
	s = strings.Trim(s, "{}")
	s = strings.ToLower(s)
	return s
}

// Converts generic connection parameters to go-pg specific parameters.
func (c GenericConn) PgParams() *PgOptions {
	pgopts := &PgOptions{Database: c.DbName, User: c.User, Password: c.Password}
	pgopts.Addr = fmt.Sprintf("%s:%d", c.Host, c.Port)
	return pgopts
}

