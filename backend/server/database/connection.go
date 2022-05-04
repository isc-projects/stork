package dbops

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	pkgerrors "github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Minimal supported database Postgres server version.
const (
	minSupportedDatabaseServerVersionMajnor = 10
	minSupportedDatabaseServerVersionMinor  = 0
	minSupportedDatabaseServerVersionPatch  = 0
)

// Common interface for go-pg DB and Tx (transaction) objects.
type DBI = pg.DBI

// Interface to a transaction used in the RollbackOnError function.
// Using this interface makes it easier to unit test this function.
type TxI interface {
	Rollback() error
}

type DBLogger struct{}

// Hook run before SQL query execution.
func (d DBLogger) BeforeQuery(c context.Context, q *pg.QueryEvent) (context.Context, error) {
	// When making queries on the system_user table we want to make sure that
	// we don't expose actual data in the logs, especially password.
	if model, ok := q.Model.(orm.TableModel); ok {
		if model != nil {
			table := model.Table()
			if table != nil && table.SQLName == "system_user" {
				// Query on the system_user table. Don't print the actual data.
				fmt.Println(q.UnformattedQuery())
				return c, nil
			}
		}
	}
	query, err := q.FormattedQuery()
	// FormattedQuery returns a tuple of query and error. The error in most cases is nil, and
	// we don't want to print it. On the other hand, all logging is printed on stdout. We want
	// to print here to stderr, so it's possible to redirect just the queries to a file.
	if err != nil {
		// Let's print errors as SQL comments. This will allow trying to run the export as a script.
		fmt.Fprintf(os.Stderr, "%s -- error:%s\n", string(query), err)
	} else {
		fmt.Fprintln(os.Stderr, string(query))
	}
	return c, nil
}

// Hook run after SQL query execution.
func (d DBLogger) AfterQuery(c context.Context, q *pg.QueryEvent) error {
	return nil
}

// Create only new PgDB instance.
func NewPgDBConn(pgParams *pg.Options, tracing bool) (*PgDB, error) {
	db := pg.Connect(pgParams)

	// Add tracing hooks if requested.
	if tracing {
		db.AddQueryHook(DBLogger{})
	}

	log.Printf("Checking connection to database")
	// Test connection to database.
	var err error
	for tries := 0; tries < 10; tries++ {
		var (
			n       int
			pgError pg.Error
		)
		_, err = db.QueryOne(pg.Scan(&n), "SELECT 1")
		if err == nil {
			break
		}
		if errors.As(err, &pgError) && pgError.Field('S') == "FATAL" {
			break
		}
		log.Printf("Problem connecting to db, trying again in 2 seconds, %d/10: %s", tries+1, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return nil, pkgerrors.Wrapf(err, "unable to connect to the database using provided credentials")
	}

	// Check that a database version is supported
	version, err := GetDatabaseServerVersion(db)
	if err != nil {
		return nil, err
	}

	minVersion := minSupportedDatabaseServerVersionMajnor*100*100 +
		minSupportedDatabaseServerVersionMinor*100 +
		minSupportedDatabaseServerVersionPatch

	if version < minVersion {
		currentPatch := version % 100
		currentMinor := (version / 100) % 100
		currentMajnor := version / (100 * 100)

		log.Warnf("Unsupported database server version: got %d.%d.%d, required at least %d.%d.%d, "+
			"Please consider upgrading Postgres server; Stork may not work correctly with this version",
			currentMajnor, currentMinor, currentPatch,
			minSupportedDatabaseServerVersionMajnor,
			minSupportedDatabaseServerVersionMinor,
			minSupportedDatabaseServerVersionPatch,
		)
	}

	return db, nil
}

// Create new instance of PgDB and migrate database if necessary to the latest schema version.
func NewPgDB(settings *DatabaseSettings) (*PgDB, error) {
	// Fetch password from the env variable or prompt for password.
	Password(settings)

	// Make a connection to DB (tracing is enabled at this stage if set to all (migrations and run-time))
	params, err := settings.PgParams()
	if err != nil {
		return nil, err
	}
	db, err := NewPgDBConn(params, settings.TraceSQL == "all")
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
		}).Info("Successfully migrated database schema")
	}

	// Enable tracing here, if we were told to enable only at run-time
	if settings.TraceSQL == "run" {
		db.AddQueryHook(DBLogger{})
	}

	log.Infof("Connected to database %s:%d, schema version: %d", settings.Host, settings.Port, newVer)
	return db, nil
}

// Fetch the connected Postgres version in numeric format.
func GetDatabaseServerVersion(db *PgDB) (int, error) {
	var version int
	_, err := db.QueryOne(pg.Scan(&version), "SELECT CAST(current_setting('server_version_num') AS integer)")
	if err != nil {
		return 0, err
	}
	return version, nil
}

// Rollback transaction if an error has occurred. This function is typically
// called using a defer statement to rollback a transaction if an error
// occurs during a transaction or the commit operation.
func RollbackOnError(tx TxI, err *error) {
	if err != nil && *err != nil {
		_ = tx.Rollback()
	}
}
