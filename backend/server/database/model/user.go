package dbmodel

import (
	"fmt"
	"github.com/go-pg/pg/v9"
	"github.com/pkg/errors"
	"isc.org/stork/server/database"
)

// Represents a user held in system_user table in the database.
type SystemUser struct {
	Id       int
	Login    string
	Email    string
	Lastname string
	Name     string
	Password string `pg:"password_hash"`

	Groups SystemGroups `pg:"many2many:system_user_to_group,fk:user_id,joinFK:group_id"`
}

type SystemUserToGroup struct {
	UserID  int `pg:",pk,not_null,on_delete:CASCADE"`
	GroupID int `pg:",pk,not_null,on_delete:CASCADE"`
}

type SystemUsers []*SystemUser

type SystemUserOrderBy int

const (
	SystemUserOrderById SystemUserOrderBy = iota
	SystemUserOrderByLoginEmail
)

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

// Creates new user in the database. The returned conflict value indicates if
// the created user information is in conflict with some existing user in the
// database, e.g. duplicated login or email.
func CreateUser(db *pg.DB, user *SystemUser) (err error, conflict bool) {
	tx, err := db.Begin()
	if err != nil {
		errors.Wrapf(err, "unable to begin transaction while trying to create user %s", user.Identity())
		return err, false
	}
	defer tx.Rollback()

	err = db.Insert(user)

	if err != nil {
		pgErr, ok := err.(pg.Error)
		if ok {
			conflict = pgErr.IntegrityViolation()
		}

		errors.Wrapf(err, "database operation error while trying to create user %s", user.Identity())
	}

	if err == nil {
		tx.Commit()
	}

	return err, conflict
}

// Updates user information in the database. The returned conflict value indicates
// if the updated data is in conflict with some other user information or the
// updated user doesn't exist.
func UpdateUser(db *pg.DB, user *SystemUser) (err error, conflict bool) {
	tx, err := db.Begin()
	if err != nil {
		errors.Wrapf(err, "unable to begin transaction while trying to update user %s", user.Identity())
		return err, false
	}
	defer tx.Rollback()

	err = db.Update(user)
	if err != nil {
		if err == pg.ErrNoRows {
			conflict = true
		} else {
			pgErr, ok := err.(pg.Error)
			if ok {
				conflict = pgErr.IntegrityViolation()
			}
		}

		errors.Wrapf(err, "database operation error while trying to update user %s", user.Identity())
	}

	if err == nil {
		tx.Commit()
	}

	return err, conflict
}

// Sets new password for the given user id.
func SetPassword(db *pg.DB, id int, password string) (err error) {
	user := SystemUser{
		Id:       id,
		Password: password,
	}

	result, err := db.Model(&user).Column("password_hash").WherePK().Update()
	if err != nil {
		errors.Wrapf(err, "database operation error while trying to set new password for the user id %d",
			id)

	} else if result.RowsAffected() == 0 {
		err = errors.Errorf("failed to update password for non existing user with id %d", id)
	}

	return err
}

// Checks if the old password matches the one in the database and modifies
// it to the new password if it does.
func ChangePassword(db *pg.DB, id int, oldPassword, newPassword string) (bool, error) {
	user := SystemUser{
		Id: id,
	}
	ok, err := db.Model(&user).
		Where("password_hash = crypt(?, password_hash) AND (id = ?)",
			oldPassword, id).Exists()

	if err != nil {
		errors.Wrapf(err, "database operation error while trying to change password of user with id %d", id)
		return false, err
	}

	if !ok {
		return false, nil
	}

	err = SetPassword(db, id, newPassword)
	return true, err
}

// Finds the user in the database by login or email and verifies that the provided password
// is correct.
func Authenticate(db *pg.DB, user *SystemUser) (bool, error) {
	// Using authentication technique described here: https://www.postgresql.org/docs/8.3/pgcrypto.html
	err := db.Model(user).Relation("Groups").
		Where("password_hash = crypt(?, password_hash) AND (login = ? OR email = ?)",
			user.Password, user.Login, user.Email).First()

	if err != nil {
		// Failing to find an entry is not really an error. It merely means that the
		// authentication failed, so return false in this case.
		if err == pg.ErrNoRows {
			return false, nil
		}
		// Other types of errors have to be logged properly.
		err = errors.Wrapf(err, "database operation error while trying to authenticate user %s", user.Identity())
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

// Fetches a collection of users from the database. The offset and limit specify the
// beginning of the page and the maximum size of the page. If these values are set
// to 0, all users are returned. Limit has to be greater
// then 0, otherwise error is returned.
func GetUsers(db *dbops.PgDB, offset, limit int, order SystemUserOrderBy) (users SystemUsers, total int64, err error) {
	total = int64(0)
	if limit == 0 {
		return nil, total, errors.New("limit should be greater than 0")
	}

	q := db.Model(&users).Relation("Groups")

	switch order {
	case SystemUserOrderByLoginEmail:
		q = q.OrderExpr("login ASC").OrderExpr("email ASC")
	default:
		q = q.OrderExpr("id ASC")
	}

	// first get total count
	totalInt, err := q.Clone().Count()
	if err != nil {
		return nil, total, errors.Wrapf(err, "problem with getting users total")
	}
	total = int64(totalInt)

	// then do actual query
	q = q.Offset(offset).Limit(limit)
	err = q.Select()

	if err != nil {
		err = errors.Wrapf(err, "problem with fetching a list of users from the database")
	}

	return users, total, err
}

// Fetches a user with a given id from the database. If the user does not exist
// the nil value is returned. The user is returned along with the list of groups
// it belongs to.
func GetUserById(db *dbops.PgDB, id int) (*SystemUser, error) {
	user := &SystemUser{}
	err := db.Model(user).Relation("Groups").Where("id = ?", id).First()
	if err == pg.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "problem with fetching user %v from the database", id)
	}
	return user, err
}

// Associates a user with a group. Currently only insertion by group id is supported.
func (u *SystemUser) AddToGroupById(db *dbops.PgDB, group *SystemGroup) (added bool, err error) {
	if group.Id > 0 {
		res, err := db.Model(&SystemUserToGroup{
			UserID:  u.Id,
			GroupID: group.Id,
		}).OnConflict("DO NOTHING").Insert()

		return res.RowsAffected() > 0, err

	} else {
		err = errors.Errorf("unable to add user to the unknown group")
	}
	return false, err
}

// Checks if the user is in the specified group. The group is matched by
// name and/or by id.
func (u *SystemUser) InGroup(group *SystemGroup) bool {
	for _, g := range u.Groups {
		if (g.Id > 0 && g.Id == group.Id) || (len(g.Name) > 0 && g.Name == group.Name) {
			return true
		}
	}
	return false
}
