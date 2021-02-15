package dbmodel

import (
	"errors"
	"fmt"

	"github.com/go-pg/pg/v9"
	"github.com/go-pg/pg/v9/orm"
	pkgerrors "github.com/pkg/errors"
	dbops "isc.org/stork/server/database"
)

// Represents a user held in system_user table in the database.
type SystemUser struct {
	ID       int
	Login    string
	Email    string
	Lastname string
	Name     string
	Password string `pg:"password_hash"`

	Groups []*SystemGroup `pg:"many2many:system_user_to_group,fk:user_id,joinFK:group_id"`
}

type SystemUserToGroup struct {
	UserID  int `pg:",pk,not_null,on_delete:CASCADE"`
	GroupID int `pg:",pk,not_null,on_delete:CASCADE"`
}

// Returns user's identity for logging purposes. It includes login, email or both.
func (user *SystemUser) Identity() string {
	// Include login, if present.
	var s string
	if len(user.Login) > 0 {
		s = fmt.Sprintf("login=%s", user.Login)
	}

	// Include email if present.
	if len(user.Email) > 0 {
		if len(s) > 0 {
			s += " "
		}
		s += fmt.Sprintf("email=%s", user.Email)
	}

	// Neither login nor email set.
	if len(s) == 0 {
		s = "unknown"
	}

	return s
}

// Creates associations of the user with its groups.
func createUserGroups(db *pg.DB, user *SystemUser) (err error) {
	var assocs []SystemUserToGroup

	if len(user.Groups) > 0 {
		for _, g := range user.Groups {
			assocs = append(assocs, SystemUserToGroup{
				UserID:  user.ID,
				GroupID: g.ID,
			})
		}

		_, err = db.Model(&assocs).OnConflict("DO NOTHING").Insert()
	}

	return err
}

// Creates new user in the database. The returned conflict value indicates if
// the created user information is in conflict with some existing user in the
// database, e.g. duplicated login or email.
func CreateUser(db *pg.DB, user *SystemUser) (conflict bool, err error) {
	tx, err := db.Begin()
	if err != nil {
		err = pkgerrors.Wrapf(err, "unable to begin transaction while trying to create user %s", user.Identity())
		return false, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	err = db.Insert(user)

	// If insert was successful, create associations of the user with groups.
	if err == nil {
		err = createUserGroups(db, user)
	}

	if err != nil {
		var pgError pg.Error
		if errors.As(err, &pgError) {
			conflict = pgError.IntegrityViolation()
		}

		err = pkgerrors.Wrapf(err, "database operation error while trying to create user %s", user.Identity())
	}

	if err == nil {
		err = tx.Commit()
	}

	return conflict, err
}

// Updates user information in the database. The returned conflict value indicates
// if the updated data is in conflict with some other user information or the
// updated user doesn't exist.
func UpdateUser(db *pg.DB, user *SystemUser) (conflict bool, err error) {
	tx, err := db.Begin()
	if err != nil {
		err = pkgerrors.Wrapf(err, "unable to begin transaction while trying to update user %s", user.Identity())
		return false, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	err = db.Update(user)

	// Delete existing associations of the user with groups.
	if err == nil {
		_, err = db.Model(&SystemUserToGroup{}).Where("user_id = ?", user.ID).Delete()
	}

	// Recreate the groups based on the new groups list.
	if err == nil {
		err = createUserGroups(db, user)
	}

	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			conflict = true
		} else {
			var pgError pg.Error
			if errors.As(err, &pgError) {
				conflict = pgError.IntegrityViolation()
			}
		}

		err = pkgerrors.Wrapf(err, "database operation error while trying to update user %s", user.Identity())
	}

	if err == nil {
		err = tx.Commit()
	}

	return conflict, err
}

