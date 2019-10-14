package storkdb

import (
	"github.com/go-pg/migrations/v7"
	"github.com/go-pg/pg/v9"
)

// Alias to pg.Options.
type DbConnOptions = pg.Options

// Migrates the database version down to 0 and then removes the gopg_migrations
// table.
func Toss(dbopts *DbConnOptions) (err error) {
	db := pg.Connect(dbopts)
	defer db.Close()

	// Migrate down to version 0.
	if _, _, err = migrations.Run(db, "reset"); err != nil {
		return err
	}

	// Remove the versioning table.
	_, err = db.Exec("DROP TABLE IF EXISTS gopg_migrations")

	return err
}

// Migrates the database using provided credentials. The args specify one of the
// migration operations supported by go-pg/migrations. The returned arguments
// contain new and old database version.
func Migrate(dbopts *DbConnOptions, args ...string) (oldVersion, newVersion int64, err error) {
	// Connect to the database.
	db := pg.Connect(dbopts)
	defer db.Close()

	// Run migrations.
	oldVersion, newVersion, err = migrations.Run(db, args...)

	return oldVersion, newVersion, err
}
