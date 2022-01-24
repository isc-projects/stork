package dbmodel

import (
	"math/big"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/types"

	errors "github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Custom support for decimal/numeric in Go-PG.
// It is dedicated to store integer-only numbers. The Postgres decimal/numeric
// type must be defined with scale equals to 0, e.g.: pg:"type:decimal(60,0)".
// See: https://github.com/go-pg/pg/blob/v10/example_custom_test.go
type IntegerDecimal struct {
	big.Int
}

// Interface check for serialization.
var _ types.ValueAppender = (*IntegerDecimal)(nil)

// Custom big.Int serializing to the database record.
func (d IntegerDecimal) AppendValue(b []byte, quote int) ([]byte, error) {
	if quote == 1 {
		b = append(b, '\'')
	}

	b = append(b, []byte(d.String())...)
	if quote == 1 {
		b = append(b, '\'')
	}
	return b, nil
}

// Interface check for deserialization.
var _ types.ValueScanner = (*IntegerDecimal)(nil)

// Custom decimal/numeric parsing to big.Int.
func (d *IntegerDecimal) ScanValue(rd types.Reader, n int) error {
	if n <= 0 {
		d.Int = *big.NewInt(0)
		return nil
	}

	tmp, err := rd.ReadFullTemp()
	if err != nil {
		return err
	}

	_, ok := d.Int.SetString(string(tmp), 10)
	if !ok {
		return errors.New("invalid decimal")
	}

	return nil
}

// Represents a statistic held in statistic table in the database.
type Statistic struct {
	Name string `pg:",pk"`
	// The maximal IPv6 prefix is 128.
	// How many decimal digits we need to store the total number of available addresses?
	// 2^128 = 10^x
	// 10 = 2^y
	// y = log2 10
	// 2^128 = 2^(x*y)
	// x*y = 128
	// x = 128 / y
	// x = 38.53 = 39
	// We need up to 39 digits to save the capacity of single StorkIPv6 network.
	//
	// But how many subnets may be handled by a single Kea instance?
	// Kea stores the subnet ID in uint32. It is 2^32 = 10^10 unique values.
	// Then we need up to 49 digits to save the capacity of all networks from a single Kea instance.
	//
	// But how many Kea instances may be handled by a single Stork instance?
	// Machine ID has an int64 data type, but Stork uses only positive values. In practice the range
	// is the same as for uint32. It is 10^10 unique values.
	// Then we need up to 59 digits to save the capacity of all subnets handled by Stork at the same time.
	Value IntegerDecimal `pg:"type:decimal(60,0)"`
}

// Initialize global statistics in db. If new statistic needs to be added then add it to statsList list
// and it will be automatically added to db here in this function.
func InitializeStats(db *pg.DB) error {
	// list of all stork global statistics
	statsList := []Statistic{
		{Name: "assigned-addresses"},
		{Name: "total-addresses"},
		{Name: "declined-addresses"},
		{Name: "assigned-nas"},
		{Name: "total-nas"},
		{Name: "assigned-pds"},
		{Name: "total-pds"},
		{Name: "declined-nas"},
	}

	// Check if there are new statistics vs existing ones. Add new ones to DB.
	_, err := db.Model(&statsList).OnConflict("DO NOTHING").Insert()
	if err != nil {
		err = errors.Wrapf(err, "problem with inserting default statistics")
	}
	return err
}

// Get all global statistics values.
func GetAllStats(db *pg.DB) (map[string]*big.Int, error) {
	statsList := []*Statistic{}
	q := db.Model(&statsList)
	err := q.Select()
	if err != nil {
		return nil, errors.Wrapf(err, "problem with getting all statistics")
	}

	statsMap := make(map[string]*big.Int)
	for _, s := range statsList {
		statsMap[s.Name] = &s.Value.Int
	}

	return statsMap, nil
}

// Set a list of global statistics.
func SetStats(db *pg.DB, statsMap map[string]*big.Int) error {
	statsList := []*Statistic{}
	for s, v := range statsMap {
		if v == nil {
			return errors.New("statistic value cannot be nil")
		}
		stat := &Statistic{Name: s, Value: IntegerDecimal{Int: *v}}
		statsList = append(statsList, stat)
	}

	q := db.Model(&statsList)
	_, err := q.Update()
	if err != nil {
		log.Printf("SET STATS ERR: %+v", err)
		return errors.Wrapf(err, "problem with setting statistics")
	}
	return nil
}
