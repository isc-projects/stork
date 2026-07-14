package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(
			`-- Lease table: add columns I forgot in the first go.
ALTER TABLE public.lease ADD COLUMN hw_address_source VARCHAR(255);
ALTER TABLE public.lease ADD COLUMN hw_type VARCHAR(255);
-- Lease Update table: ditto
ALTER TABLE public.lease_update ADD COLUMN hw_address_source VARCHAR(255);
ALTER TABLE public.lease_update ADD COLUMN hw_type VARCHAR(255);
			`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(
			`-- Lease table: remove columns that were not in the prior version of the schema.
ALTER TABLE public.lease DROP COLUMN hw_address_source;
ALTER TABLE public.lease DROP COLUMN hw_type;
-- Lease Update table: ditto
ALTER TABLE public.lease_update DROP COLUMN hw_address_source;
ALTER TABLE public.lease_update DROP COLUMN hw_type;
			`)
		return err
	})
}
