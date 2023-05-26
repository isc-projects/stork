package maintenance

import (
	"fmt"

	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
)

// Create user with a given name.
func CreateUser(dbi pg.DBI, userName string) error {
	if _, err := dbi.Exec(fmt.Sprintf("CREATE USER %s;", userName)); err != nil {
		return errors.Wrapf(err, `problem creating the user "%s"`, userName)
	}
	return nil
}

// Checks if a user with a given name exists in the database.
func HasUser(dbi pg.DBI, userName string) (bool, error) {
	var hasUserInt int
	if _, err := dbi.Query(pg.Scan(&hasUserInt), fmt.Sprintf("SELECT 1 FROM pg_roles WHERE rolname='%s';", userName)); err != nil {
		return false, errors.Wrapf(err, `problem with checking if the user "%s" exists`, userName)
	}
	return hasUserInt == 1, nil
}

// Drops user with a given name. It doesn't fail if the user doesn't exist.
func DropUserSafe(dbi pg.DBI, userName string) error {
	if _, err := dbi.Exec(fmt.Sprintf("DROP USER IF EXISTS %s;", userName)); err != nil {
		return errors.Wrapf(err, `problem dropping the user "%s"`, userName)
	}
	return nil
}

// Grant all privileges on a specific database to a given user.
func GrantAllPrivilegesOnDatabaseToUser(dbi pg.DBI, dbName, userName string) error {
	if _, err := dbi.Exec(fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE %s TO %s;", dbName, userName)); err != nil {
		return errors.Wrapf(err, `problem granting privileges on database "%s" to user "%s"`, dbName, userName)
	}
	return nil
}

// Changes (or set) password for a given user.
func AlterUserPassword(dbi pg.DBI, userName, password string) error {
	if _, err := dbi.Exec(fmt.Sprintf("ALTER USER %s WITH PASSWORD '%s'", userName, password)); err != nil {
		return errors.Wrapf(err, `problem setting generated password for user "%s"`, userName)
	}
	return nil
}
