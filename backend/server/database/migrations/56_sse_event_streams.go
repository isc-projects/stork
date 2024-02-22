package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
             -- Add sse_streams column to the event table.
             ALTER TABLE event ADD COLUMN sse_streams TEXT[];
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
             ALTER TABLE event DROP COLUMN IF EXISTS sse_streams;
        `)
		return err
	})
}
