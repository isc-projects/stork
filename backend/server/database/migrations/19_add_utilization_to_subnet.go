package dbmigs

import (
	"github.com/go-pg/migrations/v7"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
             -- Add utilization column to subnet.
             ALTER TABLE subnet ADD COLUMN utilization SMALLINT;
             ALTER TABLE subnet ADD COLUMN pds_utilization SMALLINT;

             -- Add utilizations column to shared_network.
             ALTER TABLE shared_network ADD COLUMN utilization SMALLINT;
             ALTER TABLE shared_network ADD COLUMN pds_utilization SMALLINT;

             -- Create a table with global stats.
             CREATE TABLE statistic (
                 name TEXT NOT NULL,
                 value BIGINT,
                 CONSTRAINT statistic_pkey PRIMARY KEY (name)
             );

             -- This table holds general information about daemon.
             CREATE TABLE base_daemon (
                 id BIGSERIAL NOT NULL PRIMARY KEY,
                 name TEXT NOT NULL,
	         created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
                 app_id INTEGER REFERENCES public.app(id) NOT NULL,
                 service_id INTEGER REFERENCES public.service(id) NOT NULL
             );

             -- This table includes a details about the Kea DHCP daemon.
             CREATE TABLE kea_dhcp_daemon (
                 id BIGSERIAL NOT NULL PRIMARY KEY,
                 service_id BIGINT NOT NULL,
                 ha_service_id BIGINT REFERENCES public.ha_service(id) NOT NULL,
                 lps_15_min INTEGER,
                 lps_24_h INTEGER,
                 utilization SMALLINT,
                 CONSTRAINT kea_dhcp_daemon_service_id FOREIGN KEY (service_id)
                     REFERENCES service (id) MATCH SIMPLE
                     ON UPDATE CASCADE
                     ON DELETE CASCADE
             );
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
             -- Remove utilization column from subnet.
             ALTER TABLE subnet DROP COLUMN utilization;
             ALTER TABLE subnet DROP COLUMN pds_utilization;

             -- Remove utilizations column from shared_network.
             ALTER TABLE shared_network DROP COLUMN utilization;
             ALTER TABLE shared_network DROP COLUMN pds_utilization;

             -- Drop a table with global stats.
             DROP TABLE statistic;

             -- Drop a table base_daemon.
             DROP TABLE base_daemon;

             -- Drop a table kea_dhcp_daemon.
             DROP TABLE kea_dhcp_daemon;
        `)
		return err
	})
}
