package dbmodel

import (
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	dbtest "isc.org/stork/server/database/test"
)

// Test that the session is added to the database correctly.
func TestAddSession(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	expireTime := time.Date(4024, 0o7, 10, 18, 36, 22, 0, time.Local)

	// Act
	session := &Session{Token: "foo", Data: []byte("bar"), Expiry: expireTime}
	err1 := AddOrUpdateSession(db, session)
	session = &Session{Token: "bar", Data: []byte("baz"), Expiry: time.Time{}}
	err2 := AddOrUpdateSession(db, session)

	// Assert
	require.NoError(t, err1)
	require.NoError(t, err2)
	sessions, _ := GetAllSessions(db)
	require.Len(t, sessions, 2)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Token < sessions[j].Token
	})
	require.Equal(t, "bar", sessions[0].Token)
	require.Equal(t, []byte("baz"), sessions[0].Data)
	require.Equal(t, time.Time{}, sessions[0].Expiry.UTC())
	require.Equal(t, "foo", sessions[1].Token)
	require.Equal(t, []byte("bar"), sessions[1].Data)
	require.Equal(t, expireTime, sessions[1].Expiry.Local())
}

// Test that the session is updated if it already exists.
func TestUpdateDataSession(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	expireTime := time.Date(4024, 0o7, 10, 18, 36, 22, 0, time.Local)

	session := &Session{Token: "foo", Data: []byte("bar"), Expiry: expireTime}
	_ = AddOrUpdateSession(db, session)

	// Act
	session.Data = []byte("baz")
	err := AddOrUpdateSession(db, session)

	// Assert
	require.NoError(t, err)
	sessions, _ := GetAllSessions(db)
	require.Len(t, sessions, 1)
	require.Equal(t, "foo", sessions[0].Token)
	require.Equal(t, []byte("baz"), sessions[0].Data)
	require.True(t, expireTime.Equal(sessions[0].Expiry))
}

func TestUpdateExpireTimeSession(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	expireTime := time.Date(4024, 0o7, 10, 18, 36, 22, 0, time.Local)

	session := &Session{Token: "foo", Data: []byte("bar"), Expiry: expireTime}
	_ = AddOrUpdateSession(db, session)

	// Act
	session.Expiry = time.Time{}
	err := AddOrUpdateSession(db, session)

	// Assert
	require.NoError(t, err)
	sessions, _ := GetAllSessions(db)
	require.Len(t, sessions, 1)
	require.Equal(t, "foo", sessions[0].Token)
	require.Equal(t, []byte("bar"), sessions[0].Data)
	require.Equal(t, time.Time{}, sessions[0].Expiry.UTC())
}

// Test that the active session is obtained correctly from the database.
func TestGetActiveSession(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Active session.
	_ = AddOrUpdateSession(db, &Session{Token: "foo", Data: []byte("foo"), Expiry: time.Now().Add(time.Hour)})
	// Expired session.
	_ = AddOrUpdateSession(db, &Session{Token: "bar", Data: []byte("bar"), Expiry: time.Time{}})

	t.Run("Active session", func(t *testing.T) {
		// Act
		session, err := GetActiveSession(db, "foo")

		// Assert
		require.NoError(t, err)
		require.NotNil(t, session)
	})

	t.Run("Expired session", func(t *testing.T) {
		// Act
		session, err := GetActiveSession(db, "bar")

		// Assert
		require.NoError(t, err)
		require.Nil(t, session)
	})

	t.Run("Non-existent session", func(t *testing.T) {
		// Act
		session, err := GetActiveSession(db, "baz")

		// Assert
		require.NoError(t, err)
		require.Nil(t, session)
	})
}

// Test that the session is deleted from the database correctly.
func TestDeleteSession(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_ = AddOrUpdateSession(db, &Session{Token: "foo", Data: []byte("foo"), Expiry: time.Now().Add(time.Hour)})
	_ = AddOrUpdateSession(db, &Session{Token: "bar", Data: []byte("bar"), Expiry: time.Now().Add(time.Hour)})
	_ = AddOrUpdateSession(db, &Session{Token: "baz", Data: []byte("baz"), Expiry: time.Time{}})
	_ = AddOrUpdateSession(db, &Session{Token: "boz", Data: []byte("boz"), Expiry: time.Time{}})

	// Act
	err1 := DeleteSession(db, "foo")
	err2 := DeleteSession(db, "baz")
	err3 := DeleteSession(db, "non-existent")

	// Assert
	require.NoError(t, err1)
	require.NoError(t, err2)
	require.NoError(t, err3)
	sessions, _ := GetAllSessions(db)
	require.Len(t, sessions, 2)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Token < sessions[j].Token
	})
	require.Equal(t, "bar", sessions[0].Token)
	require.Equal(t, "boz", sessions[1].Token)
}

