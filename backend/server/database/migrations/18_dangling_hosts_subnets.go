package dbmigs

import (
	"github.com/go-pg/migrations/v7"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
             -- Trigger function invoked upon deletion of an association between
             -- apps and a host. It removes a host if this host has no more
             -- associations with any app/
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

             -- Trigger which removes a host which no longer has any associations
             -- with apps.
             DO $$ BEGIN
                 CREATE TRIGGER trigger_wipe_dangling_host
                     AFTER DELETE ON local_host
                         FOR EACH ROW EXECUTE PROCEDURE wipe_dangling_host();
             EXCEPTION
                 WHEN duplicate_object THEN null;
             END $$;

             -- Trigger function invoked upon deletion of an association between
             -- apps and a subnet. It removes a subnet if this subnet has no more
             -- associations with any app/
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

             -- Trigger which removes a subnet which no longer has any associations
             -- with apps.
             DO $$ BEGIN
                 CREATE TRIGGER trigger_wipe_dangling_subnet
                     AFTER DELETE ON local_subnet
                         FOR EACH ROW EXECUTE PROCEDURE wipe_dangling_subnet();
             EXCEPTION
                 WHEN duplicate_object THEN null;
             END $$;
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
               DROP TRIGGER IF EXISTS trigger_wipe_dangling_subnet ON local_subnet;
               DROP FUNCTION IF EXISTS wipe_dangling_subnet;
               DROP TRIGGER IF EXISTS trigger_wipe_dangling_host ON local_host;
               DROP FUNCTION IF EXISTS wipe_dangling_host;
        `)
		return err
	})
}
