package maintenance

import (
	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
)

// Enum type for the authentication method in the pg_hba.conf file.
type PgAuthMethod string

// Main authentication methods supported by Postgres.
const (
	PgAuthMethodTrust          PgAuthMethod = "trust"
	PgAuthMethodPeer           PgAuthMethod = "peer"
	PgAuthMethodIdentityServer PgAuthMethod = "ident"
	PgAuthMethodMD5            PgAuthMethod = "md5"
	PgAuthMethodScramSHA256    PgAuthMethod = "scram-sha-256"
	// More: https://www.postgresql.org/docs/current/auth-methods.html
)

// Enum type for the connection type in the pg_hba.conf file.
type PgConnectionType string

// Possible values of the connection type.
const (
	PgConnectionLocal PgConnectionType = "local"
	PgConnectionHost  PgConnectionType = "host"
)

// Representation of the single pg_hba.conf rule.
type PgHBAEntry struct {
	tableName  struct{} `pg:"pg_hba_file_rules"` //nolint:unused
	LineNumber int64
	Type       PgConnectionType
	Database   []string `pg:",array"`
	UserName   []string `pg:",array"`
	Address    string
	Netmask    string
	AuthMethod PgAuthMethod
	Options    []string `pg:",array"`
	Error      string
}

// Get the loaded PgHBA configuration entries. It reads the rules loaded by the
// database. It doesn't read the pg_hba.conf file.
func GetPgHBAConfiguration(dbi pg.DBI) (entries []*PgHBAEntry, err error) {
	err = dbi.Model(&entries).Select()
	err = errors.Wrapf(err, "cannot fetch PgHBA configuration entries")
	return
}
