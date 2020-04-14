package dbmigs

import (
	"github.com/go-pg/migrations/v7"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
             -- Add utilization column to subnet.
             ALTER TABLE subnet ADD COLUMN adr_utilization SMALLINT;
             ALTER TABLE subnet ADD COLUMN pds_utilization SMALLINT;

             -- Add utilizations column to shared_network.
             ALTER TABLE shared_network ADD COLUMN adr_utilization SMALLINT;
             ALTER TABLE shared_network ADD COLUMN pds_utilization SMALLINT;

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
             -- Remove utilization column from subnet.
             ALTER TABLE subnet DROP COLUMN adr_utilization;
             ALTER TABLE subnet DROP COLUMN pds_utilization;

             -- Remove utilizations column from shared_network.
             ALTER TABLE shared_network DROP COLUMN adr_utilization;
             ALTER TABLE shared_network DROP COLUMN pds_utilization;

             -- Drop a table with global stats.
             DROP TABLE statistic;
        `)
		return err
	})
}
