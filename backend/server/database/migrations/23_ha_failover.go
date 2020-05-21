package dbmigs

import (
	"github.com/go-pg/migrations/v7"
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
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
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
