package oidc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/coreos/go-oidc/v3/oidc/oidctest"
	"github.com/stretchr/testify/require"
	"isc.org/stork/server/authdata"
	dbmodel "isc.org/stork/server/database/model"
	dbsession "isc.org/stork/server/database/session"
	dbtest "isc.org/stork/server/database/test"
)

// Helper function preparing test OIDC server which allows to test OIDC discovery
// and token exchange.
// It returns server URL as string which should be used as OIDC issuer URL,
// test server teardown function and an error if such occurred while generating
// RSA key.
func prepareTestOIDCServer() (string, func(), error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", nil, err
	}
	s := &oidctest.Server{
		PublicKeys: []oidctest.PublicKey{
			{
				PublicKey: priv.Public(),
				KeyID:     "test-key",
				Algorithm: oidc.RS256,
			},
		},
	}
	var serverURL string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			rawClaims := `{
				"iss": "` + serverURL + `",
				"aud": "clientID",
				"sub": "foo",
				"exp": ` + time.Now().Add(time.Hour).Format("1136239445") + `,
				"email": "foo@example.org",
				"email_verified": true,
				"nonce": "test-nonce",
				"groups": ["stork-users", "stork-super-admins"]
			}`
			token := oidctest.SignIDToken(priv, "test-key", oidc.RS256, rawClaims)
			resp := map[string]any{
				"access_token": "fake-access-token",
				"token_type":   "Bearer",
				"expires_in":   3600,
				"id_token":     token,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		default:
			s.ServeHTTP(w, r)
		}
	})
	srv := httptest.NewServer(handler)
	serverURL = srv.URL
	s.SetIssuer(serverURL)
	return serverURL, srv.Close, nil
}

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
	err := controller.Configure(url.URL{Scheme: "https"}, &dbsession.SessionMgr{})

	// Assert
	require.NoError(t, err)
	require.False(t, controller.configured)
	controller.settings.IssuerURL = "https://test.idp.org"
	err = controller.Configure(url.URL{Scheme: "https"}, &dbsession.SessionMgr{})
	require.Error(t, err)
	require.False(t, controller.configured) // ClientID is also mandatory setting.
}

// Test if OIDC controller internal configured flag is set.
func TestConfigure(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	issuerURL, srvTeardown, err := prepareTestOIDCServer()
	require.NoError(t, err)
	defer srvTeardown()
	controller := NewController(Settings{IssuerURL: issuerURL, ClientID: "clientID", ClientSecret: "client-secret"}, db)
	require.NotNil(t, controller)

	// Act
	err = controller.Configure(url.URL{Scheme: "http", Host: "[::]:8080"}, &dbsession.SessionMgr{})

	// Assert
	require.NoError(t, err)
	require.True(t, controller.configured)
	require.NotNil(t, controller.authSessionManager)
	require.Equal(t, "client-secret", controller.oauth2Config.ClientSecret)
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

// Test if OIDC middleware is transparent if OIDC was configured
// but the request URL path does not match any known OIDC-related endpoints.
func TestMiddlewareIsTransparent2(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	issuerURL, srvTeardown, err := prepareTestOIDCServer()
	require.NoError(t, err)
	defer srvTeardown()
	controller := NewController(Settings{IssuerURL: issuerURL, ClientID: "clientID"}, db)
	require.NotNil(t, controller)
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Hello", "world")
	})
	testSM, err := dbsession.NewSessionMgr(db)
	require.NoError(t, err)

	// Act
	err = controller.Configure(url.URL{Scheme: "https"}, testSM)
	require.NoError(t, err)
	handler := controller.Middleware(nextHandler)
	req := httptest.NewRequest("GET", "http://localhost/", nil) // This URL path does not match OIDC login or callback path.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	resp := w.Result()
	resp.Body.Close()

	// Assert
	require.True(t, controller.configured)
	require.Len(t, resp.Header, 1) // No other headers added by Session middleware means that OIDC middleware is transparent.
	require.Contains(t, resp.Header, "Hello")
	require.Contains(t, resp.Header.Get("Hello"), "world")
}

