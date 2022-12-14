package dbmigs

import "github.com/go-pg/migrations/v8"

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
			ALTER TABLE prefix_pool ADD COLUMN excluded_prefix cidr;
		`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			ALTER TABLE prefix_pool DROP COLUMN excluded_prefix;
		`)
		return err
	})
}
