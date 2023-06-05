package maintenance_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	dbops "isc.org/stork/server/database"
	"isc.org/stork/server/database/maintenance"
	dbtest "isc.org/stork/server/database/test"
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
