package dbmodel

import (
	"testing"

	"github.com/go-pg/pg/v10"
	"github.com/stretchr/testify/require"
	dbtest "isc.org/stork/server/database/test"
)

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
	err := AddDaemonCheckerPreferences(db, []*ConfigDaemonCheckerPreference{
		{
			DaemonID:    &daemon.ID,
			CheckerName: "foo",
			Excluded:    true,
		},
	})

	// Assert
	require.NoError(t, err)
	preferences, _ := GetDaemonCheckerPreferences(db, &daemon.ID)
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
	err := AddDaemonCheckerPreferences(db, []*ConfigDaemonCheckerPreference{})

	// Assert
	require.NoError(t, err)
	preferences, _ := GetDaemonCheckerPreferences(db, &daemon.ID)
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
	err1 := AddDaemonCheckerPreferences(db, []*ConfigDaemonCheckerPreference{
		{
			DaemonID:    &daemon.ID,
			CheckerName: "foo",
			Excluded:    true,
		},
	})
	err2 := AddDaemonCheckerPreferences(db, []*ConfigDaemonCheckerPreference{
		{
			DaemonID:    &daemon.ID,
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
		DaemonID:    &daemon.ID,
		CheckerName: "foo",
		Excluded:    true,
	}}
	_ = AddDaemonCheckerPreferences(db, preferences)

	// Act
	preferences[0].Excluded = false
	preferences[0].CheckerName = "bar"
	err := UpdateDaemonCheckerPreferences(db, preferences)

	// Assert
	require.NoError(t, err)
	preferences, _ = GetDaemonCheckerPreferences(db, &daemon.ID)
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
	err := UpdateDaemonCheckerPreferences(db, []*ConfigDaemonCheckerPreference{})

	// Assert
	require.NoError(t, err)
	preferences, _ := GetDaemonCheckerPreferences(db, &daemon.ID)
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
			DaemonID:    &daemon.ID,
			CheckerName: "foo",
			Excluded:    true,
		},
		{
			DaemonID:    &daemon.ID,
			CheckerName: "bar",
			Excluded:    true,
		},
	}
	_ = AddDaemonCheckerPreferences(db, preferences)

	// Act
	preferences[1].CheckerName = "foo"
	err := UpdateDaemonCheckerPreferences(db, []*ConfigDaemonCheckerPreference{
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
	_ = AddDaemonCheckerPreferences(db, preferences)

	// Act
	err := DeleteDaemonCheckerPreferences(db, []*ConfigDaemonCheckerPreference{
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

// Test that removing the daemon preferences of config checkers
// generates no error if the list of excluded IDs is empty.
func TestDeleteEmptyListOfDaemonCheckerPreferences(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	err := DeleteDaemonCheckerPreferences(db, []*ConfigDaemonCheckerPreference{})

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
		DaemonID:    &daemon.ID,
		CheckerName: "foo",
		Excluded:    true,
	}}
	_ = AddDaemonCheckerPreferences(db, preferences)

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
	preferences := []*ConfigDaemonCheckerPreference{
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
	_ = AddDaemonCheckerPreferences(db, preferences)

	// Act
	// Modifies bar
	preferences[1].Excluded = false
	errUpdate := UpdateDaemonCheckerPreferences(db, []*ConfigDaemonCheckerPreference{
		preferences[1],
	})
	// Removes foo
	errDelete := DeleteDaemonCheckerPreferences(db, []*ConfigDaemonCheckerPreference{
		preferences[0],
	})
	// Adds boz
	errAdd := AddDaemonCheckerPreferences(db, []*ConfigDaemonCheckerPreference{
		{
			DaemonID:    &daemon1.ID,
			CheckerName: "boz",
			Excluded:    true,
		},
	})
	// Commits changes

	// Asserts
	require.NoError(t, errUpdate)
	require.NoError(t, errDelete)
	require.NoError(t, errAdd)
	preferences, _ = GetDaemonCheckerPreferences(db, &daemon1.ID)
	require.Len(t, preferences, 3)
	require.EqualValues(t, "bar", preferences[0].CheckerName)
	require.EqualValues(t, "baz", preferences[1].CheckerName)
	require.EqualValues(t, "boz", preferences[2].CheckerName)
	preferences, _ = GetDaemonCheckerPreferences(db, &daemon2.ID)
	require.Len(t, preferences, 1)
	require.EqualValues(t, "biz", preferences[0].CheckerName)
}
