package dbops

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	storkutil "isc.org/stork/util"
)

// Minimal supported database Postgres server version.
const (
	minSupportedDatabaseServerVersionMajor = 10
	minSupportedDatabaseServerVersionMinor = 0
	minSupportedDatabaseServerVersionPatch = 0
)

// Common interface for go-pg DB and Tx (transaction) objects.
type DBI = pg.DBI

// Interface to a transaction used in the RollbackOnError function.
// Using this interface makes it easier to unit test this function.
type TxI interface {
	Rollback() error
}

// Defines the go-pg hooks to enable the SQL query logging.
// It implements the "pg.QueryHook" interface.
type DBLogger struct{}

// The type used to define context keys for database handling.
type contextKeywordDB string

const suppressQueryLoggingKeyword contextKeywordDB = "suppress-query-logging"

// Hook run before SQL query execution.
func (d DBLogger) BeforeQuery(c context.Context, q *pg.QueryEvent) (context.Context, error) {
	if HasSuppressedQueryLogging(c) {
		return c, nil
	}

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
func NewPgDBConn(settings *DatabaseSettings) (*PgDB, error) {
	pgParams, err := settings.convertToPgOptions()
	if err != nil {
		return nil, err
	}

	db := pg.Connect(pgParams)
	// Add tracing hooks if requested.
	if settings.TraceSQL != LoggingQueryPresetNone {
		db.AddQueryHook(DBLogger{})
	}

	log.Printf("Checking connection to database")
	// Test connection to database.
	for tries := 0; tries < 10; tries++ {
		var pgError pg.Error

		err = db.Ping(db.Context())
		if err == nil {
			break
		}
		err = errors.Wrapf(err, "unable to connect to the database using provided settings")

		if errors.As(err, &pgError) {
			switch {
			case pgError.Field('C') == "28P01":
				// 28P01 - It is a problem with an invalid password.
				if !storkutil.IsRunningInTerminal() {
					break
				}
				log.WithError(err).Error("Invalid database credentials (authentication error)")
				pgParams.Password, err = storkutil.GetSecretInTerminal(fmt.Sprintf("database password for user %s: ", pgParams.User))
				if err != nil {
					break
				}
				settings.Password = pgParams.Password
				continue
			case pgError.Field('R') == "auth_failed":
				// Another authentication problem. E.g.: The authentication
				// service may be temporarily unavailable.
			case pgError.Field('S') == "FATAL":
				break
			}
		}
		log.WithError(err).Warnf("Problem connecting to db, trying again in 2 seconds, %d/10", tries+1)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		db.Close()
		return nil, err
	}

	// Check that a database version is supported
	version, err := GetDatabaseServerVersion(db)
	if err != nil {
		db.Close()
		return nil, err
	}

	minVersion := minSupportedDatabaseServerVersionMajor*100*100 +
		minSupportedDatabaseServerVersionMinor*100 +
		minSupportedDatabaseServerVersionPatch

	if version < minVersion {
		currentPatch := version % 100
		currentMinor := (version / 100) % 100
		currentMajor := version / (100 * 100)

		log.Warnf("Unsupported database server version: got %d.%d.%d, required at least %d.%d.%d, "+
			"Please consider upgrading Postgres server; Stork may not work correctly with this version",
			currentMajor, currentMinor, currentPatch,
			minSupportedDatabaseServerVersionMajor,
			minSupportedDatabaseServerVersionMinor,
			minSupportedDatabaseServerVersionPatch,
		)
	}

	log.Infof("Connected to database %s", pgParams.Addr)

	return db, nil
}

// Migrate database if necessary to the latest schema version.
func NewApplicationDatabaseConn(settings *DatabaseSettings) (*PgDB, error) {
	db, err := NewPgDBConn(settings)
	if err != nil {
		return nil, err
	}

	migrateDB := db
	if settings.TraceSQL == LoggingQueryPresetRuntime {
		migrateDB = SuppressQueryLogging(db)
	}

	// Ensure that the latest database schema is installed.
	oldVer, newVer, err := MigrateToLatest(migrateDB)
	switch {
	case err != nil:
		db.Close()
		return nil, err
	case oldVer != newVer:
		log.WithFields(log.Fields{
			"old-version": oldVer,
			"new-version": newVer,
		}).Info("Successfully migrated database schema")
	default:
		log.WithField("version", newVer).Info("Database is up-to-date")
	}

	return db, nil
}

// Checks if the query logging suppression is enabled.
func HasSuppressedQueryLogging(ctx context.Context) bool {
	value := ctx.Value(suppressQueryLoggingKeyword)
	if isSuppressed, ok := value.(bool); ok {
		return isSuppressed
	}
	return false
}

// Returns a database instance with a changed context to suppress the SQL
// query logging hook.
func SuppressQueryLogging(db *PgDB) *PgDB {
	return db.WithContext(
		context.WithValue(
			db.Context(),
			suppressQueryLoggingKeyword,
			true,
		),
	)
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
