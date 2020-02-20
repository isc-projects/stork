package dbmigs

import (
	"github.com/go-pg/migrations/v7"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
             -- Access point type.
             CREATE TYPE ACCESSPOINTTYPE AS ENUM
                 ('control', 'statistics');

             -- This table holds application's access points.
             CREATE TABLE IF NOT EXISTS access_point (
                 app_id BIGINT NOT NULL,
                 machine_id BIGINT NOT NULL,
                 type ACCESSPOINTTYPE,
                 address TEXT DEFAULT 'localhost',
                 port INTEGER DEFAULT 0,
                 key TEXT DEFAULT '',
                 CONSTRAINT access_point_pkey PRIMARY KEY (app_id, type),
                 CONSTRAINT access_point_unique_idx UNIQUE (machine_id, port),
                 CONSTRAINT access_point_app_id FOREIGN KEY (app_id)
                     REFERENCES app (id) MATCH SIMPLE
                     ON UPDATE NO ACTION
                     ON DELETE CASCADE,
                 CONSTRAINT access_point_machine_id FOREIGN KEY (machine_id)
                     REFERENCES machine (id) MATCH SIMPLE
                     ON UPDATE NO ACTION
                     ON DELETE CASCADE
              );

              -- Migrate the data.
              INSERT INTO access_point (app_id, machine_id, type, address, port, key)
              SELECT      id, machine_id, 'control', ctrl_address, ctrl_port, ctrl_key
              FROM        app
              WHERE       NOT EXISTS (
                  SELECT 1 FROM access_point
                  WHERE  access_point.app_id = app.id
                  AND    access_point.machine_id = app.machine_id
                  AND    access_point.port = app.ctrl_port
              );

              ALTER TABLE app
                  DROP CONSTRAINT app_machine_id_ctrl_port_key;
           `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
               DROP TABLE IF EXISTS access_point;
               DROP TYPE IF EXISTS ACCESSPOINTTYPE;
           `)
		return err
	})
}
