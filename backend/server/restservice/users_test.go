package restservice

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"isc.org/stork/hooks"
	"isc.org/stork/hooks/server/authenticationcallouts"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/server/gen/models"
	"isc.org/stork/server/gen/restapi/operations/users"
	"isc.org/stork/server/hookmanager"
	storkutil "isc.org/stork/util"
)

// Tests that create user account without necessary identifier fields is
// rejected via REST API.
func TestCreateUserMissingIdentifier(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()
	rapi, _ := NewRestAPI(dbSettings, db)

	// Both login and email missing here.
	su := dbmodel.SystemUser{
		Lastname: "Doe",
		Name:     "John",
	}

	// New user has empty email which conflicts with the default admin user empty email.
	params := users.CreateUserParams{
		Account: &models.UserAccount{
			User:     newRestUser(su),
			Password: storkutil.Ptr(models.Password("pass")),
		},
	}

	// Attempt to create the user and verify the response is negative.
	rsp := rapi.CreateUser(ctx, params)
	require.IsType(t, &users.CreateUserDefault{}, rsp)
	defaultRsp := rsp.(*users.CreateUserDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, "Failed to create new user account: missing identifier", *defaultRsp.Payload.Message)
}

// Tests that create user account with already existing login is
// rejected via REST API.
func TestCreateUserConflictLogin(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()
	rapi, _ := NewRestAPI(dbSettings, db)

	// Email is missing here.
	su := dbmodel.SystemUser{
		Login:    "admin",
		Email:    "",
		Lastname: "Doe",
		Name:     "John",
	}

	// New user has login which conflicts with the default admin user login.
	params := users.CreateUserParams{
		Account: &models.UserAccount{
			User:     newRestUser(su),
			Password: storkutil.Ptr(models.Password("pass")),
		},
	}

	// Attempt to create the user and verify the response is negative.
	rsp := rapi.CreateUser(ctx, params)
	require.IsType(t, &users.CreateUserDefault{}, rsp)
	defaultRsp := rsp.(*users.CreateUserDefault)
	require.Equal(t, http.StatusConflict, getStatusCode(*defaultRsp))
	require.Equal(t, "User account with provided login/email already exists", *defaultRsp.Payload.Message)
}

// Tests that create user account with already existing email is
// rejected via REST API.
func TestCreateUserConflictEmail(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_, _ = dbmodel.CreateUser(db, &dbmodel.SystemUser{
		Login:    "foo",
		Email:    "foo@example.com",
		Name:     "foo",
		Lastname: "bar",
	})

	ctx := context.Background()
	rapi, _ := NewRestAPI(dbSettings, db)

	// Login is missing here.
	su := dbmodel.SystemUser{
		Login:    "bar",
		Email:    "foo@example.com",
		Lastname: "Doe",
		Name:     "John",
	}

	// New user has an email which conflicts with another user email.
	params := users.CreateUserParams{
		Account: &models.UserAccount{
			User:     newRestUser(su),
			Password: storkutil.Ptr(models.Password("pass")),
		},
	}

	// Attempt to create the user and verify the response is negative.
	rsp := rapi.CreateUser(ctx, params)
	require.IsType(t, &users.CreateUserDefault{}, rsp)
	defaultRsp := rsp.(*users.CreateUserDefault)
	require.Equal(t, http.StatusConflict, getStatusCode(*defaultRsp))
	require.Equal(t, "User account with provided login/email already exists", *defaultRsp.Payload.Message)
}

// Tests that creating a user account with an empty email is allowed when
// another account with an empty email exists.
func TestCreateUserConflictEmailEmpty(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_, _ = dbmodel.CreateUser(db, &dbmodel.SystemUser{
		Login:    "foo",
		Email:    "",
		Name:     "foo",
		Lastname: "bar",
	})

	ctx := context.Background()
	rapi, _ := NewRestAPI(dbSettings, db)

	su := dbmodel.SystemUser{
		Login:    "bar",
		Email:    "",
		Lastname: "Doe",
		Name:     "John",
	}

	// New user has an empty email but it shouldn't conflict with another user having an empty email.
	params := users.CreateUserParams{
		Account: &models.UserAccount{
			User:     newRestUser(su),
			Password: storkutil.Ptr(models.Password("pass")),
		},
	}

	// Attempt to create the user and verify the response is positive.
	rsp := rapi.CreateUser(ctx, params)
	require.IsType(t, &users.CreateUserOK{}, rsp)
}

// Tests that create user account with already existing email and empty login is
// rejected via REST API.
func TestCreateUserConflictEmailEmptyLogin(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_, _ = dbmodel.CreateUser(db, &dbmodel.SystemUser{
		Email:    "foo@example.com",
		Name:     "foo",
		Lastname: "bar",
	})

	ctx := context.Background()
	rapi, _ := NewRestAPI(dbSettings, db)

	// Login is missing here.
	su := dbmodel.SystemUser{
		Email:    "foo@example.com",
		Lastname: "Doe",
		Name:     "John",
	}

	// New user has an email which conflicts with another user email.
	params := users.CreateUserParams{
		Account: &models.UserAccount{
			User:     newRestUser(su),
			Password: storkutil.Ptr(models.Password("pass")),
		},
	}

	// Attempt to create the user and verify the response is negative.
	rsp := rapi.CreateUser(ctx, params)
	require.IsType(t, &users.CreateUserDefault{}, rsp)
	defaultRsp := rsp.(*users.CreateUserDefault)
	require.Equal(t, http.StatusConflict, getStatusCode(*defaultRsp))
	require.Equal(t, "User account with provided login/email already exists", *defaultRsp.Payload.Message)
}

