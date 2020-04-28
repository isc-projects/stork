package dbmodel

import (
	"strings"
)

type SortDirEnum int

const (
	SortDirAny SortDirEnum = iota
	SortDirAsc
	SortDirDesc
)

// Prepare an order expression based on table name, sortField and sortDir.
// If sortField does not start with a table name and . then it is prepanded.
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
