package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
            ALTER TABLE ha_service ADD COLUMN IF NOT EXISTS primary_comm_interrupted boolean;
            ALTER TABLE ha_service ADD COLUMN IF NOT EXISTS primary_connecting_clients bigint;
            ALTER TABLE ha_service ADD COLUMN IF NOT EXISTS primary_unacked_clients bigint;
            ALTER TABLE ha_service ADD COLUMN IF NOT EXISTS primary_unacked_clients_left bigint;
            ALTER TABLE ha_service ADD COLUMN IF NOT EXISTS primary_analyzed_packets bigint;

            ALTER TABLE ha_service ADD COLUMN IF NOT EXISTS secondary_comm_interrupted boolean;
            ALTER TABLE ha_service ADD COLUMN IF NOT EXISTS secondary_connecting_clients bigint;
            ALTER TABLE ha_service ADD COLUMN IF NOT EXISTS secondary_unacked_clients bigint;
            ALTER TABLE ha_service ADD COLUMN IF NOT EXISTS secondary_unacked_clients_left bigint;
            ALTER TABLE ha_service ADD COLUMN IF NOT EXISTS secondary_analyzed_packets bigint;

            -- This trigger function is invoked upon deletion of an association between
            -- daemons and services. It removes a service if it no longer has
            -- associations with any daemon.
            CREATE OR REPLACE FUNCTION wipe_dangling_service()
                RETURNS trigger
                LANGUAGE 'plpgsql'
                AS $function$
            BEGIN
                DELETE FROM service
                    WHERE service.id = OLD.service_id AND NOT EXISTS (
                        SELECT FROM daemon_to_service AS ds
                            WHERE ds.service_id = service.id
                );
                RETURN NULL;
            END;
            $function$;

            -- This trigger removes a service that no longer has associations
            -- with any daemons.
            DO $$ BEGIN
                CREATE TRIGGER trigger_wipe_dangling_service
                    AFTER DELETE ON daemon_to_service
                        FOR EACH ROW EXECUTE PROCEDURE wipe_dangling_service();
            EXCEPTION
                WHEN duplicate_object THEN null;
            END $$;
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
            DROP TRIGGER IF EXISTS trigger_wipe_dangling_service ON daemon_to_service;
            DROP FUNCTION IF EXISTS wipe_dangling_service;

            ALTER TABLE ha_service DROP COLUMN IF EXISTS secondary_analyzed_packets;
            ALTER TABLE ha_service DROP COLUMN IF EXISTS secondary_unacked_clients_left;
            ALTER TABLE ha_service DROP COLUMN IF EXISTS secondary_unacked_clients;
            ALTER TABLE ha_service DROP COLUMN IF EXISTS secondary_connecting_clients;
            ALTER TABLE ha_service DROP COLUMN IF EXISTS secondary_comm_interrupted;
            ALTER TABLE ha_service DROP COLUMN IF EXISTS secondary_comm_interrupted;

            ALTER TABLE ha_service DROP COLUMN IF EXISTS primary_analyzed_packets;
            ALTER TABLE ha_service DROP COLUMN IF EXISTS primary_unacked_clients_left;
            ALTER TABLE ha_service DROP COLUMN IF EXISTS primary_unacked_clients;
            ALTER TABLE ha_service DROP COLUMN IF EXISTS primary_connecting_clients;
            ALTER TABLE ha_service DROP COLUMN IF EXISTS primary_comm_interrupted;
        `)
		return err
	})
}
