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

             CREATE EXTENSION IF NOT EXISTS pgcrypto;

             CREATE OR REPLACE FUNCTION system_user_hash_password()
               RETURNS trigger
               LANGUAGE plpgsql
               AS $function$
             BEGIN
               IF NEW.password_hash IS NOT NULL THEN
                 NEW.password_hash := crypt(NEW.password_hash, gen_salt('md5'));
               END IF;
               RETURN NEW;
             END;
             $function$;

             CREATE TRIGGER system_user_before_insert_update
             BEFORE INSERT OR UPDATE ON system_user
               FOR EACH ROW EXECUTE PROCEDURE system_user_hash_password();

             INSERT INTO public.system_user (login, password_hash, name, lastname)
               VALUES ('admin', 'admin', 'admin', 'admin');`)
		return err

	}, func(db migrations.DB) error {
		_, err := db.Exec(
			`DELETE FROM public.system_user
               WHERE login = 'admin';

             DROP TRIGGER IF EXISTS system_user_before_insert_update ON system_user;

             DROP FUNCTION IF EXISTS system_user_hash_password;

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
