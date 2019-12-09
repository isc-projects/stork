package restservice

import (
	"context"
	"testing"
	"github.com/stretchr/testify/require"
	"isc.org/stork/server/database/model"
	"isc.org/stork/server/database/test"
	"isc.org/stork/server/gen/models"
	"isc.org/stork/server/gen/restapi/operations/users"
)

// Tests that user account can be created via REST API.
func TestCreateUser(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()
	rapi, _ := NewRestAPI(nil, dbSettings, db, nil)

	su := dbmodel.SystemUser{
		Email: "jb@example.org",
		Lastname: "Born",
		Login: "jb",
		Name: "John",
	}

	params := users.CreateUserParams{
		Account: &models.UserAccount{
			User: NewRestUser(su),
			Password: models.Password("pass"),
		},
	}

	// Create the user and verify the response.
	rsp := rapi.CreateUser(ctx, params)
	require.IsType(t, &users.CreateUserOK{}, rsp)
	okRsp := rsp.(*users.CreateUserOK)
	require.Greater(t, *okRsp.Payload.ID, int64(0))
	require.Equal(t, *okRsp.Payload.Email, su.Email)
	require.Equal(t, *okRsp.Payload.Lastname, su.Lastname)
	require.Equal(t, *okRsp.Payload.Login, su.Login)
	require.Equal(t, *okRsp.Payload.Name, su.Name)

	// Also check that the user is indeed in the database.
	returned, err := dbmodel.GetUserById(db, int(*okRsp.Payload.ID))
	require.NoError(t, err)
	require.Equal(t, su.Login, returned.Login)

	// An attempt to create the same user should fail with HTTP error 409.
	rsp = rapi.CreateUser(ctx, params)
	require.IsType(t, &users.CreateUserDefault{}, rsp)
	defaultRsp := rsp.(*users.CreateUserDefault)
	require.Equal(t, 409, getStatusCode(*defaultRsp))
}

// Tests that user account can be updated via REST API.
func TestUpdateUser(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()
	rapi, err := NewRestAPI(nil, dbSettings, db, nil)

	// Create new user in the database.
	su := dbmodel.SystemUser{
		Email: "jan@example.org",
		Lastname: "Kowalski",
		Name:     "Jan",
		Password: "pass",
	}
	err, con := su.Persist(db)
	require.False(t, con)
	require.NoError(t, err)

	// Modify some values and update the user.
	su.Lastname = "Born"
	params := users.UpdateUserParams{
		Account: &models.UserAccount{
			User: NewRestUser(su),
			Password: models.Password("pass"),
		},
	}
	rsp := rapi.UpdateUser(ctx, params)
	require.IsType(t, &users.UpdateUserOK{}, rsp)

	// Also check that the user has been updated in the database.
	returned, err := dbmodel.GetUserById(db, su.Id)
	require.NoError(t, err)
	require.Equal(t, su.Lastname, returned.Lastname)

	// An attempt to update non-existing user (non macthing ID) should
	// result in an error 409.
	su.Id = 123
	params = users.UpdateUserParams{
		Account: &models.UserAccount{
			User: NewRestUser(su),
			Password: models.Password("pass"),
		},
	}
	rsp = rapi.UpdateUser(ctx, params)
	require.IsType(t, &users.UpdateUserDefault{}, rsp)
	defaultRsp := rsp.(*users.UpdateUserDefault)
	require.Equal(t, 409, getStatusCode(*defaultRsp))
}

// Tests that user password can be updated via REST API.
func TestUpdateUserPassword(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctx := context.Background()
	rapi, err := NewRestAPI(nil, dbSettings, db, nil)

	// Create new user in the database.
	user := &dbmodel.SystemUser{
		Email: "jan@example.org",
		Lastname: "Kowalski",
		Name:     "Jan",
		Password: "pass",
	}
	err, con := user.Persist(db)
	require.False(t, con)
	require.NoError(t, err)

	// Update user password via the API.
	params := users.UpdateUserPasswordParams{
		ID: int64(user.Id),
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
		ID: int64(user.Id),
		Passwords: &models.PasswordChange{
			Newpassword: models.Password("updated"),
			Oldpassword: models.Password("pass"),
		},
	}
	rsp = rapi.UpdateUserPassword(ctx, params)
	require.IsType(t, &users.UpdateUserPasswordDefault{}, rsp)
	defaultRsp := rsp.(*users.UpdateUserPasswordDefault)
	require.Equal(t, 400, getStatusCode(*defaultRsp))
}
