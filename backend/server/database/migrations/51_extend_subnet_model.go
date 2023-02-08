package dbmigs

import "github.com/go-pg/migrations/v8"

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			-- Add Kea-specific parameters to the subnet.
			ALTER TABLE local_subnet ADD COLUMN kea_parameters JSONB;

			-- Add DHCP options to the subnet.
			ALTER TABLE local_subnet ADD COLUMN dhcp_option_set JSONB;
			ALTER TABLE local_subnet ADD COLUMN dhcp_option_set_hash TEXT;

			-- Add Kea parameters to the address pool.
			ALTER TABLE address_pool ADD COLUMN kea_parameters JSONB;

			-- Add DHCP options to the address pool.
			ALTER TABLE address_pool ADD COLUMN dhcp_option_set JSONB;
			ALTER TABLE address_pool ADD COLUMN dhcp_option_set_hash TEXT;

			-- Add Kea parameters to the prefix pool.
			ALTER TABLE prefix_pool ADD COLUMN kea_parameters JSONB;

			-- Add DHCP options to the prefix pool.
			ALTER TABLE prefix_pool ADD COLUMN dhcp_option_set JSONB;
			ALTER TABLE prefix_pool ADD COLUMN dhcp_option_set_hash TEXT;

			-- Create the local_shared_network table.
			CREATE TABLE IF NOT EXISTS local_shared_network (
				daemon_id BIGINT NOT NULL,
				shared_network_id BIGINT,
				kea_parameters JSONB,
				dhcp_option_set JSONB,
				dhcp_option_set_hash TEXT,
				CONSTRAINT local_shared_network_pkey PRIMARY KEY (shared_network_id, daemon_id),
				CONSTRAINT local_shared_network_daemon_id FOREIGN KEY (daemon_id)
					REFERENCES daemon (id) MATCH SIMPLE
					ON UPDATE CASCADE
					ON DELETE CASCADE,
				CONSTRAINT local_shared_network_id FOREIGN KEY (shared_network_id)
					REFERENCES shared_network (id) MATCH SIMPLE
					ON UPDATE CASCADE
					ON DELETE CASCADE
			);

			-- Create an index by daemon_id.
			CREATE INDEX local_shared_network_daemon_id_idx ON local_shared_network(daemon_id);

			-- Setting NULL hash causes the server to refresh the Kea
			-- configurations in the Stork database.
			UPDATE kea_daemon SET config_hash = NULL;
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			DROP TABLE IF EXISTS local_shared_network;
			ALTER TABLE prefix_pool DROP COLUMN dhcp_option_set_hash;
			ALTER TABLE prefix_pool DROP COLUMN dhcp_option_set;
			ALTER TABLE prefix_pool DROP COLUMN kea_parameters;
			ALTER TABLE address_pool DROP COLUMN dhcp_option_set_hash;
			ALTER TABLE address_pool DROP COLUMN dhcp_option_set;
			ALTER TABLE address_pool DROP COLUMN kea_parameters;
			ALTER TABLE local_subnet DROP COLUMN dhcp_option_set_hash;
			ALTER TABLE local_subnet DROP COLUMN dhcp_option_set;
			ALTER TABLE local_subnet DROP COLUMN kea_parameters;
        `)
		return err
	})
}
