package dbmigs

import (
	"github.com/go-pg/migrations/v7"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
             -- last time when stats were collected
             ALTER TABLE public.local_subnet
               ADD COLUMN stats_collected_at TIMESTAMP WITHOUT TIME ZONE;

             -- stats collected from dhcp
             ALTER TABLE public.local_subnet
               ADD COLUMN stats JSONB;
           `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
             ALTER TABLE public.local_subnet
               DROP COLUMN stats;
             ALTER TABLE public.local_subnet
               DROP COLUMN stats_collected_at;
           `)
		return err
	})
}
