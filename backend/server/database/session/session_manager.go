package dbsession

import (
	"context"
	"database/sql"
	"net/http"
	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/postgresstore"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	"isc.org/stork/server/database"
)

// Provides session management mechanisms for Stork. It wraps the scs.SessionManager
// structure with Stork specific implementation of sessions.
type SessionMgr struct {
	scsSessionMgr *scs.SessionManager
}

// Creates new session manager instance. The new connection is created using the
// lib/pq driver via scs.SessionManager.
func NewSessionMgr(conn *dbops.DatabaseSettings) (*SessionMgr, error) {
	connParams := conn.ConnectionParams()
	db, err := sql.Open("postgres", connParams)
	if err != nil {
		return nil, errors.Wrapf(err, "error connecting to the database for session management using credentials %s", connParams)
	}

	s := scs.New()
	s.Store = postgresstore.New(db)

	mgr := &SessionMgr{scsSessionMgr: s}

	return mgr, nil
}

// This function should be invoked upon successful authentication of the user which
// is logging in to the system. It renews or creates a new session token for the user.
// The user's login and identifier are stored in the session.
func (s *SessionMgr) LoginHandler(ctx context.Context) error {
	err := s.scsSessionMgr.RenewToken(ctx)
	if err != nil {
		return errors.Wrapf(err, "error while creating new session identifier")
	}

	s.scsSessionMgr.Put(ctx, "userLogin", "admin")
	s.scsSessionMgr.Put(ctx, "userID", 1)
	return nil
}

// Destroys user session as a result of logout.
func (s *SessionMgr) LogoutHandler(ctx context.Context) error {
	err := s.scsSessionMgr.Destroy(ctx)
	if err != nil {
		return errors.Wrapf(err, "error while destroying a user session")
	}

	return nil
}

// Implements middleware which reads the session cookie, loads session data for the
// user and stores the token/ in the Cookie being sent to the user.
func (s *SessionMgr) SessionMiddleware(handler http.Handler) http.Handler {
	return s.scsSessionMgr.LoadAndSave(handler)
}

// Checks if the given session token exists in the database. This is typically used
// in unit testing to validate that the session data is persisted in the database.
func (s *SessionMgr) HasToken(token string) bool {
	_, exists, _ := s.scsSessionMgr.Store.Find(token)
	return exists
}

// Checks if the user is logged to the system. It is assumed that the session data
// is already fetched from the database and is stored in the request context.
// The returned values are: ok - if the user is logged, user identifier and user
// login.
func (s *SessionMgr) Logged(ctx context.Context) (ok bool, id int, login string) {
	id  = s.scsSessionMgr.GetInt(ctx, "userID")
	login = s.scsSessionMgr.GetString(ctx, "userLogin")
	return id > 0, id, login
}
