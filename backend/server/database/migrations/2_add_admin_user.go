package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
             -- Login is a convenient alternative to email. Also, the default
             -- admin user has no email set initially.
             ALTER TABLE public.system_user
               ADD COLUMN login text;

             -- Email can be NULL in some circumstances, e.g. for the default admin user.
             ALTER TABLE public.system_user
               ALTER COLUMN email DROP NOT NULL;

             -- Login must be unique across the database.
             ALTER TABLE public.system_user
               ADD CONSTRAINT system_user_login_unique_idx UNIQUE (login);

             -- At least one of the login or email must be present. Many times
             -- both are present.
             ALTER TABLE public.system_user
               ADD CONSTRAINT system_user_login_email_exist_check CHECK (
                 (login IS NOT NULL) OR (email IS NOT NULL)
               );

             -- Enables crypt and gen_salt functions.
             CREATE EXTENSION IF NOT EXISTS pgcrypto;

             -- Takes the user specified password and hashes it on the flight.
             -- It is called by the INSERT/UPDATE triggers on the system_user table.
             CREATE OR REPLACE FUNCTION system_user_hash_password()
               RETURNS trigger
               LANGUAGE plpgsql
               AS $function$
             BEGIN
               IF NEW.password_hash IS NOT NULL THEN
                 NEW.password_hash := crypt(NEW.password_hash, gen_salt('bf'));
               ELSIF OLD.password_hash IS NOT NULL THEN
                 NEW.password_hash := OLD.password_hash;
               END IF;
               RETURN NEW;
             END;
             $function$;

             -- Transforms a raw password to a hash before INSERT/UPDATE.
             CREATE TRIGGER system_user_before_insert_update
             BEFORE INSERT OR UPDATE ON public.system_user
               FOR EACH ROW EXECUTE PROCEDURE system_user_hash_password();

             -- The admin is the default user who can use the 'admin' password to
             -- login to the system.
             INSERT INTO public.system_user (login, password_hash, name, lastname)
               VALUES ('admin', 'admin', 'admin', 'admin');
           `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
             -- Deletes all users. It includes the admin user added in the forward
             -- migration. We cannot preserve other users in the database because
             -- below we are going to apply NOT NULL constraint on the email column.
             -- We could theoretically only remove the users that have NULL email
             -- but it would leave the database in the inconsistent state. Generating
             -- artificial emails is not a good idea too, because they will remain
             -- if the admin runs forward migrations. Removing all users ensures that
             -- the version 1 of the database has the same contents as it had at the
             -- creation time.
             DELETE FROM public.system_user;

             -- Removes the trigger hashing passwords.
             DROP TRIGGER IF EXISTS system_user_before_insert_update ON public.system_user;

             -- Removes the password hashing function.
             DROP FUNCTION IF EXISTS system_user_hash_password;

             -- Removes the check for the presence of an email or login.
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
               DROP COLUMN login;
           `)
		return err
	})
}
