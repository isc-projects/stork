package restservice

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	dbsession "isc.org/stork/server/database/session"
	dbtest "isc.org/stork/server/database/test"
	storktest "isc.org/stork/server/test"
	storktestdbmodel "isc.org/stork/server/test/dbmodel"
	"isc.org/stork/testutil"
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
	require.EqualValues(t, http.StatusNotFound, resp.StatusCode)
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

	// request non-existing static content file
	req = httptest.NewRequest("GET", "http://localhost/assets/static-page-content/xyz.abc", nil)
	w = httptest.NewRecorder()
	apiRequestReceived = false
	handler.ServeHTTP(w, req)
	resp = w.Result()
	resp.Body.Close()
	require.EqualValues(t, http.StatusNoContent, resp.StatusCode)
}

// Check if InnerMiddleware works.
func TestInnerMiddleware(t *testing.T) {
	db, settings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	rapi, err := NewRestAPI(db, settings)
	require.NoError(t, err)
	sm, err := dbsession.NewSessionMgr(db)
	require.NoError(t, err)
	defer sm.Close()
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

	sb := testutil.NewSandbox()
	defer sb.Close()

	url := url.URL{
		Scheme: "http",
		Host:   "localhost",
	}

	handler := agentInstallerMiddleware(nextHandler, url, sb.BasePath)

	// let do some request but when there is no folder with static content
	t.Run("missing package folder", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://localhost/stork-install-agent.sh", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		resp := w.Result()
		resp.Body.Close()
		require.EqualValues(t, 500, resp.StatusCode)
		require.False(t, requestReceived)
	})

	// prepare folders
	packagesDir, _ := sb.JoinDir("assets/pkgs")

	t.Run("empty package directory", func(t *testing.T) {
		// let do some request
		req := httptest.NewRequest("GET", "http://localhost/stork-install-agent.sh", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		resp := w.Result()
		resp.Body.Close()
		require.EqualValues(t, 404, resp.StatusCode)
		require.False(t, requestReceived)
	})

	t.Run("only DEB package in the package directory", func(t *testing.T) {
		f, err := os.Create(path.Join(packagesDir, "isc-stork-agent.deb"))
		require.NoError(t, err)
		f.Close()
		defer os.Remove(f.Name())

		// let do some request
		req := httptest.NewRequest("GET", "http://localhost/stork-install-agent.sh", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()
		contentRaw, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		content := string(contentRaw)

		require.EqualValues(t, 200, resp.StatusCode)
		require.False(t, requestReceived)
		require.Contains(t, content, "http://localhost/assets/pkgs/isc-stork-agent.deb")
		require.Contains(t, content, "/etc/debian_version")
		require.NotContains(t, content, "/etc/redhat-release")
		require.NotContains(t, content, "/etc/alpine-release")
	})

	t.Run("only RPM package in the package directory", func(t *testing.T) {
		f, err := os.Create(path.Join(packagesDir, "isc-stork-agent.rpm"))
		require.NoError(t, err)
		f.Close()
		defer os.Remove(f.Name())

		// let do some request
		req := httptest.NewRequest("GET", "http://localhost/stork-install-agent.sh", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()
		contentRaw, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		content := string(contentRaw)

		require.EqualValues(t, 200, resp.StatusCode)
		require.False(t, requestReceived)
		require.Contains(t, content, "http://localhost/assets/pkgs/isc-stork-agent.rpm")
		require.NotContains(t, content, "/etc/debian_version")
		require.Contains(t, content, "/etc/redhat-release")
		require.NotContains(t, content, "/etc/alpine-release")
	})

	t.Run("only APK package in the package directory", func(t *testing.T) {
		f, err := os.Create(path.Join(packagesDir, "isc-stork-agent.apk"))
		require.NoError(t, err)
		f.Close()
		defer os.Remove(f.Name())

		// let do some request
		req := httptest.NewRequest("GET", "http://localhost/stork-install-agent.sh", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()
		contentRaw, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		content := string(contentRaw)

		require.EqualValues(t, 200, resp.StatusCode)
		require.False(t, requestReceived)
		require.Contains(t, content, "http://localhost/assets/pkgs/isc-stork-agent.apk")
		require.NotContains(t, content, "/etc/debian_version")
		require.NotContains(t, content, "/etc/redhat-release")
		require.Contains(t, content, "/etc/alpine-release")
	})

	t.Run("all packages in the package directory", func(t *testing.T) {
		// create all packages
		for _, extension := range []string{".deb", ".rpm", ".apk"} {
			f, err := os.Create(path.Join(packagesDir, "isc-stork-agent"+extension))
			require.NoError(t, err)
			f.Close()
			defer os.Remove(f.Name())
		}

		// let do some request
		req := httptest.NewRequest("GET", "http://localhost/stork-install-agent.sh", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()
		contentRaw, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		content := string(contentRaw)

		require.EqualValues(t, 200, resp.StatusCode)
		require.False(t, requestReceived)
		require.Contains(t, content, "/etc/debian_version")
		require.Contains(t, content, "/etc/redhat-release")
		require.Contains(t, content, "/etc/alpine-release")
		// The request is made over HTTP, the script should contain the same
		// scheme in the URL.
		require.Contains(t, content, "stork-agent register -u http://localhost")
		require.Contains(t, content, "curl -o /tmp/isc-stork-agent.rpm \"http://localhost/assets/pkgs/isc-stork-agent.rpm\"")
		require.Contains(t, content, "curl -o /tmp/isc-stork-agent.deb \"http://localhost/assets/pkgs/isc-stork-agent.deb\"")
		require.Contains(t, content, "wget -O /tmp/isc-stork-agent.apk \"http://localhost/assets/pkgs/isc-stork-agent.apk\"")
	})

	t.Run("all packages in the package directory - HTTPS", func(t *testing.T) {
		// create all packages
		for _, extension := range []string{".deb", ".rpm", ".apk"} {
			f, err := os.Create(path.Join(packagesDir, "isc-stork-agent"+extension))
			require.NoError(t, err)
			f.Close()
			defer os.Remove(f.Name())
		}

		// change handler to use HTTPS server address
		url := url
		url.Scheme = "https"
		handler := agentInstallerMiddleware(nextHandler, url, sb.BasePath)

		// let do some request
		req := httptest.NewRequest("GET", "https://localhost/stork-install-agent.sh", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()
		contentRaw, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		content := string(contentRaw)

		require.EqualValues(t, 200, resp.StatusCode)
		require.False(t, requestReceived)
		// The request is made over HTTPS, the script should contain the same
		// scheme in the URL.
		require.Contains(t, content, "stork-agent register -u https://localhost")
		require.Contains(t, content, "curl -o /tmp/isc-stork-agent.rpm \"https://localhost/assets/pkgs/isc-stork-agent.rpm\"")
		require.Contains(t, content, "curl -o /tmp/isc-stork-agent.deb \"https://localhost/assets/pkgs/isc-stork-agent.deb\"")
		require.Contains(t, content, "wget -O /tmp/isc-stork-agent.apk \"https://localhost/assets/pkgs/isc-stork-agent.apk\"")
	})

	t.Run("unsupported request", func(t *testing.T) {
		// let request something else, it should be forwarded to nextHandler
		req := httptest.NewRequest("GET", "http://localhost/api/users", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		require.True(t, requestReceived)
	})
}

// Check if the installation script is not generated from data provided by the
// user in the HTTP request (to avoid code injection).
func TestAgentInstallerMiddlewareServerAddressFromConfig(t *testing.T) {
	// Arrange
	sb := testutil.NewSandbox()
	defer sb.Close()
	_, _ = sb.Write("assets/pkgs/isc-stork-agent.deb", "")

	requestReceived := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestReceived = true
	})

	url := url.URL{
		Scheme: "https",
		Host:   "stork.example.com:8443",
		Path:   "/subdir",
	}

	handler := agentInstallerMiddleware(nextHandler, url, sb.BasePath)

	// Act
	req := httptest.NewRequest("GET", "http://localhost/stork-install-agent.sh", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Assert
	resp := w.Result()
	defer resp.Body.Close()
	require.EqualValues(t, 200, resp.StatusCode)
	require.False(t, requestReceived)

	contentRaw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	content := string(contentRaw)

	// The script should contain the server address from the configuration,
	// not from the HTTP Host header.
	require.Contains(t, content, "https://stork.example.com:8443/subdir")
	require.NotContains(t, content, "http://localhost")
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

// Check if metricsMiddleware returns placeholder when the endpoint is disabled.
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

	// Flush should cause no panic.
	require.NotPanics(t, lrw.Flush)
}

// Test the file middleware. Includes the test to check if the middleware
// is not vulnerable to the Path Traversal attack used to check if a given path
// exists on the filesystem.
func TestFileServerMiddlewareExtensive(t *testing.T) {
	// Arrange
	sb := testutil.NewSandbox()
	defer sb.Close()
	_, _ = sb.Write("restricted/secret", "password")
	_, _ = sb.Write("public/index.html", "index")
	publicDirectory, _ := sb.Write("public/file", "open")
	publicDirectory = path.Dir(publicDirectory)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	middleware := fileServerMiddleware(nextHandler, publicDirectory)

	requestFileContent := func(path string) (string, int, error) {
		request := httptest.NewRequest("GET", fmt.Sprintf("http://localhost/%s", path), nil)
		writer := httptest.NewRecorder()
		middleware.ServeHTTP(writer, request)
		response := writer.Result()
		content, err := io.ReadAll(response.Body)
		defer response.Body.Close()
		return string(content), response.StatusCode, err
	}

	t.Run("access to the public file", func(t *testing.T) {
		// Act
		content, status, err := requestFileContent("file")

		// Assert
		require.Equal(t, "open", content)
		require.Equal(t, 200, status)
		require.NoError(t, err)
	})

	t.Run("access to the public file with traversal", func(t *testing.T) {
		// Act
		content, status, err := requestFileContent("directory/../file")

		// Assert
		require.Equal(t, "open", content)
		require.Equal(t, 200, status)
		require.NoError(t, err)
	})

	t.Run("access to the index", func(t *testing.T) {
		// Act
		content, status, err := requestFileContent("/")

		// Assert
		require.Equal(t, "index", content)
		require.Equal(t, 200, status)
		require.NoError(t, err)
	})

	t.Run("access to the non-exist file", func(t *testing.T) {
		// Act
		content, status, err := requestFileContent("/foobar")

		// Assert
		require.Equal(t, "index", content)
		require.Equal(t, 200, status)
		require.NoError(t, err)
	})

	t.Run("access to the restricted file", func(t *testing.T) {
		// Act
		content, status, err := requestFileContent("/../restricted/secret")

		// Assert
		require.Equal(t, "invalid URL path\n", content)
		require.Equal(t, 400, status)
		require.NoError(t, err)
	})

	t.Run("access to the restricted directory using a relative path", func(t *testing.T) {
		// Act
		content, status, err := requestFileContent("../restricted")

		// Assert
		require.Equal(t, "invalid URL path\n", content)
		require.Equal(t, 400, status)
		require.NoError(t, err)
	})

	t.Run("access to the restricted file using a relative path", func(t *testing.T) {
		// Act
		content, status, err := requestFileContent("../restricted/secret")

		// Assert
		require.Equal(t, "invalid URL path\n", content)
		require.Equal(t, 400, status)
		require.NoError(t, err)
	})

	t.Run("access to the non-existing file with traversal", func(t *testing.T) {
		// Act
		content, status, err := requestFileContent("/public/directory/../foobar")

		// Assert
		require.Equal(t, "invalid URL path\n", content)
		require.Equal(t, 400, status)
		require.NoError(t, err)
	})

	t.Run("access to the restricted directory", func(t *testing.T) {
		// Act
		content, status, err := requestFileContent("/../restricted")

		// Assert
		require.Equal(t, "invalid URL path\n", content)
		require.Equal(t, 400, status)
		require.NoError(t, err)
	})

	t.Run("access to the restricted non-existing file", func(t *testing.T) {
		// Act
		content, status, err := requestFileContent("/../restricted/foobar")

		// Assert
		require.Equal(t, "invalid URL path\n", content)
		require.Equal(t, 400, status)
		require.NoError(t, err)
	})

	t.Run("access to the restricted non-existing file using a relative path", func(t *testing.T) {
		// Act
		content, status, err := requestFileContent("../restricted/foobar")

		// Assert
		require.Equal(t, "invalid URL path\n", content)
		require.Equal(t, 400, status)
		require.NoError(t, err)
	})
}

// Test that the trim base URL middleware is not created if the base URL is
// root or empty.
func TestTrimBaseURLMiddlewareNotApplicable(t *testing.T) {
	// Arrange
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	t.Run("root as base URL", func(t *testing.T) {
		// Act
		middleware := trimBaseURLMiddleware(nextHandler, "/")

		// Assert
		require.Equal(t, reflect.ValueOf(nextHandler).Pointer(), reflect.ValueOf(middleware).Pointer())
	})

	t.Run("empty base URL", func(t *testing.T) {
		// Act
		middleware := trimBaseURLMiddleware(nextHandler, "")

		// Assert
		require.Equal(t, reflect.ValueOf(nextHandler).Pointer(), reflect.ValueOf(middleware).Pointer())
	})
}

// Test that the trim base URL middleware works properly.
func TestTrimBaseURLMiddleware(t *testing.T) {
	// Arrange
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	middleware := trimBaseURLMiddleware(nextHandler, "/base/")

	request := func(path string) *url.URL {
		path = strings.TrimPrefix(path, "/")
		request := httptest.NewRequest("GET", fmt.Sprintf("http://localhost/%s", path), nil)
		writer := httptest.NewRecorder()
		middleware.ServeHTTP(writer, request)
		return request.URL
	}

	t.Run("no base URL in the request URL", func(t *testing.T) {
		// Act
		url := request("/foobar")

		// Assert
		require.EqualValues(t, "/foobar", url.Path)
	})

	t.Run("root as the request URL", func(t *testing.T) {
		// Act
		url := request("/")

		// Assert
		require.EqualValues(t, "/", url.Path)
	})

	t.Run("trim base URL from the request URL to root", func(t *testing.T) {
		// Act
		url := request("/base/")

		// Assert
		require.EqualValues(t, "/", url.Path)
	})

	t.Run("trim base URL from the request URL to subdirectory", func(t *testing.T) {
		// Act
		url := request("/base/endpoint")

		// Assert
		require.EqualValues(t, "/endpoint", url.Path)
	})
}

// Test that the request body size is limited by the middleware.
func TestMaxBodySizeMiddleware(t *testing.T) {
	// Arrange
	// Creates a middleware that limits the request body size to 1024 bytes.
	// Returns the created middleware and a function to get the body of the
	// incoming request.
	createMiddleware := func() (http.Handler, func() io.ReadCloser) {
		var outputRequest *http.Request
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			outputRequest = r
		})
		middleware := bodySizeLimiterMiddleware(nextHandler, 1024)
		return middleware, func() io.ReadCloser {
			if outputRequest == nil {
				return nil
			}
			return outputRequest.Body
		}
	}

	request := func(body string) *http.Request {
		req := httptest.NewRequest("POST", "http://localhost/", strings.NewReader(body))
		return req
	}

	t.Run("request body size is within the limit", func(t *testing.T) {
		// Act
		middleware, getBody := createMiddleware()
		req := request(strings.Repeat("a", 1024))
		writer := httptest.NewRecorder()
		middleware.ServeHTTP(writer, req)

		// Assert
		resp := writer.Result()
		require.EqualValues(t, 200, resp.StatusCode)
		incomingBody := getBody()
		defer incomingBody.Close()
		_, err := io.ReadAll(incomingBody)
		require.NoError(t, err)
	})

	t.Run("request body size exceeds the limit", func(t *testing.T) {
		// Act
		middleware, getBody := createMiddleware()
		req := request(strings.Repeat("a", 1025))
		writer := httptest.NewRecorder()
		middleware.ServeHTTP(writer, req)

		// Assert
		resp := writer.Result()
		require.EqualValues(t, 200, resp.StatusCode)
		incomingBody := getBody()
		defer incomingBody.Close()
		// Check that the body is truncated to the limit.
		buffer := make([]byte, 1024)
		n, err := io.ReadAtLeast(incomingBody, buffer, 1024)
		require.NoError(t, err)
		require.EqualValues(t, 1024, n)

		// Read a byte above the content length to ensure no more data is
		// available.
		_, err = io.ReadAtLeast(incomingBody, buffer, 1)
		require.Error(t, err, "http: request body too large")
	})

	t.Run("request body size is zero", func(t *testing.T) {
		// Act
		middleware, getBody := createMiddleware()
		req := request("")
		writer := httptest.NewRecorder()
		middleware.ServeHTTP(writer, req)

		// Assert
		resp := writer.Result()
		// Zero content length is treated as an unknown content length.
		require.EqualValues(t, 200, resp.StatusCode)
		incomingBody := getBody()
		defer incomingBody.Close()
		content, err := io.ReadAll(incomingBody)
		require.NoError(t, err)
		require.Empty(t, content)
	})

	t.Run("request body is empty", func(t *testing.T) {
		// Act
		middleware, getBody := createMiddleware()
		req := httptest.NewRequest("POST", "http://localhost/", nil)
		writer := httptest.NewRecorder()
		middleware.ServeHTTP(writer, req)

		// Assert
		resp := writer.Result()
		require.EqualValues(t, 200, resp.StatusCode)
		incomingBody := getBody()
		require.Equal(t, http.NoBody, incomingBody)
	})

	t.Run("request content length exceeds the limit", func(t *testing.T) {
		// Act
		middleware, getBody := createMiddleware()
		req := request("foo")
		req.ContentLength = 2048 // Set content length to 2048 bytes.
		writer := httptest.NewRecorder()
		middleware.ServeHTTP(writer, req)

		// Assert
		resp := writer.Result()
		require.EqualValues(t, 200, resp.StatusCode)
		incomingBody := getBody()
		defer incomingBody.Close()
		content, err := io.ReadAll(incomingBody)
		require.NoError(t, err)
		require.EqualValues(t, "foo", content)
	})

	t.Run("request body exceeds the limit but content length does not", func(t *testing.T) {
		// Act
		middleware, getBody := createMiddleware()
		req := request(strings.Repeat("a", 2048)) // Body is 2048 bytes.
		req.ContentLength = 1024                  // Content length is set to 1024 bytes.
		writer := httptest.NewRecorder()
		middleware.ServeHTTP(writer, req)

		// Assert
		resp := writer.Result()
		require.EqualValues(t, 200, resp.StatusCode)
		incomingBody := getBody()
		defer incomingBody.Close()
		// Check that the body is truncated to the content length.
		buffer := make([]byte, 1024)
		n, err := io.ReadAtLeast(incomingBody, buffer, 1024)

		require.NoError(t, err)
		require.EqualValues(t, 1024, n)

		// Read a byte above the content length to ensure no more data is
		// available.
		_, err = io.ReadAtLeast(incomingBody, buffer, 1)
		require.Error(t, err, "http: request body too large")
	})

	t.Run("content length is unknown", func(t *testing.T) {
		// Act
		middleware, getBody := createMiddleware()
		req := request("foo")
		req.ContentLength = -1 // Set content length to unknown.
		writer := httptest.NewRecorder()
		middleware.ServeHTTP(writer, req)

		// Assert
		resp := writer.Result()
		require.EqualValues(t, 200, resp.StatusCode)
		incomingBody := getBody()
		defer incomingBody.Close()
		content, err := io.ReadAll(incomingBody)
		require.NoError(t, err)
		require.EqualValues(t, "foo", content)
	})
}

// Check if HTTP security headers are added to the response by middleware.
func TestSecurityHeadersMiddleware(t *testing.T) {
	// Arrange
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	// Act
	handler := securityHeadersMiddleware(nextHandler)
	req := httptest.NewRequest("GET", "http://localhost/api/version", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	resp := w.Result()
	resp.Body.Close()

	// Assert
	require.Contains(t, resp.Header, "X-Frame-Options")
	require.Contains(t, resp.Header, "X-Content-Type-Options")
	require.Contains(t, resp.Header, "Strict-Transport-Security")
	require.Contains(t, resp.Header, "Content-Security-Policy")
	require.Contains(t, resp.Header.Get("X-Frame-Options"), "DENY")
	require.Contains(t, resp.Header.Get("X-Content-Type-Options"), "nosniff")
	require.Contains(t, resp.Header.Get("Strict-Transport-Security"), "max-age=")
	require.Contains(t, resp.Header.Get("Strict-Transport-Security"), "includeSubdomains")
	require.Contains(t, resp.Header.Get("Content-Security-Policy"), "frame-ancestors 'none'")
}
