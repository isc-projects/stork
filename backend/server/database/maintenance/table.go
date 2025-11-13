package maintenance

import (
	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
)

// Drops a given table. It doesn't fail if the table doesn't exist.
func DropTableIfExists(dbi pg.DBI, tableName string) error {
	if _, err := dbi.Exec("DROP TABLE IF EXISTS ?", pg.Ident(tableName)); err != nil {
		return errors.Wrapf(err, `problem dropping table "%s"`, tableName)
	}
	return nil
}

// Drops a given sequence. It doesn't fail if the sequence doesn't exist.
func DropSequenceIfExists(dbi pg.DBI, tableName string) error {
	if _, err := dbi.Exec("DROP SEQUENCE IF EXISTS ?", pg.Ident(tableName)); err != nil {
		return errors.Wrapf(err, `problem dropping sequence "%s"`, tableName)
	}
	return nil
}
