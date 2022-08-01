package dbmodel

import (
	"errors"

	"github.com/go-pg/pg/v10"
	pkgerrors "github.com/pkg/errors"
)

// Structure representing a global exclusion of a single config checker.
type ConfigCheckerGlobalExclude struct {
	ID          int64
	CheckerName string
}

// Structure representing an exclusion or inclusion of a single config checker
// for a specific daemon.
type ConfigCheckerDaemonPreference struct {
	ID          int64
	DaemonID    int64
	CheckerName string
	Excluded    bool
}

// Returns all global exclusions of the review chckers.
func GetGloballyExcludedCheckers(db *pg.DB) (exclusions []*ConfigCheckerGlobalExclude, err error) {
	err = db.Model(&exclusions).Select()

	if err != nil && !errors.Is(err, pg.ErrNoRows) {
		err = pkgerrors.Wrap(err, "problem selecting global exclusions of checkers")
		return
	}

	return exclusions, nil
}

// Adds a global exclusion of the review checker.
func AddGloballyExcludedCheckers(db *pg.DB, exclusions []*ConfigCheckerGlobalExclude) error {
	if len(exclusions) == 0 {
		return nil
	}
	_, err := db.Model(&exclusions).Insert()
	return pkgerrors.Wrap(err, "problem inseting global exclusions of checkers")
}

// Removes a global exclusion of the review checker.
func RemoveGloballyExcludedChekers(db *pg.DB, exclusions []*ConfigCheckerGlobalExclude) error {
	_, err := db.Model(&exclusions).Delete()
	return pkgerrors.Wrap(err, "problem deleting global exclusions of checkers")
}

// Returns the preferences of including/excluding review checker for a specific daemon.
func GetCheckerPreferencesByDaemon(db *pg.DB, daemonID int64) (preferences []*ConfigCheckerDaemonPreference, err error) {
	err = db.Model(&preferences).
		Where("config_checker_daemon_preference.daemon_id = ?", daemonID).
		Select()

	if err != nil && !errors.Is(err, pg.ErrNoRows) {
		err = pkgerrors.Wrap(err, "problem selecting checker preferences for a given daemon")
		return
	}

	return preferences, nil
}

// Adds the preferences of including/excluding review checkers for a specific daemon.
func AddCheckerPreferencesForDaemon(db *pg.DB, preferences []*ConfigCheckerDaemonPreference) error {
	_, err := db.Model(preferences).Insert()
	return pkgerrors.Wrap(err, "problem inserting checker preferences")
}

// Updates the preferences of including/excluding review checkers for a specific daemon.
func UpdateCheckerPreferencesForDaemon(db *pg.DB, int64, preferences []*ConfigCheckerDaemonPreference) error {
	_, err := db.Model(preferences).WherePK().Update()
	return pkgerrors.Wrap(err, "problem updating checker preferences")
}

// Removes the preferences of including/excluding review checker for a specific daemon.
func RemoveCheckerPreferencesForDaemon(db *pg.DB, preferences []*ConfigCheckerDaemonPreference) error {
	_, err := db.Model(preferences).WherePK().Delete()
	return pkgerrors.Wrap(err, "problem deleting checker preferences")
}

// Combines the global exclusions with the including/excluding preferences for
// a specific daemon. Returns the names of excluded checkers.
func MergeExcludedCheckerNames(globalExcludes []*ConfigCheckerGlobalExclude, daemonPreferences []*ConfigCheckerDaemonPreference) []string {
	daemonExcludes := make(map[string]bool)

	for _, globalExclude := range globalExcludes {
		daemonExcludes[globalExclude.CheckerName] = true
	}

	for _, preference := range daemonPreferences {
		daemonExcludes[preference.CheckerName] = preference.Excluded
	}

	var names []string
	for name, isExcluded := range daemonExcludes {
		if isExcluded {
			names = append(names, name)
		}
	}
	return names
}
