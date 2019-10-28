package dbtest

import(
	"os"
	"isc.org/stork/server/database"
	"isc.org/stork/server/database/migrations"
)

// Common set of database connection options which may be converted to a string
// of space separated options used by SQL drivers.
var GenericConnOptions = dbops.GenericConn{
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

// Reset the database schema to the latest version and remove any data added by tests.
func ResetSchema() {
	dbmigs.ResetToLatest(&PgConnOptions)
}

// Remove the database schema.
func TossSchema() {
	dbmigs.Toss(&PgConnOptions)
}
