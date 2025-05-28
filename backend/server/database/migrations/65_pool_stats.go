package dbmigs

import "github.com/go-pg/migrations/v8"

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			-- Add new columns to store statistics and utilizations for address
			-- and prefix pools.
			-- Make sure that when stats are stored then timestamp is also set.
			ALTER TABLE public.address_pool
				ADD COLUMN stats JSONB,
				ADD COLUMN stats_collected_at TIMESTAMP WITHOUT TIME ZONE,
				ADD COLUMN utilization SMALLINT DEFAULT 0,
				ADD CONSTRAINT stats_and_stats_collected_at_both_not_null CHECK (
					(stats IS NULL) = (stats_collected_at IS NULL)
				);
			ALTER TABLE public.prefix_pool
				ADD COLUMN stats JSONB,
				ADD COLUMN stats_collected_at TIMESTAMP WITHOUT TIME ZONE,
				ADD COLUMN utilization SMALLINT DEFAULT 0,
				ADD CONSTRAINT stats_and_stats_collected_at_both_not_null CHECK (
					(stats IS NULL) = (stats_collected_at IS NULL)
				);
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			ALTER TABLE public.address_pool
				DROP CONSTRAINT IF EXISTS stats_and_stats_collected_at_both_not_null,
				DROP COLUMN IF EXISTS utilization,
				DROP COLUMN IF EXISTS stats,
				DROP COLUMN IF EXISTS stats_collected_at;
			ALTER TABLE public.prefix_pool
				DROP CONSTRAINT IF EXISTS stats_and_stats_collected_at_both_not_null,
				DROP COLUMN IF EXISTS utilization,
				DROP COLUMN IF EXISTS stats,
				DROP COLUMN IF EXISTS stats_collected_at;
		`)
		return err
	})
}
