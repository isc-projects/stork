package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
             ALTER TABLE ha_service DROP CONSTRAINT IF EXISTS ha_service_primary_id;
             ALTER TABLE ha_service ADD CONSTRAINT ha_service_primary_id FOREIGN KEY (primary_id)
                 REFERENCES app (id) MATCH SIMPLE
                     ON UPDATE CASCADE
                     ON DELETE SET NULL;

             ALTER TABLE ha_service DROP CONSTRAINT IF EXISTS ha_service_secondary_id;
             ALTER TABLE ha_service ADD CONSTRAINT ha_service_secondary_id FOREIGN KEY (secondary_id)
                 REFERENCES app (id) MATCH SIMPLE
                     ON UPDATE CASCADE
                     ON DELETE SET NULL;

             -- The following columns were created with the TIME type instead
             -- of the TIMESTAMP type. Since TIME does not cast to TIMESTAMP, the
             -- columns must be dropped and then re-created with the appropriate
             -- type.
             ALTER TABLE ha_service DROP COLUMN IF EXISTS primary_status_collected_at;
             ALTER TABLE ha_service DROP COLUMN IF EXISTS secondary_status_collected_at;
             ALTER TABLE ha_service ADD COLUMN
                 primary_status_collected_at TIMESTAMP WITHOUT TIME ZONE;
             ALTER TABLE ha_service ADD COLUMN
                 secondary_status_collected_at TIMESTAMP WITHOUT TIME ZONE;

             -- Add new columns holding scopes served by the primary and the
             -- secondary/standby.
             ALTER TABLE ha_service ADD COLUMN primary_last_scopes TEXT[];
             ALTER TABLE ha_service ADD COLUMN secondary_last_scopes TEXT[];

             -- Add new columns indicating whether the servers are reachable via the
             -- control channel.
             ALTER TABLE ha_service ADD COLUMN primary_reachable boolean DEFAULT FALSE;
             ALTER TABLE ha_service ADD COLUMN secondary_reachable boolean DEFAULT FALSE;

             -- Add new columns which mark the time when the last failover
             -- event took place.
             ALTER TABLE ha_service ADD COLUMN
                 primary_last_failover_at TIMESTAMP WITHOUT TIME ZONE;
             ALTER TABLE ha_service ADD COLUMN
                 secondary_last_failover_at TIMESTAMP WITHOUT TIME ZONE;
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
             ALTER TABLE ha_service DROP CONSTRAINT IF EXISTS ha_service_secondary_id;
             ALTER TABLE ha_service ADD CONSTRAINT ha_service_secondary_id FOREIGN KEY (secondary_id)
                 REFERENCES app (id) MATCH SIMPLE
                     ON UPDATE NO ACTION
                     ON DELETE NO ACTION;
             ALTER TABLE ha_service DROP CONSTRAINT IF EXISTS ha_service_primary_id;
             ALTER TABLE ha_service ADD CONSTRAINT ha_service_primary_id FOREIGN KEY (primary_id)
                 REFERENCES app (id) MATCH SIMPLE
                     ON UPDATE NO ACTION
                     ON DELETE NO ACTION;
             ALTER TABLE ha_service DROP COLUMN IF EXISTS secondary_last_failover_at;
             ALTER TABLE ha_service DROP COLUMN IF EXISTS primary_last_failover_at;
             ALTER TABLE ha_service DROP COLUMN IF EXISTS secondary_reachable;
             ALTER TABLE ha_service DROP COLUMN IF EXISTS primary_reachable;
             ALTER TABLE ha_service DROP COLUMN IF EXISTS secondary_last_scopes;
             ALTER TABLE ha_service DROP COLUMN IF EXISTS primary_last_scopes;
             ALTER TABLE ha_service ALTER COLUMN
                 primary_status_collected_at TYPE TIME WITHOUT TIME ZONE;
             ALTER TABLE ha_service ALTER COLUMN
                 secondary_status_collected_at TYPE TIME WITHOUT TIME ZONE;
        `)
		return err
	})
}
