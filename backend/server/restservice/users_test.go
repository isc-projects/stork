package restservice

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/server/gen/models"
	"isc.org/stork/server/gen/restapi/operations/users"
	storktest "isc.org/stork/server/test"
)

// Tests that user account without necessary fields is rejected via REST API.
func TestCreateUserNegative(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, _ := NewRestAPI(nil, dbSettings, db, nil, fec, nil, fd)

	// Both login and email missing here.
	su := dbmodel.SystemUser{
		Lastname: "Doe",
		Name:     "John",
	}

	params := users.CreateUserParams{
		Account: &models.UserAccount{
			User:     newRestUser(su),
			Password: models.Password("pass"),
		},
	}

	// Attempt to create the user and verify the response is negative.
	rsp := rapi.CreateUser(ctx, params)
	require.IsType(t, &users.CreateUserDefault{}, rsp)
	defaultRsp := rsp.(*users.CreateUserDefault)
	require.Equal(t, 409, getStatusCode(*defaultRsp))
}

// Tests that user account can be created via REST API.
func TestCreateUser(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, _ := NewRestAPI(nil, dbSettings, db, nil, fec, nil, fd)

	// Try empty request, variant 1 - it should raise an error
	params := users.CreateUserParams{}
	rsp := rapi.CreateUser(ctx, params)
	require.IsType(t, &users.CreateUserDefault{}, rsp)
	defaultRsp := rsp.(*users.CreateUserDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, "failed to create new user account: missing data", *defaultRsp.Payload.Message)

	// Try empty request, variant 2 - it should raise an error
	params = users.CreateUserParams{
		Account: &models.UserAccount{},
	}
	rsp = rapi.CreateUser(ctx, params)
	require.IsType(t, &users.CreateUserDefault{}, rsp)
	defaultRsp = rsp.(*users.CreateUserDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, "failed to create new user account: missing data", *defaultRsp.Payload.Message)

	// Try empty request, variant 3 - it should raise an error
	params = users.CreateUserParams{
		Account: &models.UserAccount{
			User: &models.User{},
		},
	}
	rsp = rapi.CreateUser(ctx, params)
	require.IsType(t, &users.CreateUserDefault{}, rsp)
	defaultRsp = rsp.(*users.CreateUserDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, "failed to create new user account: missing data", *defaultRsp.Payload.Message)

	// Create the user and verify the response.
	su := dbmodel.SystemUser{
		Email:    "jb@example.org",
		Lastname: "Born",
		Login:    "jb",
		Name:     "John",
	}
	params = users.CreateUserParams{
		Account: &models.UserAccount{
			User:     newRestUser(su),
			Password: models.Password("pass"),
		},
	}

	rsp = rapi.CreateUser(ctx, params)
	require.IsType(t, &users.CreateUserOK{}, rsp)
	okRsp := rsp.(*users.CreateUserOK)
	require.Greater(t, *okRsp.Payload.ID, int64(0))
	require.Equal(t, *okRsp.Payload.Email, su.Email)
	require.Equal(t, *okRsp.Payload.Lastname, su.Lastname)
	require.Equal(t, *okRsp.Payload.Login, su.Login)
	require.Equal(t, *okRsp.Payload.Name, su.Name)

	// Also check that the user is indeed in the database.
	returned, err := dbmodel.GetUserByID(db, int(*okRsp.Payload.ID))
	require.NoError(t, err)
	require.Equal(t, su.Login, returned.Login)

	// An attempt to create the same user should fail with HTTP error 409.
	rsp = rapi.CreateUser(ctx, params)
	require.IsType(t, &users.CreateUserDefault{}, rsp)
	defaultRsp = rsp.(*users.CreateUserDefault)
	require.Equal(t, 409, getStatusCode(*defaultRsp))
}

