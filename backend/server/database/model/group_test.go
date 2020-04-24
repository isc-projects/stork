package dbmodel

import (
	"testing"

	"github.com/stretchr/testify/require"
	dbtest "isc.org/stork/server/database/test"
)

// Test that all system groups can be fetched from the database.
func TestGetGroups(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	groups, total, err := GetGroupsByPage(db, 0, 10, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	// There are two predefined groups.
	require.Len(t, groups, 2)

	// Groups are supposed to be ordered by id.
	require.Equal(t, 1, groups[0].ID)
	require.Equal(t, "super-admin", groups[0].Name)
	require.Equal(t, 2, groups[1].ID)
	require.Equal(t, "admin", groups[1].Name)
}
