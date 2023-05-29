package maintenance_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/server/database/maintenance"
	dbtest "isc.org/stork/server/database/test"
)

// Test that the PgHBA configuration entries are fetched properly for the
// maintenance user.
func TestGetPgHBAConfigurationEntries(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCaseWithMaintenanceCredentials(t)
	defer teardown()

	// Act
	entries, err := maintenance.GetPgHBAConfiguration(db)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, entries)
	require.NotEmpty(t, entries)
}
