package dbmigs

import "github.com/go-pg/migrations/v8"

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			CREATE TABLE IF NOT EXISTS zone (
				id BIGSERIAL NOT NULL,
				name TEXT NOT NULL,
				rname TEXT NOT NULL,
				CONSTRAINT zone_pkey PRIMARY KEY (id)
			);

			CREATE UNIQUE INDEX zone_name_unique_idx ON zone(name);
			CREATE INDEX zone_rname_idx ON zone(rname);

			CREATE TABLE IF NOT EXISTS local_zone (
				id BIGSERIAL NOT NULL,
				daemon_id BIGINT NOT NULL,
				zone_id BIGINT NOT NULL,
				view TEXT NOT NULL DEFAULT '_default',
				class TEXT NOT NULL,
				serial BIGINT NOT NULL,
				type TEXT NOT NULL,
				loaded_at TIMESTAMP WITHOUT TIME ZONE NOT NULL,
				CONSTRAINT local_zone_pkey PRIMARY KEY (daemon_id, zone_id, view),
				CONSTRAINT local_zone_daemon_id FOREIGN KEY (daemon_id)
					REFERENCES daemon (id) MATCH SIMPLE
					ON UPDATE CASCADE
					ON DELETE CASCADE,
				CONSTRAINT local_zone_zone_id FOREIGN KEY (zone_id)
				REFERENCES zone (id) MATCH SIMPLE
					ON UPDATE CASCADE
					ON DELETE CASCADE
			);

			CREATE TABLE IF NOT EXISTS zone_inventory_state (
				id BIGSERIAL NOT NULL,
				daemon_id BIGINT NOT NULL,
				created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT timezone('utc'::text, now()),
				state JSONB NOT NULL,
				CONSTRAINT zone_inventory_state_daemon_id FOREIGN KEY (daemon_id)
				REFERENCES daemon (id) MATCH SIMPLE
					ON UPDATE CASCADE
					ON DELETE CASCADE
			);

			CREATE UNIQUE INDEX zone_inventory_state_daemon_id_unique_idx
				ON zone_inventory_state (daemon_id);
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			DROP TABLE IF EXISTS zone_inventory_state;
			DROP TABLE IF EXISTS local_zone;
			DROP TABLE IF EXISTS zone;
		`)
		return err
	})
}