// Test if OIDC in-memory session storage can be read and written to.
func TestSessionStorage(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	issuerURL, srvTeardown, err := prepareTestOIDCServer()
	require.NoError(t, err)
	defer srvTeardown()
	controller := NewController(Settings{IssuerURL: issuerURL, ClientID: "clientID"}, db)
	require.NotNil(t, controller)
	var ctx context.Context
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx = r.Context()
	})
	testSM, err := dbsession.NewSessionMgr(db)
	require.NoError(t, err)
	err = controller.Configure(url.URL{Scheme: "https"}, testSM)
	require.NoError(t, err)
	handler := controller.Middleware(nextHandler)
	// Additionally force adding session manager context. It shouldn't be added for non-OIDC related URL paths.
	handler = controller.dbSessionManager.SessionMiddleware(controller.authSessionManager.LoadAndSave(handler))
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

// Test if in case of not configured controller and no existing session context
// getAuthSessionMap does not panic and returns nil.
func TestGetAuthSessionMapReturnsNil(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	controller := NewController(Settings{IssuerURL: "https://test.idp.org"}, db)
	require.NotNil(t, controller)

	// Act
	sessionMap := controller.getAuthSessionMap(context.TODO())

	// Assert
	require.Nil(t, sessionMap)
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

// Test if getMappedGroups works fine.
func TestGetMappedGroups(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	issuerURL, srvTeardown, err := prepareTestOIDCServer()
	require.NoError(t, err)
	defer srvTeardown()
	settings := Settings{
		IssuerURL:           issuerURL,
		MandatoryAllowGroup: "stork-access",
		ClientID:            "clientID",
	}
	controller := NewController(settings, db)
	require.NotNil(t, controller)
	testSM, err := dbsession.NewSessionMgr(db)
	require.NoError(t, err)
	err = controller.Configure(url.URL{Scheme: "https"}, testSM)
	require.NoError(t, err)
	require.True(t, controller.configured)
	receivedGroups := []string{
		"stork-access", "router-admins",
	}
	var allowed bool
	var mappedGroups []authdata.UserGroupID

	// Act & Assert
	t.Run("mandatory allow group configured and group mapping not configured - belongs to allow group", func(t *testing.T) {
		allowed, mappedGroups = controller.getMappedGroups(&receivedGroups)
		require.True(t, allowed)
		require.NotNil(t, mappedGroups)
		require.Empty(t, mappedGroups)
	})

	t.Run("mandatory allow group configured and group mapping not configured - not belongs to allow group", func(t *testing.T) {
		receivedGroups = []string{
			"router-admins", "kea-admins",
		}
		allowed, mappedGroups = controller.getMappedGroups(&receivedGroups)
		require.False(t, allowed)
		require.NotNil(t, mappedGroups)
		require.Empty(t, mappedGroups)
	})

	t.Run("mandatory allow group not configured and group mapping not configured", func(t *testing.T) {
		settings = Settings{
			IssuerURL:           "https://test.idp.org",
			MandatoryAllowGroup: "",
		}
		controller.settings = settings
		allowed, mappedGroups = controller.getMappedGroups(&receivedGroups)
		require.False(t, allowed)
		require.NotNil(t, mappedGroups)
		require.Empty(t, mappedGroups)
	})

	t.Run("mandatory allow group not configured and group mapping configured - not belongs to any default group", func(t *testing.T) {
		settings = Settings{
			IssuerURL:           "https://test.idp.org",
			MandatoryAllowGroup: "",
			EnableGroupMapping:  true,
		}
		controller.settings = settings
		allowed, mappedGroups = controller.getMappedGroups(&receivedGroups)
		require.False(t, allowed)
		require.NotNil(t, mappedGroups)
		require.Empty(t, mappedGroups)
	})

	t.Run("mandatory allow group not configured and group mapping configured - not belongs to any group", func(t *testing.T) {
		settings = Settings{
			IssuerURL:           "https://test.idp.org",
			MandatoryAllowGroup: "",
			EnableGroupMapping:  true,
			GroupMapping: GroupMapping{
				SuperAdmin: CommaSeparatedStrings{"stork-super-admins-1", "stork-super-admins-2"},
				Admin:      CommaSeparatedStrings{"stork-admins-1", "stork-admins-2"},
				ReadOnly:   CommaSeparatedStrings{"stork-ro-1", "stork-ro-2"},
			},
		}
		controller.settings = settings
		allowed, mappedGroups = controller.getMappedGroups(&receivedGroups)
		require.False(t, allowed)
		require.NotNil(t, mappedGroups)
		require.Empty(t, mappedGroups)
	})

	t.Run("mandatory allow group not configured and group mapping configured - belongs to all groups 1", func(t *testing.T) {
		receivedGroups = []string{
			"stork-super-admins-1", "stork-admins-1", "stork-ro-1",
		}
		allowed, mappedGroups = controller.getMappedGroups(&receivedGroups)
		require.False(t, allowed)
		require.NotNil(t, mappedGroups)
		require.NotEmpty(t, mappedGroups)
		require.Contains(t, mappedGroups, authdata.UserGroupIDSuperAdmin)
		require.Contains(t, mappedGroups, authdata.UserGroupIDAdmin)
		require.Contains(t, mappedGroups, authdata.UserGroupIDReadOnly)
	})

	t.Run("mandatory allow group not configured and group mapping configured - belongs to all groups 2", func(t *testing.T) {
		receivedGroups = []string{
			"stork-super-admins-2", "stork-admins-2", "stork-ro-2",
		}
		allowed, mappedGroups = controller.getMappedGroups(&receivedGroups)
		require.False(t, allowed)
		require.NotNil(t, mappedGroups)
		require.NotEmpty(t, mappedGroups)
		require.Contains(t, mappedGroups, authdata.UserGroupIDSuperAdmin)
		require.Contains(t, mappedGroups, authdata.UserGroupIDAdmin)
		require.Contains(t, mappedGroups, authdata.UserGroupIDReadOnly)
	})

	t.Run("mandatory allow group configured and group mapping configured - belongs to all groups but does not belong to allow group", func(t *testing.T) {
		controller.settings.MandatoryAllowGroup = "super-heroes"
		allowed, mappedGroups = controller.getMappedGroups(&receivedGroups)
		require.False(t, allowed)
		require.NotNil(t, mappedGroups)
		require.NotEmpty(t, mappedGroups)
		require.Contains(t, mappedGroups, authdata.UserGroupIDSuperAdmin)
		require.Contains(t, mappedGroups, authdata.UserGroupIDAdmin)
		require.Contains(t, mappedGroups, authdata.UserGroupIDReadOnly)
	})

	t.Run("mandatory allow group configured and group mapping configured - belongs to all groups and belongs to allow group", func(t *testing.T) {
		receivedGroups = []string{
			"stork-super-admins-2", "stork-admins-2", "stork-ro-2", "super-heroes",
		}
		allowed, mappedGroups = controller.getMappedGroups(&receivedGroups)
		require.True(t, allowed)
		require.NotNil(t, mappedGroups)
		require.NotEmpty(t, mappedGroups)
		require.Contains(t, mappedGroups, authdata.UserGroupIDSuperAdmin)
		require.Contains(t, mappedGroups, authdata.UserGroupIDAdmin)
		require.Contains(t, mappedGroups, authdata.UserGroupIDReadOnly)
	})
}

// Test if OIDC middleware is not transparent if OIDC was configured
// and the request URL path matches OIDC login endpoint.
func TestMiddlewareHandlesLoginEndpoint(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	issuerURL, srvTeardown, err := prepareTestOIDCServer()
	require.NoError(t, err)
	defer srvTeardown()
	controller := NewController(Settings{IssuerURL: issuerURL, ClientID: "clientID"}, db)
	require.NotNil(t, controller)
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Hello", "world")
	})
	testSM, err := dbsession.NewSessionMgr(db)
	require.NoError(t, err)

	// Act
	err = controller.Configure(url.URL{Scheme: "https"}, testSM)
	require.NoError(t, err)
	handler := controller.Middleware(nextHandler)
	req := httptest.NewRequest("GET", "http://localhost"+loginURLPath, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	resp := w.Result()
	resp.Body.Close()

	// Assert
	require.True(t, controller.configured)
	require.Greater(t, len(resp.Header), 2)
	// Check auth_session cookie.
	require.Contains(t, resp.Header, "Set-Cookie")
	require.Contains(t, resp.Header.Get("Set-Cookie"), "auth_session")
	// Check redirect Location header. It should have the client_id value in the redirection URL.
	require.Contains(t, resp.Header, "Location")
	require.Contains(t, resp.Header.Get("Location"), "clientID")
	require.NotContains(t, resp.Header, "Hello")
}