// Test that all active sessions are obtained correctly from the database.
func TestGetAllActiveSessions(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_ = AddOrUpdateSession(db, &Session{Token: "foo", Data: []byte("foo"), Expiry: time.Now().Add(time.Hour)})
	_ = AddOrUpdateSession(db, &Session{Token: "bar", Data: []byte("bar"), Expiry: time.Now().Add(time.Hour)})
	_ = AddOrUpdateSession(db, &Session{Token: "baz", Data: []byte("baz"), Expiry: time.Time{}})
	_ = AddOrUpdateSession(db, &Session{Token: "boz", Data: []byte("boz"), Expiry: time.Time{}})

	// Act
	sessions, err := GetAllActiveSessions(db)

	// Assert
	require.NoError(t, err)
	require.Len(t, sessions, 2)
	tokens := []string{sessions[0].Token, sessions[1].Token}
	require.Contains(t, tokens, "foo")
	require.Contains(t, tokens, "bar")
}

// Test that active sessions is an empty list if the database is empty.
func TestGetAllActiveSessionsEmpty(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	sessions, err := GetAllActiveSessions(db)

	// Assert
	require.NoError(t, err)
	require.Len(t, sessions, 0)
}

// Test that active sessions is an empty list if all sessions are expired.
func TestGetAllActiveSessionsOnlyExpired(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_ = AddOrUpdateSession(db, &Session{Token: "foo", Data: []byte("foo"), Expiry: time.Time{}})
	_ = AddOrUpdateSession(db, &Session{Token: "bar", Data: []byte("bar"), Expiry: time.Time{}})
	_ = AddOrUpdateSession(db, &Session{Token: "baz", Data: []byte("baz"), Expiry: time.Time{}})

	// Act
	sessions, err := GetAllActiveSessions(db)

	// Assert
	require.NoError(t, err)
	require.Len(t, sessions, 0)
}

// Test that all sessions are obtained correctly from the database.
func TestGetAllSessions(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_ = AddOrUpdateSession(db, &Session{Token: "foo", Data: []byte("foo"), Expiry: time.Now().Add(time.Hour)})
	_ = AddOrUpdateSession(db, &Session{Token: "bar", Data: []byte("bar"), Expiry: time.Now().Add(time.Hour)})
	_ = AddOrUpdateSession(db, &Session{Token: "baz", Data: []byte("baz"), Expiry: time.Time{}})

	// Act
	sessions, err := GetAllSessions(db)

	// Assert
	require.NoError(t, err)
	require.Len(t, sessions, 3)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Token < sessions[j].Token
	})
	require.Equal(t, "bar", sessions[0].Token)
	require.Equal(t, "baz", sessions[1].Token)
	require.Equal(t, "foo", sessions[2].Token)
}

// Test that all sessions is an empty list if the database is empty.
func TestGetAllSessionsEmpty(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	sessions, err := GetAllSessions(db)

	// Assert
	require.NoError(t, err)
	require.Len(t, sessions, 0)
}

// Test that all expired sessions are removed from the database.
func TestDeleteAllExpiredSessions(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_ = AddOrUpdateSession(db, &Session{Token: "foo", Data: []byte("foo"), Expiry: time.Now().Add(time.Hour)})
	_ = AddOrUpdateSession(db, &Session{Token: "bar", Data: []byte("bar"), Expiry: time.Now().Add(time.Hour)})
	_ = AddOrUpdateSession(db, &Session{Token: "baz", Data: []byte("baz"), Expiry: time.Time{}})
	_ = AddOrUpdateSession(db, &Session{Token: "boz", Data: []byte("boz"), Expiry: time.Time{}})

	// Act
	err := DeleteAllExpiredSessions(db)

	// Assert
	require.NoError(t, err)
	sessions, _ := GetAllSessions(db)
	require.Len(t, sessions, 2)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Token < sessions[j].Token
	})
	require.Equal(t, "bar", sessions[0].Token)
	require.Equal(t, "foo", sessions[1].Token)
}
