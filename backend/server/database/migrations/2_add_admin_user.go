package dbmigs

import (
	"github.com/go-pg/migrations/v7"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(
			`ALTER TABLE public.system_user
               ADD COLUMN login text;
             ALTER TABLE public.system_user
               ALTER COLUMN email DROP NOT NULL;
             ALTER TABLE public.system_user
               ADD CONSTRAINT system_user_login_unique_idx UNIQUE (login);
             ALTER TABLE public.system_user
               ADD CONSTRAINT system_user_login_email_exist_check CHECK (
                 (login IS NOT NULL) OR (email IS NOT NULL)
               );
             INSERT INTO public.system_user (login, password_hash, name, lastname)
               VALUES ('admin', crypt('admin', gen_salt('md5')), 'admin', 'admin');`)
		return err

	}, func(db migrations.DB) error {
		_, err := db.Exec(
			`DELETE FROM public.system_user
               WHERE login = 'admin';

             ALTER TABLE public.system_user
               DROP CONSTRAINT system_user_login_email_exist_check;

             ALTER TABLE public.system_user
               DROP CONSTRAINT system_user_login_unique_idx;

             ALTER TABLE public.system_user
               ALTER COLUMN email SET NOT NULL;

             ALTER TABLE public.system_user
               DROP COLUMN login;`)
		return err
	})
}
