package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(
			`-- Events table.
             CREATE TABLE IF NOT EXISTS public.event (
                 id           SERIAL PRIMARY KEY,
	         created_at   TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
                 text         TEXT NOT NULL,
                 level        INTEGER NOT NULL,
                 relations    JSONB
             );
           `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(
			`-- Remove table with events.
             DROP TABLE IF EXISTS public.event;`)
		return err
	})
}
