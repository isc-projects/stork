package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(
			`-- Leases table.
             CREATE TABLE IF NOT EXISTS public.lease (
                 id                 BIGSERIAL NOT NULL,
                 ip_version         SMALLINT NOT NULL,
                 client_id          BYTEA,
                 hostname           TEXT,
                 hw_address         MACADDR,
                 duid               BYTEA,
                 ip_address         INET NOT NULL,
                 type               VARCHAR(255),
                 -- TODO: figure out how to make this a TIMESTAMP WITH TIME ZONE
                 -- without also breaking the JSON representation
                 cltt               INT NOT NULL,
                 state              SMALLINT NOT NULL,
                 user_context       JSONB,
                 valid_lifetime     INT NOT NULL,
                 iaid               INT,
                 preferred_lifetime INT,
                 subnet_id          BIGINT,
                 stork_subnet_id    BIGINT,
                 fqdn_fwd           BOOLEAN,
                 fqdn_rev           BOOLEAN,
                 prefix_length      SMALLINT,
                 daemon_id          BIGINT,
                 CONSTRAINT lease_pkey PRIMARY KEY (id),
                 CONSTRAINT lease_subnet_fkey FOREIGN KEY (stork_subnet_id)
	                 REFERENCES subnet (id) MATCH SIMPLE
                         ON UPDATE CASCADE
                         ON DELETE SET NULL,
                 CONSTRAINT lease_daemons_fkey FOREIGN KEY (daemon_id)
                     REFERENCES daemon (id) MATCH SIMPLE
                         ON UPDATE CASCADE
                         ON DELETE SET NULL,
                 CONSTRAINT lease_ip_mac_daemon_unique UNIQUE (ip_address, hw_address, daemon_id),
                 CONSTRAINT lease_ip_duid_daemon_unique UNIQUE (ip_address, duid, daemon_id),
                 CONSTRAINT lease_ip_clid_daemon_unique UNIQUE (ip_address, client_id, daemon_id)
             );

             -- Index on daemon_id and client last transaction time in order to efficiently find the most recent observed CLTT. This is used when asking agents to send all the leases that the server hasn't seen yet.
             CREATE INDEX lease_daemon_cltt_idx ON lease(daemon_id, cltt);
             -- Index on client last transaction time, valid lifetime, and id in order to efficiently find leases which are expired (even if not in the expired state).
             CREATE INDEX lease_expiry_idx ON lease(cltt, valid_lifetime, id);

             -- Lease updates table.
             CREATE TABLE IF NOT EXISTS public.lease_update (
                 id                 BIGSERIAL NOT NULL,
                 ip_version         SMALLINT NOT NULL,
                 client_id          BYTEA,
                 hostname           TEXT,
                 hw_address         MACADDR,
                 duid               BYTEA,
                 ip_address         INET NOT NULL,
                 type               VARCHAR(255),
                 cltt               INT NOT NULL,
                 state              SMALLINT NOT NULL,
                 user_context       JSONB,
                 valid_lifetime     INT NOT NULL,
                 iaid               INT,
                 preferred_lifetime INT,
                 subnet_id          BIGINT,
                 stork_subnet_id    BIGINT,
                 fqdn_fwd           BOOLEAN,
                 fqdn_rev           BOOLEAN,
                 prefix_length      SMALLINT,
                 daemon_id          BIGINT,
                 CONSTRAINT lease_updates_pkey PRIMARY KEY (id),
                 CONSTRAINT lease_updates_subnet_fkey FOREIGN KEY (stork_subnet_id)
	                 REFERENCES subnet (id) MATCH SIMPLE
                         ON UPDATE CASCADE
                         ON DELETE SET NULL,
                 CONSTRAINT lease_updates_daemons_fkey FOREIGN KEY (daemon_id)
                     REFERENCES daemon (id) MATCH SIMPLE
                         ON UPDATE CASCADE
                         ON DELETE SET NULL
             );

             -- Index on daemon_id and client last transaction time in order to efficiently find the most recent observed CLTT. This is used when asking agents to send all the leases that the server hasn't seen yet.
             CREATE INDEX lease_update_daemon_cltt_idx ON lease_update(daemon_id, cltt);
             -- Index on client last transaction time, valid lifetime, and id in order to efficiently find leases which are expired (even if not in the expired state).
             CREATE INDEX lease_update_expiry_idx ON lease_update(cltt, valid_lifetime, id);
           `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(
			`-- Remove table with leases.
             DROP TABLE IF EXISTS public.lease;
             -- Remove table with lease updates.
             DROP TABLE IF EXISTS public.lease_update;
             -- Remove lease puller interval setting.
             DELETE FROM setting WHERE name = 'kea_leases_puller_interval';
        `)
		return err
	})
}
