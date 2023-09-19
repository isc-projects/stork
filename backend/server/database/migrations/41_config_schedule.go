package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
            -- Create a table holding scheduled configuration changes.
            CREATE TABLE IF NOT EXISTS scheduled_config_change (
                id BIGSERIAL NOT NULL PRIMARY KEY,
                created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT timezone('utc'::text, now()) NOT NULL,
                deadline_at TIMESTAMP WITHOUT TIME ZONE NOT NULL,
                executed BOOLEAN DEFAULT FALSE,
                error TEXT,
                user_id BIGINT NOT NULL,
                updates JSONB NOT NULL,
                CONSTRAINT scheduled_config_change_user FOREIGN KEY (user_id)
                    REFERENCES public.system_user(id)
                        ON UPDATE CASCADE
                        ON DELETE CASCADE
            );
            CREATE INDEX scheduled_config_change_deadline_idx ON scheduled_config_change USING btree (deadline_at);
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
            DROP TABLE IF EXISTS scheduled_config_change;
        `)
		return err
	})
}
