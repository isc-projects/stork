package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
             -- Add utilization column to subnet.
             ALTER TABLE subnet ADD COLUMN addr_utilization SMALLINT;
             ALTER TABLE subnet ADD COLUMN pd_utilization SMALLINT;

             -- Add utilizations column to shared_network.
             ALTER TABLE shared_network ADD COLUMN addr_utilization SMALLINT;
             ALTER TABLE shared_network ADD COLUMN pd_utilization SMALLINT;

             -- Create a table with global stats.
             CREATE TABLE statistic (
                 name TEXT NOT NULL,
                 value BIGINT,
                 CONSTRAINT statistic_pkey PRIMARY KEY (name)
             );
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
             -- Drop a table with global stats.
             DROP TABLE statistic;

             -- Remove utilizations column from shared_network.
             ALTER TABLE shared_network DROP COLUMN pd_utilization;
             ALTER TABLE shared_network DROP COLUMN addr_utilization;

             -- Remove utilization column from subnet.
             ALTER TABLE subnet DROP COLUMN pd_utilization;
             ALTER TABLE subnet DROP COLUMN addr_utilization;

        `)
		return err
	})
}
