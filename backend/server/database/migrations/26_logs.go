package dbmigs

import (
	"github.com/go-pg/migrations/v7"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(
			`-- Create table holding logging targets for daemons. The logging
             -- target specifies where the logs are stored by the daemon.
             CREATE TABLE IF NOT EXISTS log_target (
                 id bigserial NOT NULL PRIMARY KEY,
                 daemon_id bigint NOT NULL,
	             created_at timestamp without time zone NOT NULL DEFAULT timezone('utc'::text, now()),
                 name text,
                 output text NOT NULL,
                 CONSTRAINT log_target_daemon_id_fkey FOREIGN KEY (daemon_id)
                     REFERENCES daemon (id) MATCH SIMPLE
                     ON UPDATE CASCADE
                     ON DELETE CASCADE
             );
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(
			`-- Remove log_taget table.
             DROP TABLE IF EXISTS log_target;`)
		return err
	})
}
