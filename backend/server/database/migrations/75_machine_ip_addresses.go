package dbmigs

import "github.com/go-pg/migrations/v8"

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			-- This table holds the list of IP addresses detected on the
			-- monitored machines.
			CREATE TABLE IF NOT EXISTS public.machine_ip_address (
				id BIGSERIAL NOT NULL,
				machine_id BIGINT NOT NULL,
				ip_address TEXT NOT NULL,
				CONSTRAINT machine_ip_address_pkey PRIMARY KEY (id),
				CONSTRAINT machine_ip_address_machine_id_ip_address_unique UNIQUE (machine_id, ip_address),
				CONSTRAINT machine_ip_address_machine_id_fkey FOREIGN KEY (machine_id)
					REFERENCES public.machine (id) MATCH SIMPLE
						ON UPDATE CASCADE
						ON DELETE CASCADE
			);

		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			DROP TABLE IF EXISTS public.machine_ip_address;
		`)
		return err
	})
}
