package dbmigs

import "github.com/go-pg/migrations/v8"

// The migration replaces the unique indexes on the system user table with a
// constraints that force uniqueness rules only on users authorized with the
// internal method.
func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			-- Drop current unique constraints on the system user table.
			ALTER TABLE public.system_user
				DROP CONSTRAINT IF EXISTS system_user_email_unique_idx;
			ALTER TABLE public.system_user
				DROP CONSTRAINT IF EXISTS system_user_login_unique_idx;

			-- Add partial unique indexes to enforce uniqueness only for users
			-- with the internal auth method.
			CREATE UNIQUE INDEX system_user_email_unique_idx ON public.system_user (email) WHERE auth_method = 'internal';
			CREATE UNIQUE INDEX system_user_login_unique_idx ON public.system_user (login) WHERE auth_method = 'internal';
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			-- Drop the partial unique indexes and restore the original unique constraints.
			DROP INDEX IF EXISTS system_user_email_unique_idx;
			DROP INDEX IF EXISTS system_user_login_unique_idx;

			ALTER TABLE public.system_user
				ADD CONSTRAINT system_user_email_unique_idx UNIQUE (auth_method, email),
				ADD CONSTRAINT system_user_login_unique_idx UNIQUE (auth_method, login);
		`)
		return err
	})
}
