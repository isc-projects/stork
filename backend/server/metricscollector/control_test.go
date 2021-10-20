package metricscollector

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that the control is properly created.
func TestConstructControler(t *testing.T) {
	// Act
	control := NewControl()

	// Assert
	require.NotNil(t, control)
}

// Test that the HTTP handler is created.
func TestCreateHttpHandler(t *testing.T) {
	// Arrange
	control := NewControl()
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	// Act
	handler := control.SetupHandler(nextHandler)

	// Assert
	require.NotNil(t, handler)
}

// Test that the handler responses with proper content
func TestHandlerResponse(t *testing.T) {
	// Arrange
	control := NewControl()
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	handler := control.SetupHandler(nextHandler)
	req := httptest.NewRequest("GET", "http://localhost/abc", nil)
	w := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(w, req)
	resp := w.Result()
	defer resp.Body.Close()
	bodyRaw, err := ioutil.ReadAll(resp.Body)
	body := string(bodyRaw)

	// Assert
	require.EqualValues(t, 200, resp.StatusCode)
	require.NoError(t, err)
	require.NotEmpty(t, body)
}
