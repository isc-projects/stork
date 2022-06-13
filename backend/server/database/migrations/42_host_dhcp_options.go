package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
            ALTER TABLE local_host ADD COLUMN dhcp_option_set JSONB;
            ALTER TABLE local_host ADD COLUMN dhcp_option_set_hash TEXT;
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
            ALTER TABLE local_host DROP COLUMN dhcp_option_set_hash;
            ALTER TABLE local_host DROP COLUMN dhcp_option_set;
        `)
		return err
	})
}
