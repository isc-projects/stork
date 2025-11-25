package dbmodel

import (
	"errors"
	"time"

	"github.com/go-pg/pg/v10"
	pkgerrors "github.com/pkg/errors"
	dbops "isc.org/stork/server/database"
)

// A structure reflecting information about a logger used by a daemon.
type LogTarget struct {
	ID        int64 // Logger ID
	Name      string
	Severity  string
	Output    string
	CreatedAt time.Time

	DaemonID int64
	Daemon   *Daemon `pg:"rel:has-one"`
}

// Retrieves log target from the database by id.
func GetLogTargetByID(db dbops.DBI, id int64) (*LogTarget, error) {
	logTarget := LogTarget{}
	err := db.Model(&logTarget).
		Relation("Daemon.Machine").
		Relation("Daemon.AccessPoints").
		Where("log_target.id = ?", id).
		Select()
	if errors.Is(err, pg.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, pkgerrors.Wrapf(err, "problem getting log target with ID %d", id)
	}
	return &logTarget, nil
}

// Deletes log targets by daemon ID except the log targets with IDs in keepIDs slice.
func deleteLogTargetsByDaemonIDExcept(db dbops.DBI, daemonID int64, keepIDs []int64) error {
	q := db.Model(&LogTarget{}).
		Where("log_target.daemon_id = ?", daemonID)
	if len(keepIDs) > 0 {
		q = q.Where("log_target.id NOT IN (?)", pg.In(keepIDs))
	}
	_, err := q.Delete()
	return pkgerrors.Wrapf(err, "problem deleting log targets for daemon ID %d, keeping IDs: %v", daemonID, keepIDs)
}

// Adds a log target to the database.
func addLogTarget(db dbops.DBI, logTarget *LogTarget) error {
	_, err := db.Model(logTarget).Insert()
	return pkgerrors.Wrapf(err, "problem adding log target %+v", logTarget)
}

// Updates a log target in the database.
func updateLogTarget(db dbops.DBI, logTarget *LogTarget) error {
	result, err := db.Model(logTarget).
		ExcludeColumn("created_at").
		WherePK().
		Update()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem updating log target %+v", logTarget)
	} else if result.RowsAffected() <= 0 {
		err = pkgerrors.Wrapf(ErrNotExists, "log target with ID %d does not exist", logTarget.ID)
	}
	return err
}

// Adds or updates a log target in the database.
// If the log target has no id yet, it means that it is not yet present in the
// database and should be inserted. Otherwise, it is updated.
func addOrUpdateLogTarget(db dbops.DBI, logTarget *LogTarget) error {
	if logTarget.ID > 0 {
		return updateLogTarget(db, logTarget)
	}
	return addLogTarget(db, logTarget)
}
