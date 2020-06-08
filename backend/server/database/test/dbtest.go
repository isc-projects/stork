package dbtest

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

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

	// Connect to base `postgres` database to be able to create test database.
	pgConnOptions.Database = "postgres"
	db, err := dbops.NewPgDbConn(&pgConnOptions)
	if db == nil {
		log.Fatalf("unable to create database instance: %+v", err)
	}
	require.NoError(t, err)

	// Create test database from template. Template db is storktest (no tests should use it directly).
	// Test database name is storktest + big random number e.g.: storktest9817239871871478571.
	rand.Seed(time.Now().UnixNano())
	dbName := fmt.Sprintf("storktest%d", rand.Int63())

	cmd := fmt.Sprintf(`DROP DATABASE IF EXISTS %s;`, dbName)
	_, err = db.Exec(cmd)
	require.NoError(t, err)

	cmd = fmt.Sprintf(`CREATE DATABASE %s TEMPLATE storktest;`, dbName)
	_, err = db.Exec(cmd)
	require.NoError(t, err)

	db.Close()

	// Create an instance of the test database.
	pgConnOptions.Database = dbName
	genericConnOptions.BaseDatabaseSettings.DbName = dbName

	db, err = dbops.NewPgDbConn(&pgConnOptions)
	if db == nil {
		log.Fatalf("unable to create database instance: %+v", err)
	}
	require.NoError(t, err)

	// enable tracing sql queries if requested
	if _, ok := os.LookupEnv("STORK_DATABASE_TRACE"); ok {
		db.AddQueryHook(dbops.DbLogger{})
	}

	return db, &genericConnOptions, func() {
		db.Close()
	}
}
