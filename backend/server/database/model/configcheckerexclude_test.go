package dbmodel

import (
	"testing"

	"github.com/go-pg/pg/v10"
	"github.com/stretchr/testify/require"
	dbtest "isc.org/stork/server/database/test"
)

// Test that the global exclusions of the config checkers are returned properly.
func TestGetGloballyExcludedCheckers(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_ = AddGloballyExcludedCheckers(db, []*ConfigCheckerGlobalExclude{
		{
			CheckerName: "foo",
		},
		{
			CheckerName: "bar",
		},
	})

	// Act
	exclusions, err := GetGloballyExcludedCheckers(db)

	// Assert
	require.NoError(t, err)
	require.Len(t, exclusions, 2)
	require.EqualValues(t, 1, exclusions[0].ID)
	require.EqualValues(t, "foo", exclusions[0].CheckerName)
	require.EqualValues(t, 2, exclusions[1].ID)
	require.EqualValues(t, "bar", exclusions[1].CheckerName)
}

// Test that an empty list is returned for missing the global exclusions of
// the config checkers.
func TestGetGloballyExcludedCheckersForEmptyData(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	exclusions, err := GetGloballyExcludedCheckers(db)

	// Assert
	require.NoError(t, err)
	require.Empty(t, exclusions)
}

// Test that the global exclusions of the config checkers are inserted properly.
func TestAddGloballyExcludedCheckers(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	err := AddGloballyExcludedCheckers(db, []*ConfigCheckerGlobalExclude{
		{
			CheckerName: "foo",
		},
		{
			CheckerName: "bar",
		},
	})

	// Assert
	require.NoError(t, err)
	exclusions, _ := GetGloballyExcludedCheckers(db)
	require.Len(t, exclusions, 2)
}

// Test that adding empty list of the global exclusions of the config checkers
// generates no error.
func TestAddGloballyExcludedCheckersForEmptyList(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	err := AddGloballyExcludedCheckers(db, []*ConfigCheckerGlobalExclude{})

	// Assert
	require.NoError(t, err)
	exclusions, _ := GetGloballyExcludedCheckers(db)
	require.Empty(t, exclusions)
}

// Test that adding the duplicated global exclusions of the config checkers
// generates an error.
func TestAddDuplicatedGloballyExcludedCheckersCausesError(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	err := AddGloballyExcludedCheckers(db, []*ConfigCheckerGlobalExclude{
		{CheckerName: "foo"},
		{CheckerName: "foo"},
	})

	// Assert
	require.Error(t, err)
	exclusions, _ := GetGloballyExcludedCheckers(db)
	require.Empty(t, exclusions)
}

// Test that adding the same global exclusions of the config checkers
// in separate queries generates an error on the second call.
func TestAddDoubleGloballyExcludedCheckersCausesError(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	err1 := AddGloballyExcludedCheckers(db, []*ConfigCheckerGlobalExclude{
		{CheckerName: "foo"},
	})
	err2 := AddGloballyExcludedCheckers(db, []*ConfigCheckerGlobalExclude{
		{CheckerName: "foo"},
	})

	// Assert
	require.NoError(t, err1)
	require.Error(t, err2)
	exclusions, _ := GetGloballyExcludedCheckers(db)
	require.Len(t, exclusions, 1)
}

// Test that the global exclusions of the config checkers are deleted properly.
func TestRemoveGloballyExcludedCheckers(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	_ = AddGloballyExcludedCheckers(db, []*ConfigCheckerGlobalExclude{
		{CheckerName: "foo"},
		{CheckerName: "bar"},
	})
	exclusions, _ := GetGloballyExcludedCheckers(db)

	// Act
	err := RemoveGloballyExcludedChekers(db, []*ConfigCheckerGlobalExclude{
		exclusions[1],
	})

	// Assert
	require.NoError(t, err)
	exclusions, _ = GetGloballyExcludedCheckers(db)
	require.Len(t, exclusions, 1)
	require.EqualValues(t, "foo", exclusions[0].CheckerName)
}

