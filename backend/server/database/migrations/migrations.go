package dbmigs

import (
	"github.com/go-pg/migrations/v7"
	"github.com/go-pg/pg/v9"

	"isc.org/stork/server/database"
)

// Migrates the database version down to 0 and then removes the gopg_migrations
// table.
func Toss(dbopts *dbops.PgOptions) error {
	db := pg.Connect(dbopts)

	// Check if the migrations table exists. If it doesn't, there is nothing to do.
	var n int
	_, err := db.QueryOne(pg.Scan(&n), "SELECT count(*) FROM gopg_migrations")
	if err != nil {
		return nil
	}

	// Migrate the database down to 0.
	_, _, err = migrateAndStayConnected(db, "reset")
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
	db := pg.Connect(dbopts)
	oldVersion, newVersion, err = migrateAndStayConnected(db, args...)
	db.Close()

	return oldVersion, newVersion, err
}

// Migrates the database and returns the connection.
func migrateAndStayConnected(db *pg.DB, args ...string) (oldVersion, newVersion int64, err error) {
	// Run migrations.
	oldVersion, newVersion, err = migrations.Run(db, args...)
	return oldVersion, newVersion, err
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
