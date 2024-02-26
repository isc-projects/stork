package dbmigs

import "github.com/go-pg/migrations/v8"

// The migration changes the primary key of the local_host table to be
// single-column key, replaces the ip_reservation table's reference to host
// table with reference to local_host, and moves the hostname column from host
// to local_host table.
func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
				-- Modify the local_host table.

				-- Add a new column for the primary key of the local_host table.
				ALTER TABLE local_host ADD COLUMN id bigserial NOT NULL;

				-- Drop the old primary key constraint.
				ALTER TABLE local_host DROP CONSTRAINT local_host_pkey;

				-- Add a new primary key constraint.
				ALTER TABLE local_host ADD CONSTRAINT local_host_pkey PRIMARY KEY (id);

				-- Add a unique constraint for the previous primary key columns
				-- (host_id, daemon_id, and data_source).
				ALTER TABLE local_host
					ADD CONSTRAINT local_host_host_id_daemon_id_data_source_unique_idx
						UNIQUE (host_id, daemon_id, data_source);

				-- Modify the ip_reservation table.

				-- Drop the foreign key constraint from the ip_reservation table.
				ALTER TABLE ip_reservation DROP CONSTRAINT ip_reservation_host_fkey;

				-- Add a new column for the local_host reference.
				ALTER TABLE ip_reservation ADD COLUMN local_host_id bigint;

				-- Drop the existing unique index.
				ALTER TABLE ip_reservation
					DROP CONSTRAINT ip_reservation_host_id_address_unique_idx;

				-- Fill the new column with the local_host references. We
				-- select a first local_host.id for each host_id.
				UPDATE ip_reservation
					SET local_host_id = lh.id
					FROM (
						SELECT DISTINCT ON (host_id) id, host_id
						FROM local_host
					) lh
					WHERE ip_reservation.host_id = lh.host_id;

				-- Duplicate the IP reservations for each daemon.
				INSERT INTO ip_reservation (created_at, address, host_id, local_host_id)
				SELECT ir2.created_at, ir2.address, ir2.host_id, lh.id 
				FROM local_host lh
				LEFT JOIN ip_reservation ir ON lh.id = ir.local_host_id
				LEFT JOIN local_host lh2 ON lh.host_id = lh2.host_id
				RIGHT JOIN ip_reservation ir2 ON lh2.id = ir2.local_host_id 
				WHERE ir.local_host_id IS NULL;

				-- Drop the eventually orphaned IP reservations (the
				-- reservations than not belong to any daemon).
				DELETE FROM ip_reservation WHERE local_host_id IS NULL;

				-- Add the not-null constraint to the new column.
				ALTER TABLE ip_reservation
					ALTER COLUMN local_host_id SET NOT NULL;

				-- Add a foreign key constraint to the local_host table.
				ALTER TABLE ip_reservation
					ADD CONSTRAINT ip_reservation_local_host_fkey
						FOREIGN KEY (local_host_id)
						REFERENCES local_host (id)
						ON UPDATE CASCADE
						ON DELETE CASCADE;

				-- Drop the host_id column.
				ALTER TABLE ip_reservation DROP COLUMN host_id;

				-- Re-create the unique index.
				ALTER TABLE ip_reservation
					ADD CONSTRAINT ip_reservation_local_host_id_address_unique_idx
						UNIQUE (local_host_id, address);

				-- Modify the host and local_host tables.

				-- Move the hostname column from the host table to the
				-- local_host table.
				ALTER TABLE local_host ADD COLUMN hostname TEXT;
				UPDATE local_host lh
					SET hostname = h.hostname
					FROM host h
					WHERE lh.host_id = h.id;
				ALTER TABLE host DROP COLUMN hostname;

				-- Reset the Kea config hashes to re-process the host
				-- reservations.
				UPDATE kea_daemon SET config_hash = NULL;
			`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
				-- Modify the host and local_host tables.

				-- Move the hostname column from the local_host table back to
				-- the host table.
				ALTER TABLE host ADD COLUMN hostname TEXT;
				UPDATE host h
					SET hostname = lh.hostname
					FROM local_host lh
					WHERE lh.host_id = h.id;
				ALTER TABLE local_host DROP COLUMN hostname;

				-- Modify the ip_reservation table.

				-- Add the host_id column back to the ip_reservation table.
				ALTER TABLE ip_reservation ADD COLUMN host_id bigint;

				-- Fill the new column with the host references. We select a
				-- first host_id for each local_host.
				UPDATE ip_reservation
					SET host_id = lh.host_id
					FROM local_host lh
					WHERE ip_reservation.local_host_id = lh.id;

				-- Drop the duplicated IP reservations for given address and
				-- host id.
				DELETE FROM ip_reservation d
				WHERE d.id IN (
					SELECT ir.id
					FROM (
						SELECT DISTINCT ON (host_id, address) id, host_id, address
						FROM ip_reservation 
					) keep
					JOIN ip_reservation ir ON ir.address = keep.address AND ir.host_id = keep.host_id
					WHERE ir.id != keep.id
				);

				-- Set the not-null constraint to the new column.
				ALTER TABLE ip_reservation ALTER COLUMN host_id SET NOT NULL;

				-- Add a foreign key constraint to the host table.
				ALTER TABLE ip_reservation
					ADD CONSTRAINT ip_reservation_host_fkey
						FOREIGN KEY (host_id)
						REFERENCES host (id)
						ON UPDATE CASCADE
						ON DELETE CASCADE;

				-- Drop the local_host_id column.
				ALTER TABLE ip_reservation DROP COLUMN local_host_id;

				-- Re-create the unique index.
				ALTER TABLE ip_reservation
					ADD CONSTRAINT ip_reservation_host_id_address_unique_idx
						UNIQUE (host_id, address);

				-- Modify the local host table.

				-- Drop the unique constraint for the previous primary key
				-- columns (host_id, daemon_id, and data_source).
				ALTER TABLE local_host
					DROP CONSTRAINT local_host_host_id_daemon_id_data_source_unique_idx;

				-- Drop the single-column primary key constraint.
				ALTER TABLE local_host DROP CONSTRAINT local_host_pkey;

				-- Drop the new column for the primary key of the local_host
				-- table.
				ALTER TABLE local_host DROP COLUMN id;

				-- Add the old primary key constraint back.
				ALTER TABLE local_host
					ADD CONSTRAINT local_host_pkey
						PRIMARY KEY (host_id, daemon_id, data_source);

				-- Reset the Kea config hashes to re-process the host
				-- reservations.
				UPDATE kea_daemon SET config_hash = NULL;
			`)
		return err
	})
}
