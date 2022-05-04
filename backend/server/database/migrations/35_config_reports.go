package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
            -- Creates a table holding configuration review results.
            CREATE TABLE IF NOT EXISTS config_report (
                id BIGSERIAL PRIMARY KEY,
                created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT timezone('utc'::text, now()),
                checker_name TEXT NOT NULL,
                content TEXT NOT NULL,
                daemon_id BIGINT NOT NULL,
                CONSTRAINT config_report_daemon_id FOREIGN KEY (daemon_id)
                    REFERENCES daemon (id)
                    ON UPDATE CASCADE
                    ON DELETE NO ACTION
            );

            -- Creates a table mapping the configuration review results to the daemons
            -- for which the review was conducted.
            CREATE TABLE IF NOT EXISTS daemon_to_config_report (
                daemon_id BIGINT NOT NULL,
                config_report_id BIGINT NOT NULL,
                order_index BIGINT NOT NULL DEFAULT 0,
                CONSTRAINT daemon_to_config_report_pkey PRIMARY KEY (daemon_id, config_report_id),
                CONSTRAINT daemon_to_config_report_daemon_id FOREIGN KEY (daemon_id)
                    REFERENCES daemon (id)
                    ON UPDATE CASCADE
                    ON DELETE NO ACTION,
                CONSTRAINT daemon_to_config_report_config_report_id FOREIGN KEY (config_report_id)
                    REFERENCES config_report (id)
                    ON UPDATE CASCADE
                    ON DELETE CASCADE
            );

            -- Deletes all configuration reports associated with a deleted daemon.
            -- A cascade action cannot be used for this operation, because not only
            -- should the associations between the config reports and the
            -- deleted daemon be removed, but also all reports from the config_report table.
            CREATE OR REPLACE FUNCTION delete_daemon_config_reports()
                RETURNS trigger
                LANGUAGE 'plpgsql'
                AS $function$
            BEGIN
                DELETE FROM config_report AS c USING daemon_to_config_report AS d
                    WHERE c.id = d.config_report_id AND d.daemon_id = OLD.id;
                RETURN NULL;
            END;
            $function$;

            -- Creates a trigger fired when a daemon is deleted. This trigger removes
            -- all configuration review reports associated with the daemon. Once
            -- a report is deleted, the cascade action on the daemon_to_config_report
            -- deletes all associations with this report.
            DO $$ BEGIN
                CREATE TRIGGER trigger_delete_daemon_config_reports
                    AFTER DELETE ON daemon
                        FOR EACH ROW EXECUTE PROCEDURE delete_daemon_config_reports();
            EXCEPTION
                WHEN duplicate_object THEN null;
            END $$;
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
            DROP TRIGGER IF EXISTS trigger_delete_daemon_config_reports on daemon;
            DROP FUNCTION IF EXISTS delete_daemon_config_reports;
            DROP TABLE IF EXISTS daemon_to_config_report;
            DROP TABLE IF EXISTS config_report;
        `)
		return err
	})
}
