package dbsession

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"os"

	"github.com/stretchr/testify/require"

	"isc.org/stork/server/database"
	"isc.org/stork/server/database/migrations"
)

var genericConnOptions = dbops.GenericConn{
	DbName: "storktest",
	User: "storktest",
	Password: "storktest",
	Host: "localhost",
	Port: 5432,
}

var pgConnOptions dbops.PgOptions

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
	pgConnOptions := genericConnOptions.PgParams()

	// Check if we're running tests in Gitlab CI. If so, the host
	// running the database should be set to "postgres".
	// See https://docs.gitlab.com/ee/ci/services/postgres.html.
	if _, ok := os.LookupEnv("POSTGRES_DB"); ok {
		genericConnOptions.Host = "postgres"
		pgConnOptions.Addr = "postgres:5432"
	}

	// Toss the schema, including removal of the versioning table.
	dbmigs.Toss(pgConnOptions)
	dbmigs.Migrate(pgConnOptions, "init")
	dbmigs.Migrate(pgConnOptions, "up")

	// Run tests.
	c := m.Run()
	os.Exit(c)
}

func TestMiddlewareNewSession(t *testing.T) {
	mgr, _ := NewSessionMgr(&genericConnOptions)

	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()

	handler := func(w http.ResponseWriter, r *http.Request) {
		mgr.LoginHandler(r.Context())
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
