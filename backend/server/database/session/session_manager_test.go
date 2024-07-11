package dbsession

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
)

// Attempts to find a cookie in the HTTP response.
func getCookie(response *http.Response, name string) (bool, string) {
	for _, c := range response.Cookies() {
		if c.Name == name {
			return true, c.Value
		}
	}

	// Cookie not found.
	return false, ""
}

// Tests that new session is created via the middleware.
func TestMiddlewareNewSession(t *testing.T) {
	// Reset database schema.
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	mgr, err := NewSessionMgr(db)
	require.NoError(t, err)
	defer mgr.Close()

	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()

	// Create user to be logged to the system.
	user := &dbmodel.SystemUser{
		ID:                     1,
		Login:                  "johnw",
		Email:                  "johnw@example.org",
		Lastname:               "White",
		Name:                   "John C",
		AuthenticationMethodID: dbmodel.AuthenticationMethodIDInternal,

		Groups: []*dbmodel.SystemGroup{
			{
				ID:   5,
				Name: "abc",
			},
			{
				ID:   25,
				Name: "def",
			},
		},
	}

	// Run the middleware.
	handler := func(w http.ResponseWriter, r *http.Request) {
		// Simulate user login to the system.
		err := mgr.LoginHandler(r.Context(), user)
		require.NoError(t, err)

		// Check that the user has been logged
		logged, userSession := mgr.Logged(r.Context())

		// ... and that user data was stored in the session.
		require.True(t, logged)
		require.NotNil(t, userSession)
		require.Equal(t, user.ID, userSession.ID)
		require.Equal(t, user.Login, userSession.Login)
		require.Equal(t, user.Email, userSession.Email)
		require.Equal(t, user.Lastname, userSession.Lastname)
		require.Equal(t, user.Name, userSession.Name)
		require.Equal(t, user.AuthenticationMethodID, userSession.AuthenticationMethodID)

		require.Len(t, userSession.Groups, 2)
		require.True(t, userSession.InGroup(&dbmodel.SystemGroup{ID: 5}))
		require.True(t, userSession.InGroup(&dbmodel.SystemGroup{ID: 25}))
	}

	middlewareFunc := mgr.SessionMiddleware(http.HandlerFunc(handler))
	middlewareFunc.ServeHTTP(w, req)
	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, resp.StatusCode, 200)

	// Check that the session cookie was set in the response.
	hasCookie, value := getCookie(resp, "session")
	require.True(t, hasCookie)

	// Check that the session token was stored in the database.
	require.True(t, mgr.HasToken(value))
}

func TestLoad(t *testing.T) {
	// Reset database schema.
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	mgr, err := NewSessionMgr(db)
	require.NoError(t, err)
	defer mgr.Close()

	ctx := context.Background()

	_, err = mgr.Load(ctx, "")
	require.NoError(t, err)
}

func TestLogOutUser(t *testing.T) {
	// Reset database schema.
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Create session manager.
	mgr, err := NewSessionMgr(db)
	require.NoError(t, err)
	defer mgr.Close()

	ctx := context.Background()

	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()

	// Create user to be logged to the system.
	user := &dbmodel.SystemUser{
		ID:       1,
		Login:    "johnw",
		Email:    "johnw@example.org",
		Lastname: "White",
		Name:     "John C",

		Groups: []*dbmodel.SystemGroup{
			{
				ID:   5,
				Name: "abc",
			},
			{
				ID:   25,
				Name: "def",
			},
		},
	}

	// Flag which indicates if login should be performed on the response handler.
	performLogin := true

	// Run the middleware.
	handler := func(w http.ResponseWriter, r *http.Request) {
		if performLogin {
			// Simulate user login to the system.
			err := mgr.LoginHandler(r.Context(), user)
			require.NoError(t, err)

			// Check that the user has been logged
			logged, userSession := mgr.Logged(r.Context())

			// ... and that user data was stored in the session.
			require.True(t, logged)
			require.NotNil(t, userSession)
			require.Equal(t, user.ID, userSession.ID)
			require.Equal(t, user.Login, userSession.Login)
			require.Equal(t, user.Email, userSession.Email)
			require.Equal(t, user.Lastname, userSession.Lastname)
			require.Equal(t, user.Name, userSession.Name)

			require.Len(t, userSession.Groups, 2)
			require.True(t, userSession.InGroup(&dbmodel.SystemGroup{ID: 5}))
			require.True(t, userSession.InGroup(&dbmodel.SystemGroup{ID: 25}))
		}

		// Store the context so that it can be checked after the session is
		// stored in the database.
		ctx = r.Context()
	}

	middlewareFunc := mgr.SessionMiddleware(http.HandlerFunc(handler))
	middlewareFunc.ServeHTTP(w, req)
	resp := w.Result()
	resp.Body.Close()
	require.Equal(t, resp.StatusCode, 200)

	// The context has proper data in sync with the data in the database.
	// The user should have a valid session.
	logged, su := mgr.Logged(ctx)
	require.True(t, logged)
	require.Equal(t, user.ID, su.ID)

	// The LogoutUser will remove the session from the database, but it won't
	// update the data in the context used by the function call.
	// Usually the function is called using the context of the administrator which
	// is deleting a user and the updated context is the context of the user
	// being deleted. To get the right context, the context needs to be retrieved
	// from the handler once again.
	err = mgr.LogoutUser(ctx, user)
	require.NoError(t, err)

	// Update the flag so calling the handler will only retrieve the updated
	// context.
	performLogin = false

	// Perform the request and retrieve the updated context.
	middlewareFunc.ServeHTTP(w, req)
	resp = w.Result()
	resp.Body.Close()
	require.Equal(t, resp.StatusCode, 200)

	// The context has proper data in sync with the data in the database.
	// The user should not have a valid session.
	logged, _ = mgr.Logged(ctx)
	require.False(t, logged)
}
