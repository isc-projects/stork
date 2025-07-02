package dbmigs

import "github.com/go-pg/migrations/v8"

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			-- Drop the unused trigger.
			DROP TRIGGER ha_service_before_insert_update ON ha_service;
			-- Drop the function that is no longer called.
			DROP FUNCTION ha_service_type_set();
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			-- Create the function.
			CREATE FUNCTION ha_service_type_set() RETURNS TRIGGER AS $$
			BEGIN
				NEW.service_type := 'ha_dhcp';
				RETURN NEW;
			END;
			$$ LANGUAGE plpgsql;

			-- Create the trigger.
			CREATE TRIGGER ha_service_before_insert_update
			BEFORE INSERT OR UPDATE ON ha_service
			FOR EACH ROW EXECUTE FUNCTION ha_service_type_set();
		`)
		return err
	})
}
