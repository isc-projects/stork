package oidc

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

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
	require.NotNil(t, controller.authSessionManager)
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

// Test if OIDC in-memory session storage can be read and written to.
func TestSessionStorage(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	controller := NewController(Settings{IssuerURL: "https://test.idp.org"}, db)
	require.NotNil(t, controller)
	var ctx context.Context
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx = r.Context()
	})
	testSM, err := dbsession.NewSessionMgr(db)
	require.NoError(t, err)
	controller.Configure(url.URL{Scheme: "https"}, testSM)
	handler := controller.Middleware(nextHandler)
	req := httptest.NewRequest("GET", "http://localhost/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	resp := w.Result()
	resp.Body.Close()
	require.NotNil(t, controller)
	require.NotNil(t, controller.authSessionManager)
	authSession := AuthSession{
		CodeVerifier: "codeVerifier",
		Nonce:        "nonce",
		ReturnURL:    "/",
		CreatedAt:    time.Now().Add(-16 * time.Minute),
	}
	state := "state"

	// Act
	sessionMap := controller.getAuthSessionMap(ctx)

	// Assert
	require.NotNil(t, sessionMap)
	require.Empty(t, sessionMap)
	sessionMap[state] = authSession

	controller.putAuthSessionMap(ctx, sessionMap)

	sessionMap2 := controller.getAuthSessionMap(ctx)
	require.NotNil(t, sessionMap2)
	require.NotEmpty(t, sessionMap2)
	authSession2 := sessionMap2[state]
	require.Equal(t, authSession, authSession2)

	// Check if the cleanup works fine.
	controller.cleanupSessions(ctx)

	sessionMap3 := controller.getAuthSessionMap(ctx)
	require.NotNil(t, sessionMap3)
	require.Empty(t, sessionMap3)
}

// Test if generateRandBytes works fine.
func TestGenerateRandBytes(t *testing.T) {
	// Arrange & Act
	testBytes, err := generateRandBytes(32)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, testBytes)
	require.Len(t, testBytes, 32)
}

// Test if generateRandBase64Str works fine.
func TestGenerateRandBase64Str(t *testing.T) {
	// Arrange & Act
	testString, err := generateRandBase64Str()

	// Assert
	require.NoError(t, err)
	require.NotNil(t, testString)
	require.NotEmpty(t, testString)
	decoded, err := base64.RawURLEncoding.DecodeString(testString)
	require.NoError(t, err)
	require.NotNil(t, decoded)
	require.NotEmpty(t, decoded)
	require.Len(t, decoded, 32)
}

// Test if generatePKCE works fine.
func TestGeneratePKCE(t *testing.T) {
	// Arrange & Act
	codeVerifier, codeChallenge, err := generatePKCE()

	// Assert
	require.NoError(t, err)
	require.NotNil(t, codeVerifier)
	require.NotNil(t, codeChallenge)
	require.NotEmpty(t, codeVerifier)
	require.NotEmpty(t, codeChallenge)
	decoded, err := base64.RawURLEncoding.DecodeString(codeVerifier)
	require.NoError(t, err)
	require.NotNil(t, decoded)
	require.NotEmpty(t, decoded)
	require.Len(t, decoded, 32)
}
