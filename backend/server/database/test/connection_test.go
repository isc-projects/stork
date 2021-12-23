package dbtest

import (
	"testing"

	"github.com/stretchr/testify/require"
	dbops "isc.org/stork/server/database"
)

// Tests the logic that fetches database server version.
func TestGetDatabaseServerVersion(t *testing.T) {
	db, _, teardown := SetupDatabaseTestCase(t)
	defer teardown()

	version, err := dbops.GetDatabaseServerVersion(db)

	require.NoError(t, err)
	require.GreaterOrEqual(t, version, 100000)
	require.Less(t, version, 200000)
}
