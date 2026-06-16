package dbmigs

import "github.com/go-pg/migrations/v8"

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			CREATE TABLE IF NOT EXISTS public.zone_transfer_state (
				id BIGSERIAL NOT NULL,
				daemon_id BIGINT NOT NULL,
				created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT timezone('utc'::text, now()),
				view_name TEXT NOT NULL,
				zone_name TEXT NOT NULL,
				serial BIGINT NULL,
				client TEXT NOT NULL,
				server TEXT NULL,
				messages_count BIGINT NULL,
				records_count BIGINT NULL,
				bytes_count BIGINT NULL,
				duration BIGINT NULL,
				status INTEGER NOT NULL,
				start_time TIMESTAMP WITHOUT TIME ZONE NOT NULL,
				completion_time TIMESTAMP WITHOUT TIME ZONE NULL,
				message TEXT NULL,
				CONSTRAINT zone_transfer_state_pkey PRIMARY KEY (id),
				CONSTRAINT zone_transfer_state_daemon_id_view_name_zone_name_client_start_time_unique
					UNIQUE (daemon_id, view_name, zone_name, client, start_time),
				CONSTRAINT zone_transfer_state_daemon_id_fkey FOREIGN KEY (daemon_id)
					REFERENCES public.daemon (id) MATCH SIMPLE
					ON UPDATE CASCADE
					ON DELETE CASCADE
			);
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			DROP TABLE IF EXISTS public.zone_transfer_state;
		`)
		return err
	})
}
