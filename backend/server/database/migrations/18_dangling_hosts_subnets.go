package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
             -- This trigger function is invoked upon deletion of an association between
             -- apps and a host. It removes a host if it has no more
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

             -- This trigger removes a host that no longer has associations
             -- with any apps.
             DO $$ BEGIN
                 CREATE TRIGGER trigger_wipe_dangling_host
                     AFTER DELETE ON local_host
                         FOR EACH ROW EXECUTE PROCEDURE wipe_dangling_host();
             EXCEPTION
                 WHEN duplicate_object THEN null;
             END $$;

             -- This trigger function is invoked upon deletion of an association between
             -- apps and a subnet. It removes a subnet if it has no more
             -- associations with any app.
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

             -- This trigger removes a subnet that no longer has associations
             -- with any apps.
             DO $$ BEGIN
                 CREATE TRIGGER trigger_wipe_dangling_subnet
                     AFTER DELETE ON local_subnet
                         FOR EACH ROW EXECUTE PROCEDURE wipe_dangling_subnet();
             EXCEPTION
                 WHEN duplicate_object THEN null;
             END $$;

             -- This sequence returns numbers used during bulk updates of data within
             -- the database.
             CREATE SEQUENCE IF NOT EXISTS bulk_update_seq;
             SELECT nextval('bulk_update_seq');

             -- This column holds an update sequence value. When Stork fetches many
             -- hosts via the host_cmds hook library it sets the same value for all
             -- updated hosts, making it possible to determine which of the values have not
             -- been updated. Such records can be later removed. The value set for
             -- the updated hosts is taken from the bulk_update_seq.
             ALTER TABLE local_host ADD COLUMN
                 update_seq BIGINT NOT NULL DEFAULT currval('bulk_update_seq');

             -- This index queries hosts by a sequence number assigned to the
             -- local hosts. Using the sequence number, it is possible to
             -- delete those associations between apps and hosts which no
             -- longer exist. In this case, all associations for which the
             -- sequence number is unequal to the sequence number of the last
             -- update are deleted. To make it efficient, an index is required
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
