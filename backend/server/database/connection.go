package dbops

import (
	"context"
	"github.com/go-pg/pg/v9"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type dbLogger struct { }


// Hook run before SQL query execution.
func (d dbLogger) BeforeQuery(c context.Context, q *pg.QueryEvent) (context.Context, error) {
	log.Println(q.FormattedQuery())
	return c, nil
}

// Hook run after SQL query execution.
func (d dbLogger) AfterQuery(c context.Context, q *pg.QueryEvent) error {
	return nil
}

// Create only new PgDB instance.
func NewPgDbConn(pgParams *pg.Options) (*PgDB, error) {
	db := pg.Connect(pgParams)

	// Test connection to database.
	var n int
	_, err := db.QueryOne(pg.Scan(&n), "SELECT 1")
	if err != nil {
		return nil, errors.Wrapf(err, "unable to connect to the database using provided credentials")
	}
	return db, nil
}

// Create new instance of PgDB and migrate database if necessary to the latest schema version.
func NewPgDB(settings *DatabaseSettings) (*PgDB, error) {
	// Fetch password from the env variable or prompt for password.
	Password(settings)

	// Make a connection to DB
	db, err := NewPgDbConn(settings.PgParams())
	if err != nil {
		return nil, err
	}

	// Ensure that the latest database schema is installed.
	oldVer, newVer, err := MigrateToLatest(db)
	if err != nil {
		return nil, err

	} else if oldVer != newVer {
		log.WithFields(log.Fields{
			"old-version": oldVer,
			"new-version": newVer,
		}).Info("successfully migrated database schema")
	}

	// Add tracing hooks if requested.
	if (settings.TraceSql) {
		db.AddQueryHook(dbLogger{})
	}

	log.Infof("connected to database %s:%d, schema version: %d", settings.Host, settings.Port, newVer)
	return db, nil
}
