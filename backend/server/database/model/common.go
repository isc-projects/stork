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
