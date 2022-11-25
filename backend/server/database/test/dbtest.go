package dbtest

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

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
	settings := &dbops.DatabaseCLIFlagsWithMaintenance{
		DatabaseCLIFlags: dbops.DatabaseCLIFlags{
			DBName: "storktest",
			User:   "storktest",
			Host:   "/var/run/postgres",
		},
		MaintenanceDBName:   "storktest",
		MaintenanceUser:     "storktest",
		MaintenancePassword: "storktest",
	}

	settings.ReadFromEnvironment()

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
	db, err := dbops.NewPgDBConn(settings.ConvertToMaintenanceDatabaseSettings())
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
	rand.Seed(time.Now().UnixNano())
	//nolint:gosec
	dbName := fmt.Sprintf("%s%d", settings.DBName, rand.Int63())

	cmd := fmt.Sprintf(`DROP DATABASE IF EXISTS %s;`, dbName)
	_, err = db.Exec(cmd)
	if t != nil {
		require.NoError(t, err)
	} else if b != nil && err != nil {
		b.Fatalf("%s", err)
	}

	cmd = fmt.Sprintf(`CREATE DATABASE %s TEMPLATE %s;`, dbName, settings.DBName)
	_, err = db.Exec(cmd)
	if t != nil {
		require.NoError(t, err)
	} else if b != nil && err != nil {
		b.Fatalf("%s", err)
	}

	db.Close()

	// Create an instance of the test database.
	testDBSettings := settings.ConvertToDatabaseSettings()
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
