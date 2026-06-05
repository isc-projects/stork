package dbmigs

import "github.com/go-pg/migrations/v8"

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			-- Add a new column to store the metadata for users authenticating externally (LDAP, OIDC, etc.).
			ALTER TABLE public.system_user
				ADD COLUMN meta JSONB;
			-- todo: fill data with logins and emails?
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			ALTER TABLE public.kea_daemon
				DROP COLUMN IF EXISTS meta;
			-- todo: fill logins and emails from metadata?
		`)
		return err
	})
}
