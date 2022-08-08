package dbmigs

import "github.com/go-pg/migrations/v8"

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			CREATE TABLE config_daemon_checker_preference (
				daemon_id BIGINT,
				checker_name TEXT NOT NULL,
				excluded BOOLEAN,
				CONSTRAINT config_daemon_checker_preference_pkey PRIMARY KEY (daemon_id, checker_name),
				CONSTRAINT config_daemon_checker_preference_daemon_id FOREIGN KEY (daemon_id)
					REFERENCES daemon (id)
                    ON UPDATE CASCADE
                    ON DELETE CASCADE
			);
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			DROP TABLE config_daemon_checker_preference;
        `)
		return err
	})
}
