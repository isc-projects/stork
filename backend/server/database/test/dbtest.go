package dbtest

import (
	"fmt"
	"math/rand"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	dbops "isc.org/stork/server/database"
)

func failOnError(testArg interface{}, err error) {
	if t, ok := (testArg).(*testing.T); ok {
		require.NoError(t, err)
	} else if b, ok := (testArg).(*testing.B); ok {
		if err != nil {
			b.Fatalf("%+v", err)
		}
	} else {
		panic("Specified test parameter must have type *testing.T or *testing.B")
	}
}

// Prepares unit test setup by re-creating the database schema and
// returns pointer to the teardown function.
func createDatabaseTestCase() (settings *dbops.DatabaseSettings, maintenanceSettings *dbops.DatabaseSettings, err error) {
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

	// Connect to maintenance database to be able to create test database.
	maintenanceSettings, err = flags.ConvertToMaintenanceDatabaseSettings()
	if err != nil {
		return
	}

	db, err := dbops.NewPgDBConn(maintenanceSettings)
	if db == nil {
		log.
			WithField("database", maintenanceSettings.DBName).
			WithField("user", maintenanceSettings.User).
			Fatalf("Unable to create database instance: %+v", err)
	}
	if nil != err {
		return
	}

	defer db.Close()

	// Create test database from template. Template db is storktest (no tests should use it directly).
	// Test database name is usually storktest + big random number e.g.: storktest9817239871871478571.
	templateDBName := flags.DBName

	dbName := fmt.Sprintf("%s%d", templateDBName, rand.Int63()) //nolint:gosec

	cmd := fmt.Sprintf(`DROP DATABASE IF EXISTS %s;`, dbName)
	_, err = db.Exec(cmd)
	if err != nil {
		return
	}

	cmd = fmt.Sprintf(`CREATE DATABASE %s TEMPLATE %s;`, dbName, templateDBName)
	_, err = db.Exec(cmd)
	if err != nil {
		return
	}

	// Create an instance of the test database.
	settings, err = flags.ConvertToDatabaseSettings()
	if err != nil {
		return
	}

	settings.DBName = dbName
	maintenanceSettings.DBName = dbName

	return settings, maintenanceSettings, nil
}

func prepareDBInstance(settings *dbops.DatabaseSettings) (*dbops.PgDB, func(), error) {
	db, err := dbops.NewPgDBConn(settings)
	if err != nil {
		return nil, nil, err
	}

	return db, func() {
		db.Close()
	}, nil
}

// Prepares unit test setup by re-creating the database schema and
// returns pointer to the teardown function. The specified argument
// must be of a *testing.T or *testing.B type.
func SetupDatabaseTestCase(testArg interface{}) (*dbops.PgDB, *dbops.DatabaseSettings, func()) {
	settings, _, err := createDatabaseTestCase()
	failOnError(testArg, err)
	db, teardown, err := prepareDBInstance(settings)
	failOnError(testArg, err)
	return db, settings, teardown
}

// Prepares unit test setup by re-creating the database schema and
// returns pointer to the teardown function. The specified argument
// must be of a *testing.T or *testing.B type. The database uses the maintenance
// credentials.
func SetupDatabaseTestCaseAsMaintenance(testArg interface{}) (*dbops.PgDB, *dbops.DatabaseSettings, func()) {
	_, settings, err := createDatabaseTestCase()
	failOnError(testArg, err)
	db, teardown, err := prepareDBInstance(settings)
	failOnError(testArg, err)
	return db, settings, teardown
}
