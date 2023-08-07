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
	require.Equal(t, SuperAdminGroupID, groups[0].ID)
	require.Equal(t, "super-admin", groups[0].Name)
	require.Equal(t, AdminGroupID, groups[1].ID)
	require.Equal(t, "admin", groups[1].Name)

	// check sorting field and order ascending
	groups, total, err = GetGroupsByPage(db, 0, 10, nil, "name", SortDirAsc)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, groups, 2)
	require.Equal(t, "admin", groups[0].Name)
	require.Equal(t, "super-admin", groups[1].Name)

	// check sorting field and order descending
	groups, total, err = GetGroupsByPage(db, 0, 10, nil, "name", SortDirDesc)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, groups, 2)
	require.Equal(t, "super-admin", groups[0].Name)
	require.Equal(t, "admin", groups[1].Name)

	// check filtering by text
	text := "super"
	groups, total, err = GetGroupsByPage(db, 0, 10, &text, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, groups, 1)
	require.Equal(t, "super-admin", groups[0].Name)
}