// Tests that create user account with empty params is rejected via REST API.
func TestCreateUserEmptyParams(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()
	rapi, _ := NewRestAPI(dbSettings, db)

	// Try empty params - it should raise an error
	params := users.CreateUserParams{}
	rsp := rapi.CreateUser(ctx, params)
	require.IsType(t, &users.CreateUserDefault{}, rsp)
	defaultRsp := rsp.(*users.CreateUserDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, "Failed to create new user account: missing data", *defaultRsp.Payload.Message)
}

// Test that create user account with empty first and last names is rejected
// via REST API.
func TestCreateUserEmptyFirstAndLastNames(t *testing.T) {
	// Arrange
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()
	rapi, _ := NewRestAPI(dbSettings, db)

	params := users.CreateUserParams{
		Account: &models.UserAccount{
			Password: storkutil.Ptr(models.Password("pass")),
			User: &models.User{
				Login: "foo",
			},
		},
	}

	// Act
	rsp := rapi.CreateUser(ctx, params)

	// Assert
	require.IsType(t, &users.CreateUserDefault{}, rsp)
	defaultRsp := rsp.(*users.CreateUserDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, "Failed to create new user account: missing data", *defaultRsp.Payload.Message)
}

// Tests that create user account with empty request is rejected via REST API.
func TestCreateUserEmptyRequest(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()
	rapi, _ := NewRestAPI(dbSettings, db)

	// Try empty request - it should raise an error
	params := users.CreateUserParams{
		Account: &models.UserAccount{},
	}
	rsp := rapi.CreateUser(ctx, params)
	require.IsType(t, &users.CreateUserDefault{}, rsp)
	defaultRsp := rsp.(*users.CreateUserDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, "Failed to create new user account: missing data", *defaultRsp.Payload.Message)
}

// Tests that create user account with missing data is rejected via REST API.
func TestCreateUserMissingData(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()
	rapi, _ := NewRestAPI(dbSettings, db)

	// Try missing data - it should raise an error
	params := users.CreateUserParams{
		Account: &models.UserAccount{
			User: &models.User{},
		},
	}
	rsp := rapi.CreateUser(ctx, params)
	require.IsType(t, &users.CreateUserDefault{}, rsp)
	defaultRsp := rsp.(*users.CreateUserDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, "Failed to create new user account: missing data", *defaultRsp.Payload.Message)
}

// Tests that user account can be created via REST API.
func TestCreateUser(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()
	rapi, _ := NewRestAPI(dbSettings, db)

	// Create the user and verify the response.
	su := dbmodel.SystemUser{
		Email:    "jb@example.org",
		Lastname: "Born",
		Login:    "jb",
		Name:     "John",
	}
	params := users.CreateUserParams{
		Account: &models.UserAccount{
			User:     newRestUser(su),
			Password: storkutil.Ptr(models.Password("pass")),
		},
	}

	rsp := rapi.CreateUser(ctx, params)
	require.IsType(t, &users.CreateUserOK{}, rsp)
	okRsp := rsp.(*users.CreateUserOK)
	require.Greater(t, *okRsp.Payload.ID, int64(0))
	require.Equal(t, okRsp.Payload.Email, su.Email)
	require.Equal(t, okRsp.Payload.Lastname, su.Lastname)
	require.Equal(t, okRsp.Payload.Login, su.Login)
	require.Equal(t, okRsp.Payload.Name, su.Name)

	// Also check that the user is indeed in the database.
	returned, err := dbmodel.GetUserByID(db, int(*okRsp.Payload.ID))
	require.NoError(t, err)
	require.Equal(t, su.Login, returned.Login)

	// An attempt to create the same user should fail with HTTP error 409.
	rsp = rapi.CreateUser(ctx, params)
	require.IsType(t, &users.CreateUserDefault{}, rsp)
	defaultRsp := rsp.(*users.CreateUserDefault)
	require.Equal(t, 409, getStatusCode(*defaultRsp))
}

// Tests that delete user account with empty params is rejected via REST API.
func TestDeleteUserEmptyParams(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()

	rapi, err := NewRestAPI(dbSettings, db)
	require.NoError(t, err)

	// Create session manager.
	ctx, err = rapi.SessionManager.Load(ctx, "")
	require.NoError(t, err)

	// Create new super-admin user in the database.
	su := dbmodel.SystemUser{
		Email:    "john@example.org",
		Lastname: "White",
		Name:     "John",
		Groups: []*dbmodel.SystemGroup{
			{
				ID: dbmodel.SuperAdminGroupID,
			},
		},
	}
	con, err := dbmodel.CreateUser(db, &su)
	require.False(t, con)
	require.NoError(t, err)

	err = rapi.SessionManager.LoginHandler(ctx, &su)
	require.NoError(t, err)

	// Try empty params - it should raise an error
	params := users.DeleteUserParams{}
	rsp := rapi.DeleteUser(ctx, params)
	require.IsType(t, &users.DeleteUserDefault{}, rsp)
	defaultRsp := rsp.(*users.DeleteUserDefault)
	require.Equal(t, http.StatusNotFound, getStatusCode(*defaultRsp))
	require.Equal(t, "Failed to find user with ID 0 in the database", *defaultRsp.Payload.Message)
}

