package dbmodel

import (
	"fmt"
	"testing"

	faker "github.com/brianvoe/gofakeit"
	"github.com/stretchr/testify/require"

	dbops "isc.org/stork/server/database"
	dbtest "isc.org/stork/server/database/test"
)

// Generates a bunch of users and stores them in the database.
func generateTestUsers(t *testing.T, db *dbops.PgDB) {
	for i := 0; i < 100; i++ {
		login := faker.Word()
		login = fmt.Sprintf("%s%d", login, i)
		user := &SystemUser{
			Login:    login,
			Email:    fmt.Sprintf("%s@example.org", login),
			Lastname: faker.LastName(),
			Name:     faker.FirstName(),
			Password: faker.Word(),
		}
		err, _ := CreateUser(db, user)
		require.NoError(t, err, "failed for index %d, login %s", i, user.Login)
	}
}

// Tests that default system user can be authenticated.
func TestDefaultUserAuthenticate(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Use default credentials of the admin user.
	user := &SystemUser{
		Login:    "admin",
		Password: "admin",
	}
	authOk, err := Authenticate(db, user)
	require.NoError(t, err)
	require.True(t, authOk)

	// The default user is by default in the super-admin group, which
	// should be returned.
	require.True(t, user.InGroup(&SystemGroup{Name: "super-admin"}))

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
		Email:    "jan@example.org",
		Lastname: "Kowalski",
		Name:     "Jan",
		Password: "pass",
	}
	err, con := CreateUser(db, user)
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
	err, con = UpdateUser(db, user)
	require.False(t, con)
	require.NoError(t, err)

	// Authentication using old password should fail.
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
	err, con = UpdateUser(db, user)
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
func TestUpdateNoUser(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	user := &SystemUser{
		Id:       123456,
		Email:    "jan@example.org",
		Lastname: "Kowalski",
		Name:     "Jan",
		Password: "pass",
	}
	err, con := UpdateUser(db, user)
	require.True(t, con)
	require.Error(t, err)
}

// Tests that conflict flag is returned when the inserted user is in
// conflict with existing user.
func TestCreateConflict(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	user := &SystemUser{
		Login:    "jankowal",
		Email:    "jan@example.org",
		Lastname: "Kowalski",
		Name:     "Jan",
		Password: "pass",
	}
	err, con := CreateUser(db, user)
	require.False(t, con)
	require.NoError(t, err)

	user = &SystemUser{
		Email:    "jan@example.org",
		Lastname: "Kowalski",
		Name:     "Jan",
		Password: "pass",
	}
	err, con = CreateUser(db, user)
	require.True(t, con)
	require.Error(t, err)
}

// Tests that password can be modified.
func TestSetPassword(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Create new user.
	user := &SystemUser{
		Email:    "jan@example.org",
		Lastname: "Kowalski",
		Name:     "Jan",
		Password: "pass",
	}
	err, con := CreateUser(db, user)
	require.False(t, con)
	require.NoError(t, err)

	require.Greater(t, user.Id, 0)

	// Set new password for the user.
	err = SetPassword(db, user.Id, "newpass")
	require.NoError(t, err)

	// Authenticate with the new password.
	user.Password = "newpass"
	authOk, err := Authenticate(db, user)
	require.NoError(t, err)
	require.True(t, authOk)

	// Also check that authentication with invalid password fails.
	user.Password = "NewPass"
	authOk, err = Authenticate(db, user)
	require.NoError(t, err)
	require.False(t, authOk)

	// And finally check the original password doesn't work anymore.
	user.Password = "pass"
	authOk, err = Authenticate(db, user)
	require.NoError(t, err)
	require.False(t, authOk)
}

// Tests that password update fails when user does not exist.
func TestSetPasswordNoUser(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	err := SetPassword(db, 123, "newpass")
	require.Error(t, err)
}

// Test that the password is successfully changed if the current password
// is valid and that the password is not changed if the current password
// specified is invalid.
func TestChangePassword(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Create new user.
	user := &SystemUser{
		Email:    "jan@example.org",
		Lastname: "Kowalski",
		Name:     "Jan",
		Password: "pass",
	}
	err, con := CreateUser(db, user)
	require.False(t, con)
	require.NoError(t, err)

	// Provide invalid current password. The original password should not change.
	auth, err := ChangePassword(db, user.Id, "invalid", "newpass")
	require.False(t, auth)
	require.NoError(t, err)

	// Still should authenticate with the current password.
	authOk, err := Authenticate(db, user)
	require.NoError(t, err)
	require.True(t, authOk)

	// Provide valid password. The original password should be modified.
	auth, err = ChangePassword(db, user.Id, "pass", "newpass")
	require.True(t, auth)
	require.NoError(t, err)

	// Authenticate with the new password.
	user.Password = "newpass"
	authOk, err = Authenticate(db, user)
	require.NoError(t, err)
	require.True(t, authOk)
}

// Tests that all system users can be fetched from the database.
func TestGetUsers(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	generateTestUsers(t, db)

	users, total, err := GetUsers(db, 0, 1000, SystemUserOrderById)
	require.NoError(t, err)
	require.Equal(t, 101, len(users))
	require.Equal(t, int64(101), total)

	var prevId int = 0
	for _, u := range users {
		// Make sure that by default the users are ordered by ID.
		require.Greater(t, u.Id, prevId)
		prevId = u.Id
	}
}

