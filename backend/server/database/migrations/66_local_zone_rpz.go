package dbmigs

import "github.com/go-pg/migrations/v8"

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			-- Add new column to store the flag indicating if the zone is a
			-- response policy zone.
			ALTER TABLE public.local_zone
				ADD COLUMN rpz BOOLEAN DEFAULT FALSE;
			CREATE INDEX local_zone_rpz_idx ON local_zone(rpz);
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			ALTER TABLE public.local_zone
				DROP COLUMN IF EXISTS rpz;
		`)
		return err
	})
}
