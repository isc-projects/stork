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
                    REFERENCES system_user (id) MATCH FULL
                    ON UPDATE CASCADE
                    ON DELETE CASCADE
			);

			-- Move password hashes to the new table.
			INSERT INTO system_user_password(id, password_hash)
			SELECT id, password_hash
			FROM system_user;

			-- Drop an existing password hash trigger.
			DROP TRIGGER system_user_before_insert_update ON system_user;

			-- Drop the old password hash column.
			ALTER TABLE system_user DROP COLUMN password_hash;

			-- Recreate the trigger on the new column.
			CREATE TRIGGER system_user_password_before_insert_update
              BEFORE INSERT OR UPDATE ON system_user_password
                FOR EACH ROW EXECUTE PROCEDURE system_user_hash_password();

			-- Add a column for an authentication method.
			ALTER TABLE system_user ADD COLUMN auth_method TEXT DEFAULT 'internal' NOT NULL;

			-- Update constraints for login and email.
			ALTER TABLE system_user
               DROP CONSTRAINT system_user_login_unique_idx;

			ALTER TABLE system_user
               ADD CONSTRAINT system_user_login_unique_idx UNIQUE (auth_method, login);

			ALTER TABLE system_user
               DROP CONSTRAINT system_user_email_unique_idx;

			ALTER TABLE system_user
               ADD CONSTRAINT system_user_email_unique_idx UNIQUE (auth_method, email);
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			-- We cannot drop the rows representing the external users
			-- because it causes to drop the related data.

			-- Modify logins and emails of the external users to ensure they
			-- are unique.
			UPDATE system_user o
			SET email = n.email || '.' || n.auth_method,
				login = n.login || '_' || n.auth_method
			FROM system_user n
			WHERE o.id = n.id AND o.auth_method != 'internal';

			-- Restore the original unique indexes.
			ALTER TABLE system_user
               DROP CONSTRAINT system_user_login_unique_idx;

			ALTER TABLE system_user
               ADD CONSTRAINT system_user_login_unique_idx UNIQUE (login);

			ALTER TABLE system_user
               DROP CONSTRAINT system_user_email_unique_idx;

			ALTER TABLE system_user
               ADD CONSTRAINT system_user_email_unique_idx UNIQUE (email);

			-- Drop the authentication method column.
			ALTER TABLE system_user DROP COLUMN auth_method;

			-- Drop trigger on the password table.
			DROP TRIGGER system_user_password_before_insert_update ON system_user_password;

			-- Create the password hash column in the system user table.
			-- Generate the random password for all rows.
			ALTER TABLE system_user
			ADD COLUMN password_hash TEXT NOT NULL DEFAULT crypt(md5(random()::text), gen_salt('bf'));

			-- Drop the default statement.
			ALTER TABLE system_user ALTER COLUMN password_hash DROP DEFAULT;

			-- Recreate trigger on the password hash column.
			CREATE TRIGGER system_user_before_insert_update
              BEFORE INSERT OR UPDATE ON system_user_password
                FOR EACH ROW EXECUTE PROCEDURE system_user_hash_password();

			-- Restore the password hashes in the system user table.
			UPDATE system_user
			SET password_hash = system_user_password.password_hash
			FROM system_user_password
			WHERE system_user.id = system_user_password.id;

			-- Drop the password table.
			DROP TABLE system_user_password;
		`)
		return err
	})
}
