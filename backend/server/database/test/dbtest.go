package dbtest

import (
	"fmt"
	"math/rand"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	dbops "isc.org/stork/server/database"
)

// go-pg specific database connection options.
// Prepares unit test setup by re-creating the database schema and
// returns pointer to the teardown function. The specified argument
// must be of a *testing.T or *testing.B type.
func SetupDatabaseTestCase(testArg interface{}) (*dbops.PgDB, *dbops.DatabaseSettings, func()) {
	// Default configuration
	flags := &dbops.DatabaseCLIFlagsWithMaintenance{
		DatabaseCLIFlags: dbops.DatabaseCLIFlags{
			DBName: "storktest",
			User:   "storktest",
			Host:   "/var/run/postgresql",
			Port:   5432,
		},
		MaintenanceDBName: "postgres",
		MaintenanceUser:   "postgres",
	}

	flags.ReadFromEnvironment()

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

	// Connect to maintenance database to be able to create test database.
	settings, err := flags.ConvertToMaintenanceDatabaseSettings()
	if err != nil {
		b.Fatalf("Invalid database settings: %+v", err)
	}

	db, err := dbops.NewPgDBConn(settings)
	if db == nil {
		log.Fatalf("Unable to create database instance: %+v", err)
	}
	if t != nil {
		require.NoError(t, err)
	} else if b != nil && err != nil {
		b.Fatalf("%s", err)
	}

	// Create test database from template. Template db is storktest (no tests should use it directly).
	// Test database name is usually storktest + big random number e.g.: storktest9817239871871478571.
	templateDBName := flags.DBName

	dbName := fmt.Sprintf("%s%d", templateDBName, rand.Int63()) //nolint:gosec

	cmd := fmt.Sprintf(`DROP DATABASE IF EXISTS %s;`, dbName)
	_, err = db.Exec(cmd)
	if t != nil {
		require.NoError(t, err)
	} else if b != nil && err != nil {
		b.Fatalf("%s", err)
	}

	cmd = fmt.Sprintf(`CREATE DATABASE %s TEMPLATE %s;`, dbName, templateDBName)
	_, err = db.Exec(cmd)
	if t != nil {
		require.NoError(t, err)
	} else if b != nil && err != nil {
		b.Fatalf("%s", err)
	}

	db.Close()

	// Create an instance of the test database.
	testDBSettings, err := flags.ConvertToDatabaseSettings()
	if err != nil {
		b.Fatalf("Invalid database settings: %+v", err)
	}

	testDBSettings.DBName = dbName

	db, err = dbops.NewPgDBConn(testDBSettings)
	if db == nil {
		log.Fatalf("Unable to connect to the database instance: %+v", err)
	}
	if t != nil {
		require.NoError(t, err)
	} else if b != nil && err != nil {
		b.Fatalf("%s", err)
	}

	return db, testDBSettings, func() {
		db.Close()
	}
}
