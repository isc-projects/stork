package dbmigs

import (
	"github.com/go-pg/migrations/v7"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
             ALTER TABLE public.app
               ADD COLUMN ctrl_address TEXT NOT NULL DEFAULT 'localhost';
           `)
		return err

	}, func(db migrations.DB) error {
		_, err := db.Exec(`
             ALTER TABLE public.app
               DROP COLUMN ctrl_address;
           `)
		return err
	})
}
