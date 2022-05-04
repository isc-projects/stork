package dbmodel

import (
	"time"

	"github.com/go-pg/pg/v10"
	errors "github.com/pkg/errors"
)

// The number of responses a daemon sent during an interval of time.
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
	q = q.Order("kea_daemon_id", "start_time")
	err := q.Select()
	if err != nil {
		return nil, errors.Wrapf(err, "problem getting all RPS intervals")
	}

	return rpsIntervals, nil
}

// Returns an array of the total RPS for a given daemon within a given time frame
// One element for the given daemon id where:
// RpsInterval.StartTime = 0 (unused)
// RpsInterval.Responses = total of number of responses
// RpsInterval.Duration = total of the interval durations.
func GetTotalRpsOverIntervalForDaemon(db *pg.DB, startTime time.Time, endTime time.Time, daemonID int64) ([]*RpsInterval, error) {
	rpsTotals := []*RpsInterval{}

	q := db.Model(&rpsTotals)
	q = q.Column("kea_daemon_id")
	q = q.ColumnExpr("sum(responses) as responses")
	q = q.ColumnExpr("sum(duration) as duration")
	q = q.Group("kea_daemon_id")
	q = q.Where("kea_daemon_id = ? and start_time >= ? and start_time <= ?", daemonID, startTime, endTime)

	err := q.Select()
	if err != nil {
		return nil, errors.Wrapf(err, "problem getting RPS interval for daemon: %d", daemonID)
	}

	return rpsTotals, nil
}

// Add an interval to the database.
func AddRpsInterval(db *pg.DB, rpsInterval *RpsInterval) error {
	_, err := db.Model(rpsInterval).Insert()
	if err != nil {
		err = errors.Wrapf(err, "problem inserting rpsInterval %+v", rpsInterval)
	}
	return err
}

// Delete all records whose start_time is older than a given time.
func AgeOffRpsInterval(db *pg.DB, startTime time.Time) error {
	// Delete records.
	_, err := db.Model(&RpsInterval{}).Where("start_time < ?", startTime).Delete()

	if err == nil {
		err = errors.Wrapf(err, "problem deleting from rpsInterval")
	}
	return err
}
