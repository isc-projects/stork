package dbmigs

import "github.com/go-pg/migrations/v8"

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			-- Create an index on the type column of the local_zone table.
			CREATE INDEX local_zone_type_idx ON local_zone(type);
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			DROP INDEX IF EXISTS local_zone_type_idx;
		`)
		return err
	})
}
