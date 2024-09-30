package dbmigs

import "github.com/go-pg/migrations/v8"

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			-- Add a new column to indicate if the user must change the password.
			ALTER TABLE public.system_user ADD COLUMN change_password BOOLEAN NOT NULL DEFAULT FALSE;

			-- Set the change password flag if admin user has the default password.
			UPDATE public.system_user su
			SET change_password = TRUE
			FROM system_user_password p
			WHERE su.id = p.id
				AND su.login = 'admin'
				AND p.password_hash = crypt('admin', p.password_hash);
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			ALTER TABLE public.system_user DROP COLUMN change_password;
		`)
		return err
	})
}
