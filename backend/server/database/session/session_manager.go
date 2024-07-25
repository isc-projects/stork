package dbsession

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/alexedwards/scs/v2"
	"github.com/go-pg/pg/v10"
	"github.com/sirupsen/logrus"

	// Imports and registers the "postgres" driver used by database/sql.
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	dbmodel "isc.org/stork/server/database/model"
)

// Provides session management mechanisms for Stork. It wraps the scs.SessionManager
// structure with Stork-specific implementation of sessions.
type SessionMgr struct {
	scsSessionMgr *scs.SessionManager
	teardown      func()
}

// Creates new session manager instance.
func NewSessionMgr(db pg.DBI) (*SessionMgr, error) {
	s := scs.New()
	s.ErrorFunc = func(w http.ResponseWriter, r *http.Request, err error) {
		// Use logrus instead of the standard logger.
		logrus.WithError(err).Error("an error occurred in the session manager")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
	store := NewStore(db)
	s.Store = store
	teardown := func() { store.StopCleanup() }

	mgr := &SessionMgr{scsSessionMgr: s, teardown: teardown}

	return mgr, nil
}

// Stops the cleanup goroutine of the session store.
func (s *SessionMgr) Close() {
	s.teardown()
}

// This function should be invoked upon successful authentication of the user which
// is logging in to the system. It renews or creates a new session token for the user.
// The user's login and identifier are stored in the session.
func (s *SessionMgr) LoginHandler(ctx context.Context, user *dbmodel.SystemUser) error {
	err := s.scsSessionMgr.RenewToken(ctx)
	if err != nil {
		return errors.Wrapf(err, "error while creating new session identifier")
	}

	s.scsSessionMgr.Put(ctx, "userID", user.ID)
	s.scsSessionMgr.Put(ctx, "userLogin", user.Login)
	s.scsSessionMgr.Put(ctx, "userEmail", user.Email)
	s.scsSessionMgr.Put(ctx, "userLastname", user.Lastname)
	s.scsSessionMgr.Put(ctx, "userName", user.Name)
	s.scsSessionMgr.Put(ctx, "authenticationMethodID", user.AuthenticationMethodID)

	// If any user groups are associated with the user, store them
	// as a list of comma separated values.
	if len(user.Groups) > 0 {
		var groups string
		for i, g := range user.Groups {
			if i > 0 {
				groups += ","
			}
			groups += strconv.Itoa(g.ID)
		}
		s.scsSessionMgr.Put(ctx, "userGroupIds", groups)
	}

	return nil
}

// Destroys user session as a result of logout.
func (s *SessionMgr) LogoutHandler(ctx context.Context) error {
	err := s.scsSessionMgr.Destroy(ctx)

	return errors.Wrapf(err, "error while destroying a user session")
}

// Logout specific user.
func (s *SessionMgr) LogoutUser(ctx context.Context, user *dbmodel.SystemUser) error {
	err := s.scsSessionMgr.Iterate(ctx, func(ctx context.Context) error {
		id := s.scsSessionMgr.GetInt(ctx, "userID")

		if id == user.ID {
			return s.LogoutHandler(ctx)
		}

		return nil
	})

	return errors.Wrapf(err, "error while destroying a user session")
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
func (s *SessionMgr) Logged(ctx context.Context) (ok bool, user *dbmodel.SystemUser) {
	id := s.scsSessionMgr.GetInt(ctx, "userID")
	// User has no session.
	if id == 0 {
		return false, nil
	}

	// User has a session.
	user = &dbmodel.SystemUser{ID: id}
	user.Login = s.scsSessionMgr.GetString(ctx, "userLogin")
	user.Email = s.scsSessionMgr.GetString(ctx, "userEmail")
	user.Lastname = s.scsSessionMgr.GetString(ctx, "userLastname")
	user.Name = s.scsSessionMgr.GetString(ctx, "userName")
	user.AuthenticationMethodID = s.scsSessionMgr.GetString(ctx, "authenticationMethodID")

	// Retrieve comma separated list of groups.
	userGroups := s.scsSessionMgr.GetString(ctx, "userGroupIds")

	if len(userGroups) > 0 {
		groups := strings.Split(userGroups, ",")
		for _, g := range groups {
			if id, err := strconv.Atoi(g); err == nil {
				user.Groups = append(user.Groups, &dbmodel.SystemGroup{ID: id})
			}
		}
	}

	return true, user
}

// This function is only for testing purposes to prepare request context.
func (s *SessionMgr) Load(ctx context.Context, token string) (context.Context, error) {
	ctx2, err := s.scsSessionMgr.Load(ctx, token)
	return ctx2, err
}
