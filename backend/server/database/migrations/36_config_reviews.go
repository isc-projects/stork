package dbmigs

import (
	"github.com/go-pg/migrations/v7"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
            CREATE TABLE IF NOT EXISTS config_review (
                id BIGSERIAL PRIMARY KEY,
                daemon_id BIGINT UNIQUE,
                created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT timezone('utc'::text, now()),
                config_hash TEXT NOT NULL,
                signature TEXT NOT NULL,
                CONSTRAINT config_review_daemon_id FOREIGN KEY (daemon_id)
                    REFERENCES daemon (id)
                    ON UPDATE CASCADE
                    ON DELETE CASCADE
            );
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
           DROP TABLE IF EXISTS config_review;
        `)
		return err
	})
}
