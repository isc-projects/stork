package dbmigs

import (
	"github.com/go-pg/migrations/v7"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
               DROP TRIGGER IF EXISTS trigger_wipe_dangling_subnet ON local_subnet;
               DROP FUNCTION IF EXISTS wipe_dangling_subnet;
               DROP TRIGGER IF EXISTS trigger_wipe_dangling_host ON local_host;
               DROP FUNCTION IF EXISTS wipe_dangling_host;
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
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
        `)
		return err
	})
}
