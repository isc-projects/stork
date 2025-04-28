package dbmigs

import "github.com/go-pg/migrations/v8"

// This migration adds read-only user group.
// Users that belong to this group cannot perform Create, Update nor Delete actions.
func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			-- Add UNIQUE constraint on group name. There shouldn't be more than one group with the same name.
			ALTER TABLE system_group ADD CONSTRAINT system_group_name_unique_idx UNIQUE (name);
			-- Stork uses hard-coded IDs for super-admin, admin and read-only groups.
			-- Thus, insert new group with explicitly given PK ID=3.
			-- Relying on a sequence could potentially end up with other ID value assigned to the read-only group.
			INSERT INTO system_group (id, name, description) VALUES (3, 'read-only', 'This group of users can only have read access to system components and APIs. Users that belong to this group cannot perform Create, Update nor Delete actions.');
			-- Reset the primary key sequence to be in sync with max ID in the table.
			SELECT setval('system_group_id_seq', MAX(id)) FROM system_group;
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			DELETE FROM system_group WHERE name = 'read-only';
			-- Revert UNIQUE constraint on group name.
			ALTER TABLE system_group DROP CONSTRAINT IF EXISTS system_group_name_unique_idx;
			-- Reset the primary key sequence to be in sync with max ID in the table.
			SELECT setval('system_group_id_seq', MAX(id)) FROM system_group;
		`)
		return err
	})
}
