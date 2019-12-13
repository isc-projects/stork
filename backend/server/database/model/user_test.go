package dbmodel

import (
	"fmt"
	"testing"

	faker "github.com/brianvoe/gofakeit"
	"github.com/stretchr/testify/require"

	"isc.org/stork/server/database"
	"isc.org/stork/server/database/test"
)

// Generates a bunch of users and stores them in the database.
func generateTestUsers(t *testing.T, db *dbops.PgDB) {
	for i := 0; i < 100; i++ {
		login := faker.Word()
		login = fmt.Sprintf("%s%d", login, i)
		user := &SystemUser{
			Login: login,
			Email: fmt.Sprintf("%s@example.org", login),
			Lastname: faker.LastName(),
			Name: faker.FirstName(),
			Password: faker.Word(),
		}
		err, _ := user.Persist(db)
		require.NoError(t, err, "failed for index %d, login %s", i, user.Login)
	}
}

// Tests that default system user can be authenticated.
func TestDefaultUserAuthenticate(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

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
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Create new user.
	user := &SystemUser{
		Email: "jan@example.org",
		Lastname: "Kowalski",
		Name:     "Jan",
		Password: "pass",
	}
	err, con := user.Persist(db)
	require.False(t, con)
	require.NoError(t, err)

	require.Greater(t, user.Id, 0)

	authOk, err := Authenticate(db, user)
	require.NoError(t, err)
	require.True(t, authOk)
	// Returned password should be empty.
	require.Empty(t, user.Password)

	// Modifying user's password should be possible.
	user.Password = "new password"
	err, con = user.Persist(db)
	require.False(t, con)
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
	err, con = user.Persist(db)
	require.False(t, con)
	require.NoError(t, err)

	// Make sure that we can still authenticate (because the password hasn't changed).
	user.Password = "new password"
	authOk, err = Authenticate(db, user)
	require.NoError(t, err)
	require.True(t, authOk)
}

// Tests that it is indicated when the user being udpdated is not found in
// the database.
func TestPersistNoUser(t *testing.T) {
	db, _, teardown  := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	user := &SystemUser{
		Id: 123456,
		Email: "jan@example.org",
		Lastname: "Kowalski",
		Name:     "Jan",
		Password: "pass",
	}
	err, con := user.Persist(db)
	require.True(t, con)
	require.Error(t, err)
}

// Tests that conflict flag is returned when the inserted user is in
// conflict with existing user.
func TestPersistConflict(t *testing.T) {
	db, _, teardown  := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	user := &SystemUser{
		Login: "jankowal",
		Email: "jan@example.org",
		Lastname: "Kowalski",
		Name:     "Jan",
		Password: "pass",
	}
	err, con := user.Persist(db)
	require.False(t, con)
	require.NoError(t, err)

	user = &SystemUser{
		Email: "jan@example.org",
		Lastname: "Kowalski",
		Name:     "Jan",
		Password: "pass",
	}
	err, con = user.Persist(db)
	require.True(t, con)
	require.Error(t, err)
}


// Tests that all system users can be fetched from the database.
func TestGetUsersByPage(t *testing.T) {
	db, _, teardown  := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	generateTestUsers(t, db)

	users, total, err := GetUsersByPage(db, 0, 1000, SystemUserOrderById)
	require.NoError(t, err)
	require.Equal(t, 101, len(users))
	require.Equal(t, int64(101), total)

	var prevId int = 0
	for _, u := range users {
		// Make sure that by default the users are ordered by ID.
		require.Greater(t, u.Id, prevId); prevId = u.Id
	}
}

// Tests that users can be fetched and sorted by login or email.
func TestGetUsersByPageSortByLoginEmail(t *testing.T) {
	db, _, teardown  := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	generateTestUsers(t, db)

	users, total, err := GetUsersByPage(db, 0, 1000, SystemUserOrderByLoginEmail)
	require.NoError(t, err)
	require.Equal(t, 101, len(users))
	require.Equal(t, int64(101), total)

	prevLogin := ""
	for _, u := range users {
		// Make sure that by default the users are ordered by ID.
		require.Greater(t, u.Login, prevLogin); prevLogin = u.Login
	}
}

// Tests that a page of users can be fetched.
func TestGetUsersByPagePage(t *testing.T) {
	db, _, teardown  := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	generateTestUsers(t, db)

	users, total, err := GetUsersByPage(db, 50, 10, SystemUserOrderById)
	require.NoError(t, err)
	require.Equal(t, 10, len(users))
	require.Equal(t, 51, users[0].Id)
	require.Equal(t, int64(101), total)

	var prevId int = 0
	for _, u := range users {
		// Make sure that by default the users are ordered by ID.
		require.Greater(t, u.Id, prevId); prevId = u.Id
	}
}

// Tests that last page of users can be fetched without issues.
func TestGetUsersByPageLastPage(t *testing.T) {
	db, _, teardown  := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	generateTestUsers(t, db)

	users, total, err := GetUsersByPage(db, 90, 20, SystemUserOrderById)
	require.NoError(t, err)
	require.Equal(t, 11, len(users))
	require.Equal(t, 91, users[0].Id)
	require.Equal(t, int64(101), total)

	var prevId int = 0
	for _, u := range users {
		// Make sure that by default the users are ordered by ID.
		require.Greater(t, u.Id, prevId); prevId = u.Id
	}
}

// Tests that user can be fetched by Id.
func TestGetUserById(t *testing.T) {
	db, _, teardown  := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	generateTestUsers(t, db)

	users, total, err := GetUsersByPage(db, 0, 1000, SystemUserOrderById)
	require.NoError(t, err)
	require.Equal(t, int64(101), total)

	user, err := GetUserById(db, users[0].Id)
	require.NoError(t, err)
	require.NotNil(t, user)

	user, err = GetUserById(db, 1234567)
	require.NoError(t, err)
	require.Nil(t, user)
}
