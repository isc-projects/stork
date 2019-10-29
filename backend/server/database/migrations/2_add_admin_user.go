package dbmigs

import (
	"github.com/go-pg/migrations/v7"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(
			`-- Login is convenient alternative to email. Also, the default
             -- admin user has no email set initially.
             ALTER TABLE public.system_user
               ADD COLUMN login text;

             -- Email can be NULL in some circumstances, e.g. for default admin user.
             ALTER TABLE public.system_user
               ALTER COLUMN email DROP NOT NULL;

             -- Login must be unique accross the database.
             ALTER TABLE public.system_user
               ADD CONSTRAINT system_user_login_unique_idx UNIQUE (login);

             -- At least one of the login or email must be present. Many times
             -- both will be present.
             ALTER TABLE public.system_user
               ADD CONSTRAINT system_user_login_email_exist_check CHECK (
                 (login IS NOT NULL) OR (email IS NOT NULL)
               );

             -- Enables crypt and gen_salt functions.
             CREATE EXTENSION IF NOT EXISTS pgcrypto;

             -- Takes the user specified password and hashes it on the flight.
             -- It is called by the INSERT/UPDATE triggers on system_user table.
             CREATE OR REPLACE FUNCTION system_user_hash_password()
               RETURNS trigger
               LANGUAGE plpgsql
               AS $function$
             BEGIN
               IF NEW.password_hash IS NOT NULL THEN
                 NEW.password_hash := crypt(NEW.password_hash, gen_salt('md5'));
               ELSIF OLD.password_hash IS NOT NULL THEN
                 NEW.password_hash := OLD.password_hash;
               END IF;
               RETURN NEW;
             END;
             $function$;

             -- Transform raw password to a hash before INSERT/UPDATE.
             CREATE TRIGGER system_user_before_insert_update
             BEFORE INSERT OR UPDATE ON system_user
               FOR EACH ROW EXECUTE PROCEDURE system_user_hash_password();

             -- The admin is the default user which can use 'admin' password to
             -- login to the system.
             INSERT INTO public.system_user (login, password_hash, name, lastname)
               VALUES ('admin', 'admin', 'admin', 'admin');`)
		return err

	}, func(db migrations.DB) error {
		_, err := db.Exec(
			`-- Delete the default user.
             DELETE FROM public.system_user
               WHERE login = 'admin';

             -- Remove trigger hashing passwords.
             DROP TRIGGER IF EXISTS system_user_before_insert_update ON system_user;

             -- Remove password hashing function.
             DROP FUNCTION IF EXISTS system_user_hash_password;

             -- Don't check for email or login being present.
             ALTER TABLE public.system_user
               DROP CONSTRAINT system_user_login_email_exist_check;

             -- User login is no longer unique.
             ALTER TABLE public.system_user
               DROP CONSTRAINT system_user_login_unique_idx;

             -- Email is now the only identifier so it must not be null.
             ALTER TABLE public.system_user
               ALTER COLUMN email SET NOT NULL;

             -- Login is no longer used.
             ALTER TABLE public.system_user
               DROP COLUMN login;`)
		return err
	})
}
