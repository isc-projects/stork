package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
            -- Checks that the deleted operation does not remove last super-admin user.
            -- It is called by the DELETE triggers on the system_user table.
            CREATE OR REPLACE FUNCTION system_user_check_last_user()
              RETURNS trigger
              LANGUAGE plpgsql
              AS $function$
            DECLARE
              user_count integer;
              user_group integer;
            BEGIN
              SELECT group_id FROM system_user_to_group WHERE user_id = OLD.id INTO user_group;
              IF user_group != 1 THEN
                RETURN OLD;
              END IF;
              SELECT COUNT(*) FROM system_user_to_group WHERE group_id = 1 AND user_id != OLD.id LIMIT 1 INTO user_count;
              IF user_count = 0 THEN
                RAISE EXCEPTION 'deleting last super-admin user is forbidden';
              END IF;
              RETURN OLD;
            END;
            $function$;

            -- Check last super-admin user before delete.
            CREATE TRIGGER system_user_before_delete
            BEFORE DELETE ON public.system_user
              FOR EACH ROW EXECUTE PROCEDURE system_user_check_last_user();
            `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
            -- Removes the trigger checking last super-admin user.
            DROP TRIGGER IF EXISTS system_user_before_delete ON public.system_user;

            -- Removes the check last super-admin user function.
            DROP FUNCTION IF EXISTS system_user_check_last_user;
            `)
		return err
	})
}
