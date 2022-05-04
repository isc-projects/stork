package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
            -- There was an issue in this trigger introduced with the previous
            -- migration: it was fired AFTER delete instead of BEFORE delete.
            -- It must be fired BEFORE delete to avoid constraint errors.
            DROP TRIGGER IF EXISTS trigger_delete_daemon_config_reports on daemon;

            -- There was also an issue with this function in which it did not return the
            -- OLD value.
            CREATE OR REPLACE FUNCTION delete_daemon_config_reports()
                RETURNS trigger
                LANGUAGE 'plpgsql'
                AS $function$
            BEGIN
                DELETE FROM config_report AS c USING daemon_to_config_report AS d
                    WHERE c.id = d.config_report_id AND d.daemon_id = OLD.id;
                RETURN OLD;
            END;
            $function$;

            -- Recreates the trigger but fires it BEFORE delete.
            CREATE TRIGGER trigger_delete_daemon_config_reports
                BEFORE DELETE ON daemon
                    FOR EACH ROW EXECUTE PROCEDURE delete_daemon_config_reports();

            -- Finally, this creates the new config_review table.
            CREATE TABLE IF NOT EXISTS config_review (
                id BIGSERIAL PRIMARY KEY,
                daemon_id BIGINT UNIQUE,
                created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT timezone('utc'::text, now()),
                config_hash TEXT NOT NULL,
                signature TEXT NOT NULL,
                CONSTRAINT config_review_daemon_id FOREIGN KEY (daemon_id)
                    REFERENCES daemon (id)
                    ON UPDATE CASCADE
                    ON DELETE CASCADE
            );
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
            DROP TABLE IF EXISTS config_review;
            DROP TRIGGER IF EXISTS trigger_delete_daemon_config_reports on daemon;
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
            CREATE TRIGGER trigger_delete_daemon_config_reports
                AFTER DELETE ON daemon
                    FOR EACH ROW EXECUTE PROCEDURE delete_daemon_config_reports();
        `)
		return err
	})
}
