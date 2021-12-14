package dbmigs

import (
	"github.com/go-pg/migrations/v7"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
            -- Cast the integer statistics to floats - the same type that original type from Kea.
            ALTER TABLE statistic ALTER COLUMN value TYPE DOUBLE PRECISION;
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
			-- Limit to bigint safe range.
			CREATE FUNCTION pg_temp.clamp(val double precision) RETURNS bigint AS $$
				BEGIN
					RETURN CAST($1 as bigint);
				EXCEPTION
					WHEN numeric_value_out_of_range THEN
						RETURN CASE 
							WHEN val> 0 THEN +9223372036854775807
							ELSE -9223372036854775808 END;
				END;
			$$ LANGUAGE plpgsql;
			
			ALTER TABLE statistic ALTER COLUMN "value" TYPE BIGINT USING pg_temp.clamp(value);
			
			DROP FUNCTION pg_temp.clamp;
        `)
		return err
	})
}
