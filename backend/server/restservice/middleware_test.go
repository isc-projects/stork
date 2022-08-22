package restservice

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
	dbsession "isc.org/stork/server/database/session"
	dbtest "isc.org/stork/server/database/test"
	storktest "isc.org/stork/server/test"
	storktestdbmodel "isc.org/stork/server/test/dbmodel"
)

// Check if fileServerMiddleware works and handles requests correctly.
func TestFileServerMiddleware(t *testing.T) {
	apiRequestReceived := false
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiRequestReceived = true
	})

	handler := fileServerMiddleware(apiHandler, "./non-existing-static/")

	// let request some static file, as it does not exist 404 code should be returned
	req := httptest.NewRequest("GET", "http://localhost/abc", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	resp := w.Result()
	resp.Body.Close()
	require.EqualValues(t, 404, resp.StatusCode)
	require.False(t, apiRequestReceived)

	// let request some API URL, it should be forwarded to apiHandler
	req = httptest.NewRequest("GET", "http://localhost/api/users", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.True(t, apiRequestReceived)

	// request for swagger.json also should be forwarded to apiHandler
	req = httptest.NewRequest("GET", "http://localhost/swagger.json", nil)
	w = httptest.NewRecorder()
	apiRequestReceived = false
	handler.ServeHTTP(w, req)
	require.True(t, apiRequestReceived)
}

// Check if InnerMiddleware works.
func TestInnerMiddleware(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	rapi, err := NewRestAPI(dbSettings, db)
	require.NoError(t, err)
	sm, err := dbsession.NewSessionMgr(&rapi.DBSettings.BaseDatabaseSettings)
	require.NoError(t, err)
	rapi.SessionManager = sm

	handler := rapi.InnerMiddleware(nil)
	require.NotNil(t, handler)
}

// Check if fileServerMiddleware works and handles requests correctly.
func TestSSEMiddleware(t *testing.T) {
	requestReceived := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestReceived = true
	})

	fec := &storktestdbmodel.FakeEventCenter{}

	handler := sseMiddleware(nextHandler, fec)

	// let request sse
	req := httptest.NewRequest("GET", "http://localhost/sse", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	resp := w.Result()
	resp.Body.Close()
	require.EqualValues(t, 200, resp.StatusCode)
	require.False(t, requestReceived)

	// let request something else than sse, it should be forwarded to nextHandler
	req = httptest.NewRequest("GET", "http://localhost/api/users", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.True(t, requestReceived)
}

// Check if agentInstallerMiddleware works and handles requests correctly.
func TestAgentInstallerMiddleware(t *testing.T) {
	requestReceived := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestReceived = true
	})

	tmpDir, err := os.MkdirTemp("", "mdlw")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	handler := agentInstallerMiddleware(nextHandler, tmpDir)

	// let do some request but when there is no folder with static content
	req := httptest.NewRequest("GET", "http://localhost/stork-install-agent.sh", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	resp := w.Result()
	resp.Body.Close()
	require.EqualValues(t, 500, resp.StatusCode)
	require.False(t, requestReceived)

	// prepare folders
	os.Mkdir(path.Join(tmpDir, "assets"), 0o755)
	packagesDir := path.Join(tmpDir, "assets/pkgs")
	os.Mkdir(packagesDir, 0o755)

	// let do some request
	req = httptest.NewRequest("GET", "http://localhost/stork-install-agent.sh", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	resp = w.Result()
	resp.Body.Close()
	require.EqualValues(t, 404, resp.StatusCode)
	require.False(t, requestReceived)

	// create packages
	for _, extension := range []string{".deb", ".apk", ".rpm"} {
		f, err := os.Create(path.Join(packagesDir, "isc-stork-agent"+extension))
		require.NoError(t, err)
		f.Close()
	}

	// let do some request
	req = httptest.NewRequest("GET", "http://localhost/stork-install-agent.sh", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	resp = w.Result()
	resp.Body.Close()
	require.EqualValues(t, 200, resp.StatusCode)
	require.False(t, requestReceived)

	// let request something else, it should be forwarded to nextHandler
	req = httptest.NewRequest("GET", "http://localhost/api/users", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.True(t, requestReceived)
}

// Check if metricsMiddleware works and handles requests correctly.
func TestMetricsMiddleware(t *testing.T) {
	// Arrange
	metrics := storktest.NewFakeMetricsCollector()
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	handler := metricsMiddleware(nextHandler, metrics)

	// Act
	req := httptest.NewRequest("GET", "http://localhost/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Assert
	require.EqualValues(t, 1, metrics.RequestCount)
}

// Check if metricsMiddelware returns placeholder when the endpoint is disabled.
func TestMetricsMiddlewarePlaceholder(t *testing.T) {
	// Arrange
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	handler := metricsMiddleware(nextHandler, nil)

	// Act
	req := httptest.NewRequest("GET", "http://localhost/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	resp := w.Result()
	content, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, 503, resp.StatusCode)
	require.EqualValues(t, "The metrics collector endpoint is disabled.", content)
}

// Dumb response writer struct with functions to enable testing
// loggingResponseWriter.
type dumbRespWriter struct{}

func (r *dumbRespWriter) Write(b []byte) (int, error) {
	return len(b), nil
}

func (r *dumbRespWriter) WriteHeader(statusCode int) {
}

func (r *dumbRespWriter) Header() http.Header {
	return map[string][]string{}
}

// Check if helpers to logging middleware works.
func TestLoggingMiddlewareHelpers(t *testing.T) {
	lrw := &loggingResponseWriter{
		rw:           &dumbRespWriter{},
		responseData: &responseData{},
	}

	// check write
	size, err := lrw.Write([]byte("abc"))
	require.NoError(t, err)
	require.EqualValues(t, 3, size)

	// check WriteHeader
	lrw.WriteHeader(400)

	// check Header
	hdr := lrw.Header()
	require.Empty(t, hdr)
}
