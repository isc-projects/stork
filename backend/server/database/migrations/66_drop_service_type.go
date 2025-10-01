package dbmigs

import "github.com/go-pg/migrations/v8"

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			-- Drop the unused trigger.
			DROP TRIGGER ha_service_before_insert_update ON ha_service;
			-- Drop the function that is no longer called.
			DROP FUNCTION ha_service_type_set();
			-- Drop the column that is no used anymore.
			ALTER TABLE service DROP COLUMN service_type;
			-- Drop the service type enum.
			DROP TYPE SERVICETYPE;
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			-- Recreate a service type column.
			CREATE TYPE SERVICETYPE AS ENUM ('ha_dhcp');
			ALTER TABLE service ADD COLUMN service_type SERVICETYPE;
			UPDATE service SET service_type = 'ha_dhcp';

			-- Create the function.
			CREATE FUNCTION ha_service_type_set()
				RETURNS trigger
				language 'plpgsql'
				AS $function$
			BEGIN
				NEW.service_type := 'ha_dhcp';
				RETURN NEW;
			END;
			$function$;

			-- Create the trigger.
			CREATE TRIGGER ha_service_before_insert_update
			BEFORE INSERT OR UPDATE ON ha_service
			FOR EACH ROW EXECUTE PROCEDURE ha_service_type_set();
		`)
		return err
	})
}
