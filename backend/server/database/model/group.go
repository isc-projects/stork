package dbmodel

import (
	"errors"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	pkgerrors "github.com/pkg/errors"
	dbops "isc.org/stork/server/database"
)

// List of the user group IDs used in the server.
const (
	SuperAdminGroupID int = 1
	AdminGroupID      int = 2
)

// Represents a group of users having some specific permissions.
type SystemGroup struct {
	ID          int
	Name        string
	Description string

	Users []*SystemUser `pg:"many2many:system_user_to_group,fk:group_id,join_fk:user_id"`
}

// Fetches a collection of groups from the database. The offset and
// limit specify the beginning of the page and the maximum size of the
// page. The filterText can be used to match the name of description
// of a group. The nil value disables such filtering. sortField allows
// indicating sort column in database and sortDir allows selection the
// order of sorting. If sortField is empty then id is used for
// sorting.  in SortDirAny is used then ASC order is used. This
// function returns a collection of groups, the total number of groups
// and error.
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

	// prepare sorting expression, offset and limit
	ordExpr := prepareOrderExpr("system_group", sortField, sortDir)
	q = q.OrderExpr(ordExpr)
	q = q.Offset(int(offset))
	q = q.Limit(int(limit))

	total, err := q.SelectAndCount()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return []SystemGroup{}, 0, nil
		}
		err = pkgerrors.Wrapf(err, "error while fetching a list of groups from the database")
	}

	return groups, int64(total), err
}
