package dbmigs

import "github.com/go-pg/migrations/v8"

func init() {
	// Stork shouldn't exceed 60 decimal places in any statistic, even if fully loaded.
	// See comments and calculations in backend/server/database/model/stats.go.
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
		ALTER TABLE statistic ALTER COLUMN "value" TYPE DECIMAL(60,0);
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
		ALTER TABLE statistic ALTER COLUMN "value" TYPE BIGINT
			-- This clamps the values to bigint bounds.
			USING
				CASE WHEN value IS NULL THEN NULL
				ELSE
					GREATEST(
						LEAST(
							"value",
							-(((2^(8*pg_column_size(1::bigint)-2))::bigint << 1)+1)
						),
						(2^(8*pg_column_size(1::bigint)-2))::bigint << 1
					)
				END;
        `)
		return err
	})
}
