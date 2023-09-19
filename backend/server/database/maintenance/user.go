package maintenance

import (
	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
)

// The password encryption used by Postgres.
type PgPasswordEncryption string

// The password encryptions supported by Postgres.
const (
	PgPasswordEncryptionMD5         PgPasswordEncryption = "md5"
	PgPasswordEncryptionScramSHA256 PgPasswordEncryption = "scram-sha-256" //nolint:gosec
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
func DropUserIfExists(dbi pg.DBI, userName string) error {
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

// Grant all privileges on a specific schema to a given user.
func GrantAllPrivilegesOnSchemaToUser(dbi pg.DBI, schemaName, userName string) error {
	if _, err := dbi.Exec("GRANT ALL PRIVILEGES ON SCHEMA ? TO ?;", pg.Ident(schemaName), pg.Ident(userName)); err != nil {
		return errors.Wrapf(err, `problem granting privileges on schema "%s" to user "%s"`, schemaName, userName)
	}
	return nil
}

// Revoke all privileges on a specific schema from a given user.
func RevokeAllPrivilegesOnSchemaFromUser(dbi pg.DBI, schemaName, userName string) error {
	if _, err := dbi.Exec("REVOKE ALL PRIVILEGES ON SCHEMA ? FROM ?;", pg.Ident(schemaName), pg.Ident(userName)); err != nil {
		return errors.Wrapf(err, `problem revoking privileges on schema "%s" from user "%s"`, schemaName, userName)
	}
	return nil
}

// Changes (or set) password for a given user.
func AlterUserPassword(dbi pg.DBI, userName, password string) error {
	if _, err := dbi.Exec("ALTER USER ? WITH PASSWORD ?", pg.Ident(userName), password); err != nil {
		return errors.Wrapf(err, `problem altering password for user "%s"`, userName)
	}
	return nil
}

// Changes the password encryption method.
// It must be called before altering password to have an effect.
// Postgres 14 support altering encryption per user e.g.:
//
//	ALTER USER my_user SET password_encryption = 'scram-sha-256';
//
// but it is not available for the lower versions (but it doesn't cause an
// error).
func SetPasswordEncryption(dbi pg.DBI, passwordEncryption PgPasswordEncryption) error {
	if _, err := dbi.Exec("SET password_encryption = ?", passwordEncryption); err != nil {
		return errors.Wrap(err, `problem setting password encryption`)
	}
	return nil
}

// Returns current password encryption.
func ShowPasswordEncryption(dbi pg.DBI) (PgPasswordEncryption, error) {
	var passwordEncryption string
	if _, err := dbi.Query(pg.Scan(&passwordEncryption), "SHOW password_encryption;"); err != nil {
		return "", errors.Wrap(err, `problem reading password encryption`)
	}
	return PgPasswordEncryption(passwordEncryption), nil
}
