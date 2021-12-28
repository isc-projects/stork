package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
            -- We now have daemon_id column which associates a host with a
            -- daemon. Each daemon belongs to an app, so app_id column is
            -- now redundant.
            ALTER TABLE local_host DROP COLUMN app_id;

            -- Need to create new primary key using daemon_id.
            ALTER TABLE local_host
                ADD CONSTRAINT local_host_pkey PRIMARY KEY (host_id, daemon_id);

            -- Similar to local_host, the local_subnet is now associated with
            -- a daemon, so we can remove the app_id column.
            ALTER TABLE local_subnet DROP COLUMN app_id;

            -- Recreate the primary key.
            ALTER TABLE local_subnet
                ADD CONSTRAINT local_subnet_pkey PRIMARY KEY (subnet_id, daemon_id);
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
            -- Add back the app_id column.
            ALTER TABLE local_subnet
                ADD COLUMN app_id BIGINT;

            -- Copy appropriate app_id values from the daemon table.
            UPDATE local_subnet AS ls
                SET (app_id) = (SELECT app_id FROM daemon AS d WHERE d.id = ls.daemon_id);

            -- Now that we filled the column with valid app_id values we can
            -- mark the colum NOT NULL.
            ALTER TABLE local_subnet ALTER COLUMN app_id SET NOT NULL;

            -- Remove current primary key.
            ALTER TABLE local_subnet
                DROP CONSTRAINT local_subnet_pkey;

            -- Add back the primary key including app_id column.
            ALTER TABLE local_subnet
                ADD CONSTRAINT local_subnet_pkey PRIMARY KEY (app_id, subnet_id);

            -- Add the foreign key for app_id.
            ALTER TABLE local_subnet
                ADD CONSTRAINT local_subnet_app_id FOREIGN KEY (app_id)
                    REFERENCES app(id)
                        ON UPDATE CASCADE
                        ON DELETE CASCADE;

            --  Add back the app_id column.
            ALTER TABLE local_host
                ADD COLUMN app_id BIGINT;

            -- Copy appropriate app_id values from the daemon table.
            UPDATE local_host AS lh
                SET (app_id) = (SELECT app_id FROM daemon AS d WHERE d.id = lh.daemon_id);

            -- Now that we filled the column with valid app_id values we can
            -- mark the colum NOT NULL.
            ALTER TABLE local_host ALTER COLUMN app_id SET NOT NULL;

            -- Remove current primary key.
            ALTER TABLE local_host
                DROP CONSTRAINT local_host_pkey;

            -- Add back the primary key including app_id column.
            ALTER TABLE local_host
                ADD CONSTRAINT local_host_pkey PRIMARY KEY (host_id, app_id);

            -- Add the foreign key for app_id.
            ALTER TABLE local_host
                ADD CONSTRAINT local_host_app_id FOREIGN KEY (app_id)
                    REFERENCES app(id)
                        ON UPDATE CASCADE
                        ON DELETE CASCADE;
        `)
		return err
	})
}
