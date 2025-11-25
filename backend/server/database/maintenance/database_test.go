package maintenance_test

import (
	"fmt"
	"testing"

	"github.com/go-pg/pg/v10"
	"github.com/stretchr/testify/require"
	dbops "isc.org/stork/server/database"
	"isc.org/stork/server/database/maintenance"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/testutil"
)

// Test that the database is created properly.
func TestCreateDatabase(t *testing.T) {
	// Arrange
	db, settings, teardown := dbtest.SetupDatabaseTestCaseWithMaintenanceCredentials(t)
	defer teardown()
	databaseName := fmt.Sprintf("%s_create", settings.DBName)

	// Act
	created, err := maintenance.CreateDatabase(db, databaseName)

	// Assert
	require.NoError(t, err)
	require.True(t, created)
	settings.DBName = databaseName
	db, err = dbops.NewPgDBConn(settings)
	require.NoError(t, err)
	db.Close()
}

// Test that there is no possibility of SQL injection when creating a database.
func TestCreateDatabaseSQLInjection(t *testing.T) {
	// Arrange
	db, settings, teardown := dbtest.SetupDatabaseTestCaseWithMaintenanceCredentials(t)
	defer teardown()
	databaseName := fmt.Sprintf("injection_test; DROP DATABASE %s; --", settings.DBName)

	// Act
	created, err := maintenance.CreateDatabase(db, databaseName)

	// Assert
	// Previously, the SQL query for creating the database was constructed via
	// string formatting, which allowed SQL injection. In this case, attempting
	// to create a database with a name that includes an injected SQL, resulted
	// in sending to Postgres server two commands: one original to create the
	// database, and the second that was injected.
	// Fortunately, the Postgres server rejected both commands because the
	// CREATE DATABASE command cannot be executed inside a transaction block.
	// The error contained a message: "cannot run inside a transaction block".
	//
	// The code has been fixed to use parameterized queries, so SQL injection
	// is no longer possible. Now, there is no error, and the database is
	// created successfully. The injected part is treated as part of the
	// database name.
	require.NoError(t, err)
	require.True(t, created)

	// Verify that the database is created.
	settings.DBName = databaseName
	db, err = dbops.NewPgDBConn(settings)
	require.NoError(t, err)
	db.Close()
}

// Test that if the database already exists, no error is returned.
func TestCreateDatabaseAlreadyExist(t *testing.T) {
	// Arrange
	db, settings, teardown := dbtest.SetupDatabaseTestCaseWithMaintenanceCredentials(t)
	defer teardown()

	// Act
	created, err := maintenance.CreateDatabase(db, settings.DBName)

	// Assert
	require.NoError(t, err)
	require.False(t, created)
}

// Test that the database from template is created properly.
func TestCreateDatabaseFromTemplate(t *testing.T) {
	// Arrange
	db, settings, teardown := dbtest.SetupDatabaseTestCaseWithMaintenanceCredentials(t)
	defer teardown()
	databaseName := fmt.Sprintf("%s_create_from_template", settings.DBName)

	// Act
	created, err := maintenance.CreateDatabaseFromTemplate(db, databaseName, settings.DBName)

	// Assert
	require.NoError(t, err)
	require.True(t, created)
	settings.DBName = databaseName
	db, err = dbops.NewPgDBConn(settings)
	require.NoError(t, err)
	db.Close()
}

// Test that the database is deleted properly.
func TestDropDatabaseIfExistsForExistingDatabase(t *testing.T) {
	// Arrange
	db, settings, teardown := dbtest.SetupDatabaseTestCaseWithMaintenanceCredentials(t)
	defer teardown()
	databaseName := fmt.Sprintf("%s_drop_safe_existing", settings.DBName)
	_, _ = maintenance.CreateDatabase(db, databaseName)

	// Act
	err := maintenance.DropDatabaseIfExists(db, databaseName)

	// Assert
	require.NoError(t, err)
	settings.DBName = databaseName
	_, err = dbops.NewPgDBConn(settings)
	require.ErrorContains(t, err, fmt.Sprintf("database \"%s\" does not exist", databaseName))
}

// Test that dropping non-existing database causes no error.
func TestDropDatabaseIfExistsForNonExistingDatabase(t *testing.T) {
	// Arrange
	db, settings, teardown := dbtest.SetupDatabaseTestCaseWithMaintenanceCredentials(t)
	defer teardown()
	databaseName := fmt.Sprintf("%s_drop_safe_non_existing", settings.DBName)

	// Act
	err := maintenance.DropDatabaseIfExists(db, databaseName)

	// Assert
	require.NoError(t, err)
	settings.DBName = databaseName
	_, err = dbops.NewPgDBConn(settings)
	require.ErrorContains(t, err, fmt.Sprintf(`database "%s" does not exist`, databaseName))
}

// Test that the database can be restored from a dump.
func TestRestoreDatabaseFromDump(t *testing.T) {
	// Arrange
	sb := testutil.NewSandbox()
	defer sb.Close()

	dumpFilePath, _ := sb.Write("dump.sql", `
		-- Unsupported parameter should be ignored.
		SET unsupported_parameter = 'unsupported';
		-- Commands allowed only for the superuser and owner should be ignored.
		COMMENT ON SCHEMA public IS 'standard public schema';
		CREATE SCHEMA public;

		CREATE TABLE test_table (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL
		);
		INSERT INTO test_table (name) VALUES ('test1'), ('test2');
	`)

	db, _, teardown := dbtest.SetupDatabaseTestCaseWithMaintenanceCredentials(t)
	defer teardown()

	// Act
	err := maintenance.RestoreDatabaseFromDump(db, dumpFilePath)

	// Assert
	require.NoError(t, err)

	var count int
	_, err = db.QueryOne(pg.Scan(&count), "SELECT COUNT(*) FROM test_table;")
	require.NoError(t, err)
	require.Equal(t, 2, count)
}
