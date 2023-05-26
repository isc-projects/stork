package maintenance_test

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
	dbops "isc.org/stork/server/database"
	"isc.org/stork/server/database/maintenance"
	dbtest "isc.org/stork/server/database/test"
)

// Test that the pgcrypto database extension is successfully created.
func TestCreateCryptoExtension(t *testing.T) {
	// Connect to the database with full privileges.
	db, originalSettings, teardown := dbtest.SetupDatabaseTestCaseWithMaintenanceCredentials(t)
	defer teardown()

	// Create a database and the user with the same name.
	dbName := fmt.Sprintf("storktest%d", rand.Int63())
	_, err := maintenance.CreateDatabase(db, dbName)
	require.NoError(t, err)

	// Try to connect to this database using the user name.
	opts := *originalSettings
	opts.DBName = dbName
	db2, err := dbops.NewPgDBConn(&opts)
	require.NoError(t, err)
	require.NotNil(t, db2)

	// The new database should initially lack pgcrypto extension.
	hasExtension, err := maintenance.HasExtension(db2, "pgcrypto")
	require.NoError(t, err)
	require.False(t, hasExtension)

	// Create the pgcrypto extension.
	err = maintenance.CreateExtension(db2, "pgcrypto")
	require.NoError(t, err)

	// Make sure the extension is now present.
	hasExtension, err = maintenance.HasExtension(db2, "pgcrypto")
	require.NoError(t, err)
	require.True(t, hasExtension)

	// An attempt to create an unknown extension should fail.
	err = maintenance.CreateExtension(db2, "unknown")
	require.Error(t, err)
}
