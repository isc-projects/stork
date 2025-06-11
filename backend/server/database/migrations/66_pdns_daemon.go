package dbmigs

import "github.com/go-pg/migrations/v8"

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			-- This table holds PowerDNS daemon-specific information.
			CREATE TABLE IF NOT EXISTS pdns_daemon (
			id BIGSERIAL NOT NULL,
			daemon_id BIGINT NOT NULL,
			CONSTRAINT pdns_daemon_pkey PRIMARY KEY (id),
			CONSTRAINT pdns_daemon_id_unique UNIQUE (daemon_id),
			CONSTRAINT pdns_daemon_id_fkey FOREIGN KEY (daemon_id)
				REFERENCES daemon (id) MATCH SIMPLE
					ON UPDATE CASCADE
					ON DELETE CASCADE
			);
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
				DROP TABLE IF EXISTS pdns_daemon;
			`)
		return err
	})
}
