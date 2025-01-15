package dbmigs

import "github.com/go-pg/migrations/v8"

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			CREATE TABLE IF NOT EXISTS zone (
				id BIGSERIAL NOT NULL,
				name TEXT,
                CONSTRAINT zone_pkey PRIMARY KEY (id)
			);
			CREATE TABLE IF NOT EXISTS local_zone (
				id BIGSERIAL NOT NULL,
                daemon_id BIGINT NOT NULL,
                zone_id BIGINT NOT NULL,
                CONSTRAINT local_zone_pkey PRIMARY KEY (daemon_id, zone_id),
                CONSTRAINT local_zone_daemon_id FOREIGN KEY (daemon_id)
                    REFERENCES daemon (id) MATCH SIMPLE
                    ON UPDATE CASCADE
                    ON DELETE CASCADE,
                CONSTRAINT local_zone_zone_id FOREIGN KEY (zone_id)
                    REFERENCES zone (id) MATCH SIMPLE
                    ON UPDATE CASCADE
                    ON DELETE CASCADE
			);
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			DROP TABLE IF EXISTS local_zone;
			DROP TABLE IF EXISTS zone;
		`)
		return err
	})
}