// Test that removing an empty list of the global exclusions of the config
// checkers generates no error.
func TestRemoveEmptyGloballyExcludedCheckers(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	err := RemoveGloballyExcludedChekers(db, []*ConfigCheckerGlobalExclude{})

	// Assert
	require.NoError(t, err)
}

// Test that removing the non-existent global exclusions of the config checkers
// generates no error.
func TestRemoveNonExistentGloballyExcludedCheckers(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	err := RemoveGloballyExcludedChekers(db, []*ConfigCheckerGlobalExclude{
		{ID: 42, CheckerName: "foo"},
	})

	// Assert
	require.NoError(t, err)
}

// Creates one demon entry in the database.
func addTestDaemon(db *pg.DB) (*Daemon, error) {
	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	if err != nil {
		return nil, err
	}

	app := &App{
		ID:   0,
		Type: AppTypeKea,
		Daemons: []*Daemon{
			NewKeaDaemon(DaemonNameDHCPv4, true),
		},
		MachineID: m.ID,
	}

	daemons, err := AddApp(db, app)
	if err != nil {
		return nil, err
	}
	daemon := daemons[0]
	daemon.App = app
	return daemon, nil
}

// Test that the including/excluding preferences for a specific daemon are
// added properly.
func TestAddCheckerPreferencesForDaemon(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemon, _ := addTestDaemon(db)

	// Act
	err := AddCheckerPreferencesForDaemon(db, []*ConfigCheckerDaemonPreference{
		{
			DaemonID:    daemon.ID,
			CheckerName: "foo",
			Excluded:    true,
		},
	})

	// Assert
	require.NoError(t, err)
	preferences, _ := GetCheckerPreferencesByDaemon(db, daemon.ID)
	require.Len(t, preferences, 1)
	require.EqualValues(t, "foo", preferences[0].CheckerName)
	require.EqualValues(t, daemon.ID, preferences[0].DaemonID)
	require.True(t, preferences[0].Excluded)
}

// Test that adding the empty list of the including/excluding preferences
// generates no error.
func TestAddEmptyListOfCheckerPreferencesForDaemon(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemon, _ := addTestDaemon(db)

	// Act
	err := AddCheckerPreferencesForDaemon(db, []*ConfigCheckerDaemonPreference{})

	// Assert
	require.NoError(t, err)
	preferences, _ := GetCheckerPreferencesByDaemon(db, daemon.ID)
	require.Empty(t, preferences)
}

// Test that adding a preference with already existing checker name causes
// an error.
func TestAddCheckerPreferencesWithDoubledName(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemon, _ := addTestDaemon(db)

	// Act
	err1 := AddCheckerPreferencesForDaemon(db, []*ConfigCheckerDaemonPreference{
		{
			DaemonID:    daemon.ID,
			CheckerName: "foo",
			Excluded:    true,
		},
	})
	err2 := AddCheckerPreferencesForDaemon(db, []*ConfigCheckerDaemonPreference{
		{
			DaemonID:    daemon.ID,
			CheckerName: "foo",
			Excluded:    true,
		},
	})

	// Assert
	require.NoError(t, err1)
	require.Error(t, err2)
}

// Test that the checker preferences for a specific daemon are updated properly.
func TestUpdateCheckerPreferencesForDaemon(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemon, _ := addTestDaemon(db)
	preferences := []*ConfigCheckerDaemonPreference{{
		DaemonID:    daemon.ID,
		CheckerName: "foo",
		Excluded:    true,
	}}
	_ = AddCheckerPreferencesForDaemon(db, preferences)

	// Act
	preferences[0].Excluded = false
	preferences[0].CheckerName = "bar"
	err := UpdateCheckerPreferencesForDaemon(db, preferences)

	// Assert
	require.NoError(t, err)
	preferences, _ = GetCheckerPreferencesByDaemon(db, daemon.ID)
	require.Len(t, preferences, 1)
	require.EqualValues(t, "bar", preferences[0].CheckerName)
	require.False(t, preferences[0].Excluded)
}

