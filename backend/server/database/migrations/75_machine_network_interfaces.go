package dbmigs

import "github.com/go-pg/migrations/v8"

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			-- This table holds the list of interfaces detected on the
			-- monitored machines. The interface has one-to-many relationship
			-- with the machine_network_interface_ip_address table.
			CREATE TABLE IF NOT EXISTS public.machine_network_interface (
				id BIGSERIAL NOT NULL,
				machine_id BIGINT NOT NULL,
				name TEXT NOT NULL,
				flags INT NOT NULL DEFAULT 0,
				hardware_address BYTEA NULL,
				CONSTRAINT machine_network_interface_pkey PRIMARY KEY (id),
				CONSTRAINT machine_network_interface_machine_id_name_unique UNIQUE (machine_id, name),
				CONSTRAINT machine_network_interface_machine_id_fkey FOREIGN KEY (machine_id)
					REFERENCES public.machine (id) MATCH SIMPLE
					ON UPDATE CASCADE
					ON DELETE CASCADE
			);

			-- This table holds the list of IP addresses assigned to one of the network
			-- interfaces detected on a monitored machine. The IP address has one-to-one
			-- relationship with the machine_network_interface table.
			CREATE TABLE IF NOT EXISTS public.machine_network_interface_ip_address (
				machine_network_interface_id BIGINT NOT NULL,
				ip_address INET NOT NULL,
				PRIMARY KEY (machine_network_interface_id, ip_address),
				CONSTRAINT machine_network_interface_ip_address_interface_id_fkey FOREIGN KEY (machine_network_interface_id)
					REFERENCES public.machine_network_interface (id) MATCH SIMPLE
					ON UPDATE CASCADE
					ON DELETE CASCADE
			);

			-- Create index on IP address to speed up queries for machines by IP address.
			CREATE INDEX IF NOT EXISTS machine_network_interface_ip_address_idx
				ON public.machine_network_interface_ip_address (ip_address);
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			DROP INDEX IF EXISTS machine_network_interface_ip_address_idx;
			DROP TABLE IF EXISTS public.machine_network_interface_ip_address;
			DROP TABLE IF EXISTS public.machine_network_interface;
		`)
		return err
	})
}
