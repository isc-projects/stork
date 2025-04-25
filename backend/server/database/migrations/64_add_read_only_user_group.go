package dbmigs

import "github.com/go-pg/migrations/v8"

// This migration adds read-only user group.
// Users that belong to this group cannot perform Create, Update nor Delete actions.
func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			-- Stork uses hard-coded IDs for super-admin, admin and read-only groups.
			-- Alter the sequence to have a pool of 100 reserved IDs for pre-defined system groups.
			-- In the future a super-admin will be able to add/remove/update groups and set custom privileges for them.
			-- Increasing the sequence will prevent ID conflicts when dynamically assigning for future groups.
			ALTER SEQUENCE system_group_id_seq RESTART WITH 100;
			INSERT INTO system_group (id, name, description) VALUES (3, 'read-only', 'This group of users can only have read access to system components and APIs. Users that belong to this group cannot perform Create, Update nor Delete actions.');
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			DELETE FROM system_group WHERE name = 'read-only' AND id = 3;
			-- Reset the primary key sequence to be in sync with max ID in the table.
			SELECT setval('system_group_id_seq', MAX(id)) FROM system_group;
		`)
		return err
	})
}
