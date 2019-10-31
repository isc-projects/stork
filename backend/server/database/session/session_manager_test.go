package dbsession

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"github.com/stretchr/testify/require"
	"isc.org/stork/server/database/model"
	"isc.org/stork/server/database/test"
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
	teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown(t)

	mgr, err := NewSessionMgr(&dbtest.GenericConnOptions)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()

	// Create user to be logged to the system.
	user := &dbmodel.SystemUser{
		Id: 1,
		Login: "johnw",
		Email: "johnw@example.org",
		Lastname: "White",
		Name: "John C",
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
		require.Equal(t, user.Id, userSession.Id)
		require.Equal(t, user.Login, userSession.Login)
		require.Equal(t, user.Email, userSession.Email)
		require.Equal(t, user.Lastname, userSession.Lastname)
		require.Equal(t, user.Name, userSession.Name)
	}

	middlewareFunc := mgr.SessionMiddleware(http.HandlerFunc(handler))
	middlewareFunc.ServeHTTP(w, req)
	resp := w.Result()
	require.Equal(t, resp.StatusCode, 200)

	// Check that the session cookie was set in the response.
	hasCookie, value := getCookie(resp, "session")
	require.True(t, hasCookie)

	// Check that the session token was stored in the database.
	require.True(t, mgr.HasToken(value))
}
