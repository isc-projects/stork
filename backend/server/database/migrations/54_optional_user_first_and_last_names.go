package dbmigs

import "github.com/go-pg/migrations/v8"

// The migration drops the non null constraint on the name and lastname columns.
func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			ALTER TABLE public.system_user
				ALTER COLUMN name DROP NOT NULL,
				ALTER COLUMN lastname DROP NOT NULL;
			
			UPDATE public.system_user
			SET name = NULL
			WHERE name = '';

			UPDATE public.system_user
			SET lastname = NULL
			WHERE lastname = '';

			UPDATE public.system_user
			SET email = NULL
			WHERE email = '';
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			UPDATE public.system_user
			SET name = ''
			WHERE name IS NULL;

			UPDATE public.system_user
			SET lastname = ''
			WHERE lastname IS NULL;

			ALTER TABLE public.system_user
				ALTER COLUMN name SET NOT NULL,
				ALTER COLUMN lastname SET NOT NULL;
		`)
		return err
	})
}
