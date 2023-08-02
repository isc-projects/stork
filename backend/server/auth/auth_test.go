package auth

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
)

// Helper function checking if the user belonging to the specified group
// has access to the resource.
func authorizeAccept(t *testing.T, groupID int, path, method string) bool {
	// Create user with ID 5 and specified group id if the group id is
	// positive.
	user := &dbmodel.SystemUser{
		ID: 5,
	}
	if groupID > 0 {
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
	require.False(t, authorizeAccept(t, 2, "/users?start=0&limit=10", "GET"))
	require.False(t, authorizeAccept(t, 2, "/users/list", "GET"))
	require.False(t, authorizeAccept(t, 2, "/users/4/password", "GET"))
	require.False(t, authorizeAccept(t, 2, "/users//4/password/", "GET"))
	require.True(t, authorizeAccept(t, 2, "/users/5", "GET"))
	require.True(t, authorizeAccept(t, 2, "/users/5/password", "GET"))
	require.True(t, authorizeAccept(t, 2, "/users//5//password", "GET"))

	// Super-admin has no such restrictions.
	require.True(t, authorizeAccept(t, 1, "/users?start=0&limit=10", "GET"))
	require.True(t, authorizeAccept(t, 1, "/users/list", "GET"))
	require.True(t, authorizeAccept(t, 1, "/users/4/password", "GET"))
	require.True(t, authorizeAccept(t, 1, "/users//4/password/", "GET"))
	require.True(t, authorizeAccept(t, 1, "/users/5", "GET"))
	require.True(t, authorizeAccept(t, 1, "/users/5/password", "GET"))
	require.True(t, authorizeAccept(t, 1, "/users//5//password", "GET"))

	// Admin group have no restriction on machines.
	require.True(t, authorizeAccept(t, 2, "/machines/1/", "GET"))

	// But someone who belongs to no groups would not be able
	// to access machines.
	require.False(t, authorizeAccept(t, 0, "/machines/1/", "GET"))

	// The same in case of someone belonging to non existing group.
	require.False(t, authorizeAccept(t, 3, "/machines/1/", "GET"))

	// Someone who belongs to no groups would be able to log out.
	require.True(t, authorizeAccept(t, 0, "/sessions", "DELETE"))

	// Someone who belongs to no groups would be able to their and only their profile.
	require.True(t, authorizeAccept(t, 0, "/users/5", "GET"))
	require.True(t, authorizeAccept(t, 0, "/users/5/password", "GET"))
	require.False(t, authorizeAccept(t, 0, "/users/4", "GET"))
	require.False(t, authorizeAccept(t, 0, "/users/4/password", "GET"))
}
