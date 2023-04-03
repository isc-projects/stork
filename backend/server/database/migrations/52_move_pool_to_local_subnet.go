package dbmigs

import "github.com/go-pg/migrations/v8"

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			-- Delete the subnets and let Stork fetch them from the Kea servers.
			DELETE FROM subnet;
			UPDATE kea_daemon SET config_hash = NULL;

			-- We need a new primary key to reference the pools from the local_subnet.
			ALTER TABLE local_subnet DROP CONSTRAINT local_subnet_pkey;
			CREATE UNIQUE INDEX local_subnet_subnet_daemon_idx ON local_subnet (subnet_id, daemon_id);
			ALTER TABLE local_subnet ADD COLUMN id BIGSERIAL PRIMARY KEY;

			-- Move the address pools from the subnet to the local_subnet table.
			ALTER TABLE address_pool DROP COLUMN subnet_id;
			ALTER TABLE address_pool ADD COLUMN local_subnet_id BIGINT NOT NULL
				CONSTRAINT address_pool_local_subnet_fkey
					REFERENCES local_subnet (id) MATCH SIMPLE
					ON UPDATE CASCADE
					ON DELETE CASCADE;

			-- Move the prefix pools from the subnet to the local_subnet table.
			ALTER TABLE prefix_pool DROP COLUMN subnet_id;
			ALTER TABLE prefix_pool ADD COLUMN local_subnet_id BIGINT NOT NULL
				CONSTRAINT prefix_pool_local_subnet_fkey
					REFERENCES local_subnet (id) MATCH SIMPLE
					ON UPDATE CASCADE
					ON DELETE CASCADE;
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			DELETE FROM subnet;
			UPDATE kea_daemon SET config_hash = NULL;

			ALTER TABLE prefix_pool DROP COLUMN local_subnet_id;
			ALTER TABLE prefix_pool ADD COLUMN subnet_id BIGINT NOT NULL
				CONSTRAINT prefix_pool_subnet_fkey
					REFERENCES subnet (id) MATCH SIMPLE
					ON UPDATE CASCADE
					ON DELETE CASCADE;

			ALTER TABLE address_pool DROP COLUMN local_subnet_id;
			ALTER TABLE address_pool ADD COLUMN subnet_id BIGINT NOT NULL
				CONSTRAINT address_pool_subnet_fkey
					REFERENCES subnet (id) MATCH SIMPLE
					ON UPDATE CASCADE
					ON DELETE CASCADE;

			ALTER TABLE local_subnet DROP COLUMN id;

			DROP INDEX local_subnet_subnet_daemon_idx;

			ALTER TABLE local_subnet ADD CONSTRAINT local_subnet_pkey
				PRIMARY KEY(subnet_id, daemon_id);
        `)
		return err
	})
}
