package dbmodel

import (
	"errors"

	"github.com/go-pg/pg/v10"
	pkgerrors "github.com/pkg/errors"
	dbops "isc.org/stork/server/database"
)

// Structure representing an exclusion or inclusion of a single config checker
// for a specific daemon.
type ConfigDaemonCheckerPreference struct {
	DaemonID    *int64 `pg:",pk"`
	CheckerName string `pg:",pk"`
	Excluded    bool
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
	err = dbi.Model(&preferences).
		Where("config_daemon_checker_preference.daemon_id = ?", daemonID).
		Select()

	if err != nil && !errors.Is(err, pg.ErrNoRows) {
		err = pkgerrors.Wrap(err, "problem selecting checker preferences for a given daemon")
		return
	}

	return preferences, nil
}

// Adds the daemon preferences of config checkers.
func AddDaemonCheckerPreferences(dbi dbops.DBI, preferences []*ConfigDaemonCheckerPreference) error {
	if len(preferences) == 0 {
		return nil
	}
	_, err := dbi.Model(&preferences).Insert()
	return pkgerrors.Wrap(err, "problem inserting checker preferences")
}

// Updates the daemon preferences of config checkers.
func UpdateDaemonCheckerPreferences(dbi dbops.DBI, preferences []*ConfigDaemonCheckerPreference) error {
	if len(preferences) == 0 {
		return nil
	}
	_, err := dbi.Model(&preferences).WherePK().Update()
	return pkgerrors.Wrap(err, "problem updating checker preferences")
}

// Deletes all daemon preferences of config checkers for a given daemon except
// these from a given list of IDs.
func DeleteDaemonCheckerPreferences(dbi dbops.DBI, preferences []*ConfigDaemonCheckerPreference) error {
	if len(preferences) == 0 {
		return nil
	}
	_, err := dbi.Model(&preferences).Delete()
	return pkgerrors.Wrap(err, "problem deleting checker preferences")
}
