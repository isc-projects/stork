package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
            -- Adds a use-secure-protocol column to store the TLS status of the Kea CA.
            ALTER TABLE access_point ADD COLUMN use_secure_protocol BOOLEAN DEFAULT false;
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
            ALTER TABLE access_point DROP COLUMN use_secure_protocol;
        `)
		return err
	})
}
