package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
             ALTER TABLE daemon ADD COLUMN monitored BOOLEAN DEFAULT TRUE;
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
             ALTER TABLE daemon DROP COLUMN monitored;
        `)
		return err
	})
}
