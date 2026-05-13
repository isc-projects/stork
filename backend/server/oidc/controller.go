package oidc

import (
	"net/http"
	"net/url"

	dbops "isc.org/stork/server/database"
	dbsession "isc.org/stork/server/database/session"
)

// High-level component controlling OpenID Connect authentication.
// It provides a middleware which handles HTTP requests related
// to OIDC authentication flow.
type Controller struct {
	settings         Settings
	configured       bool
	db               *dbops.PgDB
	dbSessionManager *dbsession.SessionMgr
}

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
// It requires server URL used to construct OIDC redirection URI and the session
// manager to create sessions for authenticated users.
func (ctl *Controller) Configure(serverURL url.URL, dbSessionManager *dbsession.SessionMgr) {
	if ctl.settings.IssuerURL == "" {
		// Mandatory setting is missing. Controller will remain not configured.
		return
	}
	ctl.dbSessionManager = dbSessionManager
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
	return ctl.dbSessionManager.SessionMiddleware(next)
}
