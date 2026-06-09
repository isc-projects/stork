package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(
			`-- Drop old (incorrect) constraints.
			ALTER TABLE public.lease DROP CONSTRAINT lease_ip_mac_daemon_unique;
			ALTER TABLE public.lease DROP CONSTRAINT lease_ip_duid_daemon_unique;
			ALTER TABLE public.lease DROP CONSTRAINT lease_ip_clid_daemon_unique;
			-- Create new constraint. Leases should only be unique by IP address,
			-- and the previous constraints allowed one IP to be recorded as leased
			-- to both a hardware address and a client ID simultaneously.
			ALTER TABLE public.lease ADD CONSTRAINT lease_ip_daemon_unique UNIQUE (ip_address, daemon_id);
			`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(
			`-- Remove new constraint.
			ALTER TABLE public.lease DROP CONSTRAINT lease_ip_daemon_unique;
			ALTER TABLE public.lease ADD CONSTRAINT lease_ip_mac_daemon_unique UNIQUE (ip_address, hw_address, daemon_id);
			ALTER TABLE public.lease ADD CONSTRAINT lease_ip_duid_daemon_unique UNIQUE (ip_address, duid, daemon_id);
			ALTER TABLE public.lease ADD CONSTRAINT lease_ip_clid_daemon_unique UNIQUE (ip_address, client_id, daemon_id);
			`)
		return err
	})
}
