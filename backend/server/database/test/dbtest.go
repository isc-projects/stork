package dbtest

import (
	"log"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	dbops "isc.org/stork/server/database"
)

// go-pg specific database connection options.
// Prepares unit test setup by re-creating the database schema and
// returns pointer to the teardown function.
func SetupDatabaseTestCase(t *testing.T) (*dbops.PgDB, *dbops.DatabaseSettings, func()) {
	// Common set of database connection options which may be converted to a string
	// of space separated options used by SQL drivers.
	genericConnOptions := dbops.DatabaseSettings{
		BaseDatabaseSettings: dbops.BaseDatabaseSettings{
			DbName:   "storktest",
			User:     "storktest",
			Password: "storktest",
			Host:     "localhost",
			Port:     5432,
		},
	}

	// Convert generic options to go-pg options.
	pgConnOptions := *genericConnOptions.PgParams()

	// Check if we're running tests in Gitlab CI. If so, the host
	// running the database should be set to "postgres".
	// See https://docs.gitlab.com/ee/ci/services/postgres.html.
	if addr, ok := os.LookupEnv("POSTGRES_ADDR"); ok {
		splitAddr := strings.Split(addr, ":")
		if len(splitAddr) > 0 {
			genericConnOptions.Host = splitAddr[0]
		}
		if len(splitAddr) > 1 {
			if p, err := strconv.Atoi(splitAddr[1]); err == nil {
				genericConnOptions.Port = p
			}
		}
		pgConnOptions.Addr = addr
	}

	// Create an instance of the database using test credentials.
	db, err := dbops.NewPgDbConn(&pgConnOptions)
	if db == nil {
		log.Fatalf("unable to create database instance: %+v", err)
	}
	require.NoError(t, err)

	// enable tracing sql queries if requested
	if _, ok := os.LookupEnv("STORK_DATABASE_TRACE"); ok {
		db.AddQueryHook(dbops.DbLogger{})
	}

	createSchema(t, db)

	return db, &genericConnOptions, func() {
		TossSchema(t, db)
		defer db.Close()
	}
}

// Create the database schema to the latest version.
func createSchema(t *testing.T, db *dbops.PgDB) {
	TossSchema(t, db)
	_, _, err := dbops.Migrate(db, "init")
	require.NoError(t, err)
	_, _, err = dbops.Migrate(db, "up")
	require.NoError(t, err)
}

// Remove the database schema.
func TossSchema(t *testing.T, db *dbops.PgDB) {
	err := dbops.Toss(db)
	require.NoError(t, err)
}
