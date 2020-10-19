package dbmodel

import (
	"github.com/go-pg/pg/v9"
	errors "github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Represents a statistic held in statistic table in the database.
type Statistic struct {
	Name  string `pg:",pk"`
	Value int64
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
func GetAllStats(db *pg.DB) (map[string]int64, error) {
	statsList := []*Statistic{}
	q := db.Model(&statsList)
	err := q.Select()
	if err != nil {
		return nil, errors.Wrapf(err, "problem with getting all statistics")
	}

	statsMap := make(map[string]int64)
	for _, s := range statsList {
		statsMap[s.Name] = s.Value
	}

	return statsMap, nil
}

// Set a list of global statistics.
func SetStats(db *pg.DB, statsMap map[string]int64) error {
	statsList := []*Statistic{}
	for s, v := range statsMap {
		stat := &Statistic{Name: s, Value: v}
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