// Tests that delete user account with invalid user ID is rejected via REST API.
func TestDeleteUserInvalidUserID(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()

	rapi, err := NewRestAPI(dbSettings, db)
	require.NoError(t, err)

	// Create session manager.
	ctx, err = rapi.SessionManager.Load(ctx, "")
	require.NoError(t, err)

	// Create new super-admin user in the database.
	su := dbmodel.SystemUser{
		Email:    "john@example.org",
		Lastname: "White",
		Name:     "John",
		Groups: []*dbmodel.SystemGroup{
			{
				ID: dbmodel.SuperAdminGroupID,
			},
		},
	}
	con, err := dbmodel.CreateUser(db, &su)
	require.False(t, con)
	require.NoError(t, err)

	err = rapi.SessionManager.LoginHandler(ctx, &su)
	require.NoError(t, err)

	// Try using invalid user ID - it should raise an error
	params := users.DeleteUserParams{
		ID: int64(su.ID + 1),
	}

	rsp := rapi.DeleteUser(ctx, params)
	require.IsType(t, &users.DeleteUserDefault{}, rsp)
	defaultRsp := rsp.(*users.DeleteUserDefault)
	require.Equal(t, http.StatusNotFound, getStatusCode(*defaultRsp))
	require.Equal(t, "Failed to find user with ID 3 in the database", *defaultRsp.Payload.Message)
}

// Tests that delete user account with same user ID as current session is rejected via REST API.
func TestDeleteUserSameUserAsSession(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()

	rapi, err := NewRestAPI(dbSettings, db)
	require.NoError(t, err)

	// Create session manager.
	ctx, err = rapi.SessionManager.Load(ctx, "")
	require.NoError(t, err)

	// Create new super-admin user in the database.
	su := dbmodel.SystemUser{
		Email:    "john@example.org",
		Lastname: "White",
		Name:     "John",
		Groups: []*dbmodel.SystemGroup{
			{
				ID: dbmodel.SuperAdminGroupID,
			},
		},
	}
	con, err := dbmodel.CreateUser(db, &su)
	require.False(t, con)
	require.NoError(t, err)

	err = rapi.SessionManager.LoginHandler(ctx, &su)
	require.NoError(t, err)

	// Try same user ID as current session - it should raise an error - trying to delete itself
	params := users.DeleteUserParams{
		ID: int64(su.ID),
	}

	rsp := rapi.DeleteUser(ctx, params)
	require.IsType(t, &users.DeleteUserDefault{}, rsp)
	defaultRsp := rsp.(*users.DeleteUserDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, "User account with provided login/email tries to delete itself", *defaultRsp.Payload.Message)
}

// Tests that user account can be deleted via REST API.
func TestDeleteUser(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()

	rapi, err := NewRestAPI(dbSettings, db)
	require.NoError(t, err)

	// Create session manager.
	ctx, err = rapi.SessionManager.Load(ctx, "")
	require.NoError(t, err)

	// Create new super-admin user in the database.
	su := dbmodel.SystemUser{
		Email:    "john@example.org",
		Lastname: "White",
		Name:     "John",
		Groups: []*dbmodel.SystemGroup{
			{
				ID: dbmodel.SuperAdminGroupID,
			},
		},
	}
	con, err := dbmodel.CreateUser(db, &su)
	require.False(t, con)
	require.NoError(t, err)

	// Create new user in the database.
	su2 := dbmodel.SystemUser{
		Email:    "jan@example.org",
		Lastname: "Kowalski",
		Name:     "Jan",
	}
	con, err = dbmodel.CreateUser(db, &su2)
	require.False(t, con)
	require.NoError(t, err)

	err = rapi.SessionManager.LoginHandler(ctx, &su)
	require.NoError(t, err)
	params := users.DeleteUserParams{
		ID: int64(su2.ID),
	}

	rsp := rapi.DeleteUser(ctx, params)
	require.IsType(t, &users.DeleteUserOK{}, rsp)
	okRsp := rsp.(*users.DeleteUserOK)
	require.Greater(t, *okRsp.Payload.ID, int64(0))
	require.Equal(t, okRsp.Payload.Email, su2.Email)
	require.Equal(t, okRsp.Payload.Lastname, su2.Lastname)
	require.Equal(t, okRsp.Payload.Login, su2.Login)
	require.Equal(t, okRsp.Payload.Name, su2.Name)

	// Also check that the user is indeed not in the database.
	returned, err := dbmodel.GetUserByID(db, int(*okRsp.Payload.ID))
	require.NoError(t, err)
	require.Nil(t, returned)

	// An attempt to delete the same user should fail with HTTP error 409.
	rsp = rapi.DeleteUser(ctx, params)
	require.IsType(t, &users.DeleteUserDefault{}, rsp)
	defaultRsp := rsp.(*users.DeleteUserDefault)
	require.Equal(t, http.StatusNotFound, getStatusCode(*defaultRsp))
	require.Equal(t, "Failed to find user with ID 3 in the database", *defaultRsp.Payload.Message)
}