// Tests that user account can be updated via REST API.
func TestUpdateUser(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}

	rapi, err := NewRestAPI(nil, dbSettings, db, nil, fec, nil, fd)
	require.NoError(t, err)

	// Create new user in the database.
	su := dbmodel.SystemUser{
		Email:    "jan@example.org",
		Lastname: "Kowalski",
		Name:     "Jan",
		Password: "pass",
	}
	con, err := dbmodel.CreateUser(db, &su)
	require.False(t, con)
	require.NoError(t, err)

	// Try empty request, variant 1 - it should raise an error
	params := users.UpdateUserParams{}
	rsp := rapi.UpdateUser(ctx, params)
	require.IsType(t, &users.UpdateUserDefault{}, rsp)
	defaultRsp := rsp.(*users.UpdateUserDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, "failed to update user account: missing data", *defaultRsp.Payload.Message)

	// Try empty request, variant 2 - it should raise an error
	params = users.UpdateUserParams{
		Account: &models.UserAccount{},
	}
	rsp = rapi.UpdateUser(ctx, params)
	require.IsType(t, &users.UpdateUserDefault{}, rsp)
	defaultRsp = rsp.(*users.UpdateUserDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, "failed to update user account: missing data", *defaultRsp.Payload.Message)

	// Try empty request, variant 3 - it should raise an error
	params = users.UpdateUserParams{
		Account: &models.UserAccount{
			User: &models.User{},
		},
	}
	rsp = rapi.UpdateUser(ctx, params)
	require.IsType(t, &users.UpdateUserDefault{}, rsp)
	defaultRsp = rsp.(*users.UpdateUserDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, "failed to update user account: missing data", *defaultRsp.Payload.Message)

	// Modify some values and update the user.
	su.Lastname = "Born"
	params = users.UpdateUserParams{
		Account: &models.UserAccount{
			User:     newRestUser(su),
			Password: models.Password("pass"),
		},
	}
	rsp = rapi.UpdateUser(ctx, params)
	require.IsType(t, &users.UpdateUserOK{}, rsp)

	// Also check that the user has been updated in the database.
	returned, err := dbmodel.GetUserByID(db, su.ID)
	require.NoError(t, err)
	require.Equal(t, su.Lastname, returned.Lastname)

	// An attempt to update non-existing user (non macthing ID) should
	// result in an error 409.
	su.ID = 123
	params = users.UpdateUserParams{
		Account: &models.UserAccount{
			User:     newRestUser(su),
			Password: models.Password("pass"),
		},
	}
	rsp = rapi.UpdateUser(ctx, params)
	require.IsType(t, &users.UpdateUserDefault{}, rsp)
	defaultRsp = rsp.(*users.UpdateUserDefault)
	require.Equal(t, 409, getStatusCode(*defaultRsp))
}

// Tests that user password can be updated via REST API.
func TestUpdateUserPassword(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(nil, dbSettings, db, nil, fec, nil, fd)
	require.NoError(t, err)

	// Create new user in the database.
	user := &dbmodel.SystemUser{
		Email:    "jan@example.org",
		Lastname: "Kowalski",
		Name:     "Jan",
		Password: "pass",
	}
	con, err := dbmodel.CreateUser(db, user)
	require.False(t, con)
	require.NoError(t, err)

	// Update user password via the API.
	params := users.UpdateUserPasswordParams{
		ID: int64(user.ID),
		Passwords: &models.PasswordChange{
			Newpassword: models.Password("updated"),
			Oldpassword: models.Password("pass"),
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
			Newpassword: models.Password("updated"),
			Oldpassword: models.Password("pass"),
		},
	}
	rsp = rapi.UpdateUserPassword(ctx, params)
	require.IsType(t, &users.UpdateUserPasswordDefault{}, rsp)
	defaultRsp := rsp.(*users.UpdateUserPasswordDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))

	// Try empty request - it should raise an error.
	params = users.UpdateUserPasswordParams{
		ID: int64(user.ID),
	}
	rsp = rapi.UpdateUserPassword(ctx, params)
	require.IsType(t, &users.UpdateUserPasswordDefault{}, rsp)
	defaultRsp = rsp.(*users.UpdateUserPasswordDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, "failed to update password for user: missing data", *defaultRsp.Payload.Message)
}

