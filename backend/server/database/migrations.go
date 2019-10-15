package storkdb

import (
	"github.com/go-pg/migrations/v7"
	"github.com/go-pg/pg/v9"
)

// Alias to pg.Options.
type DbConnOptions = pg.Options

// Migrates the database version down to 0 and then removes the gopg_migrations
// table.
func Toss(dbopts *DbConnOptions, migrationsdir string) error {
	// Migrate the database down to 0.
	db, _, _, err := migrateAndStayConnected(dbopts, migrationsdir, "reset")
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
func Migrate(dbopts *DbConnOptions, migrationsdir string, args ...string) (oldVersion, newVersion int64, err error) {
	db, oldVersion, newVersion, err := migrateAndStayConnected(dbopts, migrationsdir, args...)
	db.Close()

	return oldVersion, newVersion, err
}

// Migrates the database using provided credentials and returns the connection
// to the database.
func migrateAndStayConnected(dbopts *DbConnOptions, migrationsdir string, args ...string) (db *pg.DB, oldVersion, newVersion int64, err error) {
	// Read migration files from the specified location.
	m := migrations.NewCollection()
	if err = m.DiscoverSQLMigrations(migrationsdir); err != nil {
		return nil, 0, 0, err
	}

	// Connect to the database.
	db = pg.Connect(dbopts)

	// Run migrations.
	oldVersion, newVersion, err = m.Run(db, args...)
	return db, oldVersion, newVersion, err
}
