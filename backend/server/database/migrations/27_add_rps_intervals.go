package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(
			`-- RpsIntervals table.
             CREATE TABLE IF NOT EXISTS public.rps_interval (
                kea_daemon_id   bigint NOT NULL,
                start_time      TIMESTAMP WITHOUT TIME ZONE NOT NULL,
                duration        bigint,
                responses       bigint,
                CONSTRAINT rps_intervals_pkey PRIMARY KEY (kea_daemon_id, start_time)
             );
           `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(
			`-- Remove table on error
             DROP TABLE IF EXISTS public.rps_interval;`)
		return err
	})
}
