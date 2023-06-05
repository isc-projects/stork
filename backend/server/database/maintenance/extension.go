package maintenance

import (
	"fmt"

	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
)

// Creates a database extension if it does not exist yet.
func CreateExtension(dbi pg.DBI, extensionName string) error {
	if _, err := dbi.Exec(fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s", extensionName)); err != nil {
		return errors.Wrapf(err, `problem creating database extension "%s"`, extensionName)
	}
	return nil
}

// Checks if an extension exists in the database.
func HasExtension(dbi pg.DBI, extensionName string) (bool, error) {
	_, err := dbi.ExecOne("SELECT 1 from pg_extension WHERE extname = ?", extensionName)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, pg.ErrNoRows):
		return false, nil
	default:
		return false, errors.Wrapf(err, `problem checking if the extension "%s" exists`, extensionName)
	}
}
