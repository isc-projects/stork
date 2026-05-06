package dbmigs

import "github.com/go-pg/migrations/v8"

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			-- Add a new column to store the server tag from the Kea config.
			ALTER TABLE public.kea_daemon
				ADD COLUMN server_tag TEXT;

			-- Backfill server_tag from the config.
			UPDATE public.kea_daemon
				SET server_tag = COALESCE(
					config->'Dhcp4'->>'server-tag',
					config->'Dhcp6'->>'server-tag'
				)
				WHERE config IS NOT NULL;
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			ALTER TABLE public.kea_daemon
				DROP COLUMN IF EXISTS server_tag;
		`)
		return err
	})
}
