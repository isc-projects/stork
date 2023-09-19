package dbmigs

import "github.com/go-pg/migrations/v8"

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			-- Create a separate table for passwords.
			CREATE TABLE system_user_password(
				id integer NOT NULL,
				password_hash TEXT NOT NULL,
				CONSTRAINT system_user_password_pkey PRIMARY KEY (id),
				CONSTRAINT system_user_password_id_fkey FOREIGN KEY (id)
                    REFERENCES public.system_user (id) MATCH FULL
                    ON UPDATE CASCADE
                    ON DELETE CASCADE
			);

			-- Move password hashes to the new table.
			INSERT INTO system_user_password(id, password_hash)
			SELECT id, password_hash
			FROM public.system_user;

			-- Drop an existing password hash trigger.
			DROP TRIGGER system_user_before_insert_update ON public.system_user;

			-- Drop the old password hash column.
			ALTER TABLE public.system_user DROP COLUMN password_hash;

			-- Recreate the trigger on the new column.
			CREATE TRIGGER system_user_password_before_insert_update
              BEFORE INSERT OR UPDATE ON system_user_password
                FOR EACH ROW EXECUTE PROCEDURE system_user_hash_password();

			-- Add a column for an authentication method.
			ALTER TABLE public.system_user ADD COLUMN auth_method TEXT DEFAULT 'internal' NOT NULL;

			-- Update constraints for login and email.
			ALTER TABLE public.system_user
               DROP CONSTRAINT system_user_login_unique_idx;

			ALTER TABLE public.system_user
               ADD CONSTRAINT system_user_login_unique_idx UNIQUE (auth_method, login);

			ALTER TABLE public.system_user
               DROP CONSTRAINT system_user_email_unique_idx;

			ALTER TABLE public.system_user
               ADD CONSTRAINT system_user_email_unique_idx UNIQUE (auth_method, email);

			-- Add a column for an external ID.
			ALTER TABLE public.system_user ADD COLUMN external_id TEXT;

			ALTER TABLE public.system_user
				ADD CONSTRAINT system_user_external_id_unique_idx UNIQUE (auth_method, external_id);

			ALTER TABLE public.system_user
				ADD CONSTRAINT system_user_external_id_required_for_external_users CHECK (
					(auth_method = 'internal') = (external_id IS NULL)
				);
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			-- Drop the rows representing the external users.
			DELETE FROM public.system_user WHERE auth_method != 'internal';

			-- Drop the external ID column.
			ALTER TABLE public.system_user DROP CONSTRAINT system_user_external_id_required_for_external_users;

			ALTER TABLE public.system_user DROP CONSTRAINT system_user_external_id_unique_idx;

			ALTER TABLE public.system_user DROP COLUMN external_id;

			-- Restore the original unique indexes.
			ALTER TABLE public.system_user
               DROP CONSTRAINT system_user_login_unique_idx;

			ALTER TABLE public.system_user
               ADD CONSTRAINT system_user_login_unique_idx UNIQUE (login);

			ALTER TABLE public.system_user
               DROP CONSTRAINT system_user_email_unique_idx;

			ALTER TABLE public.system_user
               ADD CONSTRAINT system_user_email_unique_idx UNIQUE (email);

			-- Drop the authentication method column.
			ALTER TABLE public.system_user DROP COLUMN auth_method;

			-- Drop trigger on the password table.
			DROP TRIGGER system_user_password_before_insert_update ON system_user_password;

			-- Create the password hash column in the system user table.
			-- Generate the random password for all rows.
			ALTER TABLE public.system_user
			ADD COLUMN password_hash TEXT NOT NULL DEFAULT crypt(md5(random()::text), gen_salt('bf'));

			-- Drop the default statement.
			ALTER TABLE public.system_user ALTER COLUMN password_hash DROP DEFAULT;

			-- Recreate trigger on the password hash column.
			CREATE TRIGGER system_user_before_insert_update
              BEFORE INSERT OR UPDATE ON system_user_password
                FOR EACH ROW EXECUTE PROCEDURE system_user_hash_password();

			-- Restore the password hashes in the system user table.
			UPDATE public.system_user
			SET password_hash = system_user_password.password_hash
			FROM system_user_password
			WHERE public.system_user.id = system_user_password.id;

			-- Drop the password table.
			DROP TABLE system_user_password;
		`)
		return err
	})
}