// Tests that super-admin user account can be deleted via REST API.
func TestDeleteUserInGroup(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()

	rapi, err := NewRestAPI(dbSettings, db)
	require.NoError(t, err)

	// Create session manager.
	ctx, err = rapi.SessionManager.Load(ctx, "")
	require.NoError(t, err)

	// Create new super-admin user in the database.
	su := dbmodel.SystemUser{
		Email:    "john@example.org",
		Lastname: "White",
		Name:     "John",
		Groups: []*dbmodel.SystemGroup{
			{
				ID: dbmodel.SuperAdminGroupID,
			},
		},
	}
	con, err := dbmodel.CreateUser(db, &su)
	require.False(t, con)
	require.NoError(t, err)

	// Create new user in the database.
	su2 := dbmodel.SystemUser{
		Email:    "jan@example.org",
		Lastname: "Kowalski",
		Name:     "Jan",
		Groups: []*dbmodel.SystemGroup{
			{
				ID: dbmodel.SuperAdminGroupID,
			},
		},
	}
	con, err = dbmodel.CreateUser(db, &su2)
	require.False(t, con)
	require.NoError(t, err)

	err = rapi.SessionManager.LoginHandler(ctx, &su)
	require.NoError(t, err)

	params := users.DeleteUserParams{
		ID: int64(su2.ID),
	}

	rsp := rapi.DeleteUser(ctx, params)
	require.IsType(t, &users.DeleteUserOK{}, rsp)
	okRsp := rsp.(*users.DeleteUserOK)
	require.Greater(t, *okRsp.Payload.ID, int64(0))
	require.Equal(t, okRsp.Payload.Email, su2.Email)
	require.Equal(t, okRsp.Payload.Lastname, su2.Lastname)
	require.Equal(t, okRsp.Payload.Login, su2.Login)
	require.Equal(t, okRsp.Payload.Name, su2.Name)

	// Also check that the user is indeed not in the database.
	returned, err := dbmodel.GetUserByID(db, int(*okRsp.Payload.ID))
	require.NoError(t, err)
	require.Nil(t, returned)

	// An attempt to delete the same user should fail with HTTP error 409.
	rsp = rapi.DeleteUser(ctx, params)
	require.IsType(t, &users.DeleteUserDefault{}, rsp)
	defaultRsp := rsp.(*users.DeleteUserDefault)
	require.Equal(t, http.StatusNotFound, getStatusCode(*defaultRsp))
	require.Equal(t, "Failed to find user with ID 3 in the database", *defaultRsp.Payload.Message)
}

// Tests that update user account with empty params is rejected via REST API.
func TestUpdateUserEmptyParams(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()

	rapi, err := NewRestAPI(dbSettings, db)
	require.NoError(t, err)

	// Create new user in the database.
	su := dbmodel.SystemUser{
		Email:    "jan@example.org",
		Lastname: "Kowalski",
		Name:     "Jan",
	}
	con, err := dbmodel.CreateUser(db, &su)
	require.False(t, con)
	require.NoError(t, err)

	// Try empty params - it should raise an error
	params := users.UpdateUserParams{}
	rsp := rapi.UpdateUser(ctx, params)
	require.IsType(t, &users.UpdateUserDefault{}, rsp)
	defaultRsp := rsp.(*users.UpdateUserDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, "Failed to update user account: missing data", *defaultRsp.Payload.Message)
}

// Tests that update user account with empty first and last names is rejected
// via REST API for an internal user.
func TestUpdateUserEmptyParamsForInternalUser(t *testing.T) {
	// Arrange
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()
	rapi, _ := NewRestAPI(dbSettings, db)

	su := dbmodel.SystemUser{
		Email:    "jan@example.org",
		Lastname: "Kowalski",
		Name:     "Jan",
	}
	_, _ = dbmodel.CreateUser(db, &su)

	// Act
	user := newRestUser(su)
	user.Name = ""
	user.Lastname = ""

	params := users.UpdateUserParams{
		Account: &models.UserAccount{
			User: user,
		},
	}
	rsp := rapi.UpdateUser(ctx, params)

	// Assert
	require.IsType(t, &users.UpdateUserDefault{}, rsp)
	defaultRsp := rsp.(*users.UpdateUserDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, "Failed to update user account: missing first or last name", *defaultRsp.Payload.Message)
}

// Tests that update user account with empty first and last names is rejected
// via REST API for an external user.
func TestUpdateUserEmptyParamsForExternalUser(t *testing.T) {
	// Arrange
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()
	rapi, _ := NewRestAPI(dbSettings, db)

	su := dbmodel.SystemUser{
		Email:                  "jan@example.org",
		Lastname:               "Kowalski",
		Name:                   "Jan",
		AuthenticationMethodID: "external",
		ExternalID:             "42",
	}
	_, _ = dbmodel.CreateUser(db, &su)

	// Act
	user := newRestUser(su)
	user.Name = ""
	user.Lastname = ""

	params := users.UpdateUserParams{
		Account: &models.UserAccount{
			User: user,
		},
	}
	rsp := rapi.UpdateUser(ctx, params)

	// Assert
	require.IsType(t, &users.UpdateUserOK{}, rsp)
}

// Tests that update user account with empty request is rejected via REST API.
func TestUpdateUserEmptyRequest(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()

	rapi, err := NewRestAPI(dbSettings, db)
	require.NoError(t, err)

	// Create new user in the database.
	su := dbmodel.SystemUser{
		Email:    "jan@example.org",
		Lastname: "Kowalski",
		Name:     "Jan",
	}
	con, err := dbmodel.CreateUser(db, &su)
	require.False(t, con)
	require.NoError(t, err)

	// Try empty request - it should raise an error
	params := users.UpdateUserParams{
		Account: &models.UserAccount{},
	}
	rsp := rapi.UpdateUser(ctx, params)
	require.IsType(t, &users.UpdateUserDefault{}, rsp)
	defaultRsp := rsp.(*users.UpdateUserDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, "Failed to update user account: missing data", *defaultRsp.Payload.Message)
}

// Tests that update user account with missing data is rejected via REST API.
func TestUpdateUserMissingData(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()

	rapi, err := NewRestAPI(dbSettings, db)
	require.NoError(t, err)

	// Create new user in the database.
	su := dbmodel.SystemUser{
		Email:    "jan@example.org",
		Lastname: "Kowalski",
		Name:     "Jan",
	}
	con, err := dbmodel.CreateUser(db, &su)
	require.False(t, con)
	require.NoError(t, err)

	// Try missing data - it should raise an error
	params := users.UpdateUserParams{
		Account: &models.UserAccount{
			User: &models.User{},
		},
	}
	rsp := rapi.UpdateUser(ctx, params)
	require.IsType(t, &users.UpdateUserDefault{}, rsp)
	defaultRsp := rsp.(*users.UpdateUserDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, "Failed to update user account: missing data", *defaultRsp.Payload.Message)
}