// Test if sanitizeReturnURL works fine.
func TestSanitizeReturnURL(t *testing.T) {
	// Arrange & Act & Assert
	u := ""
	t.Run("trim spaces", func(t *testing.T) {
		u = sanitizeReturnURL("  test  ")
		require.EqualValues(t, "/test", u)
	})

	t.Run("replace CR and LF chars", func(t *testing.T) {
		u = sanitizeReturnURL("  te\n\rst  ")
		require.EqualValues(t, "/test", u)
	})

	t.Run("sanitize absolute URL", func(t *testing.T) {
		u = sanitizeReturnURL("https://example.org")
		require.EqualValues(t, "/", u)
	})

	t.Run("sanitize protocol-relative URL", func(t *testing.T) {
		u = sanitizeReturnURL("//example.org")
		require.EqualValues(t, "/", u)
	})

	t.Run("URL path with query param", func(t *testing.T) {
		u = sanitizeReturnURL("dns/zones/all?daemonId=17")
		require.EqualValues(t, "/dns/zones/all?daemonId=17", u)
	})

	t.Run("sanitize protocol-relative very long URL", func(t *testing.T) {
		u = sanitizeReturnURL("//example.org" + strings.Repeat("g", 300))
		require.EqualValues(t, "/", u)
	})
}

