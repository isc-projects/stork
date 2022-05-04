package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
             -- This table holds information about selected hosts within the
             -- network. The major purpose of defining a host is to make
             -- static reservations of resources such as IP addresses or
             -- delegated prefixes.
             CREATE TABLE IF NOT EXISTS host (
                 id bigserial NOT NULL,
                 created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT timezone('utc'::text, now()),
                 subnet_id bigint NULL,
                 CONSTRAINT host_pkey PRIMARY KEY (id),
                 CONSTRAINT host_subnet_id_fkey FOREIGN KEY (subnet_id)
                     REFERENCES subnet (id) MATCH SIMPLE
                         ON UPDATE CASCADE
                         ON DELETE CASCADE
             );
             CREATE INDEX host_subnet_id_idx ON host (subnet_id);

             -- This table holds static IP address or delegated prefix reservations
             -- for hosts.
             CREATE TABLE IF NOT EXISTS ip_reservation (
                 id bigserial NOT NULL,
                 created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT timezone('utc'::text, now()),
                 address cidr NOT NULL,
                 host_id bigint NOT NULL,
                 CONSTRAINT ip_reservation_pkey PRIMARY KEY (id),
                 CONSTRAINT ip_reservation_host_id_address_unique_idx UNIQUE (host_id, address),
                 CONSTRAINT ip_reservation_host_fkey FOREIGN KEY (host_id)
                     REFERENCES host (id) MATCH SIMPLE
                         ON UPDATE CASCADE
                         ON DELETE CASCADE
             );

             -- Each host may be identified by various DHCP identifiers.
             DO $$ BEGIN
                 CREATE TYPE HOSTIDTYPE AS ENUM
                     ('hw-address', 'duid', 'circuit-id', 'client-id', 'flex-id');
             EXCEPTION
                 WHEN duplicate_object THEN null;
             END $$;

             -- This table holds a mapping of one or more identifiers to a host.
             -- For example, a single host may be identified by both MAC address
             -- and circuit ID.
             CREATE TABLE IF NOT EXISTS host_identifier (
                 id bigserial NOT NULL,
                 created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT timezone('utc'::text, now()),
                 type hostidtype NOT NULL,
                 value bytea NOT NULL,
                 host_id bigint NOT NULL,
                 CONSTRAINT host_identifier_pkey PRIMARY KEY (id),
                 CONSTRAINT host_identifier_host_type_unique_idx UNIQUE (host_id, type),
                 CONSTRAINT host_identifier_host_fkey FOREIGN KEY (host_id)
                     REFERENCES host (id) MATCH SIMPLE
                         ON UPDATE CASCADE
                         ON DELETE CASCADE
             );

             -- This enum lists types of the host data sources. In Kea's case, there are
             -- two sources of information about host reservations: configuration file
             -- and API (via host_cmds).
             DO $$ BEGIN
                 CREATE TYPE HOSTDATASOURCE AS ENUM
                     ('config', 'api');
             EXCEPTION
                 WHEN duplicate_object THEN null;
             END $$;

             -- This table associates apps with host reservations in M:N relationship.
             -- It also holds additional information about the host reservation
             -- specific to the apps. In particular, it records whether the host
             -- information was fetched from the configuration file or from the
             -- hosts database (via the host commands hook library).
             CREATE TABLE IF NOT EXISTS local_host (
                 app_id bigint NOT NULL,
                 host_id bigint NOT NULL,
                 created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT timezone('utc'::text, now()),
                 data_source hostdatasource NOT NULL,
                 CONSTRAINT local_host_pkey PRIMARY KEY (host_id, app_id),
                 CONSTRAINT local_host_app_id FOREIGN KEY (app_id)
                     REFERENCES app (id) MATCH SIMPLE
                     ON UPDATE CASCADE
                     ON DELETE CASCADE
             );
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
             DROP TABLE IF EXISTS local_host;
             DROP TYPE IF EXISTS HOSTDATASOURCE;
             DROP TABLE IF EXISTS host_identifier;
             DROP TYPE IF EXISTS hostidtype;
             DROP TABLE IF EXISTS ip_reservation;
             DROP TABLE IF EXISTS host;
        `)
		return err
	})
}
