package dbmodel

import (
	"fmt"
	"testing"

	faker "github.com/brianvoe/gofakeit"
	"github.com/go-pg/pg/v10"
	pkgerrors "github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	dbops "isc.org/stork/server/database"
	dbtest "isc.org/stork/server/database/test"
)

// Generates a bunch of users and stores them in the database.
func generateTestUsers(t *testing.T, db *dbops.PgDB) {
	faker.Seed(1)
	for i := 0; i < 100; i++ {
		login := faker.Word()
		login = fmt.Sprintf("%s%d", login, i)
		user := &SystemUser{
			Login:    login,
			Email:    fmt.Sprintf("%s@example.org", login),
			Lastname: faker.LastName(),
			Name:     faker.FirstName(),
		}
		_, err := CreateUserWithPassword(db, user, faker.Word())
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
	}
	authOk, err := Authenticate(db, user, "admin")
	require.NoError(t, err)
	require.True(t, authOk)

	// The default user is by default in the super-admin group, which
	// should be returned.
	require.True(t, user.InGroup(&SystemGroup{Name: "super-admin"}))

	// Using wrong password should cause the authentication to fail.
	authOk, err = Authenticate(db, user, "wrong")
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
	}
	con, err := CreateUserWithPassword(db, user, "pass")
	require.False(t, con)
	require.NoError(t, err)

	require.Greater(t, user.ID, 0)

	authOk, err := Authenticate(db, user, "pass")
	require.NoError(t, err)
	require.True(t, authOk)

	// Modifying user's password should be possible.
	err = SetPassword(db, user.ID, "new password")
	require.NoError(t, err)

	// Authentication using old password should fail.
	authOk, err = Authenticate(db, user, "pass")
	require.NoError(t, err)
	require.False(t, authOk)

	// But it should pass with the new password.
	authOk, err = Authenticate(db, user, "new password")
	require.NoError(t, err)
	require.True(t, authOk)

	// If password is empty, it should remain unmodified in the database.
	err = SetPassword(db, user.ID, "")
	require.NoError(t, err)

	// Make sure that we can still authenticate (because the password hasn't changed).
	authOk, err = Authenticate(db, user, "new password")
	require.NoError(t, err)
	require.True(t, authOk)
}

// Tests that it is indicated when the user being updated is not found in
// the database.
func TestUpdateNoUser(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	user := &SystemUser{
		ID:       123456,
		Email:    "jan@example.org",
		Lastname: "Kowalski",
		Name:     "Jan",
	}
	con, err := UpdateUser(db, user)
	require.True(t, con)
	require.Error(t, err)
}

// Test that the authentication method and external ID cannot be changed.
func TestUpdateUserFixedMembers(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	user := &SystemUser{
		Email:                  "foo@bar",
		Lastname:               "Bar",
		Name:                   "Foo",
		AuthenticationMethodID: "foo",
		ExternalID:             "foo",
	}

	_, _ = CreateUser(db, user)

	// Act
	user.AuthenticationMethodID = "bar"
	user.ExternalID = "bar"
	conflict, err := UpdateUser(db, user)

	// Assert
	require.NoError(t, err)
	require.False(t, conflict)
	user, _ = GetUserByID(db, user.ID)
	require.EqualValues(t, "foo", user.AuthenticationMethodID)
	require.EqualValues(t, "foo", user.ExternalID)
}

// Test that the user is created properly.
func TestCreateUser(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	user := &SystemUser{
		Login:          "foobar",
		Email:          "foo@bar.org",
		Lastname:       "Bar",
		Name:           "Foo",
		ChangePassword: true,
	}

	// Act
	conflict, err := CreateUser(db, user)

	// Assert
	require.NoError(t, err)
	require.False(t, conflict)
	// Check if all data are inserted properly.
	dbUser, err := GetUserByID(db, user.ID)
	require.NoError(t, err)
	require.EqualValues(t, "foobar", dbUser.Login)
	require.EqualValues(t, "foo@bar.org", dbUser.Email)
	require.EqualValues(t, "Bar", dbUser.Lastname)
	require.EqualValues(t, "Foo", dbUser.Name)
	require.EqualValues(t, "internal", dbUser.AuthenticationMethodID)
	require.True(t, dbUser.ChangePassword)
	// Check if there is no password entry.
	password := &SystemUserPassword{}
	password.ID = dbUser.ID
	err = db.Model(password).WherePK().Select()
	require.ErrorIs(t, err, pg.ErrNoRows)
}

