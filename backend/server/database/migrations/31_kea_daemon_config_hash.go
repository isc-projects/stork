package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
             -- Adds the config_hash column to kea_daemon.
             ALTER TABLE kea_daemon ADD COLUMN config_hash TEXT;
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
             -- Removes the config_hash column from kea_daemon.
             ALTER TABLE kea_daemon DROP COLUMN config_hash;
        `)
		return err
	})
}
