package dbmigs

import "github.com/go-pg/migrations/v8"

// The migrations extends the primary key of the local_host table to include
// the data_source column.
func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			ALTER TABLE local_host DROP CONSTRAINT local_host_pkey;
			ALTER TABLE local_host
                ADD CONSTRAINT local_host_pkey
					PRIMARY KEY (host_id, daemon_id, data_source);
		`)
		return err
	}, func(d migrations.DB) error {
		_, err := d.Exec(`
			ALTER TABLE local_host DROP CONSTRAINT local_host_pkey;
			ALTER TABLE local_host
				ADD CONSTRAINT local_host_pkey
					PRIMARY KEY (host_id, daemon_id);
		`)
		return err
	})
}
