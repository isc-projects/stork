package dbmodel

import (
	"testing"
	"time"

	pkgerrors "github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	dbtest "isc.org/stork/server/database/test"
	storkutil "isc.org/stork/util"
)

// Test creation of new config update instance.
func TestNewConfigUpdate(t *testing.T) {
	cu := NewConfigUpdate(AppTypeKea, "host_add", 1, 2, 3)
	require.NotNil(t, cu)
	require.Equal(t, AppTypeKea, cu.Target)
	require.Equal(t, "host_add", cu.Operation)
	require.Len(t, cu.DaemonIDs, 3)
	require.Contains(t, cu.DaemonIDs, int64(1))
	require.Contains(t, cu.DaemonIDs, int64(2))
	require.Contains(t, cu.DaemonIDs, int64(3))
}

// Test adding and getting scheduled config changes with ordering by deadline.
func TestAddScheduledConfigChange(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Scheduled config changes must be associated with a user.
	user := &SystemUser{
		Login:    "test",
		Lastname: "test",
		Name:     "test",
	}
	_, err := CreateUser(db, user)
	require.NoError(t, err)
	require.NotZero(t, user.ID)

	// Add 3 config changes with different deadlines.
	change := &ScheduledConfigChange{
		CreatedAt:  storkutil.UTCNow(),
		DeadlineAt: storkutil.UTCNow().Add(time.Second * 10),
		UserID:     int64(user.ID),
		Updates: []*ConfigUpdate{
			NewConfigUpdate(AppTypeKea, "host_add", 1, 2, 3),
			NewConfigUpdate(AppTypeKea, "host_update", 3),
		},
	}
	err = AddScheduledConfigChange(db, change)
	require.NoError(t, err)

	change = &ScheduledConfigChange{
		CreatedAt:  storkutil.UTCNow(),
		DeadlineAt: storkutil.UTCNow().Add(-time.Second * 15),
		UserID:     int64(user.ID),
		Updates: []*ConfigUpdate{
			NewConfigUpdate(AppTypeKea, "host_delete", 1),
		},
	}
	err = AddScheduledConfigChange(db, change)
	require.NoError(t, err)

	change = &ScheduledConfigChange{
		CreatedAt:  storkutil.UTCNow(),
		DeadlineAt: storkutil.UTCNow().Add(-time.Second * 10),
		UserID:     int64(user.ID),
		Updates: []*ConfigUpdate{
			NewConfigUpdate(AppTypeKea, "host_delete", 2),
		},
	}
	err = AddScheduledConfigChange(db, change)
	require.NoError(t, err)

	// Get all scheduled config changes with ordering by deadline.
	returned, err := GetScheduledConfigChanges(db)
	require.NoError(t, err)

	require.Len(t, returned, 3)

	require.NotZero(t, returned[0].ID)
	require.Len(t, returned[0].Updates, 1)
	require.Equal(t, AppTypeKea, returned[0].Updates[0].Target)
	require.Equal(t, "host_delete", returned[0].Updates[0].Operation)
	require.Len(t, returned[0].Updates[0].DaemonIDs, 1)
	require.EqualValues(t, 1, returned[0].Updates[0].DaemonIDs[0])

	require.NotZero(t, returned[1].ID)
	require.Len(t, returned[1].Updates, 1)
	require.Equal(t, AppTypeKea, returned[1].Updates[0].Target)
	require.Equal(t, "host_delete", returned[1].Updates[0].Operation)
	require.Len(t, returned[1].Updates[0].DaemonIDs, 1)
	require.EqualValues(t, 2, returned[1].Updates[0].DaemonIDs[0])

	require.NotZero(t, returned[2].ID)
	require.Len(t, returned[2].Updates, 2)
	require.Equal(t, AppTypeKea, returned[0].Updates[0].Target)
	require.Equal(t, "host_add", returned[2].Updates[0].Operation)
	require.Len(t, returned[2].Updates[0].DaemonIDs, 3)
	require.EqualValues(t, 1, returned[2].Updates[0].DaemonIDs[0])
	require.EqualValues(t, 2, returned[2].Updates[0].DaemonIDs[1])
	require.EqualValues(t, 3, returned[2].Updates[0].DaemonIDs[2])
	require.Equal(t, AppTypeKea, returned[2].Updates[1].Target)
	require.Equal(t, "host_update", returned[2].Updates[1].Operation)
	require.Len(t, returned[2].Updates[1].DaemonIDs, 1)
	require.EqualValues(t, 3, returned[2].Updates[1].DaemonIDs[0])
}

