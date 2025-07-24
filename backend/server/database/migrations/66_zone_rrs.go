package dbmigs

import "github.com/go-pg/migrations/v8"

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			-- Remember time when the RRs for a zone were fetched from the DNS server
			-- using AXFR.
			ALTER TABLE local_zone
				ADD COLUMN zone_transfer_at TIMESTAMP WITHOUT TIME ZONE;

			-- Holds a collection of RRs for a zone. It is used to cache the RRs
			-- received from the DNS server over AXFR. Subsequent requests to view
			-- the RRs can retrieve this data from the database rather than running
			-- AXFR again. RR data frequently used for filtering (i.e., name, type)
			-- have suitable indexes. Indexing RDATA is troublesome because it
			-- contains the data specific to the RR type. We could consider pg_trgm
			-- extension for indexing RDATA. It, however, requires that this extension
			-- is installed on the database server. That would be one more dependency.
			CREATE TABLE IF NOT EXISTS local_zone_rr (
				id BIGSERIAL NOT NULL,
				local_zone_id BIGINT NOT NULL,
				name TEXT NOT NULL,
				ttl BIGINT NOT NULL,
				class TEXT NOT NULL,
				type TEXT NOT NULL,
				rdata TEXT NOT NULL,
				CONSTRAINT local_zone_rr_pkey PRIMARY KEY (id),
				CONSTRAINT local_zone_rr_local_zone_id FOREIGN KEY (local_zone_id)
				REFERENCES local_zone (id) MATCH SIMPLE
					ON UPDATE CASCADE
					ON DELETE CASCADE
			);
			CREATE INDEX local_zone_rr_local_zone_id_idx ON local_zone_rr(local_zone_id);
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			DROP TABLE IF EXISTS local_zone_rr;
			ALTER TABLE local_zone
				DROP COLUMN zone_transfer_at;
		`)
		return err
	})
}
