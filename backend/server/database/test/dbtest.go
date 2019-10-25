package dbtest

import(
	"os"
	"isc.org/stork/server/database"
	"isc.org/stork/server/database/migrations"
)

var GenericConnOptions = dbops.GenericConn{
	DbName: "storktest",
	User: "storktest",
	Password: "storktest",
	Host: "localhost",
	Port: 5432,
}

var PgConnOptions dbops.PgOptions


func init() {
	PgConnOptions = *GenericConnOptions.PgParams()

	// Check if we're running tests in Gitlab CI. If so, the host
	// running the database should be set to "postgres".
	// See https://docs.gitlab.com/ee/ci/services/postgres.html.
	if _, ok := os.LookupEnv("POSTGRES_DB"); ok {
		GenericConnOptions.Host = "postgres"
		PgConnOptions.Addr = "postgres:5432"
	}
}

func RecreateSchema() {
	// Toss the schema, including removal of the versioning table.
	dbmigs.Toss(&PgConnOptions)
	dbmigs.Migrate(&PgConnOptions, "init")
	dbmigs.Migrate(&PgConnOptions, "up")
}

func TossSchema() {
	dbmigs.Toss(&PgConnOptions)
}
