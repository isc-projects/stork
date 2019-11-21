package dbops

import (
	"context"
	"github.com/go-pg/pg/v9"
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


// Tests connection to the database by sending trivial query.
func testPgConnection(db *pg.DB) error {
	var n int
	_, err := db.QueryOne(pg.Scan(&n), "SELECT 1")
	return err
}


func NewPgDB2(settings *DatabaseSettings) *PgDB {
	// Fetch password from the env variable or prompt for password.
	Password(settings)

	// Make a connection to DB
	db := pg.Connect(settings.PgParams())
	if err := testPgConnection(db); err != nil {
		log.Fatalf("unable to connect to the database using provided credentials: %v", err)
	}

	// Ensure that the latest database schema is installed.
	oldVer, newVer, err := MigrateToLatest(db)
	if err != nil {
		log.Fatalf("failed to migrate database schema to the latest version: %v", err)

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
	return db
}
