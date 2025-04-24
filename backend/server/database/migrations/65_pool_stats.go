package dbmigs

import "github.com/go-pg/migrations/v8"

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			-- Add new columns to store statistics for address and prefix pools.
			ALTER TABLE public.address_pool
				ADD COLUMN stats JSONB;
			ALTER TABLE public.address_pool
				ADD COLUMN stats_collected_at TIMESTAMP WITHOUT TIME ZONE;
			ALTER TABLE public.prefix_pool
				ADD COLUMN stats JSONB;
			ALTER TABLE public.prefix_pool
				ADD COLUMN stats_collected_at TIMESTAMP WITHOUT TIME ZONE;

			-- Add new columns to store utilizations.
			ALTER TABLE public.address_pool
				ADD COLUMN utilization SMALLINT DEFAULT 0;
			ALTER TABLE public.prefix_pool
				ADD COLUMN utilization SMALLINT DEFAULT 0;

			-- Make sure that when stats are stored then timestamp is also set.
			ALTER TABLE address_pool
				ADD CONSTRAINT stats_and_stats_collected_at_both_not_null CHECK (
					(stats IS NOT NULL AND stats_collected_at IS NOT NULL)
					OR
					(stats IS NULL AND stats_collected_at IS NULL)
				);
			ALTER TABLE prefix_pool
				ADD CONSTRAINT stats_and_stats_collected_at_both_not_null CHECK (
					(stats IS NOT NULL AND stats_collected_at IS NOT NULL)
					OR
					(stats IS NULL AND stats_collected_at IS NULL)
             	);
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			ALTER TABLE public.address_pool
				DROP CONSTRAINT IF EXISTS stats_and_stats_collected_at_both_not_null;
			ALTER TABLE public.prefix_pool
				DROP CONSTRAINT IF EXISTS stats_and_stats_collected_at_both_not_null;

			ALTER TABLE public.address_pool
				DROP COLUMN IF EXISTS utilization;
			ALTER TABLE public.prefix_pool
				DROP COLUMN IF EXISTS utilization;

			ALTER TABLE public.address_pool
				DROP COLUMN IF EXISTS stats;
			ALTER TABLE public.address_pool
				DROP COLUMN IF EXISTS stats_collected_at;
			ALTER TABLE public.prefix_pool
				DROP COLUMN IF EXISTS stats;
			ALTER TABLE public.prefix_pool
				DROP COLUMN IF EXISTS stats_collected_at;
		`)
		return err
	})
}
