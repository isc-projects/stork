package oidc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/gob"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/alexedwards/scs/v2"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"isc.org/stork/server/authdata"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
	dbsession "isc.org/stork/server/database/session"
)

// High-level component controlling OpenID Connect authentication.
// It provides a middleware which handles HTTP requests related
// to OIDC authentication flow.
type Controller struct {
	settings           Settings
	configured         bool
	db                 *dbops.PgDB
	dbSessionManager   *dbsession.SessionMgr
	authSessionManager *scs.SessionManager
	oauth2Config       oauth2.Config
	tokenVerifier      *oidc.IDTokenVerifier
	metadata           Metadata
}

// Structure to cache all required information for OIDC authentication in a session.
// The data must be cached when Authentication Request is sent to OpenID Provider.
// The cache is read when response is sent back to the redirection URI.
// It is used to verify the nonce, PKCE and to retrieve the return URL for
// the OIDC authentication.
type AuthSession struct {
	CodeVerifier string
	Nonce        string
	ReturnURL    string
	CreatedAt    time.Time
}

// Constant values to be used by in-memory session manager and the middleware.
const (
	authSessionKey      = "auth_sessions"
	authSessionTimeout  = 15 * time.Minute
	callbackURLPath     = "/oidc/callback"
	loginURLPath        = "/oidc/login"
	authErrorURLPath    = "/login/auth-err"
	authRejectedURLPath = "/login/unauthorized"
)

// Constructs OIDC controller instance. It requires the settings
// and a pointer to Stork server database, which is used to insert
// and update authenticated users.
func NewController(settings Settings, db *dbops.PgDB) *Controller {
	return &Controller{
		settings: settings,
		db:       db,
		metadata: Metadata{settings: settings},
	}
}

// Configures the controller. It should be called only once at server startup.
// It requires server URL to construct OIDC redirection URI and the session
// manager to create sessions for authenticated users.
// If configuration was not successful, returns an error.
func (ctl *Controller) Configure(serverURL url.URL, dbSessionManager *dbsession.SessionMgr) error {
	if ctl.settings.IssuerURL == "" {
		// Mandatory setting is missing. Controller will remain not configured.
		log.Debug("OIDC authentication disabled")
		return nil
	}
	if ctl.settings.ClientID == "" {
		return errors.New("missing mandatory oidc-client-id setting")
	}
	ctl.dbSessionManager = dbSessionManager
	// Prepare in-memory session manager used only for storing OIDC auth data in sessions.
	// SCS uses gob encoding to store session data. To store custom type, it must be registered with gob.Register().
	gob.Register(map[string]AuthSession{})
	inMemorySessionMgr := scs.New()
	inMemorySessionMgr.Lifetime = authSessionTimeout
	inMemorySessionMgr.Cookie.Name = "auth_session"
	inMemorySessionMgr.ErrorFunc = func(w http.ResponseWriter, r *http.Request, err error) {
		// Use logrus instead of the standard logger.
		log.WithError(err).Error("an error occurred in the OIDC session manager")
	}
	ctl.authSessionManager = inMemorySessionMgr

	ctx := context.Background()
	var (
		op  *oidc.Provider
		err error
	)
	if len(ctl.settings.AuthorizationEndpoint) > 0 && len(ctl.settings.TokenEndpoint) > 0 && len(ctl.settings.JWKSURI) > 0 {
		// User provided all settings for OpenID Provider config. OIDC discovery is not needed and will be skipped.
		opConfig := oidc.ProviderConfig{
			IssuerURL: ctl.settings.IssuerURL,
			AuthURL:   ctl.settings.AuthorizationEndpoint,
			TokenURL:  ctl.settings.TokenEndpoint,
			JWKSURL:   ctl.settings.JWKSURI,
		}
		op = opConfig.NewProvider(ctx)
	} else {
		// Try to communicate with OpenID Provider Issuer and perform OIDC discovery to get information about OP endpoints.
		op, err = oidc.NewProvider(ctx, ctl.settings.IssuerURL)
		if err != nil {
			return errors.Wrapf(err, "OIDC discovery failed using issuer %s", ctl.settings.IssuerURL)
		}
	}
	tokenVerifier := op.Verifier(&oidc.Config{
		ClientID: ctl.settings.ClientID,
	})
	ctl.tokenVerifier = tokenVerifier
	// Prepare OAuth2 config.
	redirectURI := ctl.settings.RedirectURI
	if len(redirectURI) == 0 {
		constructedURI := serverURL.JoinPath(callbackURLPath)
		if constructedURI.Hostname() == "::" {
			port := constructedURI.Port()
			constructedURI.Host = "localhost:" + port
		}
		redirectURI = constructedURI.String()
	}
	logFields := log.Fields{
		"redirectURI":    redirectURI,
		"openIDProvider": ctl.settings.IssuerURL,
	}
	log.WithFields(logFields).Info("Authentication using OpenID Connect is now enabled in Stork, and users will be authenticated by OpenID Provider if the redirectURI has been registered in this provider.")
	scopes := []string{
		oidc.ScopeOpenID,
	}
	scopes = append(scopes, ctl.settings.Scopes...)
	oauth2Config := oauth2.Config{
		ClientID:    ctl.settings.ClientID,
		RedirectURL: redirectURI,
		Endpoint:    op.Endpoint(),
		Scopes:      scopes,
	}
	if ctl.settings.ClientSecret != "" {
		oauth2Config.ClientSecret = ctl.settings.ClientSecret
	}
	ctl.oauth2Config = oauth2Config

	ctl.configured = true
	return nil
}

