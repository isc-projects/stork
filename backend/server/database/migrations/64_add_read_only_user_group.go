package dbmigs

import "github.com/go-pg/migrations/v8"

// This migration adds read-only user group.
// Users that belong to this group cannot perform Create, Update nor Delete actions.
func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			INSERT INTO system_group (name, description) VALUES ('read-only', 'This group of users can only have read access to system components and APIs. Users that belong to this group cannot perform Create, Update nor Delete actions.');
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			DELETE FROM system_group WHERE name = 'read-only';
		`)
		return err
	})
}
