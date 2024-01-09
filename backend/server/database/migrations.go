package dbops

import (
	"context"
	"strconv"

	"github.com/go-pg/migrations/v8"
	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"isc.org/stork/server/database/maintenance"

	// TODO: document why it is blank imported.
	_ "isc.org/stork/server/database/migrations"
)

// Checks if the migrations table exists, i.e. the 'init' command was called.
func Initialized(db *PgDB) bool {
	var n int
	_, err := db.QueryOne(pg.Scan(&n), "SELECT count(*) FROM gopg_migrations")
	return err == nil
}

// Migrates the database version down to 0 and then removes the gopg_migrations
// table.
func Toss(db *PgDB) error {
	if db == nil {
		return errors.New("database is nil")
	}

	// Check if the migrations table exists. If it doesn't, there is nothing to do.
	if !Initialized(db) {
		return nil
	}

	// Migrate the database down to 0.
	_, _, err := Migrate(db, "reset")
	if err != nil {
		return err
	}

	// Remove the versioning table and id sequence.
	err = db.RunInTransaction(context.Background(), func(tx *pg.Tx) (err error) {
		if err := maintenance.DropTableIfExists(tx, "gopg_migrations"); err != nil {
			return err
		}
		return maintenance.DropSequenceIfExists(tx, "gopg_migrations_id_seq")
	})

	return err
}

// Migrates the database. The args specify one of the migration operations supported
// by go-pg/migrations. The returned arguments contain new and old database version as
// well as an error.
func Migrate(db *PgDB, args ...string) (oldVersion, newVersion int64, err error) {
	if len(args) > 0 && args[0] == "up" && !Initialized(db) {
		if oldVersion, newVersion, err = migrations.Run(db, "init"); err != nil {
			return oldVersion, newVersion, errors.Wrapf(err, "problem initiating database")
		}
	}

	// If down migration was specified and specific version was specified, we need to do some tricks.
	// The migrations package doesn't allow migrating down to specific version, but it can migrate
	// down one step. So we can call it multiple times until it migrated down to the version we
	// want.
	if len(args) > 1 && args[0] == "down" {
		var oldVer int64
		if oldVer, _, err = migrations.Run(db, "version"); err != nil {
			return oldVer, oldVer, errors.Wrapf(err, "problem checking database version")
		}
		toVer, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return oldVer, oldVer, errors.Wrapf(err, "can't parse -t argument %s as database version (expected integer)", args[1])
		}

		if toVer >= oldVer {
			return oldVer, oldVer, errors.Errorf("can't migrate down, current version %d, want to migrate to %d", oldVer, toVer)
		}
		startVer := oldVer
		var newVer int64
		for i := oldVer; i > toVer; i-- {
			if oldVer, newVer, err = migrations.Run(db, "down"); err != nil {
				return oldVer, oldVer, errors.Wrapf(err, "problem checking database version")
			}
		}
		return startVer, newVer, nil
	}

	oldVersion, newVersion, err = migrations.Run(db, args...)
	if err != nil {
		return oldVersion, newVersion, errors.Wrapf(err, "problem migrating database, old: %d, new: %d", oldVersion, newVersion)
	}
	return oldVersion, newVersion, nil
}

// Migrates the database to the latest version. If the migrations are not initialized
// in the database, it also performs initialization step prior to running the
// migration.
func MigrateToLatest(db *PgDB) (oldVersion, newVersion int64, err error) {
	return Migrate(db, "up")
}

// Checks what is the highest available schema version.
func AvailableVersion() int64 {
	if migrations := migrations.RegisteredMigrations(); len(migrations) > 0 {
		return migrations[len(migrations)-1].Version
	}

	return 0
}

// Returns current schema version.
func CurrentVersion(db *PgDB) (int64, error) {
	return migrations.Version(db)
}

// Prepares new database for the Stork server. This function must be called with
// the maintenance (admin) database credentials (typically postgres user and
// postgres database). The dbName and userName denote the new database name and
// the new user name created. The force flag indicates whether or not the
// function should drop an existing database and/or user before re-creating them.
// The function grants all necessary privileges to the user and creates the
// pgcrypto extension.
func CreateDatabase(settings DatabaseSettings, dbName, userName, password string, force bool) error {
	db, err := NewPgDBConn(&settings)
	if err != nil {
		return err
	}
	defer db.Close()

	if force {
		// Drop an existing database if it exists.
		if err = maintenance.DropDatabaseIfExists(db, dbName); err != nil {
			return err
		}
	}
	// Re-create the database. Note that the database creation cannot
	// be done in a transaction.
	isCreated, err := maintenance.CreateDatabase(db, dbName)
	if err != nil {
		return err
	} else if !isCreated {
		log.Infof("Database '%s' already exists", dbName)
	}

	// Close the current connection. We will have to connect to the
	// newly created database instead to create the pgcrypto extension.
	db.Close()

	// Re-use all admin credentials but connect to the new database.
	settings.DBName = dbName
	db, err = NewPgDBConn(&settings)
	if err != nil {
		return err
	}
	defer db.Close()

	// Other things can be done in a transaction.
	err = db.RunInTransaction(context.Background(), func(tx *pg.Tx) (err error) {
		hasUser := false

		// Check if the user already exists.
		hasUser, err = maintenance.HasUser(tx, userName)
		if err != nil {
			return err
		}

		if hasUser && force {
			// Revoke the privileges first.
			if err = maintenance.RevokeAllPrivilegesOnSchemaFromUser(tx, "public", userName); err != nil {
				return err
			}

			// Drop an existing user.
			if err = maintenance.DropUserIfExists(tx, userName); err != nil {
				return err
			}

			hasUser = false
		}

		// Re-create the user.
		if hasUser {
			log.Infof("User '%s' already exists", userName)
		} else if err = maintenance.CreateUser(tx, userName); err != nil {
			return err
		}

		// Grant the user full control over the database.
		if err = maintenance.GrantAllPrivilegesOnDatabaseToUser(tx, dbName, userName); err != nil {
			return err
		}

		// Grant the user full control over the schema public. It is necessary for
		// some modern Postgres installations.
		if err = maintenance.GrantAllPrivilegesOnSchemaToUser(tx, "public", userName); err != nil {
			return err
		}

		// If the password has been generated assign it to the user.
		if password != "" {
			if err = maintenance.AlterUserPassword(tx, userName, password); err != nil {
				return err
			}
		}

		// Try to create the pgcrypto extension.
		err = CreatePgCryptoExtension(db)
		if err != nil {
			return err
		}

		return nil
	})
	return err
}

// Creates a PgCrypto database extension if it does not exist yet.
func CreatePgCryptoExtension(db *pg.DB) error {
	return maintenance.CreateExtension(db, "pgcrypto")
}
