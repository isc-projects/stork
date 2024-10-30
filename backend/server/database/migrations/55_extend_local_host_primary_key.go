package dbmigs

import "github.com/go-pg/migrations/v8"

// The migration extends the primary key of the local_host table to include
// the data_source column.
func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			ALTER TABLE local_host DROP CONSTRAINT local_host_pkey;
			ALTER TABLE local_host
				ADD CONSTRAINT local_host_pkey
					PRIMARY KEY (host_id, daemon_id, data_source);

			-- Reset the Kea config hashes to re-calculate the hashes of the
			-- DHCP options of the host reservations coming from the
			-- configuration file.
			UPDATE kea_daemon SET config_hash = NULL;
		`)
		return err
	}, func(d migrations.DB) error {
		_, err := d.Exec(`
			-- Drop the local host entries where the same host is defined in
			-- multiple data sources.
			DELETE FROM local_host lh
			USING (
				SELECT host_id, daemon_id
				FROM local_host
				GROUP BY host_id, daemon_id
				HAVING COUNT(*) > 1
			) d
			WHERE lh.host_id = d.host_id
				AND lh.daemon_id = d.daemon_id
				AND lh.data_source = 'config';

			ALTER TABLE local_host DROP CONSTRAINT local_host_pkey;
			ALTER TABLE local_host
				ADD CONSTRAINT local_host_pkey
					PRIMARY KEY (host_id, daemon_id);

			-- Reset the Kea config hashes to re-calculate the hashes of the
			-- DHCP options of the host reservations coming from the
			-- configuration file.
			UPDATE kea_daemon SET config_hash = NULL;
		`)
		return err
	})
}
