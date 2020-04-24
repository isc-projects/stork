package dbmodel

import (
	"github.com/go-pg/pg/v9/orm"
	"github.com/pkg/errors"

	dbops "isc.org/stork/server/database"
)

const (
	SuperAdminGroupID int = 1
	AdminGroupID      int = 2
)

// Represents a group of users having some specific permissions.
type SystemGroup struct {
	ID          int
	Name        string
	Description string

	Users []*SystemUser `pg:"many2many:system_user_to_group,fk:group_id,joinFK:user_id"`
}

// Fetches all group definitions from the database ordered by id. It doesn't include
// users associated with the groups.
func GetGroupsByPage(db *dbops.PgDB, offset, limit int64, filterText *string, sortField string, sortDir SortDirEnum) ([]SystemGroup, int64, error) {
	var groups []SystemGroup
	q := db.Model(&groups)

	if filterText != nil {
		text := "%" + *filterText + "%"
		q = q.WhereGroup(func(qq *orm.Query) (*orm.Query, error) {
			qq = qq.WhereOr("name ILIKE ?", text)
			qq = qq.WhereOr("description ILIKE ?", text)
			return qq, nil
		})
	}

	// prepare sorting expression, offser and limit
	ordExpr := prepareOrderExpr("system_group", sortField, sortDir)
	q = q.OrderExpr(ordExpr)
	q = q.Offset(int(offset))
	q = q.Limit(int(limit))

	total, err := q.SelectAndCount()
	if err != nil {
		err = errors.Wrapf(err, "error while fetching a list of groups from the database")
	}

	return groups, int64(total), err
}