// Provides middleware handling all OIDC-related HTTP requests.
// It should be chained with other server's middlewares.
// If OIDC is not configured by end user or the HTTP request is not related to OIDC,
// it is transparent.
func (ctl *Controller) Middleware(next http.Handler) http.Handler {
	if !ctl.configured {
		// In case OIDC was not configured, make the middleware transparent.
		return next
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, loginURLPath):
			ctl.loginHandler(w, r)
		case strings.HasPrefix(r.URL.Path, callbackURLPath):
			ctl.callbackHandler(w, r)
		default:
			next.ServeHTTP(w, r)
		}
	})
	// Use special helper wrapper to prevent from setting session or auth_session cookie
	// for requests other than related to OIDC.
	return ctl.wrapOIDCSession(handler)
}

// Helper middleware which chains the HTTP handler with SCS session manager middlewares
// only if the request URL path matches any of OIDC-related endpoints.
func (ctl *Controller) wrapOIDCSession(next http.Handler) http.Handler {
	sessionHandler := ctl.dbSessionManager.SessionMiddleware(ctl.authSessionManager.LoadAndSave(next))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, loginURLPath), strings.HasPrefix(r.URL.Path, callbackURLPath):
			sessionHandler.ServeHTTP(w, r)
		default:
			next.ServeHTTP(w, r)
		}
	})
}

// Helper method reading cache from in-memory session storage.
func (ctl *Controller) getAuthSessionMap(ctx context.Context) map[string]AuthSession {
	if !ctl.configured {
		// In case OIDC was not configured, there is no session context.
		return nil
	}
	m := ctl.authSessionManager.Get(ctx, authSessionKey)
	if m == nil {
		return make(map[string]AuthSession)
	}
	return m.(map[string]AuthSession)
}

// Helper method writing cache to in-memory session storage.
func (ctl *Controller) putAuthSessionMap(ctx context.Context, m map[string]AuthSession) {
	ctl.authSessionManager.Put(ctx, authSessionKey, m)
}

// Helper method doing cleanup in in-memory session storage.
// Stale sessions are removed from the session storage.
func (ctl *Controller) cleanupSessions(ctx context.Context) {
	sessionMap := ctl.getAuthSessionMap(ctx)
	now := time.Now()
	for k, v := range sessionMap {
		// We should not keep the session alive too long.
		// OIDC authentication process is expected to be finalized within minutes.
		// If AuthResponse comes later than 15 minutes after the AuthRequest,
		// such authentication will fail, because there is no longer a session
		// to verify nonce or PKCE.
		if now.Sub(v.CreatedAt) > authSessionTimeout {
			log.Debugf("OIDC in-memory session store found stale session created at %v to be removed", v.CreatedAt)
			delete(sessionMap, k)
		}
	}
	ctl.putAuthSessionMap(ctx, sessionMap)
}

