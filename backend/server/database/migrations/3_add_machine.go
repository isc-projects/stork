package dbmigs

import (
	"github.com/go-pg/migrations/v7"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
	     	_, err := db.Exec(
	     		`-- Machines table.
             CREATE TABLE public.machine (
                 id                      SERIAL PRIMARY KEY,
	         created                 TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
	         deleted                 TIMESTAMP WITHOUT TIME ZONE,
                 address                 VARCHAR(255),
                 state                   JSONB,
	         last_visited            TIMESTAMP WITHOUT TIME ZONE,
	         error                   VARCHAR(255),
                 UNIQUE (address)
             );

             -- Machine should be deleted after creation.
             ALTER TABLE public.machine
               ADD CONSTRAINT machine_created_deleted_check CHECK (
                 (deleted > created)
             );

           `)
		return err

	}, func(db migrations.DB) error {
		_, err := db.Exec(
			`-- Remove table with machines.
             DROP TABLE public.machine;`)
		return err
	})
}
