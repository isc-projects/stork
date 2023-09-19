package dbmodel

import (
	"strings"

	"github.com/go-pg/pg/v10"
)

// Defines the sorting direction.
type SortDirEnum int

// Valid values of the sorting enum.
const (
	SortDirAny SortDirEnum = iota
	SortDirAsc
	SortDirDesc
)

// Prepare an order expression based on table name, sortField and sortDir.
// If sortField does not start with a table name and . then it is prepended.
// If sortDir is DESC then NULLS LAST is added, if it is ASC then NULLS FIRST
// is added. Without that records with NULLs in sortField would not be included
// in the result.
func prepareOrderExpr(tableName string, sortField string, sortDir SortDirEnum) string {
	orderExpr := ""
	escapedTableName := "\"" + tableName + "\""
	if sortField != "" {
		if !strings.Contains(sortField, ".") {
			orderExpr += escapedTableName + "."
		}
		orderExpr += sortField + " "
	} else {
		orderExpr = escapedTableName + ".id "
	}
	switch sortDir {
	case SortDirDesc:
		orderExpr += "DESC NULLS LAST"
	default:
		orderExpr += "ASC NULLS FIRST"
	}
	return orderExpr
}

// Convenience function which inserts new entry into a database or updates an
// existing entry. It determines whether this is new or existing entry by
// examining a value of the id parameter. The id is equal to 0 if this is
// a new entry.
func upsertInTransaction(tx *pg.Tx, id int64, model interface{}) (err error) {
	var result pg.Result
	if id == 0 {
		_, err = tx.Model(model).Insert()
	} else {
		result, err = tx.Model(model).WherePK().Update()
		if err == nil && result.RowsAffected() <= 0 {
			err = ErrNotExists
		}
	}

	return err
}
