package dbmodel

import (
	"testing"

	"github.com/go-pg/pg/v10"
	"github.com/stretchr/testify/require"
	dbtest "isc.org/stork/server/database/test"
)

// Tests that the config checker preferences are constructed properly.
func TestNewConfigCheckerPreference(t *testing.T) {
	// Act
	global := NewGlobalConfigCheckerPreference("foo")
	nonGlobal := NewDaemonConfigCheckerPreference(42, "bar", true)

	// Assert
	require.Nil(t, global.DaemonID)
	require.EqualValues(t, "foo", global.CheckerName)
	require.False(t, global.Enabled)

	require.EqualValues(t, 42, *nonGlobal.DaemonID)
	require.EqualValues(t, "bar", nonGlobal.CheckerName)
	require.True(t, nonGlobal.Enabled)
}

// Tests that the global checker preference is recognized properly.
func TestConfigCheckerPreferenceIsGlobal(t *testing.T) {
	// Arrange
	global := NewGlobalConfigCheckerPreference("foo")
	nonGlobal := NewDaemonConfigCheckerPreference(1, "bar", false)

	// Act
	globalIsGlobal := global.IsGlobal()
	nonGlobalIsGlobal := nonGlobal.IsGlobal()

	// Assert
	require.True(t, globalIsGlobal)
	require.False(t, nonGlobalIsGlobal)
}

// Test that the daemon ID assigned to the config checker preference is
// returned properly.
func TestConfigCheckerPreferenceGetDaemonID(t *testing.T) {
	// Arrange
	global := NewGlobalConfigCheckerPreference("foo")
	nonGlobal := NewDaemonConfigCheckerPreference(42, "bar", false)

	// Act
	globalDaemonID := global.GetDaemonID()
	nonGlobalDaemonID := nonGlobal.GetDaemonID()

	// Assert
	require.EqualValues(t, 0, globalDaemonID)
	require.EqualValues(t, 42, nonGlobalDaemonID)
}

// Test that the string representation of the checker preference is created
// properly.
func TestCheckerPreferenceToString(t *testing.T) {
	// Arrange
	globalPreferenceDisabled := NewGlobalConfigCheckerPreference("foo")
	globalPreferenceEnabled := NewGlobalConfigCheckerPreference("foo")
	globalPreferenceEnabled.Enabled = true
	nonGlobalPreferenceDisabled := NewDaemonConfigCheckerPreference(42, "foo", false)
	nonGlobalPreferenceEnabled := NewDaemonConfigCheckerPreference(42, "foo", true)

	// Act & Assert
	require.EqualValues(t, "foo checker is globally disabled", globalPreferenceDisabled.String())
	require.EqualValues(t, "foo checker is globally enabled", globalPreferenceEnabled.String())
	require.EqualValues(t, "foo checker is disabled for 42 daemon ID", nonGlobalPreferenceDisabled.String())
	require.EqualValues(t, "foo checker is enabled for 42 daemon ID", nonGlobalPreferenceEnabled.String())
}

// Creates two demon entries in the database. The daemons belong to different
// apps and machines.
func addTestDaemons(db *pg.DB) (*Daemon, *Daemon, error) {
	var createdDaemons []*Daemon

	for i := 0; i < 2; i++ {
		m := &Machine{
			ID:        0,
			Address:   "localhost",
			AgentPort: int64(8080 + i),
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
			},
			MachineID: m.ID,
		}

		daemons, err := AddApp(db, app)
		if err != nil {
			return nil, nil, err
		}

		daemons[0].App = app
		createdDaemons = append(createdDaemons, daemons[0])
	}

	return createdDaemons[0], createdDaemons[1], nil
}

// Test that the daemon config checker preferences are inserted properly.
// If the preference already exists it should be updated instead.
func TestAddOrUpdateCheckerPreferences(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemon, _, _ := addTestDaemons(db)

	// Act
	preferences := []*ConfigCheckerPreference{
		NewDaemonConfigCheckerPreference(daemon.ID, "foo", false),
	}
	err1 := addOrUpdateCheckerPreferences(db, preferences)
	preferences[0].Enabled = true
	preferences = append(preferences, NewDaemonConfigCheckerPreference(daemon.ID, "bar", false))
	err2 := addOrUpdateCheckerPreferences(db, preferences)

	// Assert
	require.NoError(t, err1)
	require.NoError(t, err2)
	preferences, _ = GetCheckerPreferences(db, daemon.ID)
	require.Len(t, preferences, 2)
	require.EqualValues(t, "bar", preferences[0].CheckerName)
	require.False(t, preferences[0].Enabled)
	require.EqualValues(t, "foo", preferences[1].CheckerName)
	require.True(t, preferences[1].Enabled)
}

