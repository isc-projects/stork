package dbtest

import(
	"os"
	"testing"
	"github.com/stretchr/testify/require"
	"isc.org/stork/server/database"
	"isc.org/stork/server/database/migrations"
)

// Common set of database connection options which may be converted to a string
// of space separated options used by SQL drivers.
var GenericConnOptions = dbops.DatabaseSettings{
	DbName: "storktest",
	User: "storktest",
	Password: "storktest",
	Host: "localhost",
	Port: 5432,
}

// go-pg specific database connection options.
var PgConnOptions dbops.PgOptions

func init() {
	// Convert generic options to go-pg options.
	PgConnOptions = *GenericConnOptions.PgParams()

	// Check if we're running tests in Gitlab CI. If so, the host
	// running the database should be set to "postgres".
	// See https://docs.gitlab.com/ee/ci/services/postgres.html.
	if _, ok := os.LookupEnv("POSTGRES_DB"); ok {
		GenericConnOptions.Host = "postgres"
		PgConnOptions.Addr = "postgres:5432"
	}
}

func SetupDatabaseTestCase(t *testing.T) func (t *testing.T) {
	CreateSchema(t)
	return func (t *testing.T) {
		TossSchema(t)
	}
}

// Create the database schema to the latest version.
func CreateSchema(t *testing.T) {
	TossSchema(t)
	_, _, err := dbmigs.Migrate(&PgConnOptions, "init")
	require.NoError(t, err)
	_, _, err = dbmigs.Migrate(&PgConnOptions, "up")
	require.NoError(t, err)
}

// Remove the database schema.
func TossSchema(t * testing.T) {
	_ = dbmigs.Toss(&PgConnOptions)
}
