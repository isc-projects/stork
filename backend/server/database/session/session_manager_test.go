package dbsession

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"os"

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

// Common function which cleans the environment before the tests.
func TestMain(m *testing.M) {
	// Cleanup the database before and after the test.
	dbtest.ResetSchema()
	defer dbtest.ResetSchema()

	// Run tests.
	c := m.Run()
	os.Exit(c)
}

func TestMiddlewareNewSession(t *testing.T) {
	mgr, _ := NewSessionMgr(&dbtest.GenericConnOptions)

	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()

	handler := func(w http.ResponseWriter, r *http.Request) {
		mgr.LoginHandler(r.Context())
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
	fmt.Println(value)
	require.True(t, hasCookie)
	require.True(t, mgr.HasToken(value))
}
