package dbmigs

import "github.com/go-pg/migrations/v8"

func init() {
	// Stork shouldn't exceed the 60 decimal places in any statistic, even if fully loaded.
	// See comments and calculations in backend/server/database/model/stats.go.
	// Additionally, the statistic value cannot be NULL.
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
		ALTER TABLE statistic ALTER COLUMN "value" TYPE DECIMAL(60,0);
		UPDATE statistic SET "value"=0 WHERE "value" IS NULL;
		ALTER TABLE statistic ALTER COLUMN "value" SET NOT NULL;
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
		ALTER TABLE statistic ALTER COLUMN "value" DROP NOT NULL;
		ALTER TABLE statistic ALTER COLUMN "value" TYPE BIGINT
			-- Clamp the values to bigint bounds.
			USING GREATEST(
				LEAST(
					"value",
					-(((2^(8*pg_column_size(1::bigint)-2))::bigint << 1)+1)
				),
				(2^(8*pg_column_size(1::bigint)-2))::bigint << 1
			);
        `)
		return err
	})
}
