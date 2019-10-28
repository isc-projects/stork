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

func (u *SystemUser) Identity() string {
	var s string
	if len(u.Login) > 0 {
		s = fmt.Sprintf("login=%s", u.Login)
	}

	if len(u.Email) > 0 {
		if len(s) > 0 {
			s += " "
		}
		s += fmt.Sprintf("email=%s", u.Email)
	}

	if len(s) == 0 {
		s = "unknown"
	}

	return s
}

func (u *SystemUser) Persist(db *pg.DB) error {
	_, err := db.Model(u).Insert()
	if err != nil {
		_, err = db.Model(u).Where("email = ? OR login = ?", u.Email, u.Login).Update()
	}
	if err != nil {
		errors.Wrapf(err, "database operation error while trying to insert or update user %s", u.Identity())
	}
	return err
}

func Authenticate(db *pg.DB, user *SystemUser) (bool, error) {
	err := db.Model(user).
		Where("password_hash = crypt(?, password_hash) AND (login = ? OR email = ?)", user.Password, user.Login, user.Email).
		Select()

	if err != nil {
		if err == pg.ErrNoRows {
			return false, nil
		}
		errors.Wrapf(err, "database operation error while trying to authenticate user %s", user.Identity())
		return false, err
	}

	user.Password = ""
	return true, err
}

