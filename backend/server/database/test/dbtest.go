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

// Returns credentials to the test database. The default values can be
// overridden by setting the corresponding environment variables.
// Returns standard and maintenance user credentials. Both point to the default
// test database.
func getDatabaseTestSettings() (settings *dbops.DatabaseSettings, maintenanceSettings *dbops.DatabaseSettings, err error) {
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

	maintenanceSettings, err = flags.ConvertToMaintenanceDatabaseSettings()
	if err != nil {
		return
	}

	// Create the database settings with a standard user credentials.
	settings, err = flags.ConvertToDatabaseSettings()
	if err != nil {
		return
	}

	return settings, maintenanceSettings, nil
}

// Creates unit test setup by re-creating the database schema and returns the
// settings to connect to the created database as standard and maintenance user.
func createDatabaseTestCase() (settings *dbops.DatabaseSettings, maintenanceSettings *dbops.DatabaseSettings, err error) {
	settings, maintenanceSettings, err = getDatabaseTestSettings()
	if err != nil {
		return
	}

	// Connect to maintenance database to be able to create test database.
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
	templateDBName := settings.DBName

	if maintenanceSettings.DBName == templateDBName {
		log.Warn("The maintenance database should not be the same as the " +
			"template database; otherwise, the source database may report " +
			"that other users are accessing it.")
	}

	dbName := fmt.Sprintf("%s%d", templateDBName, rand.Int63()) //nolint:gosec

	if err = maintenance.DropDatabaseIfExists(db, dbName); err != nil {
		return
	}

	// Create database from template is a faster method than creating an empty
	// database and running all migrations. Creating from template took 132.87s,
	// while creating empty database and running migrations took 201.70s.
	// However. it is less convenient, because the template database must exist
	// and be in the expected version.
	if _, err = maintenance.CreateDatabaseFromTemplate(db, dbName, templateDBName); err != nil {
		return
	}

	settings.DBName = dbName
	maintenanceSettings.DBName = dbName

	return settings, maintenanceSettings, nil
}

// Restores unit test setup from the database backup, migrates it to the latest
// schema version and returns the settings to connect to the created database
// as standard and maintenance user.
func restoreDatabaseTestCase(backupFilePath string) (settings *dbops.DatabaseSettings, maintenanceSettings *dbops.DatabaseSettings, err error) {
	settings, maintenanceSettings, err = getDatabaseTestSettings()
	if err != nil {
		return settings, maintenanceSettings, err
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
		return settings, maintenanceSettings, err
	}

	defer db.Close()

	dbName := fmt.Sprintf("%s%d", settings.DBName, rand.Int63()) //nolint:gosec

	if err = maintenance.DropDatabaseIfExists(db, dbName); err != nil {
		return settings, maintenanceSettings, err
	}

	// Create empty database.
	if _, err = maintenance.CreateDatabase(db, dbName); err != nil {
		return settings, maintenanceSettings, err
	}

	db.Close()

	settings.DBName = dbName
	maintenanceSettings.DBName = dbName

	// Reconnect to the newly created database as a maintenance user.
	db, err = dbops.NewPgDBConn(maintenanceSettings)
	if err != nil {
		return settings, maintenanceSettings, err
	}
	defer db.Close()

	// Create extensions.
	if err = dbops.CreatePgCryptoExtension(db); err != nil {
		return settings, maintenanceSettings, err
	}

	// Grant all privileges on the database to the standard user.
	if err = maintenance.GrantAllPrivilegesOnDatabaseToUser(db, dbName, settings.User); err != nil {
		return nil, nil, err
	}

	if err = maintenance.GrantAllPrivilegesOnSchemaToUser(db, "public", settings.User); err != nil {
		return nil, nil, err
	}

	db.Close()

	// Reconnect to the newly created database as a standard user and restore
	// the backup and migrate to the latest schema.
	// It is important to do it as a standard user to ensure that all required
	// privileges are granted.
	db, err = dbops.NewPgDBConn(settings)
	if err != nil {
		return settings, maintenanceSettings, err
	}
	defer db.Close()

	// Restore database from the backup file.
	if err = maintenance.RestoreDatabaseFromDump(db, backupFilePath); err != nil {
		return settings, maintenanceSettings, err
	}

	// Close the session and reconnect to reset all temporary changes in the
	// database engine made by the restore process. Otherwise, the migration
	// fails due to unknown default schema ("no schema has been selected to
	// create in").
	db.Close()
	db, err = dbops.NewPgDBConn(settings)
	if err != nil {
		return settings, maintenanceSettings, err
	}
	defer db.Close()

	// Migrate database to the latest schema version.
	_, _, err = dbops.MigrateToLatest(db)
	if err != nil {
		return settings, maintenanceSettings, err
	}

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

// Prepares unit test setup by restoring the database from a dump file and
// returns pointer to the teardown function. The specified argument
// must be of a *testing.T or *testing.B type.
func SetupDatabaseTestCaseFromDump(testArg interface{}, backupFilePath string) (*dbops.PgDB, *dbops.DatabaseSettings, func()) {
	settings, _, err := restoreDatabaseTestCase(backupFilePath)
	failOnError(testArg, err)
	db, teardown, err := prepareDBInstance(settings)
	failOnError(testArg, err)
	return db, settings, teardown
}
