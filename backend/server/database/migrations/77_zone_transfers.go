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
				bytes_per_second BIGINT NULL,
				duration BIGINT NULL,
				status TEXT NOT NULL,
				started_at TIMESTAMP WITHOUT TIME ZONE NOT NULL,
				completed_at TIMESTAMP WITHOUT TIME ZONE NULL,
				message TEXT NULL,
				local BOOLEAN NOT NULL DEFAULT FALSE,
				client_machine_id BIGINT NULL,
				server_machine_id BIGINT NULL,
				CONSTRAINT zone_transfer_state_pkey PRIMARY KEY (id),
				CONSTRAINT zone_transfer_state_daemon_id_view_name_zone_name_client_start_time_unique
					UNIQUE (daemon_id, view_name, zone_name, client, started_at),
				CONSTRAINT zone_transfer_state_daemon_id_fkey FOREIGN KEY (daemon_id)
					REFERENCES public.daemon (id) MATCH SIMPLE
					ON UPDATE CASCADE
					ON DELETE CASCADE,
				CONSTRAINT zone_transfer_state_status_check CHECK (
					status IN ('started', 'completed', 'up-to-date', 'failed', 'message')
				)
			);

			-- Create an index on the status column to speed up queries by status.
			CREATE INDEX IF NOT EXISTS zone_transfer_state_status_idx
				ON public.zone_transfer_state USING btree (status);

			-- Create an index on the client machine ID column to speed up queries
			-- by client machine ID.
			CREATE INDEX IF NOT EXISTS zone_transfer_state_client_machine_id_idx
				ON public.zone_transfer_state USING btree (client_machine_id);

			-- Create an index on the server machine ID column to speed up queries
			-- by server machine ID.
			CREATE INDEX IF NOT EXISTS zone_transfer_state_server_machine_id_idx
				ON public.zone_transfer_state USING btree (server_machine_id);

		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			DROP INDEX IF EXISTS zone_transfer_state_server_machine_id_idx;
			DROP INDEX IF EXISTS zone_transfer_state_client_machine_id_idx;
			DROP INDEX IF EXISTS zone_transfer_state_status_idx;
			DROP TABLE IF EXISTS public.zone_transfer_state;
		`)
		return err
	})
}
