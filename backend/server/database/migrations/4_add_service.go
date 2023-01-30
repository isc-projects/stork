package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
             -- Apps table.
             CREATE TABLE public.app (
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

             -- App should be deleted after creation.
             ALTER TABLE public.app
               ADD CONSTRAINT app_created_deleted_check CHECK (
                 (deleted > created)
             );
           `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
             -- Removes table with apps.
             DROP TABLE public.app;
           `)
		return err
	})
}
