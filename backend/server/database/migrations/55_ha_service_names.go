package dbmigs

import "github.com/go-pg/migrations/v8"

// The migration drops the non null constraint on the name and lastname columns.
func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			-- Delete the existing service expecting they will be added again.
			DELETE FROM service;
			-- Add a new column in the ha_service table that names the HA
			-- relationships.
			ALTER TABLE ha_service ADD COLUMN relationship TEXT NOT NULL;
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			ALTER TABLE ha_service DROP COLUMN relationship;
		`)
		return err
	})
}
