package dbmigs

import "github.com/go-pg/migrations/v8"

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			ALTER TABLE config_report DROP CONSTRAINT config_report_daemon_id;
			ALTER TABLE config_report ADD CONSTRAINT config_report_daemon_id
				FOREIGN KEY (daemon_id)
				REFERENCES daemon (id)
				ON UPDATE CASCADE
				ON DELETE CASCADE;
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
		ALTER TABLE config_report DROP CONSTRAINT config_report_daemon_id;
		ALTER TABLE config_report ADD CONSTRAINT config_report_daemon_id
			FOREIGN KEY (daemon_id)
			REFERENCES daemon (id)
			ON UPDATE CASCADE
			ON DELETE NO ACTION;
        `)
		return err
	})
}
