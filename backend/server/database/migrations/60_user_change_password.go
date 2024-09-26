package dbmigs

import "github.com/go-pg/migrations/v8"

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			ALTER TABLE public.system_user ADD COLUMN change_password BOOLEAN NOT NULL DEFAULT FALSE;
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			ALTER TABLE public.system_user DROP COLUMN change_password;
		`)
		return err
	})
}
