package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
             -- Trigger function invoked upon deletion of an association between
             -- apps and a host. It removes a host if this host has no more
             -- associations with any app.
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

             -- Sequence returning numbers used during bulk updates of data within
             -- the database.
             CREATE SEQUENCE IF NOT EXISTS bulk_update_seq;
             SELECT nextval('bulk_update_seq');

             -- This column holds an update sequence value. When Stork fetches many
             -- hosts via the host_cmds hooks library it sets the same value for all
             -- updated hosts. That allows for determination which of the values haven't
             -- been updated. Such records can be later removed. The value set for
             -- the updated hosts is taken from the bulk_update_seq.
             ALTER TABLE local_host ADD COLUMN
                 update_seq BIGINT NOT NULL DEFAULT currval('bulk_update_seq');

             -- Index for querying hosts by a sequence number assigned to the
             -- local hosts. Using the sequence number it is possible to
             -- delete those associations between apps and hosts which no
             -- longer exist. In this case we delete all associations for
             -- which the sequence number is unequal the sequence number of
             -- the last update. To make it efficient an index is required
             -- for sequence numbers.
             CREATE INDEX host_update_seq_idx ON local_host(update_seq);
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
               ALTER TABLE local_host DROP COLUMN update_seq;
               DROP SEQUENCE IF EXISTS bulk_update_seq;
               DROP TRIGGER IF EXISTS trigger_wipe_dangling_subnet ON local_subnet;
               DROP FUNCTION IF EXISTS wipe_dangling_subnet;
               DROP TRIGGER IF EXISTS trigger_wipe_dangling_host ON local_host;
               DROP FUNCTION IF EXISTS wipe_dangling_host;
        `)
		return err
	})
}
