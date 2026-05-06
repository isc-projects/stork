package dbmigs

import "github.com/go-pg/migrations/v8"

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			-- Add a new column to store the server tag from the Kea config.
			ALTER TABLE public.daemon
				ADD COLUMN server_tag TEXT NOT NULL DEFAULT '';
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			ALTER TABLE public.daemon
				DROP COLUMN IF EXISTS server_tag;
		`)
		return err
	})
}
