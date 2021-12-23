package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
             ALTER TABLE public.app
               ADD COLUMN ctrl_address TEXT DEFAULT 'localhost';

             ALTER TABLE public.app
               ADD CONSTRAINT app_ctrl_address_not_empty CHECK (
                   NOT (
                     ctrl_address IS NULL || ctrl_address = ''
                   )
               );
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