// Test that the global config checker preferences are inserted properly.
// If the preference already exists it should be updated.
func TestAddOrUpdateGlobalCheckerPreferences(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	preferences := []*ConfigCheckerPreference{
		NewGlobalConfigCheckerPreference("foo"),
	}
	err1 := addOrUpdateCheckerPreferences(db, preferences)
	preferences[0].Enabled = true
	preferences = append(preferences, NewGlobalConfigCheckerPreference("bar"))
	err2 := addOrUpdateCheckerPreferences(db, preferences)

	// Assert
	require.NoError(t, err1)
	require.NoError(t, err2)
	preferences, _ = GetCheckerPreferences(db, 0)
	require.Len(t, preferences, 2)
	require.EqualValues(t, "bar", preferences[0].CheckerName)
	require.False(t, preferences[0].Enabled)
	require.EqualValues(t, "foo", preferences[1].CheckerName)
	require.True(t, preferences[1].Enabled)
}

// Test that adding/updating empty preference list causes no panic or error.
func TestAddOrUpdateEmptyPreferenceList(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	err := addOrUpdateCheckerPreferences(db, []*ConfigCheckerPreference{})

	// Assert
	require.NoError(t, err)
}

// Test that the config checker preferences are removed properly.
func TestDeleteCheckerPreferences(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemon1, daemon2, _ := addTestDaemons(db)
	preferences := []*ConfigCheckerPreference{
		NewDaemonConfigCheckerPreference(daemon1.ID, "foo", false),
		NewDaemonConfigCheckerPreference(daemon1.ID, "bar", true),
		NewDaemonConfigCheckerPreference(daemon2.ID, "baz", true),
	}
	_ = addOrUpdateCheckerPreferences(db, preferences)

	// Act
	err := deleteCheckerPreferences(db, []*ConfigCheckerPreference{
		preferences[1],
	})

	// Assert
	require.NoError(t, err)
	preferences, _ = GetCheckerPreferences(db, daemon1.ID)
	require.Len(t, preferences, 1)
	require.EqualValues(t, "foo", preferences[0].CheckerName)
	preferences, _ = GetCheckerPreferences(db, daemon2.ID)
	require.Len(t, preferences, 1)
	require.EqualValues(t, "baz", preferences[0].CheckerName)
}

// Test that deleting the non-existing preference of config checker
// causes no error.
func TestDeleteNonExistingCheckerPreferences(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	err := deleteCheckerPreferences(db, []*ConfigCheckerPreference{
		NewGlobalConfigCheckerPreference("foo"),
	})

	// Assert
	require.NoError(t, err)
	preferences, _ := GetCheckerPreferences(db, 0)
	require.Empty(t, preferences)
}

// Test that removing the config checker preferences
// generates no error if the list of disabled preferences is empty.
func TestDeleteEmptyListOfCheckerPreferences(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	err := deleteCheckerPreferences(db, []*ConfigCheckerPreference{})

	// Assert
	require.NoError(t, err)
}

// Test that removing the daemon causes to wipe out all related checker preferences.
func TestDeleteDaemonAndRelatedCheckerPreferences(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemon1, daemon2, _ := addTestDaemons(db)
	preferences := []*ConfigCheckerPreference{
		NewDaemonConfigCheckerPreference(daemon1.ID, "foo", false),
		NewDaemonConfigCheckerPreference(daemon2.ID, "bar", false),
		NewGlobalConfigCheckerPreference("baz"),
	}
	_ = addOrUpdateCheckerPreferences(db, preferences)

	// Act
	err := DeleteApp(db, daemon1.App)

	// Assert
	// Delete the config checker preferences related to the first daemon.
	require.NoError(t, err)
	preferences, _ = GetCheckerPreferences(db, daemon1.ID)
	require.Empty(t, preferences)
	// Keep left the config checker preferences related to the second daemon.
	preferences, _ = GetCheckerPreferences(db, daemon2.ID)
	require.Len(t, preferences, 1)
	// Keep left the global config checker preferences.
	preferences, _ = GetCheckerPreferences(db, 0)
	require.Len(t, preferences, 1)
}

