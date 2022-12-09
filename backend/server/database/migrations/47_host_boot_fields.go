package dbmigs

import "github.com/go-pg/migrations/v8"

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			ALTER TABLE local_host
			    ADD COLUMN next_server TEXT,
				ADD COLUMN server_hostname TEXT,
				ADD COLUMN boot_file_name TEXT;
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			ALTER TABLE local_host DROP COLUMN boot_file_name;
			ALTER TABLE local_host DROP COLUMN server_hostname;
			ALTER TABLE local_host DROP COLUMN next_server;
        `)
		return err
	})
}
