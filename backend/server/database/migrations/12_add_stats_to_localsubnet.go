package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
             -- last time when stats were collected
             ALTER TABLE public.local_subnet
               ADD COLUMN stats_collected_at TIMESTAMP WITHOUT TIME ZONE;

             -- stats collected from dhcp
             ALTER TABLE public.local_subnet
               ADD COLUMN stats JSONB;

             -- Make sure that when stats are stored in local_subnet then timestamp stats_collected_at is also set.
             ALTER TABLE local_subnet
               ADD CONSTRAINT stats_and_stats_collected_at_both_not_null CHECK (
                 (stats IS NOT NULL AND stats_collected_at IS NOT NULL)
                 OR
                 (stats IS NULL AND stats_collected_at IS NULL)
             );
           `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
             ALTER TABLE public.local_subnet
               DROP CONSTRAINT IF EXISTS stats_and_stats_collected_at_both_not_null;
             ALTER TABLE public.local_subnet
               DROP COLUMN stats;
             ALTER TABLE public.local_subnet
               DROP COLUMN stats_collected_at;
           `)
		return err
	})
}
