package dbtest

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
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
	// Default configuration
	dbUser := "storktest"
	dbPassword := dbUser
	mainDBName := dbUser
	dbHost := "localhost"
	dbPortRaw := "5432"
	dbMaintenanceName := "postgres"

	// Read the DB credentials from the environment variables.
	envMapping := map[string]*string{
		"STORK_DATABASE_USER_NAME": &dbUser,
		"STORK_DATABASE_PASSWORD":  &dbPassword,
		"STORK_DATABASE_NAME":      &mainDBName,
		"STORK_DATABASE_HOST":      &dbHost,
		"STORK_DATABASE_PORT":      &dbPortRaw,
		"DB_MAINTENANCE_NAME":      &dbMaintenanceName,
	}

	for key, value := range envMapping {
		if envValue, ok := os.LookupEnv(key); ok {
			*value = envValue
		}
	}

	dbPort := 0
	if parsedDBPort, err := strconv.ParseInt(dbPortRaw, 10, 32); err == nil {
		dbPort = int(parsedDBPort)
	}

	// Common set of database connection options which may be converted to a string
	// of space separated options used by SQL drivers.
	genericConnOptions := dbops.DatabaseSettings{
		BaseDatabaseSettings: dbops.BaseDatabaseSettings{
			DBName:   dbMaintenanceName,
			User:     dbUser,
			Password: dbPassword,
			Host:     dbHost,
			Port:     dbPort,
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

	// Connect to maintenance database to be able to create test database.
	pgConnOptions.Database = dbMaintenanceName
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
	dbName := fmt.Sprintf("%s%d", mainDBName, rand.Int63())

	cmd := fmt.Sprintf(`DROP DATABASE IF EXISTS %s;`, dbName)
	_, err = db.Exec(cmd)
	if t != nil {
		require.NoError(t, err)
	} else if b != nil && err != nil {
		b.Fatalf("%s", err)
	}

	cmd = fmt.Sprintf(`CREATE DATABASE %s TEMPLATE %s;`, dbName, mainDBName)
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