// Test that updating an empty list of the checker preferences for a specific
// daemon generates no error.
func TestUpdateEmptyListOfCheckerPreferencesForDaemon(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemon, _ := addTestDaemon(db)

	// Act
	err := UpdateCheckerPreferencesForDaemon(db, []*ConfigCheckerDaemonPreference{})

	// Assert
	require.NoError(t, err)
	preferences, _ := GetCheckerPreferencesByDaemon(db, daemon.ID)
	require.Empty(t, preferences)
}

// Test that updating the checker preferences with a name
// that already exists for a given daemon generates  error.
func TestUpdateCheckerPreferencesForDaemonWithDuplicatedName(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemon, _ := addTestDaemon(db)
	preferences := []*ConfigCheckerDaemonPreference{
		{
			DaemonID:    daemon.ID,
			CheckerName: "foo",
			Excluded:    true,
		},
		{
			DaemonID:    daemon.ID,
			CheckerName: "bar",
			Excluded:    true,
		},
	}
	_ = AddCheckerPreferencesForDaemon(db, preferences)

	// Act
	preferences[1].CheckerName = "foo"
	err := UpdateCheckerPreferencesForDaemon(db, []*ConfigCheckerDaemonPreference{
		preferences[1],
	})

	// Assert
	require.Error(t, err)
}

// Test that the including/excluding preferences for a specific daemon are
// removed properly.
func TestRemoveCheckerPreferencesForDaemon(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemon, _ := addTestDaemon(db)
	preferences := []*ConfigCheckerDaemonPreference{{
		DaemonID:    daemon.ID,
		CheckerName: "foo",
		Excluded:    true,
	}}
	_ = AddCheckerPreferencesForDaemon(db, preferences)

	// Act
	err := RemoveCheckerPreferencesForDaemon(db, preferences)

	// Assert
	require.NoError(t, err)
	preferences, _ = GetCheckerPreferencesByDaemon(db, daemon.ID)
	require.Empty(t, preferences)
}

// Test that removing an empty list of the including/excluding preferences for a specific daemon
// generates no error.
func TestRemoveEmptyListOfCheckerPreferencesForDaemon(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	err := RemoveCheckerPreferencesForDaemon(db, []*ConfigCheckerDaemonPreference{})

	// Assert
	require.NoError(t, err)
}

// Test that removing the daemon causes to wipe out all related checker preferences.
func TestRemoveDaemonAndRelatedCheckerPreferencesForDaemon(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemon, _ := addTestDaemon(db)
	preferences := []*ConfigCheckerDaemonPreference{{
		DaemonID:    daemon.ID,
		CheckerName: "foo",
		Excluded:    true,
	}}
	_ = AddCheckerPreferencesForDaemon(db, preferences)

	// Act
	err := DeleteApp(db, daemon.App)

	// Assert
	require.NoError(t, err)
	preferences, _ = GetCheckerPreferencesByDaemon(db, daemon.ID)
	require.Empty(t, preferences)
}

// Test that the excluded checker names are merged properly.
func TestMergeExcludedCheckerNames(t *testing.T) {
	// Arrange
	globalExcludes := []*ConfigCheckerGlobalExclude{
		{
			CheckerName: "foo",
		},
		{
			CheckerName: "bar",
		},
		{
			CheckerName: "baz",
		},
	}
	daemonPreferences := []*ConfigCheckerDaemonPreference{
		{
			// Duplicates global exclude.
			CheckerName: "foo",
			Excluded:    true,
		},
		{
			// Disables global exclude.
			CheckerName: "bar",
			Excluded:    false,
		},
		{
			// Daemon-specific exclude.
			CheckerName: "boz",
			Excluded:    true,
		},
		{
			// Daemon-specific include.
			CheckerName: "biz",
			Excluded:    false,
		},
	}

	// Act
	merged := MergeExcludedCheckerNames(globalExcludes, daemonPreferences)

	// Assert
	require.Len(t, merged, 3)
	require.Contains(t, merged, "foo")
	require.Contains(t, merged, "baz")
	require.Contains(t, merged, "boz")
}
