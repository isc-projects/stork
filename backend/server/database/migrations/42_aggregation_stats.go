package dbmigs

import "github.com/go-pg/migrations/v8"

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
            -- Drop the utilization columns
			ALTER TABLE subnet DROP COLUMN addr_utilization;
			ALTER TABLE subnet DROP COLUMN pd_utilization;
			ALTER TABLE shared_network DROP COLUMN addr_utilization;
			ALTER TABLE shared_network DROP COLUMN pd_utilization;

			-- Add the statistic columns
			ALTER TABLE subnet ADD COLUMN stats JSONB;
			ALTER TABLE subnet ADD COLUMN stats_collected_at TIMESTAMP WITHOUT TIME ZONE;
			ALTER TABLE shared_network ADD COLUMN stats JSONB;
			ALTER TABLE shared_network ADD COLUMN stats_collected_at TIMESTAMP WITHOUT TIME ZONE;
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
            -- Drop the statistic columns
			ALTER TABLE subnet DROP COLUMN stats;
			ALTER TABLE subnet DROP COLUMN stats_collected_at;
			ALTER TABLE shared_network DROP COLUMN stats;
			ALTER TABLE shared_network DROP COLUMN stats_collected_at;

			-- Add the utilization columns
			ALTER TABLE subnet ADD COLUMN addr_utilization SMALLINT;
			ALTER TABLE subnet ADD COLUMN pd_utilization SMALLINT;
			ALTER TABLE shared_network ADD COLUMN addr_utilization SMALLINT;
			ALTER TABLE shared_network ADD COLUMN pd_utilization SMALLINT;
        `)
		return err
	})
}