// Test if OIDC middleware is not transparent if OIDC was configured
// and the request URL path matches OIDC callback endpoint.
func TestMiddlewareHandlesCallbackEndpoint(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	issuerURL, srvTeardown, err := prepareTestOIDCServer()
	require.NoError(t, err)
	defer srvTeardown()
	controller := NewController(Settings{IssuerURL: issuerURL, ClientID: "clientID"}, db)
	require.NotNil(t, controller)
	testSM, err := dbsession.NewSessionMgr(db)
	require.NoError(t, err)
	controller.Configure(url.URL{Scheme: "http", Path: "localhost"}, testSM)
	require.True(t, controller.configured)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// empty handler
	})
	handler := controller.Middleware(nextHandler)

	// Act
	req := httptest.NewRequest("GET", "http://localhost"+callbackURLPath+"?state=state", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	resp := w.Result()
	resp.Body.Close()

	// Assert
	require.Equal(t, 302, resp.StatusCode)
	require.Greater(t, len(resp.Header), 2)
	// Check auth_session cookie.
	require.Contains(t, resp.Header, "Set-Cookie")
	require.Contains(t, resp.Header.Get("Set-Cookie"), "auth_session")
	// Check redirect Location header. It should have the login URL path in the redirection URL.
	require.Contains(t, resp.Header, "Location")
	require.Contains(t, resp.Header.Get("Location"), "/login")
}

// Helper function populating cookies from HTTP response to new HTTP request.
// Used to keep session between httptest Requests.
func populateCookies(from *http.Response, to *http.Request) {
	for _, c := range from.Cookies() {
		to.AddCookie(c)
	}
}

