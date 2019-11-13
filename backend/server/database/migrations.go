package dbops

import (
	"github.com/go-pg/migrations/v7"
	"github.com/go-pg/pg/v9"
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
	_, err = db.Exec(
		`DROP TABLE IF EXISTS gopg_migrations;
         DROP SEQUENCE IF EXISTS gopg_migrations_id_seq`)

	return err
}

// Migrates the database. The args specify one of the migration operations supported
// by go-pg/migrations. The returned arguments contain new and old database version as
// well as an error.
func Migrate(db *PgDB, args ...string) (oldVersion, newVersion int64, err error) {
	if len(args) > 0 && args[0] == "up" && !Initialized(db) {
		if oldVersion, newVersion, err = migrations.Run(db, "init"); err != nil {
			return oldVersion, newVersion, err
		}
	}
	oldVersion, newVersion, err = migrations.Run(db, args...)
	return oldVersion, newVersion, err
}

// Migrates the database to the latest version. If the migrations are not initialized
// in the database, it also performs initialization step prior to running the
// migration.
func MigrateToLatest(db *PgDB) (oldVersion, newVersion int64, err error) {
	return Migrate(db, "up")
}

// Checks what is the highest available schema version.
func AvailableVersion() int64 {
	if regm := migrations.RegisteredMigrations(); len(regm) > 0 {
		return regm[len(regm)-1].Version
	}

	return 0
}

// Returns current schema version.
func CurrentVersion(db *PgDB) (int64, error) {
	return migrations.Version(db)
}
