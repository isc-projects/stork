package oidc

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	dbsession "isc.org/stork/server/database/session"
	dbtest "isc.org/stork/server/database/test"
)

// Test if OIDC controller can be created.
func TestNewController(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	controller := NewController(Settings{}, db)

	// Assert
	require.NotNil(t, controller)
	require.False(t, controller.configured)
}

// Test if OIDC controller internal configured flag is not set if mandatory setting is missing.
func TestConfigureSettingMissing(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	controller := NewController(Settings{}, db)
	require.NotNil(t, controller)

	// Act
	controller.Configure(url.URL{Scheme: "https"}, &dbsession.SessionMgr{})

	// Assert
	require.False(t, controller.configured)
}

// Test if OIDC controller internal configured flag is set.
func TestConfigure(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	controller := NewController(Settings{IssuerURL: "https://test.idp.org"}, db)
	require.NotNil(t, controller)

	// Act
	controller.Configure(url.URL{Scheme: "https"}, &dbsession.SessionMgr{})

	// Assert
	require.True(t, controller.configured)
}

// Test if OIDC middleware is transparent if OIDC was not configured.
func TestMiddlewareIsTransparent(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	controller := NewController(Settings{}, db)
	require.NotNil(t, controller)
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Hello", "world")
	})

	// Act
	handler := controller.Middleware(nextHandler)
	req := httptest.NewRequest("GET", "http://localhost/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	resp := w.Result()
	resp.Body.Close()

	// Assert
	require.False(t, controller.configured)
	require.Len(t, resp.Header, 1) // No other headers added by Session middleware means that OIDC middleware is transparent.
	require.Contains(t, resp.Header, "Hello")
	require.Contains(t, resp.Header.Get("Hello"), "world")
}

// Test if OIDC middleware is not transparent if OIDC was configured.
func TestMiddlewareIsNotTransparent(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	controller := NewController(Settings{IssuerURL: "https://test.idp.org"}, db)
	require.NotNil(t, controller)
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Hello", "world")
	})
	testSM, err := dbsession.NewSessionMgr(db)
	require.NoError(t, err)

	// Act
	controller.Configure(url.URL{Scheme: "https"}, testSM)
	handler := controller.Middleware(nextHandler)
	req := httptest.NewRequest("GET", "http://localhost/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	resp := w.Result()
	resp.Body.Close()

	// Assert
	require.True(t, controller.configured)
	require.Greater(t, len(resp.Header), 1) // There should be also session cookie from session manager.
	require.Contains(t, resp.Header, "Hello")
	require.Contains(t, resp.Header.Get("Hello"), "world")
}
