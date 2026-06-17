package dbmigs

import "github.com/go-pg/migrations/v8"

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			-- Add a new column to indicate if the user group associations are managed externally (LDAP, OIDC, etc).
			ALTER TABLE public.system_user ADD COLUMN externally_managed_groups BOOLEAN NOT NULL DEFAULT FALSE;
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			ALTER TABLE public.system_user DROP COLUMN externally_managed_groups;
		`)
		return err
	})
}
