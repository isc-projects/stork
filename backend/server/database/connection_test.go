package dbops_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	dbops "isc.org/stork/server/database"
	dbtest "isc.org/stork/server/database/test"
)

// Error used in the connection unit tests.
var errConnection = errors.New("an error")

// Mock transaction implementation.
type testTxi struct {
	rollbackCalled bool
}

// Registers a Rollback() call.
func (txi *testTxi) Rollback() error {
	txi.rollbackCalled = true
	return nil
}

// Tests the logic that fetches database server version.
func TestGetDatabaseServerVersion(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	version, err := dbops.GetDatabaseServerVersion(db)

	require.NoError(t, err)
	require.GreaterOrEqual(t, version, 100000)
	require.Less(t, version, 200000)
}

// Test that deferred rollback is properly handled.
func TestRollbackOnError(t *testing.T) {
	tx := &testTxi{}
	require.False(t, tx.rollbackCalled)
	err := func() (err error) {
		defer dbops.RollbackOnError(tx, &err)
		err = errConnection
		return
	}()
	require.Error(t, err)
	require.ErrorIs(t, err, errConnection)
	require.True(t, tx.rollbackCalled)
}

// Test that the rollback is not invoked when there is no error.
func TestNoRollbackOnNoError(t *testing.T) {
	tx := &testTxi{}
	var err error
	require.False(t, tx.rollbackCalled)
	dbops.RollbackOnError(tx, &err)
	require.False(t, tx.rollbackCalled)
	dbops.RollbackOnError(tx, nil)
	require.False(t, tx.rollbackCalled)
}

// Test that the application database connection is created properly
// and database is migrated to the latest version.
func TestNewApplicationDatabaseConn(t *testing.T) {
	// Arrange
	db, settings, teardown := dbtest.SetupDatabaseTestCase(t)
	tossErr := dbops.Toss(db)
	teardown()

	// Act
	db, dbErr := dbops.NewApplicationDatabaseConn(settings)

	// Assert
	require.NoError(t, dbErr)
	defer db.Close()

	require.NoError(t, tossErr)
	require.NotNil(t, db)

	version, versionErr := dbops.CurrentVersion(db)
	require.NoError(t, versionErr)
	require.NotZero(t, version)
}

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
