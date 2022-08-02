package dbmigs

import "github.com/go-pg/migrations/v8"

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			CREATE TABLE config_checker_global_exclude (
				id BIGSERIAL PRIMARY KEY,
				checker_name TEXT NOT NULL UNIQUE
			);

			CREATE TABLE config_checker_daemon_preference (
				id BIGSERIAL PRIMARY KEY,
				daemon_id BIGINT,
				checker_name TEXT NOT NULL,
				excluded BOOLEAN,
				CONSTRAINT config_checker_daemon_preference_daemon_id FOREIGN KEY (daemon_id)
					REFERENCES daemon (id)
                    ON UPDATE CASCADE
                    ON DELETE CASCADE
			);

			CREATE UNIQUE INDEX config_checker_daemon_preference_daemon_checker_idx ON config_checker_daemon_preference (daemon_id, checker_name);
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			DROP INDEX config_checker_daemon_preference_daemon_checker_idx;
			DROP TABLE config_checker_daemon_preference;
			DROP TABLE config_checker_global_exclude;
        `)
		return err
	})
}
