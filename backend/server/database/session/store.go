package dbsession

import (
	"log"
	"sync"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-pg/pg/v10"
	dbmodel "isc.org/stork/server/database/model"
)

// Implements a scs.Store interface using a Stork-specific database model.
// We abandoned the original store maintained by the scs authors because it
// used other database library (lib/pg driver for sql.DB) instead of the
// go-pg library we use in Stork. There were many differences between the
// default parameters and  the lib/pg didn't support the read/write timeouts.
// It was hard to adapt the original store to our needs, so we decided to
// implement our own store using the go-pg library.
//
// See the original store implementation:
// https://github.com/alexedwards/scs/tree/master/postgresstore.
type Store struct {
	db          pg.DBI
	stopCleanup chan bool
	stopWg      sync.WaitGroup
}

var (
	_ scs.Store         = &Store{}
	_ scs.IterableStore = &Store{}
)

// New returns a new Store instance, with a background cleanup goroutine
// that runs every 5 minutes to remove expired session data.
func NewStore(db pg.DBI) *Store {
	return NewStoreWithCleanupInterval(db, 5*time.Minute)
}

// NewWithCleanupInterval returns a new Store instance. The cleanupInterval
// parameter controls how frequently expired session data is removed by the
// background cleanup goroutine. Setting it to 0 prevents the cleanup goroutine
// from running (i.e. expired sessions will not be removed).
func NewStoreWithCleanupInterval(db pg.DBI, cleanupInterval time.Duration) *Store {
	p := &Store{db: db, stopCleanup: make(chan bool)}
	if cleanupInterval > 0 {
		go p.startCleanup(cleanupInterval)
	}
	return p
}

// Find returns the data for a given session token from the Store instance.
// If the session token is not found or is expired, the returned exists flag will
// be set to false.
func (p *Store) Find(token string) (b []byte, exists bool, err error) {
	session, err := dbmodel.GetActiveSession(p.db, token)
	if err != nil {
		return nil, false, err
	}
	if session == nil {
		return nil, false, nil
	}
	return session.Data, true, nil
}

// Commit adds a session token and data to the Store instance with the
// given expiry time. If the session token already exists, then the data and expiry
// time are updated.
func (p *Store) Commit(token string, b []byte, expiry time.Time) error {
	session := &dbmodel.Session{Token: token, Data: b, Expiry: expiry}
	err := dbmodel.AddOrUpdateSession(p.db, session)
	return err
}

// Delete removes a session token and corresponding data from the Store
// instance.
func (p *Store) Delete(token string) error {
	err := dbmodel.DeleteSession(p.db, token)
	return err
}

// All returns a map containing the token and data for all active (i.e.
// not expired) sessions in the Store instance.
func (p *Store) All() (map[string][]byte, error) {
	sessions, err := dbmodel.GetAllActiveSessions(p.db)
	if err != nil {
		return nil, err
	}

	dataByToken := make(map[string][]byte)
	for _, session := range sessions {
		dataByToken[session.Token] = session.Data
	}

	return dataByToken, nil
}

func (p *Store) startCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			err := p.deleteExpired()
			if err != nil {
				log.Println(err)
			}
		case <-p.stopCleanup:
			ticker.Stop()
			p.stopWg.Done()
			return
		}
	}
}

// StopCleanup terminates the background cleanup goroutine for the Store
// instance. It's rare to terminate this; generally Store instances and
// their cleanup goroutines are intended to be long-lived and run for the lifetime
// of your application.
//
// There may be occasions though when your use of the Store is transient.
// An example is creating a new Store instance in a test function. In this
// scenario, the cleanup goroutine (which will run forever) will prevent the
// Store object from being garbage collected even after the test function
// has finished. You can prevent this by manually calling StopCleanup.
func (p *Store) StopCleanup() {
	if p.stopCleanup != nil {
		p.stopWg.Add(1)
		p.stopCleanup <- true
		p.stopWg.Wait()
		p.stopCleanup = nil
	}
}

func (p *Store) deleteExpired() error {
	return dbmodel.DeleteAllExpiredSessions(p.db)
}
