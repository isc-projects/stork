package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
             -- Make sure that the app is deleted when the machine is deleted.
             ALTER TABLE app DROP CONSTRAINT IF EXISTS app_machine_id_fkey;
             ALTER TABLE app ADD CONSTRAINT app_machine_id_fkey FOREIGN KEY (machine_id)
                 REFERENCES machine (id) MATCH SIMPLE
                     ON UPDATE CASCADE
                     ON DELETE CASCADE;

             -- The deleted column is no longer used on app and machine tables.
             -- We need to drop the constraints first, then we can delete the
             -- columns.
             ALTER TABLE app DROP CONSTRAINT IF EXISTS app_created_deleted_check;
             ALTER TABLE app DROP COLUMN deleted;
             ALTER TABLE machine DROP CONSTRAINT IF EXISTS machine_created_deleted_check;
             ALTER TABLE machine DROP COLUMN deleted;
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
             ALTER TABLE machine ADD COLUMN deleted TIMESTAMP WITHOUT TIME ZONE;
             ALTER TABLE machine
               ADD CONSTRAINT machine_created_deleted_check CHECK (
                 (deleted > created)
             );

             ALTER TABLE app ADD COLUMN deleted TIMESTAMP WITHOUT TIME ZONE;
             ALTER TABLE app
               ADD CONSTRAINT app_created_deleted_check CHECK (
                 (deleted > created)
             );

             -- Remove cascade delete action on the foreign key between app and machine.
             ALTER TABLE app DROP CONSTRAINT IF EXISTS app_machine_id_fkey;
             ALTER TABLE app ADD CONSTRAINT app_machine_id_fkey FOREIGN KEY (machine_id)
                 REFERENCES machine (id) MATCH SIMPLE
                     ON UPDATE NO ACTION
                     ON DELETE NO ACTION;
        `)
		return err
	})
}