// Sets new password for the given user id.
func SetPassword(db *pg.DB, id int, password string) (err error) {
	user := SystemUser{
		ID:       id,
		Password: password,
	}

	result, err := db.Model(&user).Column("password_hash").WherePK().Update()
	if err != nil {
		err = pkgerrors.Wrapf(err, "database operation error while trying to set new password for the user id %d",
			id)
	} else if result.RowsAffected() == 0 {
		err = pkgerrors.Errorf("failed to update password for non existing user with id %d", id)
	}

	return err
}

// Checks if the old password matches the one in the database and modifies
// it to the new password if it does.
func ChangePassword(db *pg.DB, id int, oldPassword, newPassword string) (bool, error) {
	user := SystemUser{
		ID: id,
	}
	ok, err := db.Model(&user).
		Where("password_hash = crypt(?, password_hash) AND (id = ?)",
			oldPassword, id).Exists()
	if err != nil {
		err = pkgerrors.Wrapf(err, "database operation error while trying to change password of user with id %d", id)
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
		if errors.Is(err, pg.ErrNoRows) {
			return false, nil
		}
		// Other types of errors have to be logged properly.
		err = pkgerrors.Wrapf(err, "database operation error while trying to authenticate user %s", user.Identity())
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

// Fetches a collection of users from the database. The offset and
// limit specify the beginning of the page and the maximum size of the
// page. If these values are set to 0, all users are returned. Limit
// has to be greater then 0, otherwise error is returned. sortField
// allows indicating sort column in database and sortDir allows
// selection the order of sorting. If sortField is empty then id is
// used for sorting.  in SortDirAny is used then ASC order is used.
func GetUsersByPage(db *dbops.PgDB, offset, limit int64, filterText *string, sortField string, sortDir SortDirEnum) ([]SystemUser, int64, error) {
	if limit == 0 {
		return nil, 0, pkgerrors.New("limit should be greater than 0")
	}

	var users []SystemUser
	q := db.Model(&users).Relation("Groups")

	if filterText != nil {
		text := "%" + *filterText + "%"
		q = q.WhereGroup(func(qq *orm.Query) (*orm.Query, error) {
			qq = qq.WhereOr("login ILIKE ?", text)
			qq = qq.WhereOr("email ILIKE ?", text)
			qq = qq.WhereOr("lastname ILIKE ?", text)
			qq = qq.WhereOr("name ILIKE ?", text)
			return qq, nil
		})
	}

	// prepare sorting expression, offset and limit
	ordExpr := prepareOrderExpr("system_user", sortField, sortDir)
	q = q.OrderExpr(ordExpr)
	q = q.Offset(int(offset))
	q = q.Limit(int(limit))

	total, err := q.SelectAndCount()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return []SystemUser{}, 0, nil
		}
		err = pkgerrors.Wrapf(err, "problem with fetching a list of users from the database")
	}

	return users, int64(total), err
}

// Fetches a user with a given id from the database. If the user does not exist
// the nil value is returned. The user is returned along with the list of groups
// it belongs to.
func GetUserByID(db *dbops.PgDB, id int) (*SystemUser, error) {
	user := &SystemUser{}
	err := db.Model(user).Relation("Groups").Where("id = ?", id).First()
	if errors.Is(err, pg.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, pkgerrors.Wrapf(err, "problem with fetching user %v from the database", id)
	}
	return user, err
}

// Associates a user with a group. Currently only insertion by group id is supported.
func (user *SystemUser) AddToGroupByID(db *dbops.PgDB, group *SystemGroup) (added bool, err error) {
	if group.ID > 0 {
		res, err := db.Model(&SystemUserToGroup{
			UserID:  user.ID,
			GroupID: group.ID,
		}).OnConflict("DO NOTHING").Insert()

		return res.RowsAffected() > 0, err
	}
	err = pkgerrors.Errorf("unable to add user to the unknown group")
	return false, err
}

// Checks if the user is in the specified group. The group is matched by
// name and/or by id.
func (user *SystemUser) InGroup(group *SystemGroup) bool {
	for _, g := range user.Groups {
		if (g.ID > 0 && g.ID == group.ID) || (len(g.Name) > 0 && g.Name == group.Name) {
			return true
		}
	}
	return false
}
