package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
            -- This removes the no-longer-used update_seq column.
            ALTER TABLE local_host DROP COLUMN update_seq;

            -- This removes the unused sequence.
            DROP SEQUENCE IF EXISTS bulk_update_seq;
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
             CREATE SEQUENCE IF NOT EXISTS bulk_update_seq;
             SELECT nextval('bulk_update_seq');

             ALTER TABLE local_host ADD COLUMN
                 update_seq BIGINT NOT NULL DEFAULT currval('bulk_update_seq');

             CREATE INDEX host_update_seq_idx ON local_host(update_seq);
        `)
		return err
	})
}
