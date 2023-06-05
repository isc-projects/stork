package maintenance_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/server/database/maintenance"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
)

// Test that dropping existing table works properly.
func TestDropTableIfExistsForExistingTable(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCaseWithMaintenanceCredentials(t)
	defer teardown()

	_, err := dbmodel.GetAllSettings(db)
	require.NoError(t, err)

	// Act
	err = maintenance.DropTableIfExists(db, "setting")

	// Assert
	require.NoError(t, err)
	_, err = dbmodel.GetAllSettings(db)
	require.ErrorContains(t, err, `relation "setting" does not exist`)
}

// Test that dropping non-existing table causes no error.
func TestDropTableIfExistsForNonExistingTable(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCaseWithMaintenanceCredentials(t)
	defer teardown()

	// Act
	err := maintenance.DropTableIfExists(db, "foobar")

	// Assert
	require.NoError(t, err)
}

// Test that dropping non-existing sequence causes no error.
func TestDropSequenceIfExists(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCaseWithMaintenanceCredentials(t)
	defer teardown()

	// Act
	err := maintenance.DropSequenceIfExists(db, "stork_test_non_existing_sequence")

	// Assert
	require.NoError(t, err)
}
