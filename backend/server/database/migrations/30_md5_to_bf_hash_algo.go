package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(
			`
             -- Takes the user-specified password and hashes it on the flight.
             -- It is called by the INSERT/UPDATE triggers on the system_user table.
             -- This time use blowfish instead of md5.
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
             $function$;`)
		return err
	}, func(db migrations.DB) error {
		return nil
	})
}