// Tests that users can be fetched and sorted by login or email.
func TestGetUsersSortByLoginEmail(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	generateTestUsers(t, db)

	users, total, err := GetUsers(db, 0, 1000, SystemUserOrderByLoginEmail)
	require.NoError(t, err)
	require.Equal(t, 101, len(users))
	require.Equal(t, int64(101), total)

	prevLogin := ""
	for _, u := range users {
		// Make sure that by default the users are ordered by ID.
		require.Greater(t, u.Login, prevLogin)
		prevLogin = u.Login
	}
}

// Tests that a page of users can be fetched.
func TestGetUsersPage(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	generateTestUsers(t, db)

	users, total, err := GetUsers(db, 50, 10, SystemUserOrderById)
	require.NoError(t, err)
	require.Equal(t, 10, len(users))
	require.Equal(t, 51, users[0].Id)
	require.Equal(t, int64(101), total)

	var prevId int = 0
	for _, u := range users {
		// Make sure that by default the users are ordered by ID.
		require.Greater(t, u.Id, prevId)
		prevId = u.Id
	}
}

// Tests that last page of users can be fetched without issues.
func TestGetUsersLastPage(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	generateTestUsers(t, db)

	users, total, err := GetUsers(db, 90, 20, SystemUserOrderById)
	require.NoError(t, err)
	require.Equal(t, 11, len(users))
	require.Equal(t, 91, users[0].Id)
	require.Equal(t, int64(101), total)

	var prevId int = 0
	for _, u := range users {
		// Make sure that by default the users are ordered by ID.
		require.Greater(t, u.Id, prevId)
		prevId = u.Id
	}
}

// Tests that user can be fetched by Id.
func TestGetUserById(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	generateTestUsers(t, db)

	users, total, err := GetUsers(db, 0, 1000, SystemUserOrderById)
	require.NoError(t, err)
	require.Equal(t, int64(101), total)

	user, err := GetUserById(db, users[0].Id)
	require.NoError(t, err)
	require.NotNil(t, user)

	user, err = GetUserById(db, 1234567)
	require.NoError(t, err)
	require.Nil(t, user)
}

// Test that user associations with groups are created when the user
// is created or updated.
func TestUserGroups(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Create a user account.
	user := &SystemUser{
		Login:    "test",
		Email:    "test@example.org",
		Lastname: "Smith",
		Name:     "John",
		Password: "pass",
		Groups: SystemGroups{
			&SystemGroup{
				Id: 1,
			},
			&SystemGroup{
				Id: 2,
			},
		},
	}

	err, _ := CreateUser(db, user)
	require.NoError(t, err)
	require.Greater(t, user.Id, 0)

	// Fetch the user by id. It should also return the groups it belongs to.
	returned, err := GetUserById(db, user.Id)
	require.NotNil(t, returned)
	require.NoError(t, err)

	// Fetch the user by id. It should also return the groups it belongs to.
	returned, err = GetUserById(db, user.Id)
	require.NotNil(t, returned)
	require.NoError(t, err)

	require.Equal(t, 2, len(returned.Groups))
	require.True(t, returned.InGroup(&SystemGroup{Name: "super-admin"}))
	require.True(t, returned.InGroup(&SystemGroup{Name: "admin"}))

	// Remove the user from one of the groups.
	user.Groups = SystemGroups{
		&SystemGroup{
			Id: 2,
		},
	}

	// Updating the user should also cause the groups to be updated.
	err, _ = UpdateUser(db, user)
	require.NoError(t, err)

	returned, err = GetUserById(db, user.Id)
	require.NotNil(t, returned)
	require.NoError(t, err)

	// Fetch the user by id. It should also return new groups.
	returned, err = GetUserById(db, user.Id)
	require.NotNil(t, returned)
	require.NoError(t, err)

	// The groups should have been updated. One group should now be gone.
	require.Equal(t, 1, len(returned.Groups))
	require.False(t, returned.InGroup(&SystemGroup{Name: "super-admin"}))
	require.True(t, returned.InGroup(&SystemGroup{Name: "admin"}))
}

// Test that user can be associated with a group and then the groups
// are returned along with the user.
func TestAddToGroupById(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Create a user account.
	user := &SystemUser{
		Login:    "test",
		Email:    "test@example.org",
		Lastname: "Smith",
		Name:     "John",
		Password: "pass",
	}
	err, _ := CreateUser(db, user)
	require.NoError(t, err)
	require.Greater(t, user.Id, 0)

	// Associate the user with two prefefined groups.
	added, err := user.AddToGroupById(db, &SystemGroup{Id: 1})
	require.NoError(t, err)
	require.True(t, added)

	added, err = user.AddToGroupById(db, &SystemGroup{Id: 2})
	require.NoError(t, err)
	require.True(t, added)

	// Fetch the user by id. It should also return the groups it belongs to.
	returned, err := GetUserById(db, user.Id)
	require.NotNil(t, returned)
	require.NoError(t, err)

	require.Equal(t, 2, len(returned.Groups))
	require.True(t, returned.InGroup(&SystemGroup{Name: "super-admin"}))
	require.True(t, returned.InGroup(&SystemGroup{Name: "admin"}))

	// Another attempt to add the user to the same group should be no-op.
	added, err = user.AddToGroupById(db, &SystemGroup{Id: 1})
	require.NoError(t, err)
	require.False(t, added)
}
