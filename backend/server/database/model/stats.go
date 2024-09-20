package dbmodel

import (
	"math/big"

	"github.com/go-pg/pg/v10"
	errors "github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Represents a statistic held in statistic table in the database.
type Statistic struct {
	Name SubnetStatsName `pg:",pk"`
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
	Value *integerDecimal `pg:"type:decimal(60,0)"`
}

// Initialize global statistics in db. If new statistic needs to be added then add it to statsList list
// and it will be automatically added to db here in this function.
func InitializeStats(db *pg.DB) error {
	// list of all stork global statistics
	statsList := []Statistic{
		{Name: SubnetStatsNameAssignedAddresses, Value: newIntegerDecimalZero()},
		{Name: SubnetStatsNameTotalAddresses, Value: newIntegerDecimalZero()},
		{Name: SubnetStatsNameDeclinedAddresses, Value: newIntegerDecimalZero()},
		{Name: SubnetStatsNameAssignedNAs, Value: newIntegerDecimalZero()},
		{Name: SubnetStatsNameTotalNAs, Value: newIntegerDecimalZero()},
		{Name: SubnetStatsNameAssignedPDs, Value: newIntegerDecimalZero()},
		{Name: SubnetStatsNameTotalPDs, Value: newIntegerDecimalZero()},
		{Name: SubnetStatsNameDeclinedNAs, Value: newIntegerDecimalZero()},
	}

	// Check if there are new statistics vs existing ones. Add new ones to DB.
	_, err := db.Model(&statsList).OnConflict("DO NOTHING").Insert()
	if err != nil {
		err = errors.Wrapf(err, "problem inserting default statistics")
	}
	return err
}

// Get all global statistics values.
func GetAllStats(db *pg.DB) (map[string]*big.Int, error) {
	statsList := []*Statistic{}
	q := db.Model(&statsList)
	err := q.Select()
	if err != nil {
		return nil, errors.Wrapf(err, "problem getting all statistics")
	}

	statsMap := make(map[string]*big.Int)
	for _, s := range statsList {
		var value *big.Int
		if s.Value != nil {
			value = &s.Value.Int
		}
		statsMap[s.Name] = value
	}

	return statsMap, nil
}

// Set a list of global statistics.
func SetStats(db *pg.DB, statsMap map[string]*big.Int) error {
	statsList := []*Statistic{}
	for s, v := range statsMap {
		stat := &Statistic{Name: s, Value: newIntegerDecimal(v)}
		statsList = append(statsList, stat)
	}

	q := db.Model(&statsList)
	_, err := q.Update()
	if err != nil {
		log.Printf("SET STATS ERR: %+v", err)
		return errors.Wrapf(err, "problem setting statistics")
	}
	return nil
}
