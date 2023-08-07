package auth

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
)

const noneGroupID int = 0

// Helper function checking if the user belonging to the specified group
// has access to the resource.
func authorizeAccept(t *testing.T, groupID int, path, method string) bool {
	// Create user with ID 5 and specified group id if the group id is
	// positive.
	user := &dbmodel.SystemUser{
		ID: 5,
	}
	if groupID != noneGroupID {
		user.Groups = []*dbmodel.SystemGroup{
			{
				ID: groupID,
			},
		}
	}

	// Create request with the specified path and authorize.
	req, _ := http.NewRequestWithContext(context.Background(), method, "http://example.org/api"+path, nil)
	ok, err := Authorize(user, req)
	require.NoError(t, err)

	return ok
}

// Verify that users belonging to the super-admin and admin group
// has appropriate access privileges.
func TestAuthorize(t *testing.T) {
	// Admin group have limited access to the users' management.
	require.False(t, authorizeAccept(t, dbmodel.AdminGroupID, "/users?start=0&limit=10", "GET"))
	require.False(t, authorizeAccept(t, dbmodel.AdminGroupID, "/users/list", "GET"))
	require.False(t, authorizeAccept(t, dbmodel.AdminGroupID, "/users/4/password", "GET"))
	require.False(t, authorizeAccept(t, dbmodel.AdminGroupID, "/users//4/password/", "GET"))
	require.True(t, authorizeAccept(t, dbmodel.AdminGroupID, "/users/5", "GET"))
	require.True(t, authorizeAccept(t, dbmodel.AdminGroupID, "/users/5/password", "GET"))
	require.True(t, authorizeAccept(t, dbmodel.AdminGroupID, "/users//5//password", "GET"))

	// Super-admin has no such restrictions.
	require.True(t, authorizeAccept(t, dbmodel.SuperAdminGroupID, "/users?start=0&limit=10", "GET"))
	require.True(t, authorizeAccept(t, dbmodel.SuperAdminGroupID, "/users/list", "GET"))
	require.True(t, authorizeAccept(t, dbmodel.SuperAdminGroupID, "/users/4/password", "GET"))
	require.True(t, authorizeAccept(t, dbmodel.SuperAdminGroupID, "/users//4/password/", "GET"))
	require.True(t, authorizeAccept(t, dbmodel.SuperAdminGroupID, "/users/5", "GET"))
	require.True(t, authorizeAccept(t, dbmodel.SuperAdminGroupID, "/users/5/password", "GET"))
	require.True(t, authorizeAccept(t, dbmodel.SuperAdminGroupID, "/users//5//password", "GET"))

	// Admin group have no restriction on machines.
	require.True(t, authorizeAccept(t, dbmodel.AdminGroupID, "/machines/1/", "GET"))

	// But someone who belongs to no groups would not be able
	// to access machines.
	require.False(t, authorizeAccept(t, noneGroupID, "/machines/1/", "GET"))

	// The same in case of someone belonging to non existing group.
	require.False(t, authorizeAccept(t, 999, "/machines/1/", "GET"))

	// Someone who belongs to no groups would be able to log out.
	require.True(t, authorizeAccept(t, noneGroupID, "/sessions", "DELETE"))

	// Someone who belongs to no groups would be able to their and only their profile.
	require.True(t, authorizeAccept(t, noneGroupID, "/users/5", "GET"))
	require.True(t, authorizeAccept(t, noneGroupID, "/users/5/password", "GET"))
	require.False(t, authorizeAccept(t, noneGroupID, "/users/4", "GET"))
	require.False(t, authorizeAccept(t, noneGroupID, "/users/4/password", "GET"))
}
