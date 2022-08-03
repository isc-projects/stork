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

// Commits changes in the global exclusions of the config checkers into DB
// creating a transaction if necessary.
func CommitGloballyExcludedCheckers(dbi dbops.DBI, exclusions []*ConfigCheckerGlobalExclude) error {
	if db, ok := dbi.(*pg.DB); ok {
		return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return commitGloballyExcludedCheckers(dbi, exclusions)
		})
	}
	return commitGloballyExcludedCheckers(dbi, exclusions)
}

// Commits changes in the global exclusions of the config checkers into DB.
func commitGloballyExcludedCheckers(dbi dbops.DBI, exclusions []*ConfigCheckerGlobalExclude) error {
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
	err = addGloballyExcludedCheckers(dbi, newExclusions)
	return err
}

// Adds a global exclusion of the config checker.
func addGloballyExcludedCheckers(dbi dbops.DBI, exclusions []*ConfigCheckerGlobalExclude) error {
	if len(exclusions) == 0 {
		return nil
	}
	_, err := dbi.Model(&exclusions).Insert()
	return pkgerrors.Wrap(err, "problem inseting global exclusions of checkers")
}

// Deletes all global exclusions of the config checker except these from a given list of IDs.
func deleteAllGloballyExcludedChekers(dbi dbops.DBI, excludedIDs []int64) error {
	if len(excludedIDs) == 0 {
		return nil
	}
	_, err := dbi.Model((*ConfigCheckerGlobalExclude)(nil)).
		Where("id NOT IN (?)", pg.In(excludedIDs)).
		Delete()
	return pkgerrors.Wrap(err, "problem deleting global exclusions of checkers")
}

// Returns the daemon preferences of config checkersconfig checker.
func GetCheckerPreferencesByDaemon(dbi dbops.DBI, daemonID int64) (preferences []*ConfigCheckerDaemonPreference, err error) {
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
func addCheckerPreferencesForDaemon(db *pg.DB, preferences []*ConfigCheckerDaemonPreference) error {
	if len(preferences) == 0 {
		return nil
	}
	_, err := db.Model(&preferences).Insert()
	return pkgerrors.Wrap(err, "problem inserting checker preferences")
}

// Updates the daemon preferences of config checkers.
func updateCheckerPreferencesForDaemon(db *pg.DB, preferences []*ConfigCheckerDaemonPreference) error {
	if len(preferences) == 0 {
		return nil
	}
	_, err := db.Model(&preferences).WherePK().Update()
	return pkgerrors.Wrap(err, "problem updating checker preferences")
}

// Deletes the daemon preferences of config checkersconfig checker.
func deleteCheckerPreferencesForDaemon(db *pg.DB, preferences []*ConfigCheckerDaemonPreference) error {
	if len(preferences) == 0 {
		return nil
	}
	_, err := db.Model(&preferences).Delete()
	return pkgerrors.Wrap(err, "problem deleting checker preferences")
}

// Combines the global exclusions with the daemon preferences of config checkers.
// Returns the names of excluded checkers.
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
