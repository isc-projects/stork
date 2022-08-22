package dbmodel

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-pg/pg/v10"
	pkgerrors "github.com/pkg/errors"
	dbops "isc.org/stork/server/database"
)

// Structure representing an enabling or disabling of a single config checker.
type ConfigCheckerPreference struct {
	DaemonID    *int64
	CheckerName string
	Enabled     bool `pg:",use_zero"`
}

// Check if the preference is a global - it isn't assigned to any specific
// daemon.
func (p *ConfigCheckerPreference) IsGlobal() bool {
	return p.DaemonID == nil
}

// Returns the daemon ID related to this preference. If it is a global preference,
// zero is returned.
func (p *ConfigCheckerPreference) GetDaemonID() int64 {
	if p.DaemonID == nil {
		return 0
	}
	return *p.DaemonID
}

// Returns the string representation of the preference.
func (p *ConfigCheckerPreference) String() string {
	state := "disabled"
	if p.Enabled {
		state = "enabled"
	}

	if p.IsGlobal() {
		return fmt.Sprintf("%s checker is globally %s", p.CheckerName, state)
	}
	return fmt.Sprintf("%s checker is %s for %d daemon ID", p.CheckerName, state, p.GetDaemonID())
}

// Constructs the global checker preference.
func NewGlobalConfigCheckerPreference(checkerName string) *ConfigCheckerPreference {
	return &ConfigCheckerPreference{
		DaemonID:    nil,
		CheckerName: checkerName,
		Enabled:     false,
	}
}

// Constructs the checker preference for a specific daemon.
func NewDaemonConfigCheckerPreference(daemonID int64, checkerName string, enabled bool) *ConfigCheckerPreference {
	return &ConfigCheckerPreference{
		DaemonID:    &daemonID,
		CheckerName: checkerName,
		Enabled:     enabled,
	}
}

// Returns all config checker preferences.
func GetAllCheckerPreferences(dbi dbops.DBI) (preferences []*ConfigCheckerPreference, err error) {
	err = dbi.Model(&preferences).
		Order("checker_name").
		Select()

	if err != nil && !errors.Is(err, pg.ErrNoRows) {
		err = pkgerrors.Wrap(err, "problem selecting all checker preferences")
		return
	}

	return preferences, nil
}

// Returns config checker preferences for a given daemon. If the daemon ID is
// zero returns only global checker preferences.
func GetCheckerPreferences(dbi dbops.DBI, daemonID int64) (preferences []*ConfigCheckerPreference, err error) {
	q := dbi.Model(&preferences)
	if daemonID == 0 {
		q = q.Where("daemon_id IS NULL")
	} else {
		q = q.Where("daemon_id = ?", daemonID)
	}
	err = q.Order("checker_name").
		Select()

	if err != nil && !errors.Is(err, pg.ErrNoRows) {
		message := "problem selecting global checker preferences"
		if daemonID != 0 {
			message = fmt.Sprintf("problem selecting checker preferences for a daemon with ID: %d", daemonID)
		}

		err = pkgerrors.Wrap(err, message)
		return
	}

	return preferences, nil
}

// Adds or updates the config checker preferences.
func addOrUpdateCheckerPreferences(dbi dbops.DBI, preferences []*ConfigCheckerPreference) error {
	var daemonPreferences []*ConfigCheckerPreference
	var globalPreferences []*ConfigCheckerPreference

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

// Deletes the config checker preferences.
func deleteCheckerPreferences(dbi dbops.DBI, preferences []*ConfigCheckerPreference) error {
	if len(preferences) == 0 {
		return nil
	}

	for _, preference := range preferences {
		q := dbi.Model((*ConfigCheckerPreference)(nil))
		if preference.DaemonID != nil {
			q = q.Where("daemon_id = (?) AND checker_name = (?)", preference.DaemonID, preference.CheckerName)
		} else {
			q = q.Where("daemon_id IS NULL AND checker_name = (?)", preference.CheckerName)
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
func commitCheckerPreferences(dbi dbops.DBI, updates []*ConfigCheckerPreference, deletes []*ConfigCheckerPreference) error {
	err := addOrUpdateCheckerPreferences(dbi, updates)
	if err != nil {
		return err
	}
	return deleteCheckerPreferences(dbi, deletes)
}

// Commits the changes in config checker preferences. It accepts a list of
// preferences to add or update and a list of preferences to delete. The transaction
// is created if needed.
func CommitCheckerPreferences(dbi dbops.DBI, updates []*ConfigCheckerPreference, deletes []*ConfigCheckerPreference) error {
	if db, ok := dbi.(*pg.DB); ok {
		err := db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return commitCheckerPreferences(dbi, updates, deletes)
		})
		return err
	}
	return commitCheckerPreferences(dbi, updates, deletes)
}
