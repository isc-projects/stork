package dbmodel

import (
	"testing"

	"github.com/go-pg/pg/v10"
	"github.com/stretchr/testify/require"
	dbtest "isc.org/stork/server/database/test"
)

// Checks if the global checker preference is recognized properly.
func TestConfigCheckerPreferenceIsGlobal(t *testing.T) {
	// Arrange
	global := NewGlobalConfigCheckerPreference("foo", true)
	nonGlobal := NewDaemonConfigCheckerPreference(1, "bar", true)

	// Act
	globalIsGlobal := global.IsGlobal()
	nonGlobalIsGlobal := nonGlobal.IsGlobal()

	// Assert
	require.True(t, globalIsGlobal)
	require.False(t, nonGlobalIsGlobal)
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

// Test that the daemon preferences of config checkers are inserted properly.
// If the preference already exists it should be updated instead.
func TestAddOrUpdateDaemonCheckerPreferences(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemon, _, _ := addTestDaemons(db)

	// Act
	preferences := []*ConfigCheckerPreference{
		{
			DaemonID:    &daemon.ID,
			CheckerName: "foo",
			Excluded:    true,
		},
	}
	err1 := addOrUpdateDaemonCheckerPreferences(db, preferences)
	preferences[0].Excluded = false
	preferences = append(preferences, &ConfigCheckerPreference{
		DaemonID:    &daemon.ID,
		CheckerName: "bar",
		Excluded:    true,
	})
	err2 := addOrUpdateDaemonCheckerPreferences(db, preferences)

	// Assert
	require.NoError(t, err1)
	require.NoError(t, err2)
	preferences, _ = GetDaemonCheckerPreferences(db, &daemon.ID)
	require.Len(t, preferences, 2)
	require.EqualValues(t, "bar", preferences[0].CheckerName)
	require.True(t, preferences[0].Excluded)
	require.EqualValues(t, "foo", preferences[1].CheckerName)
	require.False(t, preferences[1].Excluded)
}

// Test that the global preferences of config checkers are inserted properly.
// If the preference already exists it should be updated instead.
func TestAddOrUpdateGlobalCheckerPreferences(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	preferences := []*ConfigCheckerPreference{
		{
			DaemonID:    nil,
			CheckerName: "foo",
			Excluded:    true,
		},
	}
	err1 := addOrUpdateDaemonCheckerPreferences(db, preferences)
	preferences[0].Excluded = false
	preferences = append(preferences, &ConfigCheckerPreference{
		DaemonID:    nil,
		CheckerName: "bar",
		Excluded:    true,
	})
	err2 := addOrUpdateDaemonCheckerPreferences(db, preferences)

	// Assert
	require.NoError(t, err1)
	require.NoError(t, err2)
	preferences, _ = GetDaemonCheckerPreferences(db, nil)
	require.Len(t, preferences, 2)
	require.EqualValues(t, "bar", preferences[0].CheckerName)
	require.True(t, preferences[0].Excluded)
	require.EqualValues(t, "foo", preferences[1].CheckerName)
	require.False(t, preferences[1].Excluded)
}

// Test that adding/updating empty preference list causes no panic or error.
func TestAddOrUpdateEmptyPreferenceList(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	err := addOrUpdateDaemonCheckerPreferences(db, []*ConfigCheckerPreference{})

	// Assert
	require.NoError(t, err)
}

// Test that the daemon preferences of config checkers are removed properly.
func TestDeleteDaemonCheckerPreferences(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemon1, daemon2, _ := addTestDaemons(db)
	preferences := []*ConfigCheckerPreference{
		{
			DaemonID:    &daemon1.ID,
			CheckerName: "foo",
			Excluded:    true,
		},
		{
			DaemonID:    &daemon1.ID,
			CheckerName: "bar",
			Excluded:    false,
		},
		{
			DaemonID:    &daemon2.ID,
			CheckerName: "baz",
			Excluded:    false,
		},
	}
	_ = addOrUpdateDaemonCheckerPreferences(db, preferences)

	// Act
	err := deleteDaemonCheckerPreferences(db, []*ConfigCheckerPreference{
		preferences[1],
	})

	// Assert
	require.NoError(t, err)
	preferences, _ = GetDaemonCheckerPreferences(db, &daemon1.ID)
	require.Len(t, preferences, 1)
	require.EqualValues(t, "foo", preferences[0].CheckerName)
	preferences, _ = GetDaemonCheckerPreferences(db, &daemon2.ID)
	require.Len(t, preferences, 1)
	require.EqualValues(t, "baz", preferences[0].CheckerName)
}

// Test that deleting the non-existing daemon preference of config checker
// causes no error.
func TestDeleteNonExistingDaemonCheckerPreferences(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	err := deleteDaemonCheckerPreferences(db, []*ConfigCheckerPreference{
		{
			CheckerName: "foo",
		},
	})

	// Assert
	require.NoError(t, err)
	preferences, _ := GetDaemonCheckerPreferences(db, nil)
	require.Empty(t, preferences)
}

// Test that removing the daemon preferences of config checkers
// generates no error if the list of excluded IDs is empty.
func TestDeleteEmptyListOfDaemonCheckerPreferences(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	err := deleteDaemonCheckerPreferences(db, []*ConfigCheckerPreference{})

	// Assert
	require.NoError(t, err)
}

// Test that removing the daemon causes to wipe out all related checker preferences.
func TestDeleteDaemonAndRelatedDaemonCheckerPreferences(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemon, _, _ := addTestDaemons(db)
	preferences := []*ConfigCheckerPreference{{
		DaemonID:    &daemon.ID,
		CheckerName: "foo",
		Excluded:    true,
	}}
	_ = addOrUpdateDaemonCheckerPreferences(db, preferences)

	// Act
	err := DeleteApp(db, daemon.App)

	// Assert
	require.NoError(t, err)
	preferences, _ = GetDaemonCheckerPreferences(db, &daemon.ID)
	require.Empty(t, preferences)
}

// Test that the changes in the daemon preferences of config checkers are
// committed properly.
func TestModifyDaemonCheckerPreferences(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemon1, daemon2, _ := addTestDaemons(db)
	preferences := []*ConfigCheckerPreference{
		{
			DaemonID:    &daemon1.ID,
			CheckerName: "foo",
			Excluded:    true,
		},
		{
			DaemonID:    &daemon1.ID,
			CheckerName: "bar",
			Excluded:    true,
		},
		{
			DaemonID:    &daemon1.ID,
			CheckerName: "baz",
			Excluded:    true,
		},
		{
			DaemonID:    &daemon2.ID,
			CheckerName: "biz",
			Excluded:    true,
		},
	}
	_ = addOrUpdateDaemonCheckerPreferences(db, preferences)

	// Act
	var updates []*ConfigCheckerPreference
	var deletes []*ConfigCheckerPreference
	// Modifies bar
	preferences[1].Excluded = false
	updates = append(updates, preferences[1])
	// Removes foo
	deletes = append(deletes, preferences[0])
	// Adds boz
	updates = append(updates, &ConfigCheckerPreference{
		DaemonID:    &daemon1.ID,
		CheckerName: "boz",
		Excluded:    true,
	})
	// Commits changes
	err := CommitDaemonCheckerPreferences(db, updates, deletes)

	// Asserts
	require.NoError(t, err)
	preferences, _ = GetDaemonCheckerPreferences(db, &daemon1.ID)
	require.Len(t, preferences, 3)
	require.EqualValues(t, "bar", preferences[0].CheckerName)
	require.EqualValues(t, "baz", preferences[1].CheckerName)
	require.EqualValues(t, "boz", preferences[2].CheckerName)
	preferences, _ = GetDaemonCheckerPreferences(db, &daemon2.ID)
	require.Len(t, preferences, 1)
	require.EqualValues(t, "biz", preferences[0].CheckerName)
}
