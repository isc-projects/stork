package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
             ALTER TABLE public.app
               ADD COLUMN ctrl_key TEXT DEFAULT '';
           `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
             ALTER TABLE public.app
               DROP COLUMN ctrl_key;
           `)
		return err
	})
}