// Tests that update user account with invalid user ID is rejected via REST API.
func TestUpdateUserInvalidUserID(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()

	rapi, err := NewRestAPI(dbSettings, db)
	require.NoError(t, err)

	// Create new user in the database.
	su := dbmodel.SystemUser{
		Email:    "jan@example.org",
		Lastname: "Kowalski",
		Name:     "Jan",
	}
	con, err := dbmodel.CreateUser(db, &su)
	require.False(t, con)
	require.NoError(t, err)

	// An attempt to update non-existing user (non matching ID) should
	// result in an error 409.
	su.ID = 123
	params := users.UpdateUserParams{
		Account: &models.UserAccount{
			User:     newRestUser(su),
			Password: storkutil.Ptr(models.Password("pass")),
		},
	}
	rsp := rapi.UpdateUser(ctx, params)
	require.IsType(t, &users.UpdateUserDefault{}, rsp)
	defaultRsp := rsp.(*users.UpdateUserDefault)
	require.Equal(t, 409, getStatusCode(*defaultRsp))
}

// Tests that user account with password can be updated via REST API.
func TestUpdateUserWithPassword(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()

	rapi, err := NewRestAPI(dbSettings, db)
	require.NoError(t, err)

	// Create new user in the database.
	su := dbmodel.SystemUser{
		Email:          "jan@example.org",
		Lastname:       "Kowalski",
		Name:           "Jan",
		ChangePassword: false,
	}
	con, err := dbmodel.CreateUserWithPassword(db, &su, "secret")
	require.False(t, con)
	require.NoError(t, err)

	// Modify some values and update the user.
	su.Lastname = "Born"
	params := users.UpdateUserParams{
		Account: &models.UserAccount{
			User:     newRestUser(su),
			Password: storkutil.Ptr(models.Password("pass")),
		},
	}
	rsp := rapi.UpdateUser(ctx, params)
	require.IsType(t, &users.UpdateUserOK{}, rsp)

	// Also check that the user has been updated in the database.
	returned, err := dbmodel.GetUserByID(db, su.ID)
	require.NoError(t, err)
	require.Equal(t, su.Lastname, returned.Lastname)

	// An attempt to update non-existing user (non matching ID) should
	// result in an error 409.
	su.ID = 123
	params = users.UpdateUserParams{
		Account: &models.UserAccount{
			User:     newRestUser(su),
			Password: storkutil.Ptr(models.Password("pass")),
		},
	}
	rsp = rapi.UpdateUser(ctx, params)
	require.IsType(t, &users.UpdateUserDefault{}, rsp)
	defaultRsp := rsp.(*users.UpdateUserDefault)
	require.Equal(t, 409, getStatusCode(*defaultRsp))
}

// Tests that user account without password can be updated via REST API.
func TestUpdateUserWithoutPassword(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()

	rapi, err := NewRestAPI(dbSettings, db)
	require.NoError(t, err)

	// Create new user in the database.
	su := dbmodel.SystemUser{
		Email:          "jan@example.org",
		Lastname:       "Kowalski",
		Name:           "Jan",
		ChangePassword: false,
	}
	con, err := dbmodel.CreateUser(db, &su)
	require.False(t, con)
	require.NoError(t, err)

	// Modify some values and update the user.
	su.Lastname = "Born"
	params := users.UpdateUserParams{
		Account: &models.UserAccount{
			User: newRestUser(su),
		},
	}
	rsp := rapi.UpdateUser(ctx, params)
	require.IsType(t, &users.UpdateUserOK{}, rsp)

	// Also check that the user has been updated in the database.
	returned, err := dbmodel.GetUserByID(db, su.ID)
	require.NoError(t, err)
	require.Equal(t, su.Lastname, returned.Lastname)

	// An attempt to update non-existing user (non matching ID) should
	// result in an error 409.
	su.ID = 123
	params = users.UpdateUserParams{
		Account: &models.UserAccount{
			User: newRestUser(su),
		},
	}
	rsp = rapi.UpdateUser(ctx, params)
	require.IsType(t, &users.UpdateUserDefault{}, rsp)
	defaultRsp := rsp.(*users.UpdateUserDefault)
	require.Equal(t, 409, getStatusCode(*defaultRsp))
}

// Tests that update user password with missing data is rejected via REST API.
func TestUpdateUserPasswordMissingData(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()
	rapi, err := NewRestAPI(dbSettings, db)
	require.NoError(t, err)

	// Create new user in the database.
	user := &dbmodel.SystemUser{
		Email:          "jan@example.org",
		Lastname:       "Kowalski",
		Name:           "Jan",
		ChangePassword: false,
	}
	con, err := dbmodel.CreateUser(db, user)
	require.False(t, con)
	require.NoError(t, err)

	// Try empty request - it should raise an error.
	params := users.UpdateUserPasswordParams{
		ID: int64(user.ID),
	}
	rsp := rapi.UpdateUserPassword(ctx, params)
	require.IsType(t, &users.UpdateUserPasswordDefault{}, rsp)
	defaultRsp := rsp.(*users.UpdateUserPasswordDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, "Failed to update password for user: missing data", *defaultRsp.Payload.Message)
}

