package dbmodel

import (
	"testing"

	"github.com/go-pg/pg/v9"
	"github.com/stretchr/testify/require"

	"isc.org/stork/server/database/test"
)

// Tests that default system user can be authenticated.
func TestDefaultUserAuthenticate(t *testing.T) {
	teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown(t)

	db := pg.Connect(&dbtest.PgConnOptions)

	// Use default credentials of the admin user.
    user := &SystemUser{
		Login: "admin",
		Password: "admin",
    }
	authOk, err := Authenticate(db, user)
	require.NoError(t, err)
	require.True(t, authOk)

	// Using wrong password should cause the authentication to fail.
	user.Password = "wrong"
	authOk, err = Authenticate(db, user)
	require.NoError(t, err)
	require.False(t, authOk)
}

// Tests that system user can be added an authenticated.
func TestNewUserAuthenticate(t *testing.T) {
	teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown(t)

	db := pg.Connect(&dbtest.PgConnOptions)

	// Create new user.
    user := &SystemUser{
		Email: "jan@example.org",
		Lastname: "Kowalski",
        Name:     "Jan",
		Password: "pass",
    }
    err := user.Persist(db)
	require.NoError(t, err)

	authOk, err := Authenticate(db, user)
	require.NoError(t, err)
	require.True(t, authOk)
	// Returned password should be empty.
	require.Empty(t, user.Password)

	// Modifying user's password should be possible.
	user.Password = "new password"
	err = user.Persist(db)
	require.NoError(t, err)

	// Authentciation using old password should fail.
	user.Password = "pass"
	authOk, err = Authenticate(db, user)
	require.NoError(t, err)
	require.False(t, authOk)

	// But it should pass with the new password.
	user.Password = "new password"
	authOk, err = Authenticate(db, user)
	require.NoError(t, err)
	require.True(t, authOk)

	// If password is empty, it should remain unmodified in the database.
	user.Password = ""
    err = user.Persist(db)
	require.NoError(t, err)

	// Make sure that we can still authenticate (because the password hasn't changed).
	user.Password = "new password"
	authOk, err = Authenticate(db, user)
	require.NoError(t, err)
	require.True(t, authOk)
}
