package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
            ALTER TABLE host ADD COLUMN IF NOT EXISTS hostname TEXT;
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
            ALTER TABLE host DROP COLUMN IF EXISTS hostname;
        `)
		return err
	})
}