// Tests that user password can be updated via REST API if user account has
// assigned the password.
func TestUpdateUserPassword(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()
	rapi, err := NewRestAPI(dbSettings, db)
	require.NoError(t, err)

	// Create new user in the database.
	user := &dbmodel.SystemUser{
		Email:    "jan@example.org",
		Lastname: "Kowalski",
		Name:     "Jan",
	}
	con, err := dbmodel.CreateUserWithPassword(db, user, "pass")
	require.False(t, con)
	require.NoError(t, err)

	// Update user password via the API.
	params := users.UpdateUserPasswordParams{
		ID: int64(user.ID),
		Passwords: &models.PasswordChange{
			Newpassword: storkutil.Ptr(models.Password("updated")),
			Oldpassword: storkutil.Ptr(models.Password("pass")),
		},
	}
	rsp := rapi.UpdateUserPassword(ctx, params)
	require.IsType(t, &users.UpdateUserPasswordOK{}, rsp)

	// An attempt to update the password again should result in
	// HTTP error 400 (Bad Request) because the old password is
	// not matching anymore.
	params = users.UpdateUserPasswordParams{
		ID: int64(user.ID),
		Passwords: &models.PasswordChange{
			Newpassword: storkutil.Ptr(models.Password("updated")),
			Oldpassword: storkutil.Ptr(models.Password("pass")),
		},
	}
	rsp = rapi.UpdateUserPassword(ctx, params)
	require.IsType(t, &users.UpdateUserPasswordDefault{}, rsp)
	defaultRsp := rsp.(*users.UpdateUserPasswordDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
}

// Tests that updating the user password via REST API causes the change
// password flag to be reset.
func TestUpdateUserPasswordResetChangePasswordFlag(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()
	rapi, err := NewRestAPI(dbSettings, db)
	require.NoError(t, err)

	// Create new user in the database.
	user := &dbmodel.SystemUser{
		Email:          "jan@example.org",
		Lastname:       "Kowalski",
		Name:           "Jan",
		ChangePassword: true,
	}
	con, err := dbmodel.CreateUserWithPassword(db, user, "pass")
	require.False(t, con)
	require.NoError(t, err)

	// Update user password via the API.
	params := users.UpdateUserPasswordParams{
		ID: int64(user.ID),
		Passwords: &models.PasswordChange{
			Newpassword: storkutil.Ptr(models.Password("updated")),
			Oldpassword: storkutil.Ptr(models.Password("pass")),
		},
	}
	rsp := rapi.UpdateUserPassword(ctx, params)
	require.IsType(t, &users.UpdateUserPasswordOK{}, rsp)
	user, _ = dbmodel.GetUserByID(db, user.ID)
	require.False(t, user.ChangePassword)
}

// Tests that user password can't be updated via REST API if user account doesn't
// have assigned the password.
func TestUpdateUserPasswordForPasswordlessUser(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()
	rapi, err := NewRestAPI(dbSettings, db)
	require.NoError(t, err)

	// Create new user in the database.
	user := &dbmodel.SystemUser{
		Email:    "jan@example.org",
		Lastname: "Kowalski",
		Name:     "Jan",
	}
	con, err := dbmodel.CreateUser(db, user)
	require.False(t, con)
	require.NoError(t, err)

	// Update user password via the API.
	params := users.UpdateUserPasswordParams{
		ID: int64(user.ID),
		Passwords: &models.PasswordChange{
			Newpassword: storkutil.Ptr(models.Password("updated")),
		},
	}
	rsp := rapi.UpdateUserPassword(ctx, params)
	require.IsType(t, &users.UpdateUserPasswordDefault{}, rsp)

	defaultRsp := rsp.(*users.UpdateUserPasswordDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
}

// Tests that multiple groups can be fetched from the database.
func TestGetGroups(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()
	rapi, _ := NewRestAPI(dbSettings, db)

	params := users.GetGroupsParams{}

	rsp := rapi.GetGroups(ctx, params)
	require.IsType(t, &users.GetGroupsOK{}, rsp)
	rspOK := rsp.(*users.GetGroupsOK)

	groups := rspOK.Payload
	require.NotNil(t, groups.Items)
	require.GreaterOrEqual(t, 2, len(groups.Items))
}

// Tests that user information can be retrieved via REST API.
func TestGetUsers(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()
	rapi, err := NewRestAPI(dbSettings, db)
	require.NoError(t, err)

	// Create new user in the database.
	user := &dbmodel.SystemUser{
		Email:                  "jd@example.org",
		Lastname:               "Doe",
		Login:                  "johndoe",
		Name:                   "John",
		AuthenticationMethodID: "LDAP",
		ExternalID:             "34ddae6b-f702-469d-8796-63c853496c49",
	}
	con, err := dbmodel.CreateUser(db, user)
	require.False(t, con)
	require.NoError(t, err)

	// Try empty request - it should work
	params := users.GetUsersParams{}

	rsp := rapi.GetUsers(ctx, params)
	require.IsType(t, &users.GetUsersOK{}, rsp)
	okRsp := rsp.(*users.GetUsersOK)
	require.EqualValues(t, 2, okRsp.Payload.Total) // we expect 2 users (admin and john doe)
	require.Len(t, okRsp.Payload.Items, 2)         // make sure there's entry with 2 users

	// Retrieve all users using the API
	var (
		limit int64 = 99999
		start int64
		text  string
	)

	params = users.GetUsersParams{
		Limit: &limit,
		Start: &start,
		Text:  &text,
	}

	rsp = rapi.GetUsers(ctx, params)
	require.IsType(t, &users.GetUsersOK{}, rsp)
	okRsp = rsp.(*users.GetUsersOK)
	require.EqualValues(t, 2, okRsp.Payload.Total) // we expect 2 users (admin and john doe)
	require.Len(t, okRsp.Payload.Items, 2)         // make sure there's entry with 2 users

	// Check the default user (admin)
	require.Equal(t, "admin", okRsp.Payload.Items[0].Login)
	require.Equal(t, "admin", okRsp.Payload.Items[0].Name)
	require.Equal(t, "admin", okRsp.Payload.Items[0].Lastname)
	require.Equal(t, "", okRsp.Payload.Items[0].Email)
	require.Equal(t, dbmodel.AuthenticationMethodIDInternal, *okRsp.Payload.Items[0].AuthenticationMethodID)
	require.Empty(t, okRsp.Payload.Items[0].ExternalID)

	// Check the user we just added
	require.Equal(t, "johndoe", okRsp.Payload.Items[1].Login)
	require.Equal(t, "John", okRsp.Payload.Items[1].Name)
	require.Equal(t, "Doe", okRsp.Payload.Items[1].Lastname)
	require.Equal(t, "jd@example.org", okRsp.Payload.Items[1].Email)
	require.Equal(t, "LDAP", *okRsp.Payload.Items[1].AuthenticationMethodID)
	require.Equal(t, "34ddae6b-f702-469d-8796-63c853496c49", okRsp.Payload.Items[1].ExternalID)

	// Make sure the ID of the new user is different.
	// TODO: implement new test that add 100 users and verifies uniqueness of their IDs
	require.NotEqual(t, *okRsp.Payload.Items[0].ID, *okRsp.Payload.Items[1].ID)
}

// Tests that user information can be retrieved via REST API.
func TestGetUser(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()
	rapi, err := NewRestAPI(dbSettings, db)
	require.NoError(t, err)

	// Create new user in the database.
	user := &dbmodel.SystemUser{
		Email:          "jd@example.org",
		Lastname:       "Doe",
		Login:          "johndoe",
		Name:           "John",
		ChangePassword: true,
		Groups:         []*dbmodel.SystemGroup{{ID: dbmodel.AdminGroupID}},
	}
	con, err := dbmodel.CreateUser(db, user)
	require.False(t, con)
	require.NoError(t, err)

	// Retrieve the user with ID=2 using the API (that's the new user user we just added)
	var id int64 = 2
	params := users.GetUserParams{
		ID: id,
	}

	rsp := rapi.GetUser(ctx, params)
	require.IsType(t, &users.GetUserOK{}, rsp)
	okRsp := rsp.(*users.GetUserOK)
	require.Equal(t, user.Email, okRsp.Payload.Email) // we expect 2 users (admin and john doe)
	require.Equal(t, id, *okRsp.Payload.ID)           // user ID
	require.Equal(t, user.Login, okRsp.Payload.Login)
	require.Equal(t, user.Name, okRsp.Payload.Name)
	require.Equal(t, user.Lastname, okRsp.Payload.Lastname)
	require.Equal(t, user.ChangePassword, okRsp.Payload.ChangePassword)

	require.Len(t, okRsp.Payload.Groups, 1)
	require.EqualValues(t, dbmodel.AdminGroupID, okRsp.Payload.Groups[0])
}

// Tests that new session can not be created using empty params.
func TestCreateSessionEmptyParams(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()
	hookManager := hookmanager.NewHookManager()
	rapi, _ := NewRestAPI(dbSettings, db, hookManager)

	user := &dbmodel.SystemUser{
		Email:    "jan@example.org",
		Lastname: "Kowalski",
		Name:     "Jan",
	}
	con, err := dbmodel.CreateUser(db, user)
	require.False(t, con)
	require.NoError(t, err)

	// try empty request - it should raise an error
	params := users.CreateSessionParams{}
	rsp := rapi.CreateSession(ctx, params)
	require.IsType(t, &users.CreateSessionBadRequest{}, rsp)
}

// Tests that new session can not be created using invalid credentials.
func TestCreateSessionInvalidCredentials(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()
	hookManager := hookmanager.NewHookManager()
	rapi, _ := NewRestAPI(dbSettings, db, hookManager)

	user := &dbmodel.SystemUser{
		Email:    "jan@example.org",
		Lastname: "Kowalski",
		Name:     "Jan",
	}
	con, err := dbmodel.CreateUserWithPassword(db, user, "pass")
	require.False(t, con)
	require.NoError(t, err)

	// for next tests there is some data required in the context, let's prepare it
	ctx2, err := rapi.SessionManager.Load(ctx, "")
	require.NoError(t, err)

	params := users.CreateSessionParams{
		Credentials: &models.SessionCredentials{
			Identifier: &user.Email,
		},
	}
	// provide wrong credentials - it should raise an error
	bad := "bad pass"
	params.Credentials.Secret = &bad
	rsp := rapi.CreateSession(ctx2, params)
	require.IsType(t, &users.CreateSessionBadRequest{}, rsp)
}

// Tests that new session can be created for a logged user that uses the
// internal authentication.
func TestCreateSession(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()
	hookManager := hookmanager.NewHookManager()
	rapi, _ := NewRestAPI(dbSettings, db, hookManager)

	user := &dbmodel.SystemUser{
		Email:    "jan@example.org",
		Lastname: "Kowalski",
		Name:     "Jan",
	}
	password := "pass"
	con, err := dbmodel.CreateUserWithPassword(db, user, password)
	require.False(t, con)
	require.NoError(t, err)

	// for next tests there is some data required in the context, let's prepare it
	ctx2, err := rapi.SessionManager.Load(ctx, "")
	require.NoError(t, err)

	params := users.CreateSessionParams{
		Credentials: &models.SessionCredentials{
			Identifier: &user.Email,
			Secret:     &password,
		},
	}
	// provide correct credentials - it should create a new session
	rsp := rapi.CreateSession(ctx2, params)
	require.IsType(t, &users.CreateSessionOK{}, rsp)
	okRsp := rsp.(*users.CreateSessionOK)
	require.Greater(t, *okRsp.Payload.ID, int64(0))

	delParams := users.DeleteSessionParams{}
	rsp = rapi.DeleteSession(ctx2, delParams)
	require.IsType(t, &users.DeleteSessionOK{}, rsp)
}

// Test that the deleting session of the user authorized by an external service
// works properly.
func TestDeleteSessionOfExternalUser(t *testing.T) {
	// Arrange
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	authenticationMethodID := "external"

	metadataMock := NewMockAuthenticationMetadata(ctrl)
	metadataMock.EXPECT().
		GetID().
		Return(authenticationMethodID)

	mock := NewMockAuthenticationCalloutCarrier(ctrl)
	mock.EXPECT().
		Unauthenticate(gomock.Any()).
		Return(nil).
		Times(1)
	mock.EXPECT().
		GetMetadata().
		Return(metadataMock)

	hookManager := hookmanager.NewHookManager()
	hookManager.RegisterCalloutCarrier(mock)
	rapi, _ := NewRestAPI(dbSettings, db, hookManager)

	ctx, _ := rapi.SessionManager.Load(context.Background(), "")

	_ = rapi.SessionManager.LoginHandler(ctx, &dbmodel.SystemUser{
		ID:                     42,
		AuthenticationMethodID: authenticationMethodID,
	})

	// Act
	rsp := rapi.DeleteSession(ctx, users.DeleteSessionParams{})

	// Assert
	require.IsType(t, &users.DeleteSessionOK{}, rsp)
}

// Tests that new session can be created for a logged user that uses the
// external authentication.
func TestCreateSessionOfExternalUser(t *testing.T) {
	// Arrange
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	identifier := "foo@example.com"
	secret := "secret"
	authenticationMethodID := "external"

	metadataMock := NewMockAuthenticationMetadata(ctrl)
	metadataMock.EXPECT().
		GetID().
		Return(authenticationMethodID)

	mock := NewMockAuthenticationCalloutCarrier(ctrl)
	mock.EXPECT().
		Authenticate(gomock.Any(), gomock.Any(), &identifier, &secret).
		Return(&authenticationcallouts.User{
			ID:       "external-id",
			Login:    "foo",
			Email:    identifier,
			Lastname: "oof",
			Name:     "ofo",
		}, nil).
		Times(1)
	mock.EXPECT().
		GetMetadata().
		Return(metadataMock)

	hookManager := hookmanager.NewHookManager()
	hookManager.RegisterCalloutCarrier(mock)
	rapi, _ := NewRestAPI(dbSettings, db, hookManager)

	ctx, _ := rapi.SessionManager.Load(context.Background(), "")

	// Act
	params := users.CreateSessionParams{
		Credentials: &models.SessionCredentials{
			Identifier:             &identifier,
			Secret:                 &secret,
			AuthenticationMethodID: &authenticationMethodID,
		},
	}
	rsp := rapi.CreateSession(ctx, params)

	// Assert
	require.IsType(t, &users.CreateSessionOK{}, rsp)
	okRsp := rsp.(*users.CreateSessionOK)
	require.Greater(t, *okRsp.Payload.ID, int64(0))
}

// Test that the internal authentication method is always returned.
func TestGetAuthenticationMethodsInternal(t *testing.T) {
	// Arrange
	dbSettings := &dbops.DatabaseSettings{}
	ctx := context.Background()
	hookManager := hookmanager.NewHookManager()
	rapi, _ := NewRestAPI(dbSettings, hookManager)

	// Act
	response := rapi.GetAuthenticationMethods(ctx, users.GetAuthenticationMethodsParams{})

	// Assert
	require.IsType(t, &users.GetAuthenticationMethodsOK{}, response)
	responseOk := response.(*users.GetAuthenticationMethodsOK)
	require.EqualValues(t, 1, responseOk.Payload.Total)
	require.Len(t, responseOk.Payload.Items, 1)
	require.EqualValues(t, "internal", responseOk.Payload.Items[0].ID)
}

// Test that the authentication methods from hooks are included in the response.
func TestGetAuthenticationMethodsFromHooks(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	var mocks []hooks.CalloutCarrier
	for i := 0; i < 3; i++ {
		metadataMock := NewMockAuthenticationMetadata(ctrl)
		metadataMock.EXPECT().
			GetID().
			Return(fmt.Sprintf("mock-%d", i))
		metadataMock.EXPECT().
			GetDescription().
			Return(fmt.Sprintf("description-%d", i))
		metadataMock.EXPECT().
			GetName().
			Return(fmt.Sprintf("name-%d", i))

		mock := NewMockAuthenticationCalloutCarrier(ctrl)
		mock.EXPECT().
			GetMetadata().
			Return(metadataMock).
			Times(1)

		mocks = append(mocks, mock)
	}
	hookManager := hookmanager.NewHookManager()
	hookManager.RegisterCalloutCarriers(mocks)

	dbSettings := &dbops.DatabaseSettings{}
	ctx := context.Background()
	rapi, _ := NewRestAPI(dbSettings, hookManager)

	// Act
	response := rapi.GetAuthenticationMethods(ctx, users.GetAuthenticationMethodsParams{})

	// Assert
	require.IsType(t, &users.GetAuthenticationMethodsOK{}, response)
	responseOk := response.(*users.GetAuthenticationMethodsOK)
	items := responseOk.Payload.Items
	require.EqualValues(t, 4, responseOk.Payload.Total)
	require.Len(t, responseOk.Payload.Items, 4)

	// The internal method should be the last one.
	require.EqualValues(t, "internal", items[len(items)-1].ID)

	for i, method := range items[:len(items)-1] {
		require.EqualValues(t, fmt.Sprintf("mock-%d", i), method.ID)
	}
}
