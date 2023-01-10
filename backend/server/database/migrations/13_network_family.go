package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
             -- Add column which specifies the family of subnets grouped within it.
             ALTER TABLE shared_network
                 ADD COLUMN inet_family INT;

             -- Set initial values.
             UPDATE shared_network
                 SET inet_family=family(subnet.prefix)
             FROM subnet
             WHERE shared_network.id=subnet.shared_network_id;

             -- Drop orphaned shared networks.
             DELETE FROM shared_network
             WHERE inet_family IS NULL;

             -- Require a value.
             ALTER TABLE shared_network
                 ALTER COLUMN inet_family SET NOT NULL;

             -- Make sure that the family is one of the IPv4 or IPv6.
             ALTER TABLE shared_network
                 ADD CONSTRAINT shared_network_family_46
                     CHECK (inet_family = 4 or inet_family = 6);

             -- Create an index to select shared networks by family and also by family
             -- and shared network name.
             CREATE INDEX shared_network_family_name_idx ON shared_network(inet_family, name);

             -- Create trigger function which verifies that the subnet prefix family matches
             -- the shared network's family if the subnet belongs to a shared network.
             CREATE OR REPLACE FUNCTION match_subnet_network_family()
               RETURNS trigger
               LANGUAGE plpgsql
               AS $function$
             DECLARE
               net_family int;
             BEGIN
               IF NEW.shared_network_id IS NOT NULL THEN
                   net_family := (SELECT inet_family FROM shared_network WHERE id = NEW.shared_network_id);
                   IF net_family != family(NEW.prefix) THEN
                       RAISE EXCEPTION 'Family of the subnet % is not matching the shared network IPv% family', NEW.prefix, net_family;
                   END IF;
               END IF;
               RETURN NEW;
             END;
             $function$;

             -- Validate the subnet prefix family against shared network family on insert or update.
             CREATE TRIGGER subnet_network_family_check
             BEFORE INSERT OR UPDATE ON subnet
               FOR EACH ROW EXECUTE PROCEDURE match_subnet_network_family();
           `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
             DROP TRIGGER IF EXISTS subnet_network_family_check ON subnet;
             DROP FUNCTION IF EXISTS match_subnet_network_family;
             DROP INDEX IF EXISTS shared_network_family_name_idx;
             ALTER TABLE shared_network
                 DROP CONSTRAINT IF EXISTS shared_network_family_46;
             ALTER TABLE shared_network
                 DROP COLUMN IF EXISTS inet_family;
        `)
		return err
	})
}
