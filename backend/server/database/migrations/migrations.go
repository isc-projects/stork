package dbmigs

import (
	"github.com/go-pg/migrations/v7"
	"github.com/go-pg/pg/v9"

	"isc.org/stork/server/database"
)

// Migrates the database version down to 0 and then removes the gopg_migrations
// table.
func Toss(dbopts *dbops.PgOptions) error {
	// Migrate the database down to 0.
	db, _, _, err := migrateAndStayConnected(dbopts, "reset")
	defer db.Close()

	if err != nil {
		return err
	}

	// Remove the versioning table.
	_, err = db.Exec("DROP TABLE IF EXISTS gopg_migrations")

	return err
}

// Migrates the database using provided credentials. The migrationsdir specifies
// the location of the migration files. The args specify one of the
// migration operations supported by go-pg/migrations. The returned arguments
// contain new and old database version as well as an error.
func Migrate(dbopts *dbops.PgOptions, args ...string) (oldVersion, newVersion int64, err error) {
	db, oldVersion, newVersion, err := migrateAndStayConnected(dbopts, args...)
	db.Close()

	return oldVersion, newVersion, err
}

// Migrates the database using provided credentials and returns the connection
// to the database.
func migrateAndStayConnected(dbopts *dbops.PgOptions, args ...string) (db *pg.DB, oldVersion, newVersion int64, err error) {
	// Connect to the database.
	db = pg.Connect(dbopts)

	// Run migrations.
	oldVersion, newVersion, err = migrations.Run(db, args...)
	return db, oldVersion, newVersion, err
}

// Checks what is the highest available schema version.
func AvailableVersion() int64 {
	if regm := migrations.RegisteredMigrations(); len(regm) > 0 {
		return regm[len(regm)-1].Version
	}

	return 0
}

// Returns current schema version.
func CurrentVersion(dbopts *dbops.PgOptions) (int64, error) {
	// Connect to the database.
	db := pg.Connect(dbopts)
	return migrations.Version(db)
}
