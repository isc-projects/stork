package dbmigs

import "github.com/go-pg/migrations/v8"

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			-- Add a new column to store out-of-pool utilization for address
			-- and prefix pools in the subnet and shared network tables.
			ALTER TABLE public.subnet
				ADD COLUMN out_of_pool_addr_utilization SMALLINT,
				ADD COLUMN out_of_pool_pd_utilization SMALLINT;
			ALTER TABLE public.shared_network
				ADD COLUMN out_of_pool_addr_utilization SMALLINT,
				ADD COLUMN out_of_pool_pd_utilization SMALLINT;
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			-- Remove the out-of-pool utilization columns from the subnet and
			-- shared_network tables.
			ALTER TABLE public.subnet
				DROP COLUMN out_of_pool_addr_utilization,
				DROP COLUMN out_of_pool_pd_utilization;
			ALTER TABLE public.shared_network
				DROP COLUMN out_of_pool_addr_utilization,
				DROP COLUMN out_of_pool_pd_utilization;
		`)
		return err
	})
}
