package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
            -- This adds a missing foreign-key-to-host table.
            ALTER TABLE local_host
                ADD CONSTRAINT local_host_to_host_id FOREIGN KEY (host_id)
                    REFERENCES host (id) MATCH SIMPLE
                        ON UPDATE CASCADE
                        ON DELETE CASCADE;

             -- The subnets or hosts that are not associated with any app should
             -- no longer be automatically deleted. Such subnets and hosts can
             -- be explicitly deleted by the Stork Server.
             DROP TRIGGER IF EXISTS trigger_wipe_dangling_subnet ON local_subnet;
             DROP FUNCTION IF EXISTS wipe_dangling_subnet;
             DROP TRIGGER IF EXISTS trigger_wipe_dangling_host ON local_host;
             DROP FUNCTION IF EXISTS wipe_dangling_host;

             -- This deletes subnet, host, and shared_network entries from the database.
             -- A new column daemon_id to local_host and local_subnet needs
             -- to be added, but it is quite difficult to retrieve the daemon_id
             -- from the data currently held in the database. It is easier to delete
             -- the existing data and let the Stork Server fetch it again,
             -- ensuring the correctness of the new daemon_id value.
             DELETE FROM subnet;
             DELETE FROM host;
             DELETE FROM shared_network;

             -- This forces the Kea server to gather Kea server configurations and
             --  recreate subnets, hosts, and shared networks.
             UPDATE kea_daemon SET config_hash = NULL;

             -- This adds new columns that link the local_subnet and local_host entries
             -- with the daemons. Previously app_id was used, but it does not work
             -- in cases when local_subnet and/or local_host should be updated for
             -- one daemon but not for the other daemon belonging to the same app.
             ALTER TABLE local_subnet ADD COLUMN daemon_id BIGINT NOT NULL;
             ALTER TABLE local_host ADD COLUMN daemon_id BIGINT NOT NULL;

             -- This creates indexes on the new columns.
             CREATE INDEX local_subnet_daemon_id_idx ON local_subnet(daemon_id);
             CREATE INDEX local_host_daemon_id_idx ON local_host(daemon_id);

            -- This adds foreign keys for the new columns.
            ALTER TABLE local_subnet
                ADD CONSTRAINT local_subnet_to_daemon_id FOREIGN KEY (daemon_id)
                    REFERENCES daemon (id) MATCH SIMPLE
                        ON UPDATE CASCADE
                        ON DELETE CASCADE;
            ALTER TABLE local_host
                ADD CONSTRAINT local_host_to_daemon_id FOREIGN KEY (daemon_id)
                    REFERENCES daemon (id) MATCH SIMPLE
                        ON UPDATE CASCADE
                        ON DELETE CASCADE;
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
             ALTER TABLE local_host DROP COLUMN IF EXISTS daemon_id;
             ALTER TABLE local_subnet DROP COLUMN IF EXISTS daemon_id;

             CREATE OR REPLACE FUNCTION wipe_dangling_host()
                 RETURNS trigger
                 LANGUAGE 'plpgsql'
                 AS $function$
             BEGIN
                 DELETE FROM host
                     WHERE host.id = OLD.host_id AND NOT EXISTS (
                         SELECT FROM local_host AS lh
                             WHERE lh.host_id = host.id
                 );
                 RETURN NULL;
             END;
             $function$;

             DO $$ BEGIN
                 CREATE TRIGGER trigger_wipe_dangling_host
                     AFTER DELETE ON local_host
                         FOR EACH ROW EXECUTE PROCEDURE wipe_dangling_host();
             EXCEPTION
                 WHEN duplicate_object THEN null;
             END $$;

             CREATE OR REPLACE FUNCTION wipe_dangling_subnet()
                 RETURNS trigger
                 LANGUAGE 'plpgsql'
                 AS $function$
             BEGIN
                 DELETE FROM subnet
                     WHERE subnet.id = OLD.subnet_id AND NOT EXISTS (
                         SELECT FROM local_subnet AS ls
                             WHERE ls.subnet_id = subnet.id
                 );
                 RETURN NULL;
             END;
             $function$;

             DO $$ BEGIN
                 CREATE TRIGGER trigger_wipe_dangling_subnet
                     AFTER DELETE ON local_subnet
                         FOR EACH ROW EXECUTE PROCEDURE wipe_dangling_subnet();
             EXCEPTION
                 WHEN duplicate_object THEN null;
             END $$;

             ALTER TABLE local_host DROP CONSTRAINT IF EXISTS local_host_to_host_id;
        `)
		return err
	})
}