// Test that the user with password is created properly.
func TestCreateUserWithPassword(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	user := &SystemUser{
		Login:    "foobar",
		Email:    "foo@bar.org",
		Lastname: "Bar",
		Name:     "Foo",
	}

	// Act
	conflict, err := CreateUserWithPassword(db, user, "barfoo")

	// Assert
	require.NoError(t, err)
	require.False(t, conflict)
	// Check if all data are inserted properly.
	dbUser, err := GetUserByID(db, user.ID)
	require.NoError(t, err)
	require.EqualValues(t, "foobar", dbUser.Login)
	require.EqualValues(t, "foo@bar.org", dbUser.Email)
	require.EqualValues(t, "Bar", dbUser.Lastname)
	require.EqualValues(t, "Foo", dbUser.Name)
	require.EqualValues(t, "internal", dbUser.AuthenticationMethodID)
	require.False(t, dbUser.ChangePassword)
	// Check if there is a password entry.
	password := &SystemUserPassword{}
	password.ID = dbUser.ID
	err = db.Model(password).WherePK().Select()
	require.NoError(t, err)
}

// Test that the user with an empty password cannot be created.
func TestCreateUserWithEmptyPassword(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	user := &SystemUser{
		Login:    "foobar",
		Email:    "foo@bar.org",
		Lastname: "Bar",
		Name:     "Foo",
	}

	// Act
	conflict, err := CreateUserWithPassword(db, user, "")

	// Assert
	require.Error(t, err)
	require.False(t, conflict)
}

// Test that logins and emails of the users from the same authentication service
// must be unique.
func TestCreateUsersWithDuplicatedPasswordFromTheSameAuthentication(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	user := &SystemUser{
		Login:    "same",
		Email:    "same",
		Name:     "n",
		Lastname: "l",
	}

	_, _ = CreateUser(db, user)

	// Act
	_, errLogin := createUser(db, &SystemUser{
		Login:    "same",
		Email:    "unique",
		Name:     "n-login",
		Lastname: "l-login",
	})
	_, errEmail := createUser(db, &SystemUser{
		Login:    "unique",
		Email:    "same",
		Name:     "n-email",
		Lastname: "l-email",
	})

	// Assert
	require.Error(t, errLogin)
	require.Error(t, errEmail)
}

// Test that logins and emails of the users from the different authentication
// services may be duplicated.
func TestCreateUsersWithDuplicatedPasswordFromDifferentAuthentications(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	user := &SystemUser{
		Login:    "same",
		Email:    "same",
		Name:     "n",
		Lastname: "l",
	}

	_, _ = CreateUser(db, user)

	// Act
	_, errLogin := CreateUser(db, &SystemUser{
		Login:                  "same",
		Email:                  "unique",
		AuthenticationMethodID: "foo",
		ExternalID:             "ex-id-1",
		Name:                   "n-login",
		Lastname:               "l-login",
	})
	_, errEmail := CreateUser(db, &SystemUser{
		Login:                  "unique",
		Email:                  "same",
		AuthenticationMethodID: "foo",
		ExternalID:             "ex-id-2",
		Name:                   "n-email",
		Lastname:               "l-email",
	})

	// Assert
	require.NoError(t, errLogin)
	require.NoError(t, errEmail)
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
	}
	con, err := CreateUser(db, user)
	require.False(t, con)
	require.NoError(t, err)

	user = &SystemUser{
		Email:    "jan@example.org",
		Lastname: "Kowalski",
		Name:     "Jan",
	}
	con, err = CreateUser(db, user)
	require.True(t, con)
	require.Error(t, err)
}

