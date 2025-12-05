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
// Sorting by any of these custom sort fields requires more complex syntax of the ORDER BY and DISTINCT ON expressions.
// E.g., sorting by more than one column is required, or Postgresql function must be used in these expressions.
type CustomSortFieldEnum string

// Valid values of the enum.
const (
	SortFieldTotalAddresses    CustomSortFieldEnum = "custom_total_addresses"    // Used for sorting records in "subnet" and "shared_network" tables.
	SortFieldAssignedAddresses CustomSortFieldEnum = "custom_assigned_addresses" // Used for sorting records in "subnet" and "shared_network" tables.
	SortFieldTotalPDs          CustomSortFieldEnum = "custom_total_pds"          // Used for sorting records in "subnet" and "shared_network" tables.
	SortFieldAssignedPDs       CustomSortFieldEnum = "custom_assigned_pds"       // Used for sorting records in "subnet" and "shared_network" tables.
	SortFieldPDUtilization     CustomSortFieldEnum = "custom_pd_utilization"     // Used for sorting records in "subnet" and "shared_network" tables.
	SortFieldRName             CustomSortFieldEnum = "custom_rname"              // Used for sorting records in "zone" table.
	SortFieldEventText         CustomSortFieldEnum = "custom_text"               // Used for sorting records in "event" table.
	SortFieldEventLevel        CustomSortFieldEnum = "custom_level"              // Used for sorting records in "event" table.
)

// Prepare an order expression based on table name, sortField and sortDir.
// If sortField does not start with a table name and . then it is prepended.
// If sortDir is DESC then NULLS LAST is added, if it is ASC then NULLS FIRST
// is added. Without that records with NULLs in sortField would not be included
// in the result.
// It returns also DISTINCT ON expression that is often used in queries together with ORDER BY.
// DISTINCT ON must contain the same fields that are used for sorting.
// This function handles CustomSortFieldEnum sortFields (all of them have "custom_" prefix) in a special way.
// Sorting by any of these custom sort fields requires more complex syntax of the ORDER BY and DISTINCT ON expressions.
func prepareOrderAndDistinctExpr(tableName string, sortField string, sortDir SortDirEnum) (orderExpr string, distinctOnExpr string) {
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
				// Below cases are common for custom sorting in "subnet" and "shared_network" tables.
				// They compute statsExpr required for sorting by IPv6 prefix delegation related fields.
				case SortFieldTotalPDs:
					statsExpr = fmt.Sprintf("(%s.stats->>'total-pds')::numeric", escapedTableName)
				case SortFieldAssignedPDs:
					statsExpr = fmt.Sprintf("(%s.stats->>'assigned-pds')::numeric", escapedTableName)
				case SortFieldPDUtilization:
					statsExpr = fmt.Sprintf("%s.pd_utilization", escapedTableName)
				default:
					// NO-OP for other custom sort fields - statsExpr is not used.
				}
				switch CustomSortFieldEnum(sortField) {
				case SortFieldTotalAddresses:
					// Sort subnets and shared networks by total addresses no matter the IP v4/v6 family.
					sortField = fmt.Sprintf("COALESCE(%[1]s.stats->>'total-nas', %[1]s.stats->>'total-addresses')::numeric", escapedTableName)
					distinctOnExpr += ", " + sortField
				case SortFieldAssignedAddresses:
					// Sort subnets and shared networks by assigned addresses no matter the IP v4/v6 family.
					sortField = fmt.Sprintf("COALESCE(%[1]s.stats->>'assigned-nas', %[1]s.stats->>'assigned-addresses')::numeric", escapedTableName)
					distinctOnExpr += ", " + sortField
				case SortFieldTotalPDs, SortFieldAssignedPDs, SortFieldPDUtilization:
					// When sorting subnets and shared networks by IPv6 prefix delegation related statistics, sort by the IP v4/v6 family first.
					// This will handle IPv4 records depending on sorting order in a similar way to ASC NULLS FIRST/DESC NULLS LAST common sorting rule.
					familyExpr := fmt.Sprintf("%s.inet_family", escapedTableName)
					if tableName == "subnet" {
						familyExpr = fmt.Sprintf("family(%s.prefix)", escapedTableName)
					}
					sortField = fmt.Sprintf("%s %s, %s", familyExpr, dirExpr, statsExpr)
					distinctOnExpr += fmt.Sprintf(", %s, %s", familyExpr, statsExpr)
				case SortFieldRName:
					// When sorting DNS zones by rname field, use the C collation.
					sortField = fmt.Sprintf("%s.rname COLLATE \"C\"", escapedTableName)
					distinctOnExpr += fmt.Sprintf(", %s.rname", escapedTableName)
				case SortFieldEventText:
					// When sorting events by text, apply the second sort by created_at field.
					sortField = fmt.Sprintf("%[1]s.text %[2]s, %[1]s.created_at", escapedTableName, dirExpr)
					distinctOnExpr += fmt.Sprintf(", %s.text", escapedTableName)
				case SortFieldEventLevel:
					// When sorting events by level, apply the second sort by created_at field.
					sortField = fmt.Sprintf("%[1]s.level %[2]s, %[1]s.created_at", escapedTableName, dirExpr)
					distinctOnExpr += fmt.Sprintf(", %s.level", escapedTableName)
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