// Test that callback handler handles error in the redirection URL.
func TestCallbackEndpointHandlesError(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	issuerURL, srvTeardown, err := prepareTestOIDCServer()
	require.NoError(t, err)
	defer srvTeardown()
	controller := NewController(Settings{IssuerURL: issuerURL, ClientID: "clientID"}, db)
	require.NotNil(t, controller)
	testSM, err := dbsession.NewSessionMgr(db)
	require.NoError(t, err)
	controller.Configure(url.URL{Scheme: "http", Path: "localhost"}, testSM)
	require.True(t, controller.configured)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// empty handler
	})
	handler := controller.Middleware(nextHandler)

	// Act
	// First send request to login endpoint to retrieve random state for the authentication.
	req := httptest.NewRequest("GET", "http://localhost"+loginURLPath, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	resp := w.Result()
	resp.Body.Close()
	require.Equal(t, 302, resp.StatusCode)
	require.Contains(t, resp.Header, "Location")
	redirectURL := resp.Header.Get("Location")
	parsedURL, err := url.Parse(redirectURL)
	require.NoError(t, err)
	state := parsedURL.Query().Get("state")

	// Send request to callback endpoint.
	req2 := httptest.NewRequest("GET", "http://localhost"+callbackURLPath+"?state="+state+"&error=123&error_description=testError", nil)
	w2 := httptest.NewRecorder()
	populateCookies(resp, req2)
	handler.ServeHTTP(w2, req2)
	resp2 := w2.Result()
	resp2.Body.Close()

	// Assert
	require.Equal(t, 302, resp2.StatusCode)
	// Check redirect Location header. It should redirect to login page showing brief error feedback message.
	require.Contains(t, resp2.Header, "Location")
	require.Contains(t, resp2.Header.Get("Location"), "/login/auth-err")
}

// Helper middleware modifying stored nonces in auth session manager.
// Useful for testing OIDC callback handler.
func nonceModifier(h http.Handler, c *Controller, nonce string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m := c.getAuthSessionMap(r.Context())
		if m != nil {
			for k, v := range m {
				v.Nonce = nonce
				m[k] = v
			}
			c.putAuthSessionMap(r.Context(), m)
		}
		h.ServeHTTP(w, r)
	})
}

// Test that callback handler handles token exchange error.
func TestCallbackEndpointHandlesTokenExchangeError(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	issuerURL, srvTeardown, err := prepareTestOIDCServer()
	require.NoError(t, err)
	defer srvTeardown()
	controller := NewController(Settings{IssuerURL: issuerURL, ClientID: "clientID"}, db)
	require.NotNil(t, controller)
	testSM, err := dbsession.NewSessionMgr(db)
	require.NoError(t, err)
	controller.Configure(url.URL{Scheme: "http", Path: "localhost"}, testSM)
	// Modify token endpoint so that token exchange should fail.
	controller.oauth2Config.Endpoint.TokenURL = "http://localhost/dummyFooBar"
	require.True(t, controller.configured)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// empty handler
	})
	handler := controller.Middleware(nextHandler)

	// Act
	// First send request to login endpoint to retrieve random state for the authentication.
	req := httptest.NewRequest("GET", "http://localhost"+loginURLPath, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	resp := w.Result()
	resp.Body.Close()
	require.Equal(t, 302, resp.StatusCode)
	require.Contains(t, resp.Header, "Location")
	redirectURL := resp.Header.Get("Location")
	parsedURL, err := url.Parse(redirectURL)
	require.NoError(t, err)
	state := parsedURL.Query().Get("state")

	// Send request to callback endpoint.
	req2 := httptest.NewRequest("GET", "http://localhost"+callbackURLPath+"?state="+state, nil)
	w2 := httptest.NewRecorder()
	populateCookies(resp, req2)
	handler.ServeHTTP(w2, req2)
	resp2 := w2.Result()
	resp2.Body.Close()

	// Assert
	require.Equal(t, 302, resp2.StatusCode)
	// Check redirect Location header. It should redirect to login page showing brief error feedback message.
	require.Contains(t, resp2.Header, "Location")
	require.Contains(t, resp2.Header.Get("Location"), "/login/auth-err")
}