// Generates and returns base64-encoded 32 random bytes as string.
// In case of io.ReadFull error, it returns empty string and an error.
func generateRandBase64Str() (result string, err error) {
	bytes := make([]byte, 32)
	_, err = rand.Read(bytes)
	if err != nil {
		err = errors.Wrap(err, "error while generating slice of random bytes")
		return
	}
	result = base64.RawURLEncoding.EncodeToString(bytes)
	return
}

// Generates and returns Proof Key for Code Exchange (PKCE) random codeVerifier
// and codeChallenge as strings.
// In case of io.ReadFull error, it returns empty strings and an error.
func generatePKCE() (codeVerifier string, codeChallenge string, err error) {
	codeVerifier, err = generateRandBase64Str()
	if err != nil {
		return
	}
	hash := sha256.Sum256([]byte(codeVerifier))
	codeChallenge = base64.RawURLEncoding.EncodeToString(hash[:])
	return
}

// Returns bool stating whether user belongs to MandatoryAllowGroup
// and a slice of configured groups user belongs to.
// Takes a slice of strings returned from OpenID Provider token endpoint
// representing groups that user belongs to in OP and based on OIDC settings
// checks association to configured groups.
func (ctl *Controller) getMappedGroups(groups *[]string) (allowed bool, mappedGroups []authdata.UserGroupID) {
	allowed = false
	mappedGroups = []authdata.UserGroupID{}
	if ctl.settings.MandatoryAllowGroup != "" || ctl.settings.EnableGroupMapping {
		for _, g := range *groups {
			if g == ctl.settings.MandatoryAllowGroup {
				allowed = true
			}
			if slices.Contains(ctl.settings.GroupMapping.ReadOnly, g) {
				mappedGroups = append(mappedGroups, authdata.UserGroupIDReadOnly)
			}
			if slices.Contains(ctl.settings.GroupMapping.Admin, g) {
				mappedGroups = append(mappedGroups, authdata.UserGroupIDAdmin)
			}
			if slices.Contains(ctl.settings.GroupMapping.SuperAdmin, g) {
				mappedGroups = append(mappedGroups, authdata.UserGroupIDSuperAdmin)
			}
		}
	}
	return
}

// Handles OIDC login endpoint which initiates OIDC authentication process.
// It prepares the authentication request, stores necessary data in session
// and redirects user-agent to authentication URL, where user will proceed
// with authentication.
// In case of any error, user-agent is redirected back to login page where simple
// error feedback should be displayed.
func (ctl *Controller) loginHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	state, err := generateRandBase64Str()
	if err != nil {
		log.WithError(err).Error("Error while generating OIDC random state")
		http.Redirect(w, r, authErrorURLPath, http.StatusFound)
		return
	}
	nonce, err := generateRandBase64Str()
	if err != nil {
		log.WithError(err).Error("Error while generating OIDC random nonce")
		http.Redirect(w, r, authErrorURLPath, http.StatusFound)
		return
	}
	codeVerifier, codeChallenge, err := generatePKCE()
	if err != nil {
		log.WithError(err).Error("Error while generating OIDC random PKCE")
		http.Redirect(w, r, authErrorURLPath, http.StatusFound)
		return
	}

	authSession := AuthSession{
		CodeVerifier: codeVerifier,
		Nonce:        nonce,
		ReturnURL:    sanitizeReturnURL(r.URL.Query().Get("returnUrl")),
		CreatedAt:    time.Now(),
	}
	sessionMap := ctl.getAuthSessionMap(ctx)
	sessionMap[state] = authSession
	ctl.putAuthSessionMap(ctx, sessionMap)

	authURL := ctl.oauth2Config.AuthCodeURL(
		state,
		oidc.Nonce(nonce),
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)

	ctl.cleanupSessions(ctx)

	http.Redirect(w, r, authURL, http.StatusFound)
}

