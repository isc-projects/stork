package dbmigs

import "github.com/go-pg/migrations/v8"

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			-- Holds a collection of DNS zones. The zone names are unique within the
			-- zone table. If multiple servers share the same zone, there are multiple
			-- entries in the local_zone table, each with the zone_id pointing to the primary
			-- key of the zone table, and daemon_id pointing to the daemon table. It creates
			-- associations between the zone and multiple daemons/servers. The rname column
			-- holds "reverse" zone names (i.e., the zone names with labels ordered backwards).
			-- Ordering the zones by rname is the same as ordering the zones in the DNS
			-- order.
			CREATE TABLE IF NOT EXISTS zone (
				id BIGSERIAL NOT NULL,
				name TEXT NOT NULL,
				rname TEXT NOT NULL,
				CONSTRAINT zone_pkey PRIMARY KEY (id)
			);
			-- Suffice to create a unique index on the name column. There is no need to
			-- create the unique index on the rname column. It simplifies the use of the
			-- ON CONFLICT clause.
			CREATE UNIQUE INDEX zone_name_unique_idx ON zone(name);
			CREATE INDEX zone_rname_idx ON zone(rname);

			-- Associates a zone with a server. It allows server-specific zone configuration,
			-- which may be different for different servers. Thus, it includes class, serial
			-- type and other values.
			CREATE TABLE IF NOT EXISTS local_zone (
				id BIGSERIAL NOT NULL,
				daemon_id BIGINT NOT NULL,
				zone_id BIGINT NOT NULL,
				view TEXT NOT NULL DEFAULT '_default',
				class TEXT NOT NULL,
				serial BIGINT NOT NULL,
				type TEXT NOT NULL,
				loaded_at TIMESTAMP WITHOUT TIME ZONE NOT NULL,
				UNIQUE(daemon_id, zone_id, view),
				CONSTRAINT local_zone_pkey PRIMARY KEY (id),
				CONSTRAINT local_zone_daemon_id FOREIGN KEY (daemon_id)
					REFERENCES daemon (id) MATCH SIMPLE
					ON UPDATE CASCADE
					ON DELETE CASCADE,
				CONSTRAINT local_zone_zone_id FOREIGN KEY (zone_id)
				REFERENCES zone (id) MATCH SIMPLE
					ON UPDATE CASCADE
					ON DELETE CASCADE
			);
			-- There is a UNIQUE constraint on the (daemon_id, zone_id and view)
			-- already. It creates a composite index that is picked by queries
			-- by daemon_id. Thus, there is no need to create a dedicated index
			-- for daemon_id. However, we need indexes by zone_id and view because
			-- there will be queries by these columns.
			CREATE INDEX local_zone_zone_id_idx ON local_zone(zone_id);
			CREATE INDEX local_zone_view_idx ON local_zone(view);

			-- Holds the state of fetching the zones from the zone inventories on
			-- the agents to the server. Each entry in this table represents a state
			-- of fetching the zones from one DNS server.
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
			-- There may be a need to query for a single daemon's state.
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
