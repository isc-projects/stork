package dbtest

import (
	"fmt"
	"math/rand"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	dbops "isc.org/stork/server/database"
	"isc.org/stork/server/database/maintenance"
)

// Helper function to perform an error assertion.
// It supports the testing (testing.T) and benchmark (testing.B) objects.
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

// Creates unit test setup by re-creating the database schema and returns the
// settings to connect to the created database as standard and maintenance user.
func createDatabaseTestCase() (settings *dbops.DatabaseSettings, maintenanceSettings *dbops.DatabaseSettings, err error) {
	// Default configuration
	flags := &dbops.DatabaseCLIFlagsWithMaintenance{
		DatabaseCLIFlags: dbops.DatabaseCLIFlags{
			DBName: "storktest",
			User:   "storktest",
			Host:   "", // Use default.
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
			WithField("host", maintenanceSettings.Host).
			WithField("database", maintenanceSettings.DBName).
			WithField("user", maintenanceSettings.User).
			WithError(err).
			Fatalf("Unable to create database instance")
	}
	if nil != err {
		return
	}

	defer db.Close()

	// Create test database from template. Template db is storktest (no tests should use it directly).
	// Test database name is usually storktest + big random number e.g.: storktest9817239871871478571.
	templateDBName := flags.DBName

	if flags.MaintenanceDBName == templateDBName {
		log.Warn("The maintenance database should not be the same as the " +
			"template database; otherwise, the source database may report " +
			"that other users are accessing it.")
	}

	dbName := fmt.Sprintf("%s%d", templateDBName, rand.Int63()) //nolint:gosec

	if err = maintenance.DropDatabaseIfExists(db, dbName); err != nil {
		return
	}

	if _, err = maintenance.CreateDatabaseFromTemplate(db, dbName, templateDBName); err != nil {
		return
	}

	// Create the database settings with a standard user credentials.
	settings, err = flags.ConvertToDatabaseSettings()
	if err != nil {
		return
	}

	settings.DBName = dbName
	maintenanceSettings.DBName = dbName

	return settings, maintenanceSettings, nil
}

// Returns a database connection object and teardown function.
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
func SetupDatabaseTestCaseWithMaintenanceCredentials(testArg interface{}) (*dbops.PgDB, *dbops.DatabaseSettings, func()) {
	_, settings, err := createDatabaseTestCase()
	failOnError(testArg, err)
	db, teardown, err := prepareDBInstance(settings)
	failOnError(testArg, err)
	return db, settings, teardown
}
