package dbsession

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
)

// Test that the active session is found correctly.
func TestStoreFind(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	store := NewStore(db)

	_ = dbmodel.AddOrUpdateSession(db, &dbmodel.Session{
		Token: "foo", Data: []byte("foo"), Expiry: time.Now().Add(time.Hour),
	})
	_ = dbmodel.AddOrUpdateSession(db, &dbmodel.Session{
		Token: "bar", Data: []byte("bar"), Expiry: time.Time{},
	})

	t.Run("Active session", func(t *testing.T) {
		// Act
		data, exists, err := store.Find("foo")

		// Assert
		require.NoError(t, err)
		require.True(t, exists)
		require.Equal(t, []byte("foo"), data)
	})

	t.Run("Expired session", func(t *testing.T) {
		// Act
		data, exists, err := store.Find("bar")

		// Assert
		require.NoError(t, err)
		require.False(t, exists)
		require.Nil(t, data)
	})

	t.Run("Non-existent session", func(t *testing.T) {
		// Act
		data, exists, err := store.Find("baz")

		// Assert
		require.NoError(t, err)
		require.False(t, exists)
		require.Nil(t, data)
	})
}

// Test that the session is added or updated correctly.
func TestStoreCommitAdd(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	store := NewStore(db)

	// Act
	err := store.Commit("foo", []byte("foo"), time.Now().Add(time.Hour))

	// Assert
	require.NoError(t, err)

	// Act
	data, exists, err := store.Find("foo")

	// Assert
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, []byte("foo"), data)
	sessions, _ := dbmodel.GetAllActiveSessions(db)
	require.Len(t, sessions, 1)
}

// Test that the session is updated correctly.
func TestStoreCommitUpdate(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	store := NewStore(db)

	_ = dbmodel.AddOrUpdateSession(db, &dbmodel.Session{
		Token: "foo", Data: []byte("foo"), Expiry: time.Now().Add(time.Hour),
	})

	// Act
	err := store.Commit("foo", []byte("bar"), time.Now().Add(time.Hour))

	// Assert
	require.NoError(t, err)
	session, _ := dbmodel.GetActiveSession(db, "foo")
	require.Equal(t, []byte("bar"), session.Data)
}

// Test that the session is deleted correctly.
func TestStoreDelete(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	store := NewStore(db)

	_ = dbmodel.AddOrUpdateSession(db, &dbmodel.Session{
		Token: "foo", Data: []byte("foo"), Expiry: time.Now().Add(time.Hour),
	})
	_ = dbmodel.AddOrUpdateSession(db, &dbmodel.Session{
		Token: "bar", Data: []byte("bar"), Expiry: time.Time{},
	})

	// Act
	err := store.Delete("foo")

	// Assert
	require.NoError(t, err)
	sessions, _ := dbmodel.GetAllSessions(db)
	require.Len(t, sessions, 1)
	require.Equal(t, "bar", sessions[0].Token)
}

// Test that all active sessions are returned correctly.
func TestStoreAll(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	store := NewStore(db)

	_ = dbmodel.AddOrUpdateSession(db, &dbmodel.Session{
		Token: "foo", Data: []byte("foo"), Expiry: time.Now().Add(time.Hour),
	})
	_ = dbmodel.AddOrUpdateSession(db, &dbmodel.Session{
		Token: "bar", Data: []byte("bar"), Expiry: time.Now().Add(time.Hour),
	})
	_ = dbmodel.AddOrUpdateSession(db, &dbmodel.Session{
		Token: "baz", Data: []byte("baz"), Expiry: time.Time{},
	})

	// Act
	dataByToken, err := store.All()

	// Assert
	require.NoError(t, err)
	require.Len(t, dataByToken, 2)
	require.Equal(t, []byte("foo"), dataByToken["foo"])
	require.Equal(t, []byte("bar"), dataByToken["bar"])
}

// Test that store cleanup expired sessions in the background.
func TestStoreCleanup(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_ = dbmodel.AddOrUpdateSession(db, &dbmodel.Session{
		Token: "foo", Data: []byte("foo"), Expiry: time.Now().Add(time.Hour),
	})
	_ = dbmodel.AddOrUpdateSession(db, &dbmodel.Session{
		Token: "bar", Data: []byte("bar"), Expiry: time.Time{},
	})

	// Act & Assert
	store := NewStoreWithCleanupInterval(db, 100*time.Millisecond)

	require.Eventually(t, func() bool {
		sessions, _ := dbmodel.GetAllSessions(db)
		return len(sessions) == 1
	}, 1*time.Second, 100*time.Millisecond)

	sessions, _ := dbmodel.GetAllSessions(db)
	require.Len(t, sessions, 1)
	require.Equal(t, "foo", sessions[0].Token)
	require.NotNil(t, store)
}

// Test that the store cleanup goroutine can be stopped.
func TestStoreCleanupStop(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	store := NewStoreWithCleanupInterval(db, 50*time.Millisecond)

	// Act
	store.StopCleanup()

	// Add an expired session.
	_ = dbmodel.AddOrUpdateSession(db, &dbmodel.Session{
		Token: "foo", Data: []byte("foo"), Expiry: time.Time{},
	})

	// Assert
	require.Never(t, func() bool {
		sessions, _ := dbmodel.GetAllSessions(db)
		// The session should be never removed.
		return len(sessions) == 0
	}, 500*time.Millisecond, 100*time.Millisecond)
}
