package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
            DELETE FROM service;
            -- Existing foreign keys point to the app table, so they
            -- must be dropped.
            ALTER TABLE ha_service DROP CONSTRAINT IF EXISTS ha_service_primary_id;
            ALTER TABLE ha_service DROP CONSTRAINT IF EXISTS ha_service_secondary_id;

            -- Recreate the foreign keys to make them point to the daemon table.
            ALTER TABLE ha_service ADD CONSTRAINT ha_service_primary_id FOREIGN KEY (primary_id)
                REFERENCES daemon (id) MATCH SIMPLE
                ON UPDATE CASCADE
                ON DELETE SET NULL;
            ALTER TABLE ha_service ADD CONSTRAINT ha_service_secondary_id FOREIGN KEY (secondary_id)
                REFERENCES daemon (id) MATCH SIMPLE
                ON UPDATE CASCADE
                ON DELETE SET NULL;

            -- The app_to_service table provides M:N relation between services and apps.
            -- This relation must be modified to be between apps and daemons.
            ALTER TABLE app_to_service DROP CONSTRAINT IF EXISTS app_to_service_app_id;

            -- The table and the app_id column must be renamed to reflect this new relationship.
            ALTER TABLE app_to_service RENAME TO daemon_to_service;
            ALTER TABLE daemon_to_service RENAME COLUMN app_id TO daemon_id;

            -- This adds the constraint back against the daemon table.
            ALTER TABLE daemon_to_service ADD CONSTRAINT daemon_to_service_daemon_id FOREIGN KEY (daemon_id)
                REFERENCES daemon (id) MATCH SIMPLE
                ON UPDATE NO ACTION
                ON DELETE CASCADE;
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
            DELETE FROM service;
            ALTER TABLE daemon_to_service DROP CONSTRAINT IF EXISTS daemon_to_service_daemon_id;
            ALTER TABLE daemon_to_service RENAME TO app_to_service;
            ALTER TABLE app_to_service RENAME COLUMN daemon_id TO app_id;
            ALTER TABLE app_to_service ADD CONSTRAINT app_to_service_app_id FOREIGN KEY (app_id)
                REFERENCES app (id) MATCH SIMPLE
                ON UPDATE CASCADE
                ON DELETE CASCADE;

            ALTER TABLE ha_service DROP CONSTRAINT IF EXISTS ha_service_primary_id;
            ALTER TABLE ha_service DROP CONSTRAINT IF EXISTS ha_service_secondary_id;

            ALTER TABLE ha_service ADD CONSTRAINT ha_service_primary_id FOREIGN KEY (primary_id)
                REFERENCES app (id) MATCH SIMPLE
                ON UPDATE CASCADE
                ON DELETE SET NULL;
            ALTER TABLE ha_service ADD CONSTRAINT ha_service_secondary_id FOREIGN KEY (secondary_id)
                REFERENCES app (id) MATCH SIMPLE
                ON UPDATE CASCADE
                ON DELETE SET NULL;
        `)
		return err
	})
}