// Sanitizes returnURL query parameter value. If any suspicious syntax is detected,
// it logs an error and returns home "/" path as string. If correct value was given,
// sanitized URL is returned as string.
func sanitizeReturnURL(returnURL string) string {
	const home = "/"
	returnURL = strings.TrimSpace(returnURL)
	returnURL = strings.NewReplacer("\n", "", "\r", "").Replace(returnURL)
	parsed, err := url.Parse(returnURL)
	safeReturnURL := ""
	if (utf8.RuneCountInString(returnURL)) <= 255 {
		safeReturnURL = returnURL
	} else {
		r := []rune(returnURL)
		safeReturnURL = string(r[:255]) + "..."
	}
	if err != nil {
		log.WithField("returnURL", safeReturnURL).WithError(err).Warn("Error while sanitizing returnURL")
		return home
	}
	if parsed.IsAbs() || strings.HasPrefix(returnURL, "//") {
		log.WithField("returnURL", safeReturnURL).Warn("Error while sanitizing returnURL - wrong format")
		return home
	}
	sanitizedPath := path.Clean(home + parsed.Path)
	if parsed.RawQuery != "" {
		return sanitizedPath + "?" + parsed.RawQuery
	}
	return sanitizedPath
}

// Extracts groups claim from raw claims and returns the groups as slice of strings.
func (ctl *Controller) extractGroupsFromClaim(rawClaims map[string]interface{}) []string {
	// Do custom unmarshaling of the groups claim, because we can't be sure
	// how the claim is formatted on the OpenID Provider side.
	// We should have the groups extracted as slice of strings.
	var groups []string
	if val, ok := rawClaims[ctl.settings.GroupsClaim]; ok {
		switch claim := val.(type) {
		case []interface{}:
			for _, g := range claim {
				if s, ok := g.(string); ok {
					groups = append(groups, s)
				}
			}
		case []string:
			groups = claim
		case string:
			groups = []string{claim}
		}
	}
	return groups
}

