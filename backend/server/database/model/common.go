package dbmodel

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/types"
	"github.com/pkg/errors"
)

// Defines the sorting direction.
type SortDirEnum string

// Valid values of the sorting enum.
const (
	SortDirAny  SortDirEnum = "any"
	SortDirAsc  SortDirEnum = "asc"
	SortDirDesc SortDirEnum = "desc"
)

// Prepare an order expression based on table name, sortField and sortDir.
// If sortField does not start with a table name and . then it is prepended.
// If sortDir is DESC then NULLS LAST is added, if it is ASC then NULLS FIRST
// is added. Without that records with NULLs in sortField would not be included
// in the result.
// It returns also DISTINCT ON expression that is often used in queries together with ORDER BY.
// DISTINCT ON must contain the same fields that are used for sorting.
// It accepts a function to handle some sortFields in a special way.
// The custom handler function is called with original sortField, escapedTableName and dirExpr.
// It returns ORDER BY expression, DISTINCT ON expression,
// and a boolean value indicating whether the custom handler processed the sortField.
func prepareOrderAndDistinctExpr(tableName string, sortField string, sortDir SortDirEnum, customOrderAndDistinct func(sortField, escapedTableName, dirExpr string) (string, string, bool)) (orderExpr string, distinctOnExpr string) {
	var dirExpr string
	switch sortDir {
	case SortDirDesc:
		dirExpr = "DESC NULLS LAST"
	default:
		dirExpr = "ASC NULLS FIRST"
	}
	escapedTableName := "\"" + tableName + "\""
	distinctOnExpr = escapedTableName + ".id"

	if sortField == "" {
		// No sort field specified, use ID.
		orderExpr = fmt.Sprintf("%s.id %s", escapedTableName, dirExpr)
		return orderExpr, distinctOnExpr
	}

	if customOrderAndDistinct != nil {
		// Check whether custom handler wants to process this sort field.
		customOrderExpr, customDistinctExpr, useCustom := customOrderAndDistinct(sortField, escapedTableName, dirExpr)
		if useCustom {
			orderExpr = customOrderExpr
			distinctOnExpr += ", " + customDistinctExpr
			return orderExpr, distinctOnExpr
		}
	}

	// Construct order and distinct expressions in the default way.
	if !strings.Contains(sortField, ".") {
		// It is a column name. Prepend table name.
		sortField = fmt.Sprintf("%s.%s", escapedTableName, sortField)
	}

	if strings.ToLower(sortField) != tableName+".id" && strings.ToLower(sortField) != escapedTableName+".id" {
		// ID is already included in distinct expression. If sorting by some other field,
		// add it to distinct expression as well.
		distinctOnExpr += ", " + sortField
	}

	orderExpr = fmt.Sprintf("%s %s", sortField, dirExpr)
	return orderExpr, distinctOnExpr
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

	return errors.WithStack(err)
}

// Type for storing utilization in a smallint column. The utilization range is
// 0-1. The value is stored by multiplying the real utilization by 1000. For
// example, if the utilization is 0.123456789 then the value stored in the
// database is 123. The value is stored in the smallint column to save space
// (2 bytes (!)).
type Utilization float64

var _ types.ValueAppender = (*Utilization)(nil)

var _ types.ValueScanner = (*Utilization)(nil)

// Converts the utilization to the integer value. The value is multiplied by
// 1000 to store it in the smallint column. The value is rounded to the nearest
// integer value.
func (u Utilization) AppendValue(b []byte, quote int) ([]byte, error) {
	if quote == 1 {
		b = append(b, '\'')
	}

	s := strconv.FormatFloat(float64(u)*1000., 'f', 0, 64)
	b = append(b, []byte(s)...)

	if quote == 1 {
		b = append(b, '\'')
	}
	return b, nil
}

// Deserializes the utilization from the database. The value is stored in
// the smallint column. The value is divided by 1000 to get the real
// utilization value. The value is rounded to the nearest integer value.
func (u *Utilization) ScanValue(rd types.Reader, n int) error {
	if n <= 0 {
		return nil
	}

	tmp, err := rd.ReadFullTemp()
	if err != nil {
		return err
	}

	i, err := strconv.ParseInt(string(tmp), 10, 64)
	if err != nil {
		return errors.Wrapf(err, "problem parsing utilization from DB: '%s'", string(tmp))
	}

	*u = Utilization(float64(i) / 1000.)

	return nil
}
