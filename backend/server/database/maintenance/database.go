package maintenance

import (
	"fmt"

	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
)

// Create database with a given name.
func CreateDatabase(db *pg.DB, dbName string) (created bool, err error) {
	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s;", dbName))
	if err != nil {
		var pgErr pg.Error
		if errors.As(err, &pgErr) && pgErr.Field('C') == "42P04" { // duplicate_database
			return false, nil
		}
		err = errors.Wrapf(err, `problem creating the database "%s"`, dbName)
		return false, err
	}
	return true, nil
}

// Create database from template with a given name.
func CreateDatabaseFromTemplate(db *pg.DB, dbName, templateName string) (created bool, err error) {
	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s TEMPLATE %s;", dbName, templateName))
	if err != nil {
		var pgErr pg.Error
		if errors.As(err, &pgErr) && pgErr.Field('C') == "42P04" { // duplicate_database
			return false, nil
		}
		err = errors.Wrapf(
			err,
			`problem creating the database "%s" from the template "%s"`,
			dbName,
			templateName,
		)
		return false, err
	}
	return true, nil
}

// Drop database with a given name. It doesn't fail if the database doesn't exist.
func DropDatabaseIfExists(db *pg.DB, dbName string) error {
	if _, err := db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", dbName)); err != nil {
		return errors.Wrapf(err, `problem dropping the database "%s"`, dbName)
	}
	return nil
}
