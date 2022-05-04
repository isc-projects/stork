package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
            -- Adds the name column to the app table.
            ALTER TABLE app ADD COLUMN name TEXT;

             -- For each app in this table, generate a name. The basic name has the
            -- [app-type]@[machine-address] format.
            UPDATE app
                SET name = CONCAT(app.type, '@', address)
             FROM machine
             WHERE app.machine_id = machine.id;

             -- For each machine running multiple instances of the same app,
             -- postfixes must be added to some apps' names to make them unique. One of
             -- the apps can be left without a postfix. The other apps running on the
            -- same machine will have the following format:
            -- [app-type]@[machine-address]%[app-id].
            DO $$
            DECLARE
                 r record;
                 last_name TEXT;
            BEGIN
                FOR r IN
                    -- This query finds all records which have duplicate name entries.
                    WITH apps AS (
                        SELECT a.*, count(*) OVER (PARTITION BY name) AS count
                        FROM app AS a
                        ORDER BY name
                    )
                     SELECT * FROM apps WHERE count > 1
                LOOP
                    -- This appends a postfix only if this is next occurrence of the same name.
                    IF (last_name = name) THEN
                         UPDATE app SET name = CONCAT(name, '%', id::TEXT) WHERE id = r.id;
                    ELSE
                        last_name = name;
                    END IF;
                END LOOP;
            END $$;

            -- This trigger function creates a default app name when a new app is added or an
            -- app is updated and the name is not specified.
            CREATE OR REPLACE FUNCTION create_default_app_name()
                RETURNS trigger
                LANGUAGE 'plpgsql'
                AS $function$
            BEGIN
                -- Trims whitespace before and after the actual name.
                SELECT REGEXP_REPLACE(NEW.name, '\s+$', '') INTO NEW.name;
                SELECT REGEXP_REPLACE(NEW.name, '^\s+', '') INTO NEW.name;

                IF NEW.name IS NULL OR NEW.name = '' THEN
                    -- Creates a base name without a postfix.
                    NEW.name = CONCAT(NEW.type, '@', (SELECT address FROM machine WHERE id = NEW.machine_id));
                    -- Checks whether the postfix is needed. It is necessary when the name already exists
                    -- without the postfix.
                    IF ((SELECT COUNT(*) FROM app WHERE name = NEW.name) > 0) THEN
                        NEW.name = CONCAT(NEW.name, '%', NEW.id::TEXT);
                    END IF;
                END IF;
            RETURN NEW;
            END;
            $function$;

            -- Creates a trigger checking whether the record lacks a name. If it does, generate one.
            DO $$ BEGIN
                CREATE TRIGGER trigger_create_default_app_name
                    BEFORE INSERT OR UPDATE ON app
                         FOR EACH ROW EXECUTE PROCEDURE create_default_app_name();
            END $$;

            -- This trigger function verifies that an app name is valid. If the app name has the following
            -- pattern [text]@[machine-address][%id], it checks that the machine with the given name
            -- exists. The special format [text]@@[machine-address] can be used to avoid this check.
            CREATE OR REPLACE FUNCTION validate_app_name()
                RETURNS trigger
                LANGUAGE 'plpgsql'
                AS $function$
            DECLARE
                machine_name TEXT;
            BEGIN
                machine_name = SUBSTRING(NEW.name, CONCAT('@', '([^\%]+)'));
                IF machine_name IS NOT NULL AND STRPOS(machine_name, '@') = 0 THEN
                    IF ((SELECT COUNT(*) FROM machine WHERE address = machine_name) = 0) THEN
                         RAISE EXCEPTION 'machine % does not exist', machine_name;
                    END IF;
                END IF;
                RETURN NEW;
            END;
            $function$;

            -- Creates a trigger validating the app name.
            DO $$ BEGIN
                CREATE TRIGGER trigger_validate_app_name
                    BEFORE INSERT OR UPDATE ON app
                        FOR EACH ROW EXECUTE PROCEDURE validate_app_name();
            END $$;

            -- This trigger function is invoked when a machine's address is updated. It updates all
            -- app names derived from the previous machine name to a new machine name.
            CREATE OR REPLACE FUNCTION replace_app_name()
                RETURNS trigger
                LANGUAGE 'plpgsql'
                AS $function$
            BEGIN
                -- Updates the app name only if the machine name was changed.
                IF NEW.address != OLD.address THEN
                    -- For each app following the pattern [text]@[machine-address], updates the
                    -- app name.
                    UPDATE app
                         SET name = REGEXP_REPLACE(name, CONCAT('@', OLD.address, '((\%\d+){0,1})$'), CONCAT('@', NEW.address, '\2'))
                    WHERE app.machine_id = NEW.id;
                END IF;
                RETURN NEW;
            END;
            $function$;

            -- Creates the trigger to update the app names as a result of a machine address change.
            DO $$ BEGIN
                CREATE TRIGGER trigger_replace_app_name
                    AFTER UPDATE ON machine
                        FOR EACH ROW EXECUTE PROCEDURE replace_app_name();
            END $$;

            ALTER TABLE app ADD CONSTRAINT app_name_unique UNIQUE (name);
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
            ALTER TABLE app DROP CONSTRAINT app_name_unique;

            DROP TRIGGER IF EXISTS trigger_replace_app_name ON machine;
            DROP FUNCTION IF EXISTS replace_app_name;

            DROP TRIGGER IF EXISTS trigger_validate_app_name on app;
            DROP FUNCTION IF EXISTS validate_app_name;

            DROP TRIGGER IF EXISTS trigger_create_default_app_name ON app;
            DROP FUNCTION IF EXISTS create_default_app_name;

            ALTER TABLE app DROP COLUMN name;
        `)
		return err
	})
}
