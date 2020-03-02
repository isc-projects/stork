package dbmigs

import (
	"github.com/go-pg/migrations/v7"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
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
             ALTER TABLE app DROP CONSTRAINT IF EXISTS app_created_deleted_check;
        `)
		return err
	})
}