// Tests that multiple groups can be fetched from the database.
func TestGetGroups(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, _ := NewRestAPI(nil, dbSettings, db, nil, fec, nil, fd)

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
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(nil, dbSettings, db, nil, fec, nil, fd)
	require.NoError(t, err)

	// Create new user in the database.
	user := &dbmodel.SystemUser{
		Email:    "jd@example.org",
		Lastname: "Doe",
		Login:    "johndoe",
		Name:     "John",
		Password: "pass",
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
	var limit int64 = 99999
	var start int64 = 0
	var text string = ""
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
	require.Equal(t, "admin", *okRsp.Payload.Items[0].Login)
	require.Equal(t, "admin", *okRsp.Payload.Items[0].Name)
	require.Equal(t, "admin", *okRsp.Payload.Items[0].Lastname)
	require.Equal(t, "", *okRsp.Payload.Items[0].Email)

	// Check the user we just added
	require.Equal(t, "johndoe", *okRsp.Payload.Items[1].Login)
	require.Equal(t, "John", *okRsp.Payload.Items[1].Name)
	require.Equal(t, "Doe", *okRsp.Payload.Items[1].Lastname)
	require.Equal(t, "jd@example.org", *okRsp.Payload.Items[1].Email)

	// Make sure the ID of the new user is different.
	// TODO: implement new test that add 100 users and verifies uniqueness of their IDs
	require.NotEqual(t, *okRsp.Payload.Items[0].ID, *okRsp.Payload.Items[1].ID)
}

// Tests that user information can be retrieved via REST API.
func TestGetUser(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(nil, dbSettings, db, nil, fec, nil, fd)
	require.NoError(t, err)

	// Create new user in the database.
	user := &dbmodel.SystemUser{
		Email:    "jd@example.org",
		Lastname: "Doe",
		Login:    "johndoe",
		Name:     "John",
		Password: "pass",
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
	require.Equal(t, user.Email, *okRsp.Payload.Email) // we expect 2 users (admin and john doe)
	require.Equal(t, id, *okRsp.Payload.ID)            // user ID
	require.Equal(t, user.Login, *okRsp.Payload.Login)
	require.Equal(t, user.Name, *okRsp.Payload.Name)
	require.Equal(t, user.Lastname, *okRsp.Payload.Lastname)

	// TODO: check that the new user belongs to a group
	// require.Len(t, okRsp.Payload.Groups, 1)
}

// Tests that new session can be created for a logged user.
func TestCreateSession(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, _ := NewRestAPI(nil, dbSettings, db, nil, fec, nil, fd)

	user := &dbmodel.SystemUser{
		Email:    "jan@example.org",
		Lastname: "Kowalski",
		Name:     "Jan",
		Password: "pass",
	}
	con, err := dbmodel.CreateUser(db, user)
	require.False(t, con)
	require.NoError(t, err)

	// try empty request - it should raise an error
	params := users.CreateSessionParams{}
	rsp := rapi.CreateSession(ctx, params)
	require.IsType(t, &users.CreateSessionBadRequest{}, rsp)

	// for next tests there is some data required in the context, let's prepare it
	ctx2, err := rapi.SessionManager.Load(ctx, "")
	require.NoError(t, err)

	// provide wrong credentials - it should raise an error
	params.Credentials.Useremail = &user.Email
	bad := "bad pass"
	params.Credentials.Userpassword = &bad
	rsp = rapi.CreateSession(ctx2, params)
	require.IsType(t, &users.CreateSessionBadRequest{}, rsp)

	// provide correct credentials - it should create a new session
	params.Credentials.Useremail = &user.Email
	params.Credentials.Userpassword = &user.Password
	rsp = rapi.CreateSession(ctx2, params)
	require.IsType(t, &users.CreateSessionOK{}, rsp)
	okRsp := rsp.(*users.CreateSessionOK)
	require.Greater(t, *okRsp.Payload.ID, int64(0))

	delParams := users.DeleteSessionParams{}
	rsp = rapi.DeleteSession(ctx2, delParams)
	require.IsType(t, &users.DeleteSessionOK{}, rsp)
}
