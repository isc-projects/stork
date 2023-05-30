package maintenance

import (
	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
)

// Create user with a given name.
func CreateUser(dbi pg.DBI, userName string) error {
	if _, err := dbi.Exec("CREATE USER ?;", pg.Ident(userName)); err != nil {
		return errors.Wrapf(err, `problem creating the user "%s"`, userName)
	}
	return nil
}

// Checks if a user with a given name exists in the database.
func HasUser(dbi pg.DBI, userName string) (bool, error) {
	var hasUserInt int
	if _, err := dbi.Query(pg.Scan(&hasUserInt), "SELECT 1 FROM pg_roles WHERE rolname = ?;", userName); err != nil {
		return false, errors.Wrapf(err, `problem with checking if the user "%s" exists`, userName)
	}
	return hasUserInt == 1, nil
}

// Drops user with a given name. It doesn't fail if the user doesn't exist.
func DropUserSafe(dbi pg.DBI, userName string) error {
	if _, err := dbi.Exec("DROP USER IF EXISTS ?;", pg.Ident(userName)); err != nil {
		return errors.Wrapf(err, `problem dropping the user "%s"`, userName)
	}
	return nil
}

// Grant all privileges on a specific database to a given user.
func GrantAllPrivilegesOnDatabaseToUser(dbi pg.DBI, dbName, userName string) error {
	if _, err := dbi.Exec("GRANT ALL PRIVILEGES ON DATABASE ? TO ?;", pg.Ident(dbName), pg.Ident(userName)); err != nil {
		return errors.Wrapf(err, `problem granting privileges on database "%s" to user "%s"`, dbName, userName)
	}
	return nil
}

// Changes (or set) password for a given user.
func AlterUserPassword(dbi pg.DBI, userName, password string) error {
	if _, err := dbi.Exec("ALTER USER ? WITH PASSWORD ?", pg.Ident(userName), password); err != nil {
		return errors.Wrapf(err, `problem setting generated password for user "%s"`, userName)
	}
	return nil
}

// Changes the password encryption method to scram-sha-256.
// It must be called before altering password to have an effect.
// Postgres 14 support altering encryption per user e.g.:
//
//	ALTER USER my_user SET password_encryption = 'scram-sha-256';
//
// but it is not available for the lower versions (but it doesn't cause an
// error).
func AlterPasswordEncryptionToScramSHA256(dbi pg.DBI) error {
	if _, err := dbi.Exec("SET password_encryption = 'scram-sha-256'"); err != nil {
		return errors.Wrap(err, `problem setting password encryption`)
	}
	return nil
}