// Handles OIDC callback endpoint which interprets redirection from OpenID Provider
// after user successfully authenticates at the OP and authorizes Stork as a
// Relying Party. It verifies the response, extracts required parameters and
// sends a request to OP token endpoint. It verifies the token response and
// extracts the claims. If the user is allowed to log in to Stork,
// DB system user entry is created or updated and a session is created
// for authenticated user. User-agent is redirected to returnURL and
// user will be able to use Stork UI.
// In case of any error, user-agent is redirected back to login page where simple
// error feedback should be displayed.
func (ctl *Controller) callbackHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctl.cleanupSessions(ctx)

	var (
		ok          bool
		authSession AuthSession
		sessionMap  map[string]AuthSession
	)
	// Get cached data for received state.
	state := r.URL.Query().Get("state")
	if len(state) > 0 {
		sessionMap = ctl.getAuthSessionMap(ctx)
		authSession, ok = sessionMap[state]
	}
	if !ok {
		log.WithField("state", state).Warn("OIDC callback endpoint received invalid or expired state")
		http.Redirect(w, r, authErrorURLPath, http.StatusFound)
		return
	}
	delete(sessionMap, state)
	ctl.putAuthSessionMap(ctx, sessionMap)
	codeVerifier := authSession.CodeVerifier
	expectedNonce := authSession.Nonce

	hasError := r.URL.Query().Has("error")
	if hasError {
		errCode := r.URL.Query().Get("error")
		errDescription := r.URL.Query().Get("error_description")
		log.Errorf("OIDC authentication error response received - error code (%s) error description (%s)", errCode, errDescription)
		http.Redirect(w, r, authErrorURLPath, http.StatusFound)
		return
	}

	// Do the exchange with token endpoint and verify the response.
	code := r.URL.Query().Get("code")
	if len(code) == 0 {
		log.Error("Code is missing in the OIDC authentication response")
		http.Redirect(w, r, authErrorURLPath, http.StatusFound)
		return
	}
	token, err := ctl.oauth2Config.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", codeVerifier))
	if err != nil {
		log.WithError(err).Error("Error while exchanging OIDC token")
		http.Redirect(w, r, authErrorURLPath, http.StatusFound)
		return
	}
	idTokenJWT, ok := token.Extra("id_token").(string)
	if !ok {
		log.Error("Error while extracting id_token from OIDC token response")
		http.Redirect(w, r, authErrorURLPath, http.StatusFound)
		return
	}
	idToken, err := ctl.tokenVerifier.Verify(ctx, idTokenJWT)
	if err != nil {
		log.WithError(err).Error("Error while verifying OIDC token response")
		http.Redirect(w, r, authErrorURLPath, http.StatusFound)
		return
	}

	if idToken.Nonce != expectedNonce {
		logFields := log.Fields{
			"idTokenNonce":  idToken.Nonce,
			"expectedNonce": expectedNonce,
		}
		log.WithFields(logFields).Error("Error while verifying OIDC token response - invalid nonce")
		http.Redirect(w, r, authErrorURLPath, http.StatusFound)
		return
	}

	// Extract the claims.
	var claims struct {
		Sub        string `json:"sub"`
		Email      string `json:"email"`
		GivenName  string `json:"given_name"`
		FamilyName string `json:"family_name"`
		Name       string `json:"name"`
		Groups     []string
	}
	err = idToken.Claims(&claims)
	if err != nil {
		log.WithError(err).Error("Error while extracting OIDC claims")
		http.Redirect(w, r, authErrorURLPath, http.StatusFound)
		return
	}
	// Extract groups claim depending on configured setting.
	if ctl.settings.GroupsClaim != "" {
		var rawClaims map[string]interface{}
		err = idToken.Claims(&rawClaims)
		if err != nil {
			log.WithError(err).Error("Error while extracting OIDC claims")
			http.Redirect(w, r, authErrorURLPath, http.StatusFound)
			return
		}
		groups := ctl.extractGroupsFromClaim(rawClaims)
		claims.Groups = groups
	}
	log.Debugf("Claims received during OIDC authentication %+v", claims)

	// Check the group mapping.
	belongsToAllowGroup, mappedGroups := ctl.getMappedGroups(&claims.Groups)
	if ctl.settings.MandatoryAllowGroup != "" && !belongsToAllowGroup {
		log.Warnf("Authentication rejected for OIDC user ID %s - user does not belong to group that is mandatory for access (%s)", claims.Sub, ctl.settings.MandatoryAllowGroup)
		http.Redirect(w, r, authRejectedURLPath, http.StatusFound)
		return
	}

	// At this point OIDC authentication to Stork is considered successful.
	// Construct user metadata, insert that to DB and create a session.
	name := claims.GivenName
	if name == "" {
		name = claims.Name
	}
	lastname := claims.FamilyName
	if lastname == "" {
		lastname = claims.Name
	}
	outputUser := authdata.User{
		ID:       claims.Sub,
		Email:    claims.Email,
		Lastname: lastname,
		Name:     name,
		Groups:   []authdata.UserGroupID{},
	}
	if ctl.settings.EnableGroupMapping {
		outputUser.Groups = mappedGroups
		outputUser.ExternallyManagedGroups = true
		if len(outputUser.Groups) == 0 {
			log.Warnf("OIDC user ID %s belongs to no group used for group mapping. User will be logged in but will not be able to use Stork.", claims.Sub)
		}
	} else {
		// In case group mapping is not configured, assign external user to Read-only group.
		outputUser.Groups = []authdata.UserGroupID{
			authdata.UserGroupIDReadOnly,
		}
	}
	systemUser, err := dbmodel.AddOrUpdateExternalUser(ctl.db, &outputUser, ctl.settings.IdentityProviderID)
	if err != nil || systemUser == nil {
		log.WithError(err).Errorf("Error creating or updating system user in DB for authenticated OIDC user ID %s", claims.Sub)
		http.Redirect(w, r, authErrorURLPath, http.StatusFound)
		return
	}
	err = ctl.dbSessionManager.LoginHandler(ctx, systemUser)
	if err != nil {
		log.WithError(err).Errorf("Error creating session for authenticated OIDC user ID %s", claims.Sub)
		http.Redirect(w, r, authErrorURLPath, http.StatusFound)
		return
	}

	http.Redirect(w, r, authSession.ReturnURL, http.StatusFound)
}

// Returns bool flag whether controller is configured or not.
func (ctl *Controller) IsConfigured() bool {
	return ctl.configured
}

// Returns metadata about OIDC authentication method.
func (ctl *Controller) GetMetadata() authdata.AuthenticationMetadata {
	return &ctl.metadata
}
