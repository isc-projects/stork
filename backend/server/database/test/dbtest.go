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
// returns pointer to the teardown function. The specified argument
// must be of a *testing.T or *testing.B type.
func SetupDatabaseTestCase(testArg interface{}) (*dbops.PgDB, *dbops.DatabaseSettings, func()) {
	// Default Postgres server password.
	pgPass := "storktest"
	// Check if user wants to use a different one.
	if envPass, ok := os.LookupEnv("STORK_DATABASE_PASSWORD"); ok {
		pgPass = envPass
	}
	// Common set of database connection options which may be converted to a string
	// of space separated options used by SQL drivers.
	genericConnOptions := dbops.DatabaseSettings{
		BaseDatabaseSettings: dbops.BaseDatabaseSettings{
			DBName:   "storktest",
			User:     "storktest",
			Password: pgPass,
			Host:     "localhost",
			Port:     5432,
		},
	}

	var (
		t  *testing.T
		b  *testing.B
		ok bool
	)
	t, ok = (testArg).(*testing.T)
	if !ok {
		b, ok = (testArg).(*testing.B)
		if !ok {
			panic("Specified test parameter must have type *testing.T or *testing.B")
		}
	}

	// Convert generic options to go-pg options.
	pgConnOptions, _ := genericConnOptions.PgParams()

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
	db, err := dbops.NewPgDBConn(pgConnOptions, false)
	if db == nil {
		log.Fatalf("Unable to create database instance: %+v", err)
	}
	if t != nil {
		require.NoError(t, err)
	} else if b != nil && err != nil {
		b.Fatalf("%s", err)
	}

	// Create test database from template. Template db is storktest (no tests should use it directly).
	// Test database name is storktest + big random number e.g.: storktest9817239871871478571.
	rand.Seed(time.Now().UnixNano())
	//nolint:gosec
	dbName := fmt.Sprintf("storktest%d", rand.Int63())

	cmd := fmt.Sprintf(`DROP DATABASE IF EXISTS %s;`, dbName)
	_, err = db.Exec(cmd)
	if t != nil {
		require.NoError(t, err)
	} else if b != nil && err != nil {
		b.Fatalf("%s", err)
	}

	cmd = fmt.Sprintf(`CREATE DATABASE %s TEMPLATE storktest;`, dbName)
	_, err = db.Exec(cmd)
	if t != nil {
		require.NoError(t, err)
	} else if b != nil && err != nil {
		b.Fatalf("%s", err)
	}

	db.Close()

	// Create an instance of the test database.
	pgConnOptions.Database = dbName
	genericConnOptions.BaseDatabaseSettings.DBName = dbName

	db, err = dbops.NewPgDBConn(pgConnOptions, false)
	if db == nil {
		log.Fatalf("Unable to create database instance: %+v", err)
	}
	if t != nil {
		require.NoError(t, err)
	} else if b != nil && err != nil {
		b.Fatalf("%s", err)
	}

	// enable tracing sql queries if requested
	if _, ok := os.LookupEnv("STORK_DATABASE_TRACE"); ok {
		db.AddQueryHook(dbops.DBLogger{})
	}

	return db, &genericConnOptions, func() {
		db.Close()
	}
}
