package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
             -- Add _at prefixes to timestamp columns.
             ALTER TABLE app RENAME COLUMN created TO created_at;
             ALTER TABLE machine RENAME COLUMN created TO created_at;
             ALTER TABLE machine RENAME COLUMN last_visited TO last_visited_at;
             ALTER TABLE shared_network RENAME COLUMN created TO created_at;
             ALTER TABLE address_pool RENAME COLUMN created TO created_at;
             ALTER TABLE prefix_pool RENAME COLUMN created TO created_at;
             ALTER TABLE service RENAME COLUMN created TO created_at;
             ALTER TABLE ha_service RENAME COLUMN primary_status_time TO primary_status_collected_at;
             ALTER TABLE ha_service RENAME COLUMN secondary_status_time TO secondary_status_collected_at;
             ALTER TABLE subnet RENAME COLUMN created TO created_at;
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
             -- Restore previous names of timestamp columns.
             ALTER TABLE app RENAME COLUMN created_at TO created;
             ALTER TABLE machine RENAME COLUMN created_at TO created;
             ALTER TABLE machine RENAME COLUMN last_visited_at TO last_visited;
             ALTER TABLE shared_network RENAME COLUMN created_at TO created;
             ALTER TABLE address_pool RENAME COLUMN created_at TO created;
             ALTER TABLE prefix_pool RENAME COLUMN created_at TO created;
             ALTER TABLE service RENAME COLUMN created_at TO created;
             ALTER TABLE ha_service RENAME COLUMN primary_status_collected_at TO primary_status_time;
             ALTER TABLE ha_service RENAME COLUMN secondary_status_collected_at TO secondary_status_time;
             ALTER TABLE subnet RENAME COLUMN created_at TO created;
        `)
		return err
	})
}
