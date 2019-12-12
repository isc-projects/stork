package dbmodel

import (
	"testing"
	"github.com/stretchr/testify/require"
	"isc.org/stork/server/database/test"
)

// Test that all system groups can be fetched from the database.
func TestGetGroups(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	groups, err := GetGroups(db)
	require.NoError(t, err)
	// There are two predefined groups.
	require.Equal(t, 2, len(groups))

	// Groups are supposed to be ordered by id.
	require.Equal(t, 1, groups[0].Id)
	require.Equal(t, "super-admin", groups[0].Name)
	require.Equal(t, 2, groups[1].Id)
	require.Equal(t, "admin", groups[1].Name)
}
