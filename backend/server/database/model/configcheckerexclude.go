package dbmodel

import (
	"context"
	"errors"

	"github.com/go-pg/pg/v10"
	pkgerrors "github.com/pkg/errors"
	dbops "isc.org/stork/server/database"
)

// Structure representing a global exclusion of a single config checker.
type ConfigCheckerGlobalExclude struct {
	ID          int64
	CheckerName string
}

// Structure representing an exclusion or inclusion of a single config checker
// for a specific daemon.
type ConfigDaemonCheckerPreference struct {
	ID          int64
	DaemonID    int64
	CheckerName string
	Excluded    bool
}

// Returns all global exclusions of the review chckers.
func GetGloballyExcludedConfigCheckers(db *pg.DB) (exclusions []*ConfigCheckerGlobalExclude, err error) {
	err = db.Model(&exclusions).Select()

	if err != nil && !errors.Is(err, pg.ErrNoRows) {
		err = pkgerrors.Wrap(err, "problem selecting global exclusions of checkers")
		return
	}

	return exclusions, nil
}

// Commits changes in the global exclusions of the config checkers into DB
// creating a transaction if necessary.
func CommitGloballyExcludedConfigCheckers(dbi dbops.DBI, exclusions []*ConfigCheckerGlobalExclude) error {
	if db, ok := dbi.(*pg.DB); ok {
		return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return commitGloballyExcludedConfigCheckers(dbi, exclusions)
		})
	}
	return commitGloballyExcludedConfigCheckers(dbi, exclusions)
}

// Commits changes in the global exclusions of the config checkers into DB.
func commitGloballyExcludedConfigCheckers(dbi dbops.DBI, exclusions []*ConfigCheckerGlobalExclude) error {
	var newExclusions []*ConfigCheckerGlobalExclude
	var existingExclusionIDs []int64
	for _, exclusion := range exclusions {
		if exclusion.ID != 0 {
			existingExclusionIDs = append(existingExclusionIDs, exclusion.ID)
		} else {
			newExclusions = append(newExclusions, exclusion)
		}
	}

	// Deletes old exclusions
	err := deleteAllGloballyExcludedChekers(dbi, existingExclusionIDs)
	if err != nil {
		return err
	}

	// Insert new exclusions
	err = addGloballyExcludedConfigCheckers(dbi, newExclusions)
	return err
}

// Adds a global exclusion of the config checker.
func addGloballyExcludedConfigCheckers(dbi dbops.DBI, exclusions []*ConfigCheckerGlobalExclude) error {
	if len(exclusions) == 0 {
		return nil
	}
	_, err := dbi.Model(&exclusions).Insert()
	return pkgerrors.Wrap(err, "problem inseting global exclusions of checkers")
}

// Deletes all global exclusions of the config checkers except these from a
// given list of IDs.
func deleteAllGloballyExcludedChekers(dbi dbops.DBI, excludedIDs []int64) error {
	q := dbi.Model((*ConfigCheckerGlobalExclude)(nil))
	if len(excludedIDs) != 0 {
		q = q.Where("id NOT IN (?)", pg.In(excludedIDs))
	} else {
		// Deletes all entries. Where clause is mandatory.
		q = q.Where("1 = 1")
	}
	_, err := q.Delete()
	return pkgerrors.Wrap(err, "problem deleting global exclusions of checkers")
}

// Returns the daemon preferences of config checker.
func GetDaemonCheckerPreferences(dbi dbops.DBI, daemonID int64) (preferences []*ConfigDaemonCheckerPreference, err error) {
	err = dbi.Model(&preferences).
		Where("config_checker_daemon_preference.daemon_id = ?", daemonID).
		Select()

	if err != nil && !errors.Is(err, pg.ErrNoRows) {
		err = pkgerrors.Wrap(err, "problem selecting checker preferences for a given daemon")
		return
	}

	return preferences, nil
}

// Adds the daemon preferences of config checkers.
func addDaemonCheckerPreferences(dbi dbops.DBI, preferences []*ConfigDaemonCheckerPreference) error {
	if len(preferences) == 0 {
		return nil
	}
	_, err := dbi.Model(&preferences).Insert()
	return pkgerrors.Wrap(err, "problem inserting checker preferences")
}

// Updates the daemon preferences of config checkers.
func updateDaemonCheckerPreferences(dbi dbops.DBI, preferences []*ConfigDaemonCheckerPreference) error {
	if len(preferences) == 0 {
		return nil
	}
	_, err := dbi.Model(&preferences).WherePK().Update()
	return pkgerrors.Wrap(err, "problem updating checker preferences")
}

// Deletes all daemon preferences of config checkers for a given daemon except
// these from a given list of IDs.
func deleteAllDaemonCheckerPreferences(dbi dbops.DBI, daemonID int64, excludedIDs []int64) error {
	q := dbi.Model((*ConfigDaemonCheckerPreference)(nil)).
		Where("daemon_id = (?)", daemonID)
	if len(excludedIDs) != 0 {
		q = q.Where("id NOT IN (?)", pg.In(excludedIDs))
	}
	_, err := q.Delete()
	return pkgerrors.Wrap(err, "problem deleting checker preferences")
}

// Commits changes in the daemon preferences of the config checkers into DB
// creating a transaction if necessary.
func CommitDaemonCheckerPreferences(dbi dbops.DBI, daemonID int64, preferences []*ConfigDaemonCheckerPreference) error {
	if db, ok := dbi.(*pg.DB); ok {
		return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return commitDaemonCheckerPreferences(dbi, daemonID, preferences)
		})
	}
	return commitDaemonCheckerPreferences(dbi, daemonID, preferences)
}

// Commits changes in the daemon preferences of the config checkers into DB.
func commitDaemonCheckerPreferences(dbi dbops.DBI, daemonID int64, preferences []*ConfigDaemonCheckerPreference) error {
	var newPreferences []*ConfigDaemonCheckerPreference
	var existingPreferences []*ConfigDaemonCheckerPreference
	var existingPreferenceIDs []int64
	for _, preference := range preferences {
		if preference.ID != 0 {
			existingPreferenceIDs = append(existingPreferenceIDs, preference.ID)
			existingPreferences = append(existingPreferences, preference)
		} else {
			newPreferences = append(newPreferences, preference)
		}
	}

	// Deletes old preferences.
	err := deleteAllDaemonCheckerPreferences(dbi, daemonID, existingPreferenceIDs)
	if err != nil {
		return err
	}

	// Updates existing preferences.
	err = updateDaemonCheckerPreferences(dbi, existingPreferences)
	if err != nil {
		return err
	}

	// Insert new preferences.
	err = addDaemonCheckerPreferences(dbi, newPreferences)
	return err
}

// Combines the global exclusions with the daemon preferences of config checkers.
// Returns the names of excluded checkers.
func MergeExcludedCheckerNames(globalExcludes []*ConfigCheckerGlobalExclude, daemonPreferences []*ConfigDaemonCheckerPreference) []string {
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
