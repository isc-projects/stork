package dbmodel

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/go-pg/pg/v10"
	pkgerrors "github.com/pkg/errors"
	dbops "isc.org/stork/server/database"
)

// Representation of the config changes scheduled by the config
// manager (see server/apps). Each scheduled config change includes
// a deadline (timestamp) indicating when this config change should
// be committed (sent to the configured daemon). A change can
// comprise one or multiple updates. For example: a single change
// can cause creation of a host reservation and an update of an
// existing subnet.
type ScheduledConfigChange struct {
	ID         int64
	CreatedAt  time.Time
	DeadlineAt time.Time

	UserID int64
	User   *SystemUser `pg:"rel:has-one"`

	Updates []*ConfigUpdate `pg:",json_use_number"`

	Executed bool
	Error    string
}

// Represents a single config update belonging to a config change.
type ConfigUpdate struct {
	// Type of the configured daemon, e.g. "kea".
	Target AppType
	// Type of the operation to perform, e.g. "host_add".
	Operation string
	// Identifiers of the daemons affected by the update. For example,
	// a host reservation can be shared by two daemons.
	DaemonIDs []int64
	// Holds information required to apply the config update, e.g.
	// commands to be sent to the configured server, information to be
	// inserted into the database etc. The contents of this field are
	// specific to the performed operation.
	Recipe *json.RawMessage
}

// Checks if any of the updates pertain to Kea.
func (c ScheduledConfigChange) HasKeaUpdates() bool {
	for _, update := range c.Updates {
		if update.Target == "kea" {
			return true
		}
	}
	return false
}

// Creates new config update instance.
func NewConfigUpdate(target AppType, operation string, daemonIDs ...int64) *ConfigUpdate {
	return &ConfigUpdate{
		Target:    target,
		Operation: operation,
		DaemonIDs: daemonIDs,
	}
}

// Inserts scheduled config change into the database in the transaction.
func addScheduledConfigChange(tx *pg.Tx, scc *ScheduledConfigChange) (err error) {
	if _, err = tx.Model(scc).Insert(); err != nil {
		err = pkgerrors.Wrapf(err, "problem with adding scheduled config change")
	}
	return
}

// Inserts a scheduled config change into the database.
func AddScheduledConfigChange(dbi dbops.DBI, scc *ScheduledConfigChange) error {
	if db, ok := dbi.(*pg.DB); ok {
		return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return addScheduledConfigChange(tx, scc)
		})
	}
	return addScheduledConfigChange(dbi.(*pg.Tx), scc)
}

// Returns all scheduled config changes.
func GetScheduledConfigChanges(dbi dbops.DBI) ([]ScheduledConfigChange, error) {
	var changes []ScheduledConfigChange
	err := dbi.Model(&changes).
		OrderExpr("deadline_at ASC").
		Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return changes, nil
		}
		err = pkgerrors.Wrapf(err, "problem with getting scheduled config changes")
	}
	return changes, err
}

// Returns scheduled and not executed config changes which deadline has expired.
func GetDueConfigChanges(dbi dbops.DBI) ([]ScheduledConfigChange, error) {
	var changes []ScheduledConfigChange
	err := dbi.Model(&changes).
		OrderExpr("deadline_at ASC").
		Where("executed = ?", false).
		Where("deadline_at < now() at time zone 'UTC'").
		Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return changes, nil
		}
		err = pkgerrors.Wrapf(err, "problem with getting due config changes")
	}
	return changes, err
}

// Marks specified config change as executed. Such changes are no longer
// returned in queries for due config changes. The errtext specifies an optional
// text describing an error that occurred during the config change execution.
func SetScheduledConfigChangeExecuted(dbi dbops.DBI, changeID int64, errtext string) error {
	change := &ScheduledConfigChange{
		ID:       changeID,
		Executed: true,
		Error:    errtext,
	}
	result, err := dbi.Model(change).
		Column("executed").
		Column("error").
		WherePK().
		Update()
	if err != nil {
		return pkgerrors.Wrapf(err, "problem with updating config change %d", changeID)
	}
	if result.RowsAffected() <= 0 {
		return pkgerrors.Wrapf(ErrNotExists, "config change with id %d does not exist", changeID)
	}
	return nil
}

// Returns time in seconds to next scheduled config change.
func GetTimeToNextScheduledConfigChange(dbi dbops.DBI) (time.Duration, bool, error) {
	var tm struct {
		Duration *float64
	}
	_, err := dbi.QueryOne(&tm,
		`SELECT MIN(EXTRACT(EPOCH FROM(deadline_at - now() at time zone 'UTC'))) AS duration
         FROM scheduled_config_change
         WHERE executed = FALSE`)
	if err != nil {
		return 0, false, pkgerrors.Wrapf(err, "problem with getting time to next config change")
	}
	if tm.Duration == nil {
		// Scheduled config changes do not exist.
		return 0, false, nil
	}
	return time.Duration(*tm.Duration) * time.Second, true, err
}

// Deletes selected scheduled config change from the database.
func DeleteScheduledConfigChange(dbi dbops.DBI, changeID int64) error {
	scc := &ScheduledConfigChange{
		ID: changeID,
	}
	result, err := dbi.Model(scc).WherePK().Delete()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem with deleting scheduled config change with id %d", changeID)
	} else if result.RowsAffected() <= 0 {
		err = pkgerrors.Wrapf(ErrNotExists, "scheduled config change with id %d does not exist", changeID)
	}
	return err
}