// Test that the user is successfully deleted.
func TestDeleteUser(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Create new user.
	user := &SystemUser{
		Email:    "jan@example.org",
		Lastname: "Kowalski",
		Name:     "Jan",
	}
	con, err := CreateUser(db, user)
	require.False(t, con)
	require.NoError(t, err)

	err = DeleteUser(db, user)
	require.NoError(t, err)

	// An attempt to delete the same user should result in an error.
	err = DeleteUser(db, user)
	require.Error(t, err)
	require.ErrorIs(t, pkgerrors.Cause(err), ErrNotExists)
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
	}
	con, err := CreateUserWithPassword(db, user, "pass")
	require.False(t, con)
	require.NoError(t, err)

	require.Greater(t, user.ID, 0)

	// Set new password for the user.
	err = SetPassword(db, user.ID, "newpass")
	require.NoError(t, err)

	// Authenticate with the new password.
	authOk, err := Authenticate(db, user, "newpass")
	require.NoError(t, err)
	require.True(t, authOk)

	// Also check that authentication with invalid password fails.
	authOk, err = Authenticate(db, user, "NewPass")
	require.NoError(t, err)
	require.False(t, authOk)

	// And finally check the original password doesn't work anymore.
	authOk, err = Authenticate(db, user, "pass")
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
	}
	con, err := CreateUserWithPassword(db, user, "pass")
	require.False(t, con)
	require.NoError(t, err)

	// Provide invalid current password. The original password should not change.
	err = ChangePassword(db, user.ID, "invalid", "newpass")
	require.ErrorIs(t, err, ErrInvalidPassword)

	// Still should authenticate with the current password.
	authOk, err := Authenticate(db, user, "pass")
	require.NoError(t, err)
	require.True(t, authOk)

	// Provide valid password. The original password should be modified.
	err = ChangePassword(db, user.ID, "pass", "newpass")
	require.NoError(t, err)

	// Authenticate with the new password.
	authOk, err = Authenticate(db, user, "newpass")
	require.NoError(t, err)
	require.True(t, authOk)
}

// Tests that changing password resets the change password flag.
func TestChangePasswordResetChangeFlag(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	user := &SystemUser{
		Email:          "user@example.com",
		Lastname:       "Foo",
		Name:           "Bar",
		Login:          "foobar",
		ChangePassword: true,
	}

	_, _ = CreateUserWithPassword(db, user, "pass")

	// Act
	err := ChangePassword(db, user.ID, "pass", "newpass")

	// Assert
	require.NoError(t, err)
	user, _ = GetUserByID(db, user.ID)
	require.False(t, user.ChangePassword)
}

// Tests that all system users can be fetched from the database.
func TestGetUsers(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	generateTestUsers(t, db)

	users, total, err := GetUsersByPage(db, 0, 1000, nil, "", SortDirAny)
	require.NoError(t, err)
	require.Len(t, users, 101)
	require.EqualValues(t, 101, total)

	prevID := 0
	for _, u := range users {
		// Make sure that by default the users are ordered by ID.
		require.Greater(t, u.ID, prevID)
		prevID = u.ID
	}
}

// Tests that users can be fetched and sorted by login.
func TestGetUsersSortByLogin(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	generateTestUsers(t, db)

	users, total, err := GetUsersByPage(db, 0, 1000, nil, "login", SortDirAsc)
	require.NoError(t, err)
	require.Len(t, users, 101)
	require.EqualValues(t, 101, total)

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

	users, total, err := GetUsersByPage(db, 50, 10, nil, "", SortDirAny)
	require.NoError(t, err)
	require.Len(t, users, 10)
	require.EqualValues(t, 51, users[0].ID)
	require.EqualValues(t, 101, total)

	prevID := 0
	for _, u := range users {
		// Make sure that by default the users are ordered by ID.
		require.Greater(t, u.ID, prevID)
		prevID = u.ID
	}
}

// Tests that last page of users can be fetched without issues.
func TestGetUsersLastPage(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	generateTestUsers(t, db)

	users, total, err := GetUsersByPage(db, 90, 20, nil, "", SortDirAny)
	require.NoError(t, err)
	require.Len(t, users, 11)
	require.EqualValues(t, 91, users[0].ID)
	require.EqualValues(t, 101, total)

	prevID := 0
	for _, u := range users {
		// Make sure that by default the users are ordered by ID.
		require.Greater(t, u.ID, prevID)
		prevID = u.ID
	}
}

// Check filtering users by text.
func TestGetUsersPageByText(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	generateTestUsers(t, db)

	text := "3"
	users, total, err := GetUsersByPage(db, 0, 100, &text, "", SortDirAny)
	require.NoError(t, err)
	require.Len(t, users, 19)
	require.EqualValues(t, 19, total)
	require.EqualValues(t, "ut3", users[0].Login)
	require.EqualValues(t, "qui32", users[5].Login)
	require.EqualValues(t, "ut37", users[10].Login)
	require.EqualValues(t, "est93", users[18].Login)
}