// Test that callback handler handles token response verification error.
func TestCallbackEndpointHandlesTokenRespVerificationError(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	issuerURL, srvTeardown, err := prepareTestOIDCServer()
	require.NoError(t, err)
	defer srvTeardown()
	// Construct controller with wrong ClientID that doesn't match with the one in fake OpenID Provider server.
	// This should cause token response verification error.
	controller := NewController(Settings{IssuerURL: issuerURL, ClientID: "wrongClientID"}, db)
	require.NotNil(t, controller)
	testSM, err := dbsession.NewSessionMgr(db)
	require.NoError(t, err)
	controller.Configure(url.URL{Scheme: "http", Path: "localhost"}, testSM)
	require.True(t, controller.configured)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// empty handler
	})
	handler := controller.Middleware(nextHandler)

	// Act
	// First send request to login endpoint to retrieve random state for the authentication.
	req := httptest.NewRequest("GET", "http://localhost"+loginURLPath, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	resp := w.Result()
	resp.Body.Close()
	require.Equal(t, 302, resp.StatusCode)
	require.Contains(t, resp.Header, "Location")
	redirectURL := resp.Header.Get("Location")
	parsedURL, err := url.Parse(redirectURL)
	require.NoError(t, err)
	state := parsedURL.Query().Get("state")

	// Send request to callback endpoint.
	req2 := httptest.NewRequest("GET", "http://localhost"+callbackURLPath+"?state="+state, nil)
	w2 := httptest.NewRecorder()
	populateCookies(resp, req2)
	handler.ServeHTTP(w2, req2)
	resp2 := w2.Result()
	resp2.Body.Close()

	// Assert
	require.Equal(t, 302, resp2.StatusCode)
	// Check redirect Location header. It should redirect to login page showing brief error feedback message.
	require.Contains(t, resp2.Header, "Location")
	require.Contains(t, resp2.Header.Get("Location"), "/login/auth-err")
}

// Test that callback handler handles invalid nonce error.
func TestCallbackEndpointHandlesWrongNonce(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	issuerURL, srvTeardown, err := prepareTestOIDCServer()
	require.NoError(t, err)
	defer srvTeardown()
	controller := NewController(Settings{IssuerURL: issuerURL, ClientID: "clientID"}, db)
	require.NotNil(t, controller)
	testSM, err := dbsession.NewSessionMgr(db)
	require.NoError(t, err)
	controller.Configure(url.URL{Scheme: "http", Path: "localhost"}, testSM)
	require.True(t, controller.configured)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// empty handler
	})
	// Use invalid nonce.
	handler := nonceModifier(controller.Middleware(nextHandler), controller, "invalid-nonce")

	// Act
	// First send request to login endpoint to retrieve random state for the authentication.
	req := httptest.NewRequest("GET", "http://localhost"+loginURLPath, nil)
	w := httptest.NewRecorder()
	controller.authSessionManager.LoadAndSave(handler).ServeHTTP(w, req)
	resp := w.Result()
	resp.Body.Close()
	require.Equal(t, 302, resp.StatusCode)
	require.Contains(t, resp.Header, "Location")
	redirectURL := resp.Header.Get("Location")
	parsedURL, err := url.Parse(redirectURL)
	require.NoError(t, err)
	state := parsedURL.Query().Get("state")

	// Send request to callback endpoint.
	req2 := httptest.NewRequest("GET", "http://localhost"+callbackURLPath+"?state="+state, nil)
	w2 := httptest.NewRecorder()
	populateCookies(resp, req2)
	controller.authSessionManager.LoadAndSave(handler).ServeHTTP(w2, req2)
	resp2 := w2.Result()
	resp2.Body.Close()

	// Assert
	require.Equal(t, 302, resp2.StatusCode)
	// Check redirect Location header. It should redirect to login page showing brief error feedback message.
	require.Contains(t, resp2.Header, "Location")
	require.Contains(t, resp2.Header.Get("Location"), "/login/auth-err")
}

