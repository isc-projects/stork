package dbmigs

import "github.com/go-pg/migrations/v8"

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			CREATE TABLE config_review_checker_excluded_global (
				id BIGSERIAL PRIMARY KEY,
				checker_name TEXT NOT NULL UNIQUE
			);

			CREATE TABLE config_review_checker_option (
				id BIGSERIAL PRIMARY KEY,
				daemon_id BIGINT,
				checker_name TEXT NOT NULL,
				excluded BOOLEAN,
				CONSTRAINT config_review_checker_option_daemon_id FOREIGN KEY (daemon_id)
					REFERENCES daemon (id)
                    ON UPDATE CASCADE
                    ON DELETE CASCADE
			);

			CREATE INDEX config_review_checker_option_daemon_checker_idx ON app (daemon_id, checker_name);
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			DROP INDEX config_review_checker_option_daemon_checker_idx;
			DROP TABLE config_review_checker_option;
			DROP TABLE config_review_checker_excluded_global;
        `)
		return err
	})
}
