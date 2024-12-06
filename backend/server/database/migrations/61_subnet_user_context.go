package dbmigs

import "github.com/go-pg/migrations/v8"

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			ALTER TABLE local_subnet
			ADD COLUMN user_context JSONB;
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			ALTER TABLE local_subnet
			DROP COLUMN user_context;
		`)
		return err
	})
}