// Test getting due config changes.
func TestGetDueConfigChanges(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Scheduled config changes must be associated with a user.
	user := &SystemUser{
		Login:    "test",
		Lastname: "test",
		Name:     "test",
	}
	_, err := CreateUser(db, user)
	require.NoError(t, err)
	require.NotZero(t, user.ID)

	// Add 3 config changes with different deadlines. Two deadlines are
	// in the past. One is in the future.
	change := &ScheduledConfigChange{
		CreatedAt:  storkutil.UTCNow(),
		DeadlineAt: storkutil.UTCNow().Add(time.Second * 10),
		UserID:     int64(user.ID),
		Updates: []*ConfigUpdate{
			NewConfigUpdate(AppTypeKea, "host_add", 1, 2, 3),
			NewConfigUpdate(AppTypeKea, "host_update", 3),
		},
	}
	err = AddScheduledConfigChange(db, change)
	require.NoError(t, err)

	change = &ScheduledConfigChange{
		CreatedAt:  storkutil.UTCNow(),
		DeadlineAt: storkutil.UTCNow().Add(-time.Second * 15),
		UserID:     int64(user.ID),
		Updates: []*ConfigUpdate{
			NewConfigUpdate(AppTypeKea, "host_delete", 1),
		},
	}
	err = AddScheduledConfigChange(db, change)
	require.NoError(t, err)

	change = &ScheduledConfigChange{
		CreatedAt:  storkutil.UTCNow(),
		DeadlineAt: storkutil.UTCNow().Add(-time.Second * 10),
		UserID:     int64(user.ID),
		Updates: []*ConfigUpdate{
			NewConfigUpdate(AppTypeKea, "host_delete", 2),
		},
	}
	err = AddScheduledConfigChange(db, change)
	require.NoError(t, err)

	// Get due config changes. We should get two.
	returned, err := GetDueConfigChanges(db)
	require.NoError(t, err)
	require.Len(t, returned, 2)

	// Returned config changes should be ordered by deadline.
	require.NotZero(t, returned[0].ID)
	require.Len(t, returned[0].Updates, 1)
	require.Equal(t, AppTypeKea, returned[0].Updates[0].Target)
	require.Equal(t, "host_delete", returned[0].Updates[0].Operation)
	require.Len(t, returned[0].Updates[0].DaemonIDs, 1)
	require.EqualValues(t, 1, returned[0].Updates[0].DaemonIDs[0])

	require.NotZero(t, returned[1].ID)
	require.Len(t, returned[1].Updates, 1)
	require.Equal(t, AppTypeKea, returned[1].Updates[0].Target)
	require.Equal(t, "host_delete", returned[1].Updates[0].Operation)
	require.Len(t, returned[1].Updates[0].DaemonIDs, 1)
	require.EqualValues(t, 2, returned[1].Updates[0].DaemonIDs[0])

	// Mark one of the changes as executed.
	err = SetScheduledConfigChangeExecuted(db, returned[0].ID, "")
	require.NoError(t, err)

	// This time only a single (not executed) change should be returned.
	returned, err = GetDueConfigChanges(db)
	require.NoError(t, err)
	require.Len(t, returned, 1)

	require.Len(t, returned[0].Updates, 1)
	require.Equal(t, AppTypeKea, returned[0].Updates[0].Target)
	require.Equal(t, "host_delete", returned[0].Updates[0].Operation)
	require.Len(t, returned[0].Updates[0].DaemonIDs, 1)
	require.EqualValues(t, 2, returned[0].Updates[0].DaemonIDs[0])
}

// Test marking the specified config change as executed.
func TestSetConfigChangeExecuted(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Scheduled config changes must be associated with a user.
	user := &SystemUser{
		Login:    "test",
		Lastname: "test",
		Name:     "test",
	}
	_, err := CreateUser(db, user)
	require.NoError(t, err)
	require.NotZero(t, user.ID)

	// Add config change.
	change := &ScheduledConfigChange{
		CreatedAt:  storkutil.UTCNow(),
		DeadlineAt: storkutil.UTCNow().Add(-time.Second * 10),
		UserID:     int64(user.ID),
		Updates: []*ConfigUpdate{
			NewConfigUpdate(AppTypeKea, "host_add", 1, 2, 3),
		},
	}
	err = AddScheduledConfigChange(db, change)
	require.NoError(t, err)

	// Make sure that the interesting fields have default values.
	require.False(t, change.Executed)
	require.Empty(t, change.Error)

	// Mark the config change executed and set the error string.
	err = SetScheduledConfigChangeExecuted(db, change.ID, "config change error")
	require.NoError(t, err)

	// An attempt to modify a non-existing change should result in an
	// error.
	err = SetScheduledConfigChangeExecuted(db, change.ID+1, "")
	require.Error(t, err)

	// Get the updated config change.
	returned, err := GetScheduledConfigChanges(db)
	require.NoError(t, err)
	require.Len(t, returned, 1)

	// Make sure that certain fields were not modified.
	require.WithinDuration(t, change.CreatedAt, returned[0].CreatedAt, time.Millisecond)
	require.WithinDuration(t, change.DeadlineAt, returned[0].DeadlineAt, time.Millisecond)
	require.Equal(t, change.UserID, returned[0].UserID)
	require.Len(t, returned[0].Updates, 1)

	// Make sure that the two interesting fields were modified.
	require.True(t, returned[0].Executed)
	require.Equal(t, "config change error", returned[0].Error)
}

