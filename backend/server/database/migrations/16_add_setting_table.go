package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
             -- Create a table of setting.
             CREATE TABLE IF NOT EXISTS setting (
                 name TEXT NOT NULL,
                 val_type INTEGER NOT NULL,
                 value TEXT NOT NULL,
                 CONSTRAINT setting_pkey PRIMARY KEY (name)
             );
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
             DROP TABLE IF EXISTS setting;
        `)
		return err
	})
}
