package dbmodel

import (
	"testing"

	"github.com/go-pg/pg/v10"
	"github.com/stretchr/testify/require"
	dbtest "isc.org/stork/server/database/test"
)

// Test that the global exclusions of the config checkers are returned properly.
func TestGetGloballyExcludedConfigCheckers(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	_ = addGloballyExcludedConfigCheckers(db, []*ConfigCheckerGlobalExclude{
		{
			CheckerName: "foo",
		},
		{
			CheckerName: "bar",
		},
	})

	// Act
	exclusions, err := GetGloballyExcludedConfigCheckers(db)

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
func TestGetGloballyExcludedConfigCheckersForEmptyData(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	exclusions, err := GetGloballyExcludedConfigCheckers(db)

	// Assert
	require.NoError(t, err)
	require.Empty(t, exclusions)
}

// Test that the global exclusions of the config checkers are inserted properly.
func TestAddGloballyExcludedConfigCheckers(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	err := addGloballyExcludedConfigCheckers(db, []*ConfigCheckerGlobalExclude{
		{
			CheckerName: "foo",
		},
		{
			CheckerName: "bar",
		},
	})

	// Assert
	require.NoError(t, err)
	exclusions, _ := GetGloballyExcludedConfigCheckers(db)
	require.Len(t, exclusions, 2)
}

// Test that adding empty list of the global exclusions of the config checkers
// generates no error.
func TestAddGloballyExcludedConfigCheckersForEmptyList(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	err := addGloballyExcludedConfigCheckers(db, []*ConfigCheckerGlobalExclude{})

	// Assert
	require.NoError(t, err)
	exclusions, _ := GetGloballyExcludedConfigCheckers(db)
	require.Empty(t, exclusions)
}

// Test that adding the duplicated global exclusions of the config checkers
// generates an error.
func TestAddDuplicatedGloballyExcludedConfigCheckersCausesError(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	err := addGloballyExcludedConfigCheckers(db, []*ConfigCheckerGlobalExclude{
		{CheckerName: "foo"},
		{CheckerName: "foo"},
	})

	// Assert
	require.Error(t, err)
	exclusions, _ := GetGloballyExcludedConfigCheckers(db)
	require.Empty(t, exclusions)
}

// Test that adding the same global exclusions of the config checkers
// in separate queries generates an error on the second call.
func TestAddDoubleGloballyExcludedConfigCheckersCausesError(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	err1 := addGloballyExcludedConfigCheckers(db, []*ConfigCheckerGlobalExclude{
		{CheckerName: "foo"},
	})
	err2 := addGloballyExcludedConfigCheckers(db, []*ConfigCheckerGlobalExclude{
		{CheckerName: "foo"},
	})

	// Assert
	require.NoError(t, err1)
	require.Error(t, err2)
	exclusions, _ := GetGloballyExcludedConfigCheckers(db)
	require.Len(t, exclusions, 1)
}

// Test that the global exclusions of the config checkers are deleted properly.
func TestDeleteGloballyExcludedConfigCheckers(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	_ = addGloballyExcludedConfigCheckers(db, []*ConfigCheckerGlobalExclude{
		{CheckerName: "foo"},
		{CheckerName: "bar"},
	})
	exclusions, _ := GetGloballyExcludedConfigCheckers(db)

	// Act
	err := deleteAllGloballyExcludedChekers(db, []int64{
		exclusions[0].ID,
	})

	// Assert
	require.NoError(t, err)
	exclusions, _ = GetGloballyExcludedConfigCheckers(db)
	require.Len(t, exclusions, 1)
	require.EqualValues(t, "foo", exclusions[0].CheckerName)
}

// Test that removing the global exclusions of the config
// checkers without excluding any entry generates no error.
func TestDeleteEmptyGloballyExcludedConfigCheckers(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	err := deleteAllGloballyExcludedChekers(db, []int64{})

	// Assert
	require.NoError(t, err)
	exclusions, _ := GetGloballyExcludedConfigCheckers(db)
	require.Empty(t, exclusions)
}

// Test that removing the non-existent global exclusions of the config checkers
// generates no error.
func TestDeleteNonExistentGloballyExcludedConfigCheckers(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	err := deleteAllGloballyExcludedChekers(db, []int64{42})

	// Assert
	require.NoError(t, err)
}

// Test that the changes in the global exclusions of the config checkers are
// committed properly.
func TestCommitGloballyExcludedConfigCheckers(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	_ = addGloballyExcludedConfigCheckers(db, []*ConfigCheckerGlobalExclude{
		{CheckerName: "foo"}, {CheckerName: "bar"},
	})
	exclusions, _ := GetGloballyExcludedConfigCheckers(db)

	// Act
	// Deletes foo
	exclusions = append(exclusions[:0], exclusions[1:]...)
	// Adds baz
	exclusions = append(exclusions, &ConfigCheckerGlobalExclude{CheckerName: "baz"})
	// Commits
	err := CommitGloballyExcludedConfigCheckers(db, exclusions)

	// Assert
	require.NoError(t, err)
	exclusions, _ = GetGloballyExcludedConfigCheckers(db)
	require.Len(t, exclusions, 2)
	require.EqualValues(t, "bar", exclusions[0].CheckerName)
	require.EqualValues(t, "baz", exclusions[1].CheckerName)
}

// Creates two demon entries in the database.
func addTestDaemons(db *pg.DB) (*Daemon, *Daemon, error) {
	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	if err != nil {
		return nil, nil, err
	}

	app := &App{
		ID:   0,
		Type: AppTypeKea,
		Daemons: []*Daemon{
			NewKeaDaemon(DaemonNameDHCPv4, true),
			NewKeaDaemon(DaemonNameDHCPv6, true),
		},
		MachineID: m.ID,
	}

	daemons, err := AddApp(db, app)
	if err != nil {
		return nil, nil, err
	}

	daemons[0].App = app
	daemons[1].App = app
	return daemons[0], daemons[1], nil
}

// Test that the daemon preferences of config checkers are added properly.
func TestAddDaemonCheckerPreferences(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemon, _, _ := addTestDaemons(db)

	// Act
	err := addDaemonCheckerPreferences(db, []*ConfigDaemonCheckerPreference{
		{
			DaemonID:    daemon.ID,
			CheckerName: "foo",
			Excluded:    true,
		},
	})

	// Assert
	require.NoError(t, err)
	preferences, _ := GetDaemonCheckerPreferences(db, daemon.ID)
	require.Len(t, preferences, 1)
	require.EqualValues(t, "foo", preferences[0].CheckerName)
	require.EqualValues(t, daemon.ID, preferences[0].DaemonID)
	require.True(t, preferences[0].Excluded)
}

// Test that adding the empty list of the daemon preferences of config checkers
// generates no error.
func TestAddEmptyListOfDaemonCheckerPreferences(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemon, _, _ := addTestDaemons(db)

	// Act
	err := addDaemonCheckerPreferences(db, []*ConfigDaemonCheckerPreference{})

	// Assert
	require.NoError(t, err)
	preferences, _ := GetDaemonCheckerPreferences(db, daemon.ID)
	require.Empty(t, preferences)
}

// Test that adding a daemon preference with already existing checker name causes
// an error.
func TestAddCheckerPreferencesWithDoubledName(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemon, _, _ := addTestDaemons(db)

	// Act
	err1 := addDaemonCheckerPreferences(db, []*ConfigDaemonCheckerPreference{
		{
			DaemonID:    daemon.ID,
			CheckerName: "foo",
			Excluded:    true,
		},
	})
	err2 := addDaemonCheckerPreferences(db, []*ConfigDaemonCheckerPreference{
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

// Test that the daemon preferences for a specific daemon are updated properly.
func TestUpdateDaemonCheckerPreferences(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemon, _, _ := addTestDaemons(db)
	preferences := []*ConfigDaemonCheckerPreference{{
		DaemonID:    daemon.ID,
		CheckerName: "foo",
		Excluded:    true,
	}}
	_ = addDaemonCheckerPreferences(db, preferences)

	// Act
	preferences[0].Excluded = false
	preferences[0].CheckerName = "bar"
	err := updateDaemonCheckerPreferences(db, preferences)

	// Assert
	require.NoError(t, err)
	preferences, _ = GetDaemonCheckerPreferences(db, daemon.ID)
	require.Len(t, preferences, 1)
	require.EqualValues(t, "bar", preferences[0].CheckerName)
	require.False(t, preferences[0].Excluded)
}

// Test that updating an empty list of the daemon preferences of config checkers
// generates no error.
func TestUpdateEmptyListOfDaemonCheckerPreferences(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemon, _, _ := addTestDaemons(db)

	// Act
	err := updateDaemonCheckerPreferences(db, []*ConfigDaemonCheckerPreference{})

	// Assert
	require.NoError(t, err)
	preferences, _ := GetDaemonCheckerPreferences(db, daemon.ID)
	require.Empty(t, preferences)
}

// Test that updating the daemon preferences of config checkers with a name
// that already exists for a given daemon generates error.
func TestUpdateDaemonCheckerPreferencesWithDuplicatedName(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemon, _, _ := addTestDaemons(db)
	preferences := []*ConfigDaemonCheckerPreference{
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
	_ = addDaemonCheckerPreferences(db, preferences)

	// Act
	preferences[1].CheckerName = "foo"
	err := updateDaemonCheckerPreferences(db, []*ConfigDaemonCheckerPreference{
		preferences[1],
	})

	// Assert
	require.Error(t, err)
}

// Test that the daemon preferences of config checkers are removed properly.
func TestDeleteDaemonCheckerPreferences(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemon1, daemon2, _ := addTestDaemons(db)
	preferences := []*ConfigDaemonCheckerPreference{
		{
			DaemonID:    daemon1.ID,
			CheckerName: "foo",
			Excluded:    true,
		},
		{
			DaemonID:    daemon1.ID,
			CheckerName: "bar",
			Excluded:    false,
		},
		{
			DaemonID:    daemon2.ID,
			CheckerName: "baz",
			Excluded:    false,
		},
	}
	_ = addDaemonCheckerPreferences(db, preferences)

	// Act
	err := deleteAllDaemonCheckerPreferences(db, daemon1.ID, []int64{1})

	// Assert
	require.NoError(t, err)
	preferences, _ = GetDaemonCheckerPreferences(db, daemon1.ID)
	require.Len(t, preferences, 1)
	require.EqualValues(t, "foo", preferences[0].CheckerName)
	preferences, _ = GetDaemonCheckerPreferences(db, daemon2.ID)
	require.Len(t, preferences, 1)
	require.EqualValues(t, "baz", preferences[0].CheckerName)
}

// Test that removing the daemon preferences of config checkers
// generates no error if the list of excluded IDs is empty.
func TestDeleteEmptyListOfDaemonCheckerPreferences(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	err := deleteAllDaemonCheckerPreferences(db, 1, []int64{})

	// Assert
	require.NoError(t, err)
}

// Test that removing the daemon causes to wipe out all related checker preferences.
func TestDeleteDaemonAndRelatedDaemonCheckerPreferences(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemon, _, _ := addTestDaemons(db)
	preferences := []*ConfigDaemonCheckerPreference{{
		DaemonID:    daemon.ID,
		CheckerName: "foo",
		Excluded:    true,
	}}
	_ = addDaemonCheckerPreferences(db, preferences)

	// Act
	err := DeleteApp(db, daemon.App)

	// Assert
	require.NoError(t, err)
	preferences, _ = GetDaemonCheckerPreferences(db, daemon.ID)
	require.Empty(t, preferences)
}

// Test that the changes in the daemon preferences of config checkers are
// committed properly.
func TestCommitDaemonCheckerPreferences(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemon1, daemon2, _ := addTestDaemons(db)
	preferences := []*ConfigDaemonCheckerPreference{
		{
			DaemonID:    daemon1.ID,
			CheckerName: "foo",
			Excluded:    true,
		},
		{
			DaemonID:    daemon1.ID,
			CheckerName: "bar",
			Excluded:    true,
		},
		{
			DaemonID:    daemon1.ID,
			CheckerName: "baz",
			Excluded:    true,
		},
		{
			DaemonID:    daemon2.ID,
			CheckerName: "biz",
			Excluded:    true,
		},
	}
	_ = addDaemonCheckerPreferences(db, preferences)

	// Act
	// Modifies bar
	preferences[1].Excluded = false
	// Removes foo
	preferences = append(preferences[:0], preferences[1:]...)
	// Adds boz
	preferences = append(preferences, &ConfigDaemonCheckerPreference{
		DaemonID:    daemon1.ID,
		CheckerName: "boz",
		Excluded:    true,
	})
	// Commits changes
	err := CommitDaemonCheckerPreferences(db, daemon1.ID, preferences)

	// Asserts
	require.NoError(t, err)
	preferences, _ = GetDaemonCheckerPreferences(db, daemon1.ID)
	require.Len(t, preferences, 3)
	require.EqualValues(t, "bar", preferences[0].CheckerName)
	require.EqualValues(t, "baz", preferences[1].CheckerName)
	require.EqualValues(t, "boz", preferences[2].CheckerName)
	preferences, _ = GetDaemonCheckerPreferences(db, daemon2.ID)
	require.Len(t, preferences, 1)
	require.EqualValues(t, "biz", preferences[0].CheckerName)
}

// Test that the changes in the daemon preferences of config checkers generates
// an error if the checker names are duplicated.
func TestCommitDuplicatedDaemonCheckerPreferences(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemon, _, _ := addTestDaemons(db)
	preferences := []*ConfigDaemonCheckerPreference{
		{
			DaemonID:    daemon.ID,
			CheckerName: "foo",
			Excluded:    true,
		},
	}
	_ = addDaemonCheckerPreferences(db, preferences)

	// Act
	// Adds duplicated foo
	preferences = append(preferences, &ConfigDaemonCheckerPreference{
		DaemonID:    daemon.ID,
		CheckerName: "foo",
		Excluded:    false,
	})
	// Add another entry
	preferences = append(preferences, &ConfigDaemonCheckerPreference{
		DaemonID:    daemon.ID,
		CheckerName: "bar",
		Excluded:    false,
	})
	// Commits changes
	err := CommitDaemonCheckerPreferences(db, daemon.ID, preferences)

	// Asserts
	require.Error(t, err)
	preferences, _ = GetDaemonCheckerPreferences(db, daemon.ID)
	// No new entry was added.
	require.Len(t, preferences, 1)
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
	daemonPreferences := []*ConfigDaemonCheckerPreference{
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
