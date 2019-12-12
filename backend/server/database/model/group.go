package dbmodel

import (
	"github.com/pkg/errors"
	"isc.org/stork/server/database"
)

const SuperAdminGroupId int = 1

// Represents a group of users having some specific permissions.
type SystemGroup struct {
	Id          int
	Name        string
	Description string

	Users []*SystemUser `pg:"many2many:system_user_to_group,fk:group_id,joinFK:user_id"`
}

type SystemGroups []*SystemGroup

// Fetches all group definitions from the database ordered by id. It doesn't include
// users associated with the groups.
func GetGroups(db *dbops.PgDB) (groups SystemGroups, err error) {
	err = db.Model(&groups).OrderExpr("id ASC").Select()
	if err != nil {
		err = errors.Wrapf(err, "error while fetching a list of groups from the database")
	}

	return groups, err
}
