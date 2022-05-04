package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
            -- The daemon_id column associates a host with a
            -- daemon. Each daemon belongs to an app, so the app_id column is
            -- now redundant.
            ALTER TABLE local_host DROP COLUMN app_id;

            -- This creates a new primary key using the daemon_id.
            ALTER TABLE local_host
                ADD CONSTRAINT local_host_pkey PRIMARY KEY (host_id, daemon_id);

            -- Similar to local_host, the local_subnet is now associated with
            -- a daemon, so the app_id column can be removed.
            ALTER TABLE local_subnet DROP COLUMN app_id;

            -- This recreates the primary key.
            ALTER TABLE local_subnet
                ADD CONSTRAINT local_subnet_pkey PRIMARY KEY (subnet_id, daemon_id);
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
            -- Adds back the app_id column.
            ALTER TABLE local_subnet
                ADD COLUMN app_id BIGINT;

            -- Copies appropriate app_id values from the daemon table.
            UPDATE local_subnet AS ls
                SET (app_id) = (SELECT app_id FROM daemon AS d WHERE d.id = ls.daemon_id);

            -- Now that the column has valid app_id values, it can be marked
            -- NOT NULL.
            ALTER TABLE local_subnet ALTER COLUMN app_id SET NOT NULL;

            -- Removes the current primary key.
            ALTER TABLE local_subnet
                DROP CONSTRAINT local_subnet_pkey;

            -- Adds back the primary key, including the app_id column.
            ALTER TABLE local_subnet
                ADD CONSTRAINT local_subnet_pkey PRIMARY KEY (app_id, subnet_id);

            -- Adds the foreign key for app_id.
            ALTER TABLE local_subnet
                ADD CONSTRAINT local_subnet_app_id FOREIGN KEY (app_id)
                    REFERENCES app(id)
                        ON UPDATE CASCADE
                        ON DELETE CASCADE;

            --  Adds back the app_id column.
            ALTER TABLE local_host
                ADD COLUMN app_id BIGINT;

            -- Copies appropriate app_id values from the daemon table.
            UPDATE local_host AS lh
                SET (app_id) = (SELECT app_id FROM daemon AS d WHERE d.id = lh.daemon_id);

            -- Now that the column has valid app_id values, it can be marked
            -- NOT NULL.
            ALTER TABLE local_host ALTER COLUMN app_id SET NOT NULL;

            -- Removes the current primary key.
            ALTER TABLE local_host
                DROP CONSTRAINT local_host_pkey;

            -- Adds back the primary key, including the app_id column.
            ALTER TABLE local_host
                ADD CONSTRAINT local_host_pkey PRIMARY KEY (host_id, app_id);

            -- Adds the foreign key for app_id.
            ALTER TABLE local_host
                ADD CONSTRAINT local_host_app_id FOREIGN KEY (app_id)
                    REFERENCES app(id)
                        ON UPDATE CASCADE
                        ON DELETE CASCADE;
        `)
		return err
	})
}
