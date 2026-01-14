package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			-- Drop the obsolete trigger and function.
			DROP TRIGGER trigger_update_machine_id ON access_point;
			DROP FUNCTION update_machine_id;

			-- Add a column in the access point table to store a new foreign key
			-- to the daemon table.
			ALTER TABLE access_point ADD COLUMN daemon_id bigint;
			-- Fill a reference to the daemon table. Use the daemon ID of the
			-- control daemon to preserve the existing capabilities to connect
			-- to the Kea CA daemon. This value is temporary and will be
			-- updated as soon as the Stork agent re-detects the Kea processes.
			UPDATE access_point
			SET daemon_id = daemon.id
			FROM app, daemon
			WHERE access_point.app_id = app.id AND app.id = daemon.app_id;
			-- Set constraints for the new column.
			ALTER TABLE access_point ALTER COLUMN daemon_id SET NOT NULL;
			ALTER TABLE access_point ADD CONSTRAINT access_point_daemon_id_fkey
				FOREIGN KEY (daemon_id)
				REFERENCES daemon(id)
				MATCH FULL
				ON UPDATE CASCADE
				ON DELETE CASCADE;
			-- Drop the unnecessary reference to the machine table.
			ALTER TABLE access_point DROP COLUMN machine_id;
			-- Change the primary key.
			ALTER TABLE access_point DROP CONSTRAINT access_point_pkey;
			ALTER TABLE access_point ADD CONSTRAINT access_point_pkey
				PRIMARY KEY (daemon_id, type);
			-- Create access points for daemons that don't have any. Copy them
			-- from the access points of the corresponding app.
			INSERT INTO access_point(app_id, "type", address, port, "key", "use_secure_protocol", daemon_id)
			SELECT ap_copy.app_id, ap_copy."type", ap_copy.address, ap_copy.port,
					ap_copy.key, ap_copy.use_secure_protocol, daemon.id AS daemon_id 
			FROM daemon
			LEFT JOIN access_point ap_existing ON ap_existing.daemon_id = daemon.id
			LEFT JOIN access_point ap_copy ON ap_copy.app_id = daemon.app_id
			WHERE ap_existing.daemon_id IS NULL AND ap_copy.daemon_id IS NOT NULL;
			-- Drop the unnecessary reference to the app table.
			ALTER TABLE access_point DROP COLUMN app_id;

			-- Add a reference to the machine table in the daemon table.
			ALTER TABLE daemon ADD COLUMN machine_id bigint;
			-- Fill the reference to the machine table.
			UPDATE daemon
			SET machine_id = machine.id
			FROM app, machine
			WHERE daemon.app_id = app.id AND app.machine_id = machine.id;
			-- Set constraints for the new column.
			ALTER TABLE daemon ALTER COLUMN machine_id SET NOT NULL;
			ALTER TABLE daemon ADD CONSTRAINT daemon_machine_id_fkey
				FOREIGN KEY (machine_id)
				REFERENCES machine(id)
				MATCH FULL
				ON UPDATE CASCADE
				ON DELETE CASCADE;
			-- Drop the unnecessary reference to the app table.
			ALTER TABLE daemon DROP COLUMN app_id;

			-- Update name of the state puller in settings.
			UPDATE setting
			SET name = 'state_puller_interval'
			WHERE name = 'apps_state_puller_interval';

			-- Change the column indicating whether the secure protocol is used
			-- to a column storing the protocol name.
			ALTER TABLE access_point ADD COLUMN protocol TEXT;
			UPDATE access_point
			SET protocol = CASE
				WHEN use_secure_protocol THEN 'https'
				ELSE 'http'
			END;
			ALTER TABLE access_point ALTER COLUMN protocol SET NOT NULL;
			ALTER TABLE access_point DROP COLUMN use_secure_protocol;

			-- Drop obsolete tables and functions.
			DROP TRIGGER trigger_create_default_app_name ON app;
			DROP TRIGGER trigger_validate_app_name ON app;
			DROP TRIGGER trigger_replace_app_name ON machine;
			DROP FUNCTION create_default_app_name;
			DROP FUNCTION validate_app_name;
			DROP FUNCTION replace_app_name;
			DROP TABLE app;
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			-- === APP ===

			-- Recreate the app table
			CREATE TABLE public.app (
				id serial4 NOT NULL,
				created_at timestamp DEFAULT (now() AT TIME ZONE 'utc'::text) NOT NULL,
				machine_id int4 NOT NULL,
				"type" varchar(10) NOT NULL,
				active bool DEFAULT false NULL,
				meta jsonb NULL,
				details jsonb NULL,
				"name" text NULL,
				CONSTRAINT app_name_unique UNIQUE (name),
				CONSTRAINT app_pkey PRIMARY KEY (id),
				CONSTRAINT app_machine_id_fkey FOREIGN KEY (machine_id)
					REFERENCES public.machine(id)
						ON DELETE CASCADE
						ON UPDATE CASCADE
			);

			-- This trigger function creates a default app name when a new app is added or an
			-- app is updated and the name is not specified.
			CREATE FUNCTION create_default_app_name()
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

			-- This trigger function verifies that an app name is valid. If the app name has the following
			-- pattern [text]@[machine-address][%id], it checks that the machine with the given name
			-- exists. The special format [text]@@[machine-address] can be used to avoid this check.
			CREATE FUNCTION validate_app_name()
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

			-- This trigger function is invoked when a machine's address is updated. It updates all
			-- app names derived from the previous machine name to a new machine name.
			CREATE FUNCTION replace_app_name()
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

			CREATE TRIGGER trigger_create_default_app_name BEFORE INSERT OR UPDATE
			ON public.app FOR EACH ROW EXECUTE PROCEDURE create_default_app_name();

			CREATE TRIGGER trigger_validate_app_name BEFORE INSERT OR UPDATE
			ON public.app FOR EACH ROW EXECUTE PROCEDURE validate_app_name();

			CREATE TRIGGER trigger_replace_app_name AFTER UPDATE
			ON machine FOR EACH ROW EXECUTE PROCEDURE replace_app_name();

			-- === PROTOCOL ===

			-- Change protocol column back to use_secure_protocol
			ALTER TABLE access_point ADD COLUMN use_secure_protocol boolean NOT NULL DEFAULT false;
			UPDATE access_point
			SET use_secure_protocol = CASE
				WHEN protocol = 'https' THEN true
				ELSE false
			END;
			ALTER TABLE access_point DROP COLUMN protocol;

			-- === SETTING ===

			-- Revert the setting name change
			UPDATE setting
			SET name = 'apps_state_puller_interval'
			WHERE name = 'state_puller_interval';

			-- === MACHINE - APP - DAEMON REFERENCES ===

			-- Add app_id column back to daemon table
			ALTER TABLE daemon ADD COLUMN app_id bigint;

			-- Create apps from existing daemons
			-- Group daemons by machine and create an app for each machine
			INSERT INTO app (created_at, machine_id, type, active, meta, name)
			SELECT d.created_at, d.machine_id,
				CASE
					WHEN (d.name = 'named') THEN 'bind9'
					WHEN (d.name = 'pdns') THEN 'pdns'
					ELSE 'kea'
				END AS "type",
				d.active,
				json_build_object('version', d.version, 'extendedVersion', d.extended_version) AS meta,
				'app-' || ap.address || '-' || ap.port AS "name"
			FROM daemon d
			RIGHT JOIN (
				SELECT MIN(ap.daemon_id) AS daemon_id
				FROM access_point ap
				WHERE ap.type = 'control'
				GROUP BY ap.address, ap.port
			) dap ON d.id = dap.daemon_id
			LEFT JOIN access_point ap ON ap.daemon_id = dap.daemon_id AND ap.type = 'control';

			-- Update daemon.app_id to reference the created apps
			UPDATE daemon
			SET app_id = da.app_id
			FROM (
				SELECT d.id AS daemon_id, app.id AS app_id
				FROM daemon d
				LEFT JOIN access_point ap ON ap.daemon_id = d.id AND ap.type = 'control'
				RIGHT JOIN app ON app.name = ('app-' || ap.address || '-' || ap.port)
			) da
			WHERE daemon.id = da.daemon_id;

			-- Set constraints for app_id
			ALTER TABLE daemon ALTER COLUMN app_id SET NOT NULL;
			ALTER TABLE daemon ADD CONSTRAINT daemon_app_id_fkey
				FOREIGN KEY (app_id)
				REFERENCES app(id)
				MATCH FULL
				ON UPDATE CASCADE
				ON DELETE CASCADE;

			-- Drop the machine_id foreign key from daemon
			ALTER TABLE daemon DROP CONSTRAINT daemon_machine_id_fkey;
			ALTER TABLE daemon DROP COLUMN machine_id;

			-- === ACCESS POINT ===

			-- Add app_id column back to access_point table
			ALTER TABLE access_point ADD COLUMN app_id bigint;

			-- Fill app_id in access_point from daemon
			UPDATE access_point
			SET app_id = daemon.app_id
			FROM daemon
			WHERE access_point.daemon_id = daemon.id;

			-- Add machine_id column back to access_point table
			ALTER TABLE access_point ADD COLUMN machine_id bigint;

			-- Fill machine_id in access_point from app
			UPDATE access_point
			SET machine_id = app.machine_id
			FROM app
			WHERE access_point.app_id = app.id;

			-- Set constraints for access_point
			ALTER TABLE access_point ALTER COLUMN app_id SET NOT NULL;
			ALTER TABLE access_point ALTER COLUMN machine_id SET NOT NULL;

			-- Remove duplicated access points (same app_id and type).
			DELETE FROM access_point
			WHERE daemon_id NOT IN (
				SELECT MIN(daemon_id)
				FROM access_point ap
				GROUP BY app_id
			);

			-- Change primary key back
			ALTER TABLE access_point DROP CONSTRAINT access_point_pkey;
			ALTER TABLE access_point ADD CONSTRAINT access_point_pkey
				PRIMARY KEY (app_id, type);

			-- Add foreign key constraints for access_point
			ALTER TABLE access_point ADD CONSTRAINT access_point_app_id
				FOREIGN KEY (app_id)
				REFERENCES app (id) MATCH SIMPLE
				ON UPDATE NO ACTION
				ON DELETE CASCADE;

			ALTER TABLE access_point ADD CONSTRAINT access_point_machine_id
				FOREIGN KEY (machine_id)
				REFERENCES machine (id) MATCH SIMPLE
				ON UPDATE NO ACTION
				ON DELETE CASCADE;

			-- Drop the daemon_id column and corresponding foreign key from access_point
			ALTER TABLE access_point DROP COLUMN daemon_id;

			-- === MACHINE ===

			-- Trigger function inserting control access point every time an
			-- app created.
			CREATE FUNCTION update_machine_id()
				RETURNS trigger
				LANGUAGE 'plpgsql'
				AS $function$
			BEGIN
				UPDATE app SET machine_id = NEW.machine_id WHERE id = NEW.app_id;
				RETURN NEW;
			END;
			$function$;

			CREATE TRIGGER trigger_update_machine_id BEFORE INSERT OR UPDATE
			ON access_point FOR EACH ROW EXECUTE PROCEDURE update_machine_id();
		`)
		return err
	})
}
