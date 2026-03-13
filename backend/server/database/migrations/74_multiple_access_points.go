package dbmigs

import "github.com/go-pg/migrations/v8"

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			-- Add a new auto-increment ID column to the access point table.
			ALTER TABLE access_point ADD COLUMN id BIGSERIAL;
			-- Remove the old composite primary key.
			ALTER TABLE access_point DROP CONSTRAINT access_point_pkey;
			-- Set the new ID column as the primary key.
			ALTER TABLE access_point ADD CONSTRAINT access_point_pkey PRIMARY KEY (id);
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			-- Delete the duplicate access points for each daemon and type,
			-- keeping only oldest of them.
			DELETE FROM access_point ap
			WHERE id NOT IN (
				SELECT MIN(id)
				FROM access_point
				GROUP BY daemon_id, type
			);
			-- Drop the ID column.
			ALTER TABLE access_point DROP COLUMN id;
			-- Restore the old composite primary key.
			ALTER TABLE access_point ADD CONSTRAINT access_point_pkey PRIMARY KEY (daemon_id, type);	
		`)
		return err
	})
}
