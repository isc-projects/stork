package dbmodel

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/go-pg/pg/v9"
)

// Represents a user held in system_user table in the database.
type SystemUser struct {
	Id           int
	Login        string
	Email        string
	Lastname     string
	Name         string
	Password     string `pg:"password_hash"`
}

// Returns user's identity for logging purposes. It includes login, email or both.
func (u *SystemUser) Identity() string {
	// Include login, if present.
	var s string
	if len(u.Login) > 0 {
		s = fmt.Sprintf("login=%s", u.Login)
	}

	// Include email if present.
	if len(u.Email) > 0 {
		if len(s) > 0 {
			s += " "
		}
		s += fmt.Sprintf("email=%s", u.Email)
	}

	// Neither login nor email set.
	if len(s) == 0 {
		s = "unknown"
	}

	return s
}

// Inserts or updates user information in the database. If user id is set, the user information
// is updated in the database. Otherwise, the new user will be inserted.
func (user *SystemUser) Persist(db *pg.DB) (err error) {
	if user.Id == 0 {
		// Add new user as the id is not set.
		err = db.Insert(user)
	} else {
		// Update existing user by primary key.
		err = db.Update(user)
	}
	if err != nil {
		errors.Wrapf(err, "database operation error while trying to persist user %s", user.Identity())
	}
	return err
}

// Finds the user in the database by login or email and verifies that the provided password
// is correct.
func Authenticate(db *pg.DB, user *SystemUser) (bool, error) {
	// Using authentication technique described here: https://www.postgresql.org/docs/8.3/pgcrypto.html
	err := db.Model(user).
		Where("password_hash = crypt(?, password_hash) AND (login = ? OR email = ?)",
		user.Password, user.Login, user.Email).Select()

	if err != nil {
		// Failing to find an entry is not really an error. It merely means that the
		// authentication failed, so return false in this case.
		if err == pg.ErrNoRows {
			return false, nil
		}
		// Other types of errors have to be logged properly.
		errors.Wrapf(err, "database operation error while trying to authenticate user %s", user.Identity())
		return false, err
	}

	// We don't want to return password hash in the password field so we
	// set it to an empty string, which serves two purposes. First, the
	// password hash is not interpreted as password. Second, when using the
	// returned SystemUser instance to update the database the password will
	// remain unmodified in the database. The database treats empty password
	// as an indication that the old password must be preserved.
	user.Password = ""
	return true, err
}

