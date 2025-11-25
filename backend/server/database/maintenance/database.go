package maintenance

import (
	"bytes"
	"fmt"
	"os"
	"regexp"

	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
)

// Create database with a given name.
func CreateDatabase(db *pg.DB, dbName string) (created bool, err error) {
	_, err = db.Exec("CREATE DATABASE ?;", pg.Ident(dbName))
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
	_, err = db.Exec("CREATE DATABASE ? TEMPLATE ?;", pg.Ident(dbName), pg.Ident(templateName))
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
	if _, err := db.Exec("DROP DATABASE IF EXISTS ?;", pg.Ident(dbName)); err != nil {
		return errors.Wrapf(err, `problem dropping the database "%s"`, dbName)
	}
	return nil
}

// Restore database from a dump file. The database must already exist.
// The dump file must be a plain SQL file. It should not have been created
// with the pg_dump's custom format. The privileges and ownerships should be
// excluded from the dump file.
//
// Warning: This function executes the SQL script without any validation. Don't
// call it in the production code.
func RestoreDatabaseFromDump(db *pg.DB, dumpFilePath string) error {
	// It is a naive method that reads the whole dump file into a memory and
	// executes it as a single command. It would be better to stream the file
	// content to the database server, but go-pg doesn't support it directly.
	// Alternatively, we could split the file into chunks or use the
	// pg_restore command line tool.
	dump, err := os.ReadFile(dumpFilePath)
	if err != nil {
		return errors.Wrapf(
			err,
			`problem reading the database dump file "%s"`,
			dumpFilePath,
		)
	}

	// Wipe out a query to create the public schema. It already exists in
	// the target database, and executing this command would fail.
	// We could drop the public schema and recreate it from the dump, but
	// that would also drop any default privileges that are not included in the
	// dump file.
	dump = bytes.ReplaceAll(dump, []byte("CREATE SCHEMA public;"), []byte("-- CREATE SCHEMA public;"))

	// Wipe out commenting on the public schema as it is allowed only by the
	// owner.
	dump = bytes.ReplaceAll(dump, []byte("COMMENT ON SCHEMA public IS 'standard public schema';"), []byte(""))

	// Some PostgreSQL versions may not support the database configuration
	// parameters included in the dump file. Wipe them out if they are not
	// supported.
	for {
		_, err = db.Exec(string(dump))

		// Check if the error is about an unrecognized configuration parameter.
		var pgErr pg.Error
		if err == nil || !errors.As(err, &pgErr) {
			break
		}
		if pgErr.Field('C') != "42704" { // undefined_object
			break
		}
		pattern := regexp.MustCompile("^unrecognized configuration parameter \"(.+)\"$")
		matches := pattern.FindStringSubmatch(pgErr.Field('M'))
		if len(matches) != 2 {
			// Unmatched error message. It must be similar problem but with the
			// restoring query.
			break
		}
		parameter := matches[1]
		// Wipe out the unrecognized parameter from the dump file.
		paramPattern := regexp.MustCompile(fmt.Sprintf(`SET %s = .+?;`, regexp.QuoteMeta(parameter)))
		dump = paramPattern.ReplaceAll(dump, []byte(""))
	}

	if err != nil {
		return errors.Wrapf(
			err,
			`problem restoring the database from the dump file "%s"`,
			dumpFilePath,
		)
	}
	return nil
}