// Tests getting time to next config change.
func TestGetTimeToNextConfigChange(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Scheduled config changes must be associated with a user.
	user := &SystemUser{
		Login:    "test",
		Lastname: "test",
		Name:     "test",
	}
	_, err := CreateUser(db, user)
	require.NoError(t, err)
	require.NotZero(t, user.ID)

	// There are no scheduled config changes. It should not cause an error. Instead,
	// the second value should be false.
	tn, exists, err := GetTimeToNextScheduledConfigChange(db)
	require.NoError(t, err)
	require.False(t, exists)
	require.Zero(t, tn)

	// Schedule 3 config changes. The first one is already executed so it should
	// be excluded from the results.
	change := &ScheduledConfigChange{
		CreatedAt:  storkutil.UTCNow(),
		DeadlineAt: storkutil.UTCNow().Add(time.Second * 10),
		UserID:     int64(user.ID),
		Executed:   true,
		Updates: []*ConfigUpdate{
			NewConfigUpdate(AppTypeKea, "host_add", 1, 2, 3),
			NewConfigUpdate(AppTypeKea, "host_update", 3),
		},
	}
	err = AddScheduledConfigChange(db, change)
	require.NoError(t, err)

	change = &ScheduledConfigChange{
		CreatedAt:  storkutil.UTCNow(),
		DeadlineAt: storkutil.UTCNow().Add(time.Second * 25),
		UserID:     int64(user.ID),
		Updates: []*ConfigUpdate{
			NewConfigUpdate(AppTypeKea, "host_delete", 1),
		},
	}
	err = AddScheduledConfigChange(db, change)
	require.NoError(t, err)

	change = &ScheduledConfigChange{
		CreatedAt:  storkutil.UTCNow(),
		DeadlineAt: storkutil.UTCNow().Add(time.Second * 100),
		UserID:     int64(user.ID),
		Updates: []*ConfigUpdate{
			NewConfigUpdate(AppTypeKea, "host_delete", 2),
		},
	}
	err = AddScheduledConfigChange(db, change)
	require.NoError(t, err)

	// Get time in seconds to next scheduled config change. It should be around
	// 25 seconds away.
	tn, exists, err = GetTimeToNextScheduledConfigChange(db)
	require.NoError(t, err)
	require.True(t, exists)
	require.GreaterOrEqual(t, tn, time.Second*15)
	require.LessOrEqual(t, tn, time.Second*25)
}

// Test deleting specified scheduled config change.
func TestDeleteScheduledConfigChange(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Scheduled config changes must be associated with a user.
	user := &SystemUser{
		Login:    "test",
		Lastname: "test",
		Name:     "test",
	}
	_, err := CreateUser(db, user)
	require.NoError(t, err)
	require.NotZero(t, user.ID)

	// Add two config changes.
	change1 := &ScheduledConfigChange{
		CreatedAt:  storkutil.UTCNow(),
		DeadlineAt: storkutil.UTCNow().Add(time.Second * 10),
		UserID:     int64(user.ID),
		Updates: []*ConfigUpdate{
			NewConfigUpdate(AppTypeKea, "host_add", 1),
		},
	}
	err = AddScheduledConfigChange(db, change1)
	require.NoError(t, err)
	require.NotZero(t, change1)

	change2 := &ScheduledConfigChange{
		CreatedAt:  storkutil.UTCNow(),
		DeadlineAt: storkutil.UTCNow().Add(time.Second * 10),
		UserID:     int64(user.ID),
		Updates: []*ConfigUpdate{
			NewConfigUpdate(AppTypeKea, "host_update", 1),
		},
	}
	err = AddScheduledConfigChange(db, change2)
	require.NoError(t, err)
	require.NotZero(t, change2)

	// Delete first change by ID.
	err = DeleteScheduledConfigChange(db, change1.ID)
	require.NoError(t, err)

	// There should be one left.
	returned, err := GetScheduledConfigChanges(db)
	require.NoError(t, err)
	require.Len(t, returned, 1)

	// An attempt to delete the same change should result in an error.
	err = DeleteScheduledConfigChange(db, change1.ID)
	require.Error(t, err)
	require.ErrorIs(t, pkgerrors.Cause(err), ErrNotExists)

	// Delete the second change.
	err = DeleteScheduledConfigChange(db, change2.ID)
	require.NoError(t, err)

	// There should be no changes left.
	returned, err = GetScheduledConfigChanges(db)
	require.NoError(t, err)
	require.Empty(t, returned)
}

// Test that it is possible to determine that any of the updates pertain
// to Kea.
func TestHasKeaUpdates(t *testing.T) {
	change := ScheduledConfigChange{
		Updates: []*ConfigUpdate{
			NewConfigUpdate("bind9", "dns", 1),
			NewConfigUpdate(AppTypeKea, "host", 2),
		},
	}
	require.True(t, change.HasKeaUpdates())
}

// Test that false is returned for a scheduled configuration change when
// no Kea update is found.
func TestHasKeaUpdatesNoKeaUpdate(t *testing.T) {
	change := ScheduledConfigChange{
		Updates: []*ConfigUpdate{
			NewConfigUpdate("bind9", "dns", 1),
		},
	}
	require.False(t, change.HasKeaUpdates())
}