// Test that callback handler handles user not allowed to access Stork.
func TestCallbackEndpointHandlesUnauthorizedUser(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	issuerURL, srvTeardown, err := prepareTestOIDCServer()
	require.NoError(t, err)
	defer srvTeardown()
	controller := NewController(Settings{IssuerURL: issuerURL, ClientID: "clientID", GroupsClaim: "groups", MandatoryAllowGroup: "foo"}, db)
	require.NotNil(t, controller)
	testSM, err := dbsession.NewSessionMgr(db)
	require.NoError(t, err)
	controller.Configure(url.URL{Scheme: "http", Path: "localhost"}, testSM)
	require.True(t, controller.configured)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// empty handler
	})
	handler := nonceModifier(controller.Middleware(nextHandler), controller, "test-nonce")

	// Act
	// First send request to login endpoint to retrieve random state for the authentication.
	req := httptest.NewRequest("GET", "http://localhost"+loginURLPath, nil)
	w := httptest.NewRecorder()
	controller.authSessionManager.LoadAndSave(handler).ServeHTTP(w, req)
	resp := w.Result()
	resp.Body.Close()
	require.Equal(t, 302, resp.StatusCode)
	require.Contains(t, resp.Header, "Location")
	redirectURL := resp.Header.Get("Location")
	parsedURL, err := url.Parse(redirectURL)
	require.NoError(t, err)
	state := parsedURL.Query().Get("state")

	// Send request to callback endpoint.
	req2 := httptest.NewRequest("GET", "http://localhost"+callbackURLPath+"?state="+state, nil)
	w2 := httptest.NewRecorder()
	populateCookies(resp, req2)
	controller.authSessionManager.LoadAndSave(handler).ServeHTTP(w2, req2)
	resp2 := w2.Result()
	resp2.Body.Close()

	// Assert
	require.Equal(t, 302, resp2.StatusCode)
	// Check redirect Location header. It should redirect to login page showing brief error feedback message.
	require.Contains(t, resp2.Header, "Location")
	require.Contains(t, resp2.Header.Get("Location"), "/login/unauthorized")
}

// Test that callback handler authorizes a user.
func TestCallbackEndpointAuthorizesUser(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	issuerURL, srvTeardown, err := prepareTestOIDCServer()
	require.NoError(t, err)
	defer srvTeardown()
	settings := Settings{
		IssuerURL:           issuerURL,
		ClientID:            "clientID",
		GroupsClaim:         "groups",
		MandatoryAllowGroup: "stork-users",
		EnableGroupMapping:  true,
		GroupMapping: GroupMapping{
			SuperAdmin: CommaSeparatedStrings{"stork-super-admins"},
		},
	}
	controller := NewController(settings, db)
	require.NotNil(t, controller)
	testSM, err := dbsession.NewSessionMgr(db)
	require.NoError(t, err)
	controller.Configure(url.URL{Scheme: "http", Path: "localhost"}, testSM)
	require.True(t, controller.configured)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// empty handler
	})
	handler := nonceModifier(controller.Middleware(nextHandler), controller, "test-nonce")

	// Act
	// First send request to login endpoint to retrieve random state for the authentication.
	req := httptest.NewRequest("GET", "http://localhost"+loginURLPath, nil)
	w := httptest.NewRecorder()
	controller.authSessionManager.LoadAndSave(handler).ServeHTTP(w, req)
	resp := w.Result()
	resp.Body.Close()
	require.Equal(t, 302, resp.StatusCode)
	require.Contains(t, resp.Header, "Location")
	redirectURL := resp.Header.Get("Location")
	parsedURL, err := url.Parse(redirectURL)
	require.NoError(t, err)
	state := parsedURL.Query().Get("state")

	// Send request to callback endpoint.
	req2 := httptest.NewRequest("GET", "http://localhost"+callbackURLPath+"?state="+state, nil)
	w2 := httptest.NewRecorder()
	populateCookies(resp, req2)
	controller.authSessionManager.LoadAndSave(handler).ServeHTTP(w2, req2)
	resp2 := w2.Result()
	resp2.Body.Close()

	// Assert
	require.Equal(t, 302, resp2.StatusCode)
	// Check redirect Location header. It should redirect to home "/" path.
	require.Contains(t, resp2.Header, "Location")
	require.EqualValues(t, resp2.Header.Get("Location"), "/")
	// Check if user was created in DB.
	dbUser, err := dbmodel.GetUserByExternalID(db, "oidc", "foo")
	require.NoError(t, err)
	require.NotNil(t, dbUser)
	dbUser, err = dbmodel.GetUserByID(db, dbUser.ID)
	require.NoError(t, err)
	require.NotNil(t, dbUser)
	require.EqualValues(t, "foo@example.org", dbUser.Email)
	require.NotNil(t, dbUser.Groups)
	require.Len(t, dbUser.Groups, 1)
	require.Equal(t, dbmodel.SuperAdminGroupID, dbUser.Groups[0].ID)
}

