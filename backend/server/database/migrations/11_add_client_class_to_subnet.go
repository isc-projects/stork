package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
             -- Add client_class column to subnet.
             ALTER TABLE subnet ADD COLUMN client_class TEXT;
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
             -- Remove client_class column from subnet.
             ALTER TABLE subnet DROP COLUMN client_class;
        `)
		return err
	})
}