// Test that the changes in the config checker preferences are
// committed properly.
func TestModifyCheckerPreferences(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemon1, daemon2, _ := addTestDaemons(db)
	preferences := []*ConfigCheckerPreference{
		NewDaemonConfigCheckerPreference(daemon1.ID, "foo", false),
		NewDaemonConfigCheckerPreference(daemon1.ID, "bar", false),
		NewDaemonConfigCheckerPreference(daemon1.ID, "baz", false),
		NewDaemonConfigCheckerPreference(daemon2.ID, "biz", false),
	}
	_ = addOrUpdateCheckerPreferences(db, preferences)

	// Act
	var updates []*ConfigCheckerPreference
	var deletes []*ConfigCheckerPreference
	// Modifies bar
	preferences[1].Enabled = true
	updates = append(updates, preferences[1])
	// Removes foo
	deletes = append(deletes, preferences[0])
	// Adds boz
	updates = append(updates, NewDaemonConfigCheckerPreference(daemon1.ID, "boz", true))
	// Commits changes
	err := CommitCheckerPreferences(db, updates, deletes)

	// Asserts
	require.NoError(t, err)
	preferences, _ = GetCheckerPreferences(db, daemon1.ID)
	require.Len(t, preferences, 3)
	require.EqualValues(t, "bar", preferences[0].CheckerName)
	require.EqualValues(t, "baz", preferences[1].CheckerName)
	require.EqualValues(t, "boz", preferences[2].CheckerName)
	preferences, _ = GetCheckerPreferences(db, daemon2.ID)
	require.Len(t, preferences, 1)
	require.EqualValues(t, "biz", preferences[0].CheckerName)
}

// Test that committing the nil preference lists causes no error.
func TestModifyCheckerPreferencesWithNils(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	err := CommitCheckerPreferences(db, nil, nil)

	// Assert
	require.NoError(t, err)
}

// Test that the global checker preferences are fetched properly.
func TestGetGlobalCheckerPreferences(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemon, _, _ := addTestDaemons(db)
	_ = addOrUpdateCheckerPreferences(db, []*ConfigCheckerPreference{
		NewGlobalConfigCheckerPreference("foo"),
		NewGlobalConfigCheckerPreference("bar"),
		NewDaemonConfigCheckerPreference(daemon.ID, "baz", false),
	})

	// Act
	preferences, err := GetCheckerPreferences(db, 0)

	// Assert
	require.NoError(t, err)
	require.Len(t, preferences, 2)
	require.EqualValues(t, "bar", preferences[0].CheckerName)
	require.False(t, preferences[0].Enabled)
	require.EqualValues(t, "foo", preferences[1].CheckerName)
	require.False(t, preferences[1].Enabled)
}

// Test that the daemon checker preferences are fetched properly.
func TestGetDaemonCheckerPreferences(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemon1, daemon2, _ := addTestDaemons(db)
	_ = addOrUpdateCheckerPreferences(db, []*ConfigCheckerPreference{
		NewDaemonConfigCheckerPreference(daemon1.ID, "foo", false),
		NewDaemonConfigCheckerPreference(daemon1.ID, "bar", true),
		NewDaemonConfigCheckerPreference(daemon2.ID, "baz", false),
	})

	// Act
	preferences, err := GetCheckerPreferences(db, daemon1.ID)

	// Assert
	require.NoError(t, err)
	require.Len(t, preferences, 2)
	require.EqualValues(t, "bar", preferences[0].CheckerName)
	require.True(t, preferences[0].Enabled)
	require.EqualValues(t, "foo", preferences[1].CheckerName)
	require.False(t, preferences[1].Enabled)
}

// Test that the all checker preferences are fetched properly.
func TestGetAllCheckerPreferences(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemon1, daemon2, _ := addTestDaemons(db)
	_ = addOrUpdateCheckerPreferences(db, []*ConfigCheckerPreference{
		NewDaemonConfigCheckerPreference(daemon1.ID, "foo", false),
		NewDaemonConfigCheckerPreference(daemon1.ID, "bar", true),
		NewDaemonConfigCheckerPreference(daemon2.ID, "baz", false),
		NewGlobalConfigCheckerPreference("boz"),
		NewGlobalConfigCheckerPreference("biz"),
	})

	// Act
	preferences, err := GetAllCheckerPreferences(db)

	// Assert
	require.NoError(t, err)
	require.Len(t, preferences, 5)
	require.EqualValues(t, "bar", preferences[0].CheckerName)
	require.True(t, preferences[0].Enabled)
	require.EqualValues(t, "baz", preferences[1].CheckerName)
	require.False(t, preferences[1].Enabled)
	require.EqualValues(t, "biz", preferences[2].CheckerName)
	require.False(t, preferences[2].Enabled)
	require.EqualValues(t, "boz", preferences[3].CheckerName)
	require.False(t, preferences[3].Enabled)
	require.EqualValues(t, "foo", preferences[4].CheckerName)
	require.False(t, preferences[4].Enabled)
}
