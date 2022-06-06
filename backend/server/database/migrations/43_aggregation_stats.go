package dbmigs

import "github.com/go-pg/migrations/v8"

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
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
        `)
		return err
	})
}
