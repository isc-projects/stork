package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
             -- Access point type.
             DO $$ BEGIN
                 CREATE TYPE ACCESSPOINTTYPE AS ENUM
                     ('control', 'statistics');
             EXCEPTION
                 WHEN duplicate_object THEN null;
             END $$;

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

              -- Trigger function inserting control access point every time an
              -- app created.
              CREATE OR REPLACE FUNCTION update_machine_id()
                  RETURNS trigger
                  LANGUAGE 'plpgsql'
                  AS $function$
              BEGIN
                  UPDATE app SET machine_id = NEW.machine_id WHERE id = NEW.app_id;
                  RETURN NEW;
              END;
              $function$;

              -- Trigger to insert control access point when an app is created.
              DO $$ BEGIN
                  CREATE TRIGGER trigger_update_machine_id
                      BEFORE INSERT OR UPDATE ON access_point
                          FOR EACH ROW EXECUTE PROCEDURE update_machine_id();
              EXCEPTION
                  WHEN duplicate_object THEN null;
              END $$;

              -- Migrate existing data.
              INSERT INTO access_point (app_id, machine_id, type, address, port, key)
              SELECT      id, machine_id, 'control', ctrl_address, ctrl_port, ctrl_key
              FROM        app
              WHERE       NOT EXISTS (
                  SELECT 1 FROM access_point
                  WHERE  access_point.app_id = app.id
                  AND    access_point.machine_id = app.machine_id
                  AND    access_point.port = app.ctrl_port
              );

              -- Drop deprecated constraints and columns.
              ALTER TABLE app
                  DROP CONSTRAINT app_machine_id_ctrl_port_key;

              ALTER TABLE app
                  DROP COLUMN ctrl_address;

              ALTER TABLE app
                  DROP COLUMN ctrl_port;

              ALTER TABLE app
                  DROP COLUMN ctrl_key;

           `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`

               -- Restore columns.
               ALTER TABLE app
                   ADD COLUMN ctrl_address TEXT DEFAULT 'localhost';

               ALTER TABLE app
                   ADD COLUMN ctrl_port INTEGER DEFAULT 0;

               ALTER TABLE app
                   ADD COLUMN ctrl_key TEXT DEFAULT '';

               -- Restore data.
               UPDATE app SET (ctrl_address, ctrl_port, ctrl_key) =
                   (SELECT address, port, key
                    FROM   access_point
                    WHERE  access_point.app_id = app.id
                    AND    access_point.type = 'control');

               -- Restore CONSTRAINT app_machine_id_ctrl_port_key.
               ALTER TABLE app
                   ADD CONSTRAINT app_machine_id_ctrl_port_key UNIQUE (machine_id, ctrl_port);

               -- Drop function and trigger.
               DROP TRIGGER IF EXISTS trigger_update_machine_id ON access_point;
               DROP FUNCTION IF EXISTS update_machine_id;

               -- Drop created tables and types.
               DROP TABLE IF EXISTS access_point;
               DROP TYPE IF EXISTS ACCESSPOINTTYPE;
           `)
		return err
	})
}
