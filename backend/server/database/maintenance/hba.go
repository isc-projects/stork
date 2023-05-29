package maintenance

import (
	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
)

type PgAuthMethod string

const (
	PgAuthMethodTrust          PgAuthMethod = "trust"
	PgAuthMethodPeer           PgAuthMethod = "peer"
	PgAuthMethodIdentityServer PgAuthMethod = "ident"
	PgAuthMethodMD5            PgAuthMethod = "md5"
	PgAuthMethodScramSHA256    PgAuthMethod = "scram-sha-256"
	// More: https://www.postgresql.org/docs/current/auth-methods.html
)

type PgConnectionType string

const (
	PgConnectionLocal = "local"
	PgConnectionHost  = "host"
)

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

// Get the loaded PgHBA configuration entries.
func GetPgHBAConfiguration(dbi pg.DBI) (entries []*PgHBAEntry, err error) {
	err = dbi.Model(&entries).Select()
	err = errors.Wrapf(err, "cannot fetch PgHBA configuration entries")
	return
}
