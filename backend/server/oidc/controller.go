package oidc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/gob"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"isc.org/stork/server/authdata"
	dbops "isc.org/stork/server/database"
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

// Constant values to be used by in-memory session manager.
const (
	authSessionKey     = "auth_sessions"
	authSessionTimeout = 15 * time.Minute
)

// Constructs OIDC controller instance. It requires the settings
// and a pointer to Stork server database, which is used to insert
// and update authenticated users.
func NewController(settings Settings, db *dbops.PgDB) *Controller {
	return &Controller{
		settings: settings,
		db:       db,
	}
}

// Configures the controller. It should be called only once at server startup.
// It requires server URL to construct OIDC redirection URI and the session
// manager to create sessions for authenticated users.
func (ctl *Controller) Configure(serverURL url.URL, dbSessionManager *dbsession.SessionMgr) {
	if ctl.settings.IssuerURL == "" {
		// Mandatory setting is missing. Controller will remain not configured.
		log.Debug("OIDC authentication disabled")
		return
	}
	if ctl.settings.ClientID == "" {
		log.Error("OIDC authentication can't be used due to missing oidc-client-id setting")
		return
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
	ctl.configured = true
}

// Provides middleware handling all OIDC-related HTTP requests.
// It should be chained with other server's middlewares.
// If OIDC is not configured by end user, it is transparent.
func (ctl *Controller) Middleware(next http.Handler) http.Handler {
	if !ctl.configured {
		// In case OIDC was not configured, make the middleware transparent.
		return next
	}
	return ctl.dbSessionManager.SessionMiddleware(ctl.authSessionManager.LoadAndSave(next))
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
		err = errors.Wrapf(err, "error while generating slice of random bytes")
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
