package dbsession

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"github.com/stretchr/testify/require"
	"isc.org/stork/server/database/test"
)

func getCookie(response *http.Response, name string) (bool, string) {
	for _, c := range response.Cookies() {
		if c.Name == name {
			return true, c.Value
		}
	}

	return false, ""
}

func TestMiddlewareNewSession(t *testing.T) {
	teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown(t)

	mgr, _ := NewSessionMgr(&dbtest.GenericConnOptions)

	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()

	handler := func(w http.ResponseWriter, r *http.Request) {
		err := mgr.LoginHandler(r.Context())
		require.NoError(t, err)
		logged, id, login := mgr.Logged(r.Context())

		require.True(t, logged)
		require.Equal(t, id, 1)
		require.Equal(t, login, "admin")
	}

	middlewareFunc := mgr.SessionMiddleware(http.HandlerFunc(handler))
	middlewareFunc.ServeHTTP(w, req)

	resp := w.Result()

	require.Equal(t, resp.StatusCode, 200)

	hasCookie, value := getCookie(resp, "session")
	require.True(t, hasCookie)
	require.True(t, mgr.HasToken(value))
}
