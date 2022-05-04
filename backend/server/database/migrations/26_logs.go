package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(
			`-- This creates a table holding logging targets for daemons. The logging
             -- target specifies where the logs are stored by the daemon.
             CREATE TABLE IF NOT EXISTS log_target (
                 id bigserial NOT NULL PRIMARY KEY,
                 daemon_id bigint NOT NULL,
                 created_at timestamp without time zone NOT NULL DEFAULT timezone('utc'::text, now()),
                 name text,
                 severity text,
                 output text NOT NULL,
                 CONSTRAINT log_target_daemon_id_fkey FOREIGN KEY (daemon_id)
                     REFERENCES daemon (id) MATCH SIMPLE
                     ON UPDATE CASCADE
                     ON DELETE CASCADE
             );

             -- This function converts logging severity to lowercase.
             CREATE OR REPLACE FUNCTION log_target_lower_severity()
                 RETURNS trigger
                 LANGUAGE  plpgsql
                 AS $function$
             BEGIN
                 NEW.severity = LOWER(NEW.severity);
                 RETURN NEW;
             END;
             $function$;

             -- This trigger is invoked before insert or update on the log_target table,
             -- which turns severity to lowercase.
             CREATE TRIGGER log_target_before_insert_update
             BEFORE INSERT OR UPDATE ON log_target
                 FOR EACH ROW EXECUTE PROCEDURE log_target_lower_severity();
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(
			`DROP TRIGGER IF EXISTS log_target_before_insert_update ON log_target;
             DROP FUNCTION IF EXISTS log_target_lower_severity;
             DROP TABLE IF EXISTS log_target;
        `)
		return err
	})
}
