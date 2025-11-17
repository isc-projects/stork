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
type SortDirEnum int

// Valid values of the sorting enum.
const (
	SortDirAny SortDirEnum = iota
	SortDirAsc
	SortDirDesc
)

// Defines custom sort field values.
type CustomSortFieldEnum string

// Valid values of the enum.
const (
	TotalAddresses    CustomSortFieldEnum = "custom_total_addresses"
	AssignedAddresses CustomSortFieldEnum = "custom_assigned_addresses"
	TotalPDs          CustomSortFieldEnum = "custom_total_pds"
	AssignedPDs       CustomSortFieldEnum = "custom_assigned_pds"
	PDUtilization     CustomSortFieldEnum = "custom_pd_utilization"
	RName             CustomSortFieldEnum = "custom_rname"
)

// Prepare an order expression based on table name, sortField and sortDir.
// If sortField does not start with a table name and . then it is prepended.
// If sortDir is DESC then NULLS LAST is added, if it is ASC then NULLS FIRST
// is added. Without that records with NULLs in sortField would not be included
// in the result.
// It returns also DISTINCT ON expression that is often used in queries together with ORDER BY.
// DISTINCT ON must contain the same fields that are used for sorting.
func prepareOrderExpr(tableName string, sortField string, sortDir SortDirEnum) (orderExpr string, distinctOnExpr string) {
	orderExpr = ""
	var dirExpr string
	switch sortDir {
	case SortDirDesc:
		dirExpr = "DESC NULLS LAST"
	default:
		dirExpr = "ASC NULLS FIRST"
	}
	escapedTableName := "\"" + tableName + "\""
	distinctOnExpr = escapedTableName + ".id"
	if sortField != "" {
		if !strings.Contains(sortField, ".") {
			if !strings.HasPrefix(sortField, "custom_") {
				orderExpr += escapedTableName + "."
				if strings.ToLower(sortField) != "id" {
					distinctOnExpr += ", " + escapedTableName + "." + sortField
				}
			} else {
				// This sort field requires custom handling.
				statsExpr := ""
				switch CustomSortFieldEnum(sortField) {
				case TotalPDs:
					statsExpr = fmt.Sprintf("(%s.stats->>'total-pds')::numeric", escapedTableName)
				case AssignedPDs:
					statsExpr = fmt.Sprintf("(%s.stats->>'assigned-pds')::numeric", escapedTableName)
				case PDUtilization:
					statsExpr = fmt.Sprintf("%s.pd_utilization", escapedTableName)
				}
				switch CustomSortFieldEnum(sortField) {
				case TotalAddresses:
					sortField = fmt.Sprintf("COALESCE(%[1]s.stats->>'total-nas', %[1]s.stats->>'total-addresses')::numeric", escapedTableName)
					distinctOnExpr += ", " + sortField
				case AssignedAddresses:
					sortField = fmt.Sprintf("COALESCE(%[1]s.stats->>'assigned-nas', %[1]s.stats->>'assigned-addresses')::numeric", escapedTableName)
					distinctOnExpr += ", " + sortField
				case TotalPDs, AssignedPDs, PDUtilization:
					familyExpr := fmt.Sprintf("%s.inet_family", escapedTableName)
					if tableName == "subnet" {
						familyExpr = fmt.Sprintf("family(%s.prefix)", escapedTableName)
					}
					sortField = fmt.Sprintf("%s %s, %s", familyExpr, dirExpr, statsExpr)
					distinctOnExpr += fmt.Sprintf(", %s, %s", familyExpr, statsExpr)
				case RName:
					sortField = fmt.Sprintf("%s.rname COLLATE \"C\"", escapedTableName)
					distinctOnExpr += fmt.Sprintf("%s.rname", escapedTableName)
				}
			}
		} else if strings.ToLower(sortField) != tableName+".id" && strings.ToLower(sortField) != escapedTableName+".id" {
			distinctOnExpr += ", " + sortField
		}
		orderExpr += sortField + " "
	} else {
		orderExpr = escapedTableName + ".id "
	}
	orderExpr += dirExpr
	return
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