// Test that callback handler authorizes a user with group mapping disabled.
func TestCallbackEndpointAuthorizesUserGroupMappingDisabled(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	issuerURL, srvTeardown, err := prepareTestOIDCServer()
	require.NoError(t, err)
	defer srvTeardown()
	settings := Settings{
		IssuerURL:           issuerURL,
		ClientID:            "clientID",
		GroupsClaim:         "groups",
		MandatoryAllowGroup: "stork-users",
	}
	controller := NewController(settings, db)
	require.NotNil(t, controller)
	testSM, err := dbsession.NewSessionMgr(db)
	require.NoError(t, err)
	controller.Configure(url.URL{Scheme: "http", Path: "localhost"}, testSM)
	require.True(t, controller.configured)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// empty handler
	})
	handler := nonceModifier(controller.Middleware(nextHandler), controller, "test-nonce")

	// Act
	// First send request to login endpoint to retrieve random state for the authentication.
	req := httptest.NewRequest("GET", "http://localhost"+loginURLPath, nil)
	w := httptest.NewRecorder()
	controller.authSessionManager.LoadAndSave(handler).ServeHTTP(w, req)
	resp := w.Result()
	resp.Body.Close()
	require.Equal(t, 302, resp.StatusCode)
	require.Contains(t, resp.Header, "Location")
	redirectURL := resp.Header.Get("Location")
	parsedURL, err := url.Parse(redirectURL)
	require.NoError(t, err)
	state := parsedURL.Query().Get("state")

	// Send request to callback endpoint.
	req2 := httptest.NewRequest("GET", "http://localhost"+callbackURLPath+"?state="+state, nil)
	w2 := httptest.NewRecorder()
	populateCookies(resp, req2)
	controller.authSessionManager.LoadAndSave(handler).ServeHTTP(w2, req2)
	resp2 := w2.Result()
	resp2.Body.Close()

	// Assert
	require.Equal(t, 302, resp2.StatusCode)
	// Check redirect Location header. It should redirect to home "/" path.
	require.Contains(t, resp2.Header, "Location")
	require.EqualValues(t, resp2.Header.Get("Location"), "/")
	// Check if user was created in DB.
	dbUser, err := dbmodel.GetUserByExternalID(db, "oidc", "foo")
	require.NoError(t, err)
	require.NotNil(t, dbUser)
	dbUser, err = dbmodel.GetUserByID(db, dbUser.ID)
	require.NoError(t, err)
	require.NotNil(t, dbUser)
	require.EqualValues(t, "foo@example.org", dbUser.Email)
	require.NotNil(t, dbUser.Groups)
	require.Len(t, dbUser.Groups, 1)
	require.Equal(t, dbmodel.ReadOnlyGroupID, dbUser.Groups[0].ID)
}
