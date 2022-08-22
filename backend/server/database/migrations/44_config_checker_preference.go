package dbmigs

import "github.com/go-pg/migrations/v8"

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			CREATE TABLE config_checker_preference (
				id BIGSERIAL PRIMARY KEY,
				daemon_id BIGINT,
				checker_name TEXT NOT NULL,
				enabled BOOLEAN NOT NULL,
				CONSTRAINT config_checker_preference_daemon_id_fk FOREIGN KEY (daemon_id)
					REFERENCES daemon (id)
                    ON UPDATE CASCADE
                    ON DELETE CASCADE
			);

			CREATE UNIQUE INDEX config_checker_preference_non_null_idx ON config_checker_preference (daemon_id, checker_name) WHERE daemon_id IS NOT NULL;
			CREATE UNIQUE INDEX config_checker_preference_nullable_idx ON config_checker_preference (checker_name) WHERE daemon_id IS NULL;
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			DROP INDEX config_checker_preference_nullable_idx;
			DROP INDEX config_checker_preference_non_null_idx;
			DROP TABLE config_checker_preference;
        `)
		return err
	})
}