// Tests that user can be fetched by Id.
func TestGetUserByID(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	generateTestUsers(t, db)

	users, total, err := GetUsersByPage(db, 0, 1000, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 101, total)

	user, err := GetUserByID(db, users[0].ID)
	require.NoError(t, err)
	require.NotNil(t, user)

	user, err = GetUserByID(db, 1234567)
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
		Groups: []*SystemGroup{
			{
				ID: SuperAdminGroupID,
			},
			{
				ID: AdminGroupID,
			},
		},
	}

	_, err := CreateUser(db, user)
	require.NoError(t, err)
	require.Greater(t, user.ID, 0)

	// Fetch the user by id. It should also return the groups it belongs to.
	returned, err := GetUserByID(db, user.ID)
	require.NotNil(t, returned)
	require.NoError(t, err)

	// Fetch the user by id. It should also return the groups it belongs to.
	returned, err = GetUserByID(db, user.ID)
	require.NotNil(t, returned)
	require.NoError(t, err)

	require.Len(t, returned.Groups, 2)
	require.True(t, returned.InGroup(&SystemGroup{Name: "super-admin"}))
	require.True(t, returned.InGroup(&SystemGroup{Name: "admin"}))

	// Remove the user from one of the groups.
	user.Groups = []*SystemGroup{
		{
			ID: 2,
		},
	}

	// Updating the user should also cause the groups to be updated.
	_, err = UpdateUser(db, user)
	require.NoError(t, err)

	returned, err = GetUserByID(db, user.ID)
	require.NotNil(t, returned)
	require.NoError(t, err)

	// Fetch the user by id. It should also return new groups.
	returned, err = GetUserByID(db, user.ID)
	require.NotNil(t, returned)
	require.NoError(t, err)

	// The groups should have been updated. One group should now be gone.
	require.Len(t, returned.Groups, 1)
	require.False(t, returned.InGroup(&SystemGroup{Name: "super-admin"}))
	require.True(t, returned.InGroup(&SystemGroup{Name: "admin"}))
}

// Test that the internal database ID is fetched by the authentication method
// and the external ID properly.
func TestGetUserIDByExternalID(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	user := &SystemUser{
		Login:                  "login",
		Email:                  "email@example.com",
		Lastname:               "Last",
		Name:                   "Name",
		AuthenticationMethodID: "method",
		ExternalID:             "externalID",
		Groups:                 []*SystemGroup{{ID: SuperAdminGroupID}},
	}

	_, _ = CreateUser(db, user)

	// Act
	nonExistUser, nonExistErr := GetUserByExternalID(db, "method", "nonExistingID")
	dbUser, err := GetUserByExternalID(db, "method", "externalID")

	// Assert
	require.NoError(t, nonExistErr)
	require.Zero(t, nonExistUser)

	require.NoError(t, err)
	require.EqualValues(t, user.ID, dbUser.ID)
	require.Len(t, dbUser.Groups, 1)
	require.EqualValues(t, SuperAdminGroupID, dbUser.Groups[0].ID)
}

// Test that user can be associated with a group and then the groups
// are returned along with the user.
func TestAddToGroupByID(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Create a user account.
	user := &SystemUser{
		Login:    "test",
		Email:    "test@example.org",
		Lastname: "Smith",
		Name:     "John",
	}
	_, err := CreateUser(db, user)
	require.NoError(t, err)
	require.Greater(t, user.ID, 0)

	// Associate the user with two predefined groups.
	added, err := user.AddToGroupByID(db, &SystemGroup{ID: 1})
	require.NoError(t, err)
	require.True(t, added)

	added, err = user.AddToGroupByID(db, &SystemGroup{ID: 2})
	require.NoError(t, err)
	require.True(t, added)

	// Fetch the user by id. It should also return the groups it belongs to.
	returned, err := GetUserByID(db, user.ID)
	require.NotNil(t, returned)
	require.NoError(t, err)

	require.Len(t, returned.Groups, 2)
	require.True(t, returned.InGroup(&SystemGroup{Name: "super-admin"}))
	require.True(t, returned.InGroup(&SystemGroup{Name: "admin"}))

	// Another attempt to add the user to the same group should be no-op.
	added, err = user.AddToGroupByID(db, &SystemGroup{ID: 1})
	require.NoError(t, err)
	require.False(t, added)
}

// Test that the user is successfully deleted if added to super-admin group.
func TestDeleteUserInGroup(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Create new user.
	user := &SystemUser{
		Email:    "jan@example.org",
		Lastname: "Kowalski",
		Name:     "Jan",
		Groups: []*SystemGroup{
			{
				ID: SuperAdminGroupID,
			},
		},
	}
	con, err := CreateUser(db, user)
	require.False(t, con)
	require.NoError(t, err)

	err = DeleteUser(db, user)
	require.NoError(t, err)

	// An attempt to delete the same user should result in an error.
	err = DeleteUser(db, user)
	require.Error(t, err)
	require.ErrorIs(t, pkgerrors.Cause(err), ErrNotExists)
}
