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
	// Read the DB credentials from the environment variables.
	dbUser := "storktest"
	if envDbUser, ok := os.LookupEnv("STORK_DATABASE_USER_NAME"); ok {
		dbUser = envDbUser
	}

	dbPassword := "storktest"
	if envPass, ok := os.LookupEnv("STORK_DATABASE_PASSWORD"); ok {
		dbPassword = envPass
	}

	mainDbName := "storktest"
	if envMainDbName, ok := os.LookupEnv("STORK_DATABASE_NAME"); ok {
		mainDbName = envMainDbName
	}

	dbHost := "localhost"
	if envDbHost, ok := os.LookupEnv("STORK_DATABASE_HOST"); ok {
		dbHost = envDbHost
	}

	dbPort := 5432
	if envDbPortRaw, ok := os.LookupEnv("STORK_DATABASE_PORT"); ok {
		envDbPort, err := strconv.ParseInt(envDbPortRaw, 10, 32)
		if err == nil {
			dbPort = int(envDbPort)
		}
	}

	dbMaintenanceName := "postgres"
	if envDbMaintenanceName, ok := os.LookupEnv("DB_MAINTENANCE_NAME"); ok {
		dbMaintenanceName = envDbMaintenanceName
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
	dbName := fmt.Sprintf("%s%d", mainDbName, rand.Int63())

	cmd := fmt.Sprintf(`DROP DATABASE IF EXISTS %s;`, dbName)
	_, err = db.Exec(cmd)
	if t != nil {
		require.NoError(t, err)
	} else if b != nil && err != nil {
		b.Fatalf("%s", err)
	}

	cmd = fmt.Sprintf(`CREATE DATABASE %s TEMPLATE %s;`, dbName, mainDbName)
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
