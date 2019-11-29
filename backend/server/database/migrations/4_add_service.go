package dbmigs

import (
	"github.com/go-pg/migrations/v7"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
	     	_, err := db.Exec(`
             -- Services table.
             CREATE TABLE public.service (
                 id                      SERIAL PRIMARY KEY,
	         created                 TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
	         deleted                 TIMESTAMP WITHOUT TIME ZONE,
                 machine_id              INTEGER REFERENCES public.machine(id) NOT NULL,
                 type                    VARCHAR(10) NOT NULL,
                 ctrl_port               INTEGER DEFAULT 0,
                 active                  BOOLEAN DEFAULT FALSE,
                 meta                    JSONB,
                 details                 JSONB,
                 UNIQUE (machine_id, ctrl_port)
             );

             -- Service should be deleted after creation.
             ALTER TABLE public.service
               ADD CONSTRAINT service_created_deleted_check CHECK (
                 (deleted > created)
             );
           `)
		return err

	}, func(db migrations.DB) error {
		_, err := db.Exec(`
             -- Remove table with services.
             DROP TABLE public.service;
           `)
		return err
	})
}
