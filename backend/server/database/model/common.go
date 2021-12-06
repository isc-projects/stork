package dbmodel

import (
	"strings"

	"github.com/go-pg/pg/v9"
)

type SortDirEnum int

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
	if sortField != "" {
		if !strings.Contains(sortField, ".") {
			orderExpr += tableName + "."
		}
		orderExpr += sortField + " "
	} else {
		orderExpr = tableName + ".id "
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
	if id == 0 {
		_, err = tx.Model(model).Insert()
	} else {
		_, err = tx.Model(model).WherePK().Update()
	}

	return err
}
