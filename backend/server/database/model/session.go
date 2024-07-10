package dbmodel

import (
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
)

type Session struct {
	tableName struct{}  `pg:"sessions"` //nolint:unused
	Token     string    `pg:",pk,notnull,use_zero"`
	Data      []byte    `pg:",notnull,use_zero"`
	Expiry    time.Time `pg:",notnull,use_zero"`
}

// Returns an active session with the given token. If the session is not found
// or has expired, returns nil and no error.
func GetActiveSession(dbi pg.DBI, token string) (*Session, error) {
	session := &Session{}
	err := dbi.Model(session).
		Where("token = ?", token).
		Where("current_timestamp < expiry").
		Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = errors.Wrapf(err, "problem getting the session with token %s", token)
		return nil, err
	}
	return session, err
}

// Adds a session to the database. If the session already exists, it will be
// updated.
func AddOrUpdateSession(dbi pg.DBI, session *Session) error {
	_, err := dbi.Model(session).OnConflict("(token) DO UPDATE").
		Set("data = EXCLUDED.data").
		Set("expiry = EXCLUDED.expiry").
		Insert()
	if err != nil {
		err = errors.Wrapf(err, "problem adding session with token %s", session.Token)
	}
	return err
}

// Deletes a session with a given token from the database.
func DeleteSession(dbi pg.DBI, token string) error {
	_, err := dbi.Model((*Session)(nil)).
		Where("token = ?", token).
		Delete()
	if err != nil {
		err = errors.Wrapf(err, "problem deleting session with token %s", token)
	}
	return err
}

// Returns all active sessions.
func GetAllActiveSessions(dbi pg.DBI) ([]*Session, error) {
	var sessions []*Session
	err := dbi.Model(&sessions).
		Where("current_timestamp < expiry").
		Select()
	if err != nil {
		err = errors.Wrap(err, "problem getting all active sessions")
	}
	return sessions, err
}

// Returns all sessions, including expired ones.
func GetAllSessions(dbi pg.DBI) ([]*Session, error) {
	var sessions []*Session
	err := dbi.Model(&sessions).Select()
	if err != nil {
		err = errors.Wrap(err, "problem getting all sessions")
	}
	return sessions, err
}

// Removes all expired sessions from the database.
func DeleteAllExpiredSessions(dbi pg.DBI) error {
	_, err := dbi.Model((*Session)(nil)).
		Where("expiry < current_timestamp").
		Delete()
	if err != nil {
		err = errors.Wrap(err, "problem removing all expired sessions")
	}
	return err
}
