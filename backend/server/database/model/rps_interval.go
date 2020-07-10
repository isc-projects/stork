package dbmodel

import (
	"time"

	"github.com/go-pg/pg/v9"
	errors "github.com/pkg/errors"
)

// The number of responses a daemon sent during an interval of time
type RpsInterval struct {
	KeaDaemonID int64     `pg:",pk"` // ID of Kea daemon
	StartTime   time.Time `pg:",pk"` // beginning of this interval
	Duration    int64     // duration of this interval (seconds)
	Responses   int64     // number of responses in this interval
}

// Get all global statistics values.
func GetAllRpsIntervals(db *pg.DB) ([]*RpsInterval, error) {
	rpsIntervals := []*RpsInterval{}
	q := db.Model(&rpsIntervals)
	err := q.Select()
	if err != nil {
		return nil, errors.Wrapf(err, "problem with getting all RPS intervals")
	}

	return rpsIntervals, nil
}

// Add an interval to the database
func AddRpsInterval(db *pg.DB, rpsInterval *RpsInterval) error {
	err := db.Insert(rpsInterval)
	if err != nil {
		err = errors.Wrapf(err, "problem with inserting rpsInterval %+v", rpsInterval)
	}
	return err
}
