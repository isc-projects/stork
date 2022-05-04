package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
            -- This table holds generic information about daemons.
            CREATE TABLE IF NOT EXISTS daemon (
                id bigserial NOT NULL,
                app_id bigint NOT NULL,
                pid integer,
                name text,
                active boolean NOT NULL DEFAULT false,
                version text,
                extended_version text,
                uptime bigint,
                created_at timestamp without time zone NOT NULL DEFAULT timezone('utc'::text, now()),
                reloaded_at timestamp without time zone,
                CONSTRAINT daemon_pkey PRIMARY KEY (id),
                CONSTRAINT daemon_app_id_fkey FOREIGN KEY (app_id)
                    REFERENCES app (id) MATCH SIMPLE
                    ON UPDATE CASCADE
                    ON DELETE CASCADE
            );

            -- This table holds common information for all Kea daemons, e.g. configuration.
            CREATE TABLE IF NOT EXISTS kea_daemon (
                id bigserial NOT NULL,
                daemon_id bigint NOT NULL,
                config jsonb,
                CONSTRAINT kea_daemon_pkey PRIMARY KEY (id),
                CONSTRAINT kea_daemon_id_unique UNIQUE (daemon_id),
                CONSTRAINT kea_daemon_id_fkey FOREIGN KEY (daemon_id)
                    REFERENCES daemon (id) MATCH SIMPLE
                    ON UPDATE CASCADE
                    ON DELETE CASCADE
            );

            -- This table holds Kea DHCP daemon-specific information.
            CREATE TABLE IF NOT EXISTS kea_dhcp_daemon (
                id bigserial NOT NULL,
                kea_daemon_id bigint NOT NULL,
                stats jsonb,
                CONSTRAINT kea_dhcp_daemon_pkey PRIMARY KEY (id),
                CONSTRAINT kea_dhcp_daemon_id_unique UNIQUE (kea_daemon_id),
                CONSTRAINT kea_dhcp_daemon_id_fkey FOREIGN KEY (kea_daemon_id)
                    REFERENCES kea_daemon (id) MATCH SIMPLE
                    ON UPDATE CASCADE
                    ON DELETE CASCADE
            );

            -- This table holds BIND 9 daemon-specific information.
            CREATE TABLE IF NOT EXISTS bind9_daemon (
                id bigserial NOT NULL,
                daemon_id bigint NOT NULL,
                stats jsonb,
                CONSTRAINT bind9_daemon_pkey PRIMARY KEY (id),
                CONSTRAINT bind9_daemon_id_unique UNIQUE (daemon_id),
                CONSTRAINT bind9_daemon_id_fkey FOREIGN KEY (daemon_id)
                    REFERENCES daemon (id) MATCH SIMPLE
                    ON UPDATE CASCADE
                    ON DELETE CASCADE
            );
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
            DROP TABLE IF EXISTS bind9_daemon;
            DROP TABLE IF EXISTS kea_dhcp_daemon;
            DROP TABLE IF EXISTS kea_daemon;
            DROP TABLE IF EXISTS daemon;
        `)
		return err
	})
}
