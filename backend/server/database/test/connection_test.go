package dbtest

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	dbops "isc.org/stork/server/database"
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
	db, _, teardown := SetupDatabaseTestCase(t)
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
