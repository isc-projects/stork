package dbops

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-pg/pg/v9"
	"github.com/go-pg/pg/v9/orm"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type DbLogger struct{}

// Hook run before SQL query execution.
func (d DbLogger) BeforeQuery(c context.Context, q *pg.QueryEvent) (context.Context, error) {
	// When making queries on the system_user table we want to make sure that
	// we don't expose actual data in the logs, especially password.
	if model, ok := q.Model.(orm.TableModel); ok {
		if model != nil {
			table := model.Table()
			if table != nil && table.Name == "system_user" {
				// Query on the system_user table. Don't print the actual data.
				fmt.Println(q.UnformattedQuery())
				return c, nil
			}
		}
	}
	query, err := q.FormattedQuery()
	// FormattedQuery returns a tuple of query and error. The error in most cases is nil, and
	// we don't want to print it. On the other hand, all logging is printed on stdout. We want
	// to print here to stder, so it's possible to redirect just the queries to a file.
	if err != nil {
		// Let's print errors as SQL comments. This will allow trying to run the export as a script.
		fmt.Fprintf(os.Stderr, "%s -- error:%s\n", query, err)
	} else {
		fmt.Fprintln(os.Stderr, query)
	}
	return c, nil
}

// Hook run after SQL query execution.
func (d DbLogger) AfterQuery(c context.Context, q *pg.QueryEvent) error {
	return nil
}

// Create only new PgDB instance.
func NewPgDbConn(pgParams *pg.Options, tracing bool) (*PgDB, error) {
	db := pg.Connect(pgParams)

	// Add tracing hooks if requested.
	if tracing {
		db.AddQueryHook(DbLogger{})
	}

	log.Printf("checking connection to database")
	// Test connection to database.
	var err error
	for tries := 0; tries < 10; tries++ {
		var n int
		_, err = db.QueryOne(pg.Scan(&n), "SELECT 1")
		if err == nil {
			break
		} else if pgErr, ok := err.(pg.Error); ok && pgErr.Field('S') == "FATAL" {
			break
		} else {
			log.Printf("problem with connecting to db, trying again in 2 seconds, %d/10: %s", tries+1, err)
		}
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return nil, errors.Wrapf(err, "unable to connect to the database using provided credentials")
	}
	return db, nil
}

// Create new instance of PgDB and migrate database if necessary to the latest schema version.
func NewPgDB(settings *DatabaseSettings) (*PgDB, error) {
	// Fetch password from the env variable or prompt for password.
	Password(settings)

	// Make a connection to DB (tracing is enabled at this stage if set to all (migrations and run-time))
	db, err := NewPgDbConn(settings.PgParams(), settings.TraceSQL == "all")
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

	// Enable tracing here, if we were told to enable only at run-time
	if settings.TraceSQL == "run" {
		db.AddQueryHook(DbLogger{})
	}

	log.Infof("connected to database %s:%d, schema version: %d", settings.Host, settings.Port, newVer)
	return db, nil
}

// Creates new transaction or returns existing transaction along with
// the appropriate rollback and commit functions. If the dbIface is
// a pointer to pg.DB object, this object is used to create new
// transaction. The rollback and commit functions contain appropriate
// rollback and commit implementations. If the dbIface points to an
// pg.Tx it means that we're in the middle of an existing transaction.
// In that case this object is returned to the caller and the rollback
// and commit functions are no-op.
func Transaction(dbIface interface{}) (tx *pg.Tx, rollback func(), commit func() error, err error) {
	db, ok := dbIface.(*pg.DB)
	if ok {
		tx, err = db.Begin()
		if err != nil {
			err = errors.Wrapf(err, "problem with starting database transaction")
		}
		rollback = func() {
			// We neither capture nor log any error here because it would
			// flood us with warnings indicating that rollback was called
			// on already committed changes. Our usage pattern is to
			// always call rollback upon exiting the function. It most
			// often occurs after commit.
			_ = tx.Rollback()
		}
		commit = func() (err error) {
			err = tx.Commit()
			if err != nil {
				err = errors.Wrapf(err, "problem with committing the transaction")
			}
			return err
		}
	} else {
		tx, ok = dbIface.(*pg.Tx)
		if !ok {
			err = errors.New("unsupported type of the database transaction object provided")
		}
		rollback = func() {}
		commit = func() (err error) {
			return nil
		}
	}
	return tx, rollback, commit, err
}
