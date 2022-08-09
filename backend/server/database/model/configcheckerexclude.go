package dbmodel

import (
	"context"
	"errors"

	"github.com/go-pg/pg/v10"
	pkgerrors "github.com/pkg/errors"
	dbops "isc.org/stork/server/database"
)

// Structure representing an exclusion or inclusion of a single config checker
// for a specific daemon.
type ConfigDaemonCheckerPreference struct {
	DaemonID    *int64
	CheckerName string
	Excluded    bool `pg:",use_zero"`
}

func (p *ConfigDaemonCheckerPreference) IsGlobal() bool {
	return p.DaemonID == nil
}

func (p *ConfigDaemonCheckerPreference) GetDaemonID() int64 {
	if p.DaemonID == nil {
		return 0
	}
	return *p.DaemonID
}

// Returns the daemon preferences of config checker.
func GetDaemonCheckerPreferences(dbi dbops.DBI, daemonID *int64) (preferences []*ConfigDaemonCheckerPreference, err error) {
	q := dbi.Model(&preferences)
	if daemonID != nil {
		q = q.Where("daemon_id = ?", daemonID)
	} else {
		q = q.Where("daemon_id IS NULL")
	}
	err = q.Order("checker_name").
		Select()

	if err != nil && !errors.Is(err, pg.ErrNoRows) {
		err = pkgerrors.Wrap(err, "problem selecting checker preferences for a given daemon")
		return
	}

	return preferences, nil
}

// Adds or updates the daemon preferences of config checkers.
func AddOrUpdateDaemonCheckerPreferences(dbi dbops.DBI, preferences []*ConfigDaemonCheckerPreference) error {
	var daemonPreferences []*ConfigDaemonCheckerPreference
	var globalPreferences []*ConfigDaemonCheckerPreference

	for _, preference := range preferences {
		if preference.DaemonID != nil {
			daemonPreferences = append(daemonPreferences, preference)
		} else {
			globalPreferences = append(globalPreferences, preference)
		}
	}

	if len(daemonPreferences) != 0 {
		_, err := dbi.Model(&daemonPreferences).
			OnConflict("(daemon_id, checker_name) WHERE daemon_id IS NOT NULL DO UPDATE").
			Insert()
		if err != nil {
			return pkgerrors.Wrap(err, "problem inserting/updating daemon checker preferences")
		}
	}
	if len(globalPreferences) != 0 {
		_, err := dbi.Model(&globalPreferences).
			OnConflict("(checker_name) WHERE daemon_id IS NULL DO UPDATE").
			Insert()
		return pkgerrors.Wrap(err, "problem inserting/updating global checker preferences")
	}
	return nil
}

// Deletes all daemon preferences of config checkers for a given daemon except
// these from a given list of IDs.
func DeleteDaemonCheckerPreferences(dbi dbops.DBI, preferences []*ConfigDaemonCheckerPreference) error {
	if len(preferences) == 0 {
		return nil
	}

	for _, preference := range preferences {
		q := dbi.Model((*ConfigDaemonCheckerPreference)(nil))
		if preference.DaemonID != nil {
			q = q.Where("daemon_id = (?) AND checker_name = (?)", preference.DaemonID, preference.CheckerName)
		} else {
			q = q.Where("daemon_id IS NONE AND checker_name = (?)", preference.CheckerName)
		}
		_, err := q.Delete()
		if err != nil {
			return pkgerrors.Wrap(err, "problem deleting checker preference")
		}
	}

	return nil
}

// Commits the changes in config checker preferences. It accepts a list of
// preferences to add or update and a list of preferences to delete.
func commitDaemonCheckerPreferences(dbi dbops.DBI, updates []*ConfigDaemonCheckerPreference, deletes []*ConfigDaemonCheckerPreference) error {
	err := AddOrUpdateDaemonCheckerPreferences(dbi, updates)
	if err != nil {
		return err
	}
	return DeleteDaemonCheckerPreferences(dbi, deletes)
}

// Commits the changes in config checker preferences. It accepts a list of
// preferences to add or update and a list of preferences to delete. The transaction
// is created if needed.
func CommitDaemonCheckerPreferences(dbi dbops.DBI, updates []*ConfigDaemonCheckerPreference, deletes []*ConfigDaemonCheckerPreference) error {
	if db, ok := dbi.(*pg.DB); ok {
		err := db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return commitDaemonCheckerPreferences(dbi, updates, deletes)
		})
		return err
	}
	return commitDaemonCheckerPreferences(dbi, updates, deletes)
}
