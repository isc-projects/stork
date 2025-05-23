package dbmodel

import (
	"strconv"
	"strings"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/types"
	"github.com/pkg/errors"
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

	// if u != nil {
	s := strconv.FormatFloat(float64(u)*1000., 'f', 0, 64)
	b = append(b, []byte(s)...)
	// } else {
	// b = append(b, "NULL"...)
	// }

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
