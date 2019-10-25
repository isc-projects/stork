package dbsession

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/postgresstore"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	"isc.org/stork/server/database"
)

type SessionMgr struct {
	scsSessionMgr *scs.SessionManager
}

func NewSessionMgr(conn *dbops.GenericConn) (*SessionMgr, error) {
	connParams := conn.ConnectionParams()
	fmt.Println(connParams)
	db, err := sql.Open("postgres", connParams)
	if err != nil {
		return nil, errors.Wrapf(err, "error connecting to the database for session management using credentials %s", connParams)
	}

	s := scs.New()
	s.Store = postgresstore.New(db)

	mgr := &SessionMgr{scsSessionMgr: s}

	return mgr, nil
}

func (s *SessionMgr) LoginHandler(ctx context.Context) error {
	err := s.scsSessionMgr.RenewToken(ctx)
	if err != nil {
		return errors.Wrapf(err, "error while creating new session identifier")
	}

	s.scsSessionMgr.Put(ctx, "userLogin", "admin")
	s.scsSessionMgr.Put(ctx, "userID", 1)
	return nil
}

func (s *SessionMgr) SessionMiddleware(handler http.Handler) http.Handler {
	return s.scsSessionMgr.LoadAndSave(handler)
}

func (s *SessionMgr) HasToken(token string) bool {
	_, exists, _ := s.scsSessionMgr.Store.Find(token)
	return exists
}

func (s *SessionMgr) Logged(ctx context.Context) (bool, int, string) {
	id := s.scsSessionMgr.GetInt(ctx, "userID")
	email := s.scsSessionMgr.GetString(ctx, "userLogin")
	return id > 0, id, email
}
