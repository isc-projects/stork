package dbops_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	dbops "isc.org/stork/server/database"
	dbtest "isc.org/stork/server/database/test"
)

// Test that the suppress query logging function returns a valid DB with a
// context containing the disabling logging keyword.
func TestSuppressQueryLogging(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	before := dbops.HasSuppressedQueryLogging(db.Context())
	db = dbops.SuppressQueryLogging(db)
	after := dbops.HasSuppressedQueryLogging(db.Context())

	// Assert
	require.False(t, before)
	require.True(t, after)
}

// Test that too long queries are rejected.
func TestQuerySizeLimit(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	hook := dbops.NewDBQuerySizeLimiterCustom(10)
	db.AddQueryHook(hook)

	t.Run("query size exceeds the limit", func(t *testing.T) {
		// Act
		_, err := db.Exec("SELECT '01234567890123456789'")

		// Assert
		require.ErrorContains(t, err, "query size exceeds 10B limit, got: 29B")
	})

	t.Run("query size is within the limit", func(t *testing.T) {
		// Act
		_, err := db.Exec("SELECT 1")

		// Assert
		require.NoError(t, err)
	})
}
