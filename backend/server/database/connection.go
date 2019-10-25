package dbops

import (
	"fmt"
	"strings"

	"github.com/go-pg/pg/v9"
	"github.com/go-pg/pg/v9/orm"
)

type GenericConn struct {
	User string
	Password string
	DbName string
	Host string
	Port int
}

// Alias to pg.Options.
type PgOptions = pg.Options

func init() {
    orm.SetTableNameInflector(func(s string) string {
        return s
    })
}

func NewGenericConn() *GenericConn {
	conn := &GenericConn{Port: 5432}
	return conn
}

func (c GenericConn) ConnectionParams() string {
	s := fmt.Sprintf("%+v", c)
	s = strings.ReplaceAll(s, ":", "=")
	s = strings.Trim(s, "{}")
	s = strings.ToLower(s)
	return s
}

func (c GenericConn) PgParams() *PgOptions {
	pgopts := &PgOptions{Database: c.DbName, User: c.User, Password: c.Password}
	pgopts.Addr = fmt.Sprintf("%s:%d", c.Host, c.Port)
	return pgopts
}

