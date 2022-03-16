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
	cu := NewConfigUpdate("kea", "host_add", 1, 2, 3)
	require.NotNil(t, cu)
	require.Equal(t, "kea", cu.Target)
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
		Password: "test",
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
			NewConfigUpdate("kea", "host_add", 1, 2, 3),
			NewConfigUpdate("kea", "host_update", 3),
		},
	}
	err = AddScheduledConfigChange(db, change)
	require.NoError(t, err)

	change = &ScheduledConfigChange{
		CreatedAt:  storkutil.UTCNow(),
		DeadlineAt: storkutil.UTCNow().Add(-time.Second * 15),
		UserID:     int64(user.ID),
		Updates: []*ConfigUpdate{
			NewConfigUpdate("kea", "host_delete", 1),
		},
	}
	err = AddScheduledConfigChange(db, change)
	require.NoError(t, err)

	change = &ScheduledConfigChange{
		CreatedAt:  storkutil.UTCNow(),
		DeadlineAt: storkutil.UTCNow().Add(-time.Second * 10),
		UserID:     int64(user.ID),
		Updates: []*ConfigUpdate{
			NewConfigUpdate("kea", "host_delete", 2),
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
	require.Equal(t, "kea", returned[0].Updates[0].Target)
	require.Equal(t, "host_delete", returned[0].Updates[0].Operation)
	require.Len(t, returned[0].Updates[0].DaemonIDs, 1)
	require.EqualValues(t, 1, returned[0].Updates[0].DaemonIDs[0])

	require.NotZero(t, returned[1].ID)
	require.Len(t, returned[1].Updates, 1)
	require.Equal(t, "kea", returned[1].Updates[0].Target)
	require.Equal(t, "host_delete", returned[1].Updates[0].Operation)
	require.Len(t, returned[1].Updates[0].DaemonIDs, 1)
	require.EqualValues(t, 2, returned[1].Updates[0].DaemonIDs[0])

	require.NotZero(t, returned[2].ID)
	require.Len(t, returned[2].Updates, 2)
	require.Equal(t, "kea", returned[0].Updates[0].Target)
	require.Equal(t, "host_add", returned[2].Updates[0].Operation)
	require.Len(t, returned[2].Updates[0].DaemonIDs, 3)
	require.EqualValues(t, 1, returned[2].Updates[0].DaemonIDs[0])
	require.EqualValues(t, 2, returned[2].Updates[0].DaemonIDs[1])
	require.EqualValues(t, 3, returned[2].Updates[0].DaemonIDs[2])
	require.Equal(t, "kea", returned[2].Updates[1].Target)
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
		Password: "test",
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
			NewConfigUpdate("kea", "host_add", 1, 2, 3),
			NewConfigUpdate("kea", "host_update", 3),
		},
	}
	err = AddScheduledConfigChange(db, change)
	require.NoError(t, err)

	change = &ScheduledConfigChange{
		CreatedAt:  storkutil.UTCNow(),
		DeadlineAt: storkutil.UTCNow().Add(-time.Second * 15),
		UserID:     int64(user.ID),
		Updates: []*ConfigUpdate{
			NewConfigUpdate("kea", "host_delete", 1),
		},
	}
	err = AddScheduledConfigChange(db, change)
	require.NoError(t, err)

	change = &ScheduledConfigChange{
		CreatedAt:  storkutil.UTCNow(),
		DeadlineAt: storkutil.UTCNow().Add(-time.Second * 10),
		UserID:     int64(user.ID),
		Updates: []*ConfigUpdate{
			NewConfigUpdate("kea", "host_delete", 2),
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
	require.Equal(t, "kea", returned[0].Updates[0].Target)
	require.Equal(t, "host_delete", returned[0].Updates[0].Operation)
	require.Len(t, returned[0].Updates[0].DaemonIDs, 1)
	require.EqualValues(t, 1, returned[0].Updates[0].DaemonIDs[0])

	require.NotZero(t, returned[1].ID)
	require.Len(t, returned[1].Updates, 1)
	require.Equal(t, "kea", returned[1].Updates[0].Target)
	require.Equal(t, "host_delete", returned[1].Updates[0].Operation)
	require.Len(t, returned[1].Updates[0].DaemonIDs, 1)
	require.EqualValues(t, 2, returned[1].Updates[0].DaemonIDs[0])
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
		Password: "test",
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
			NewConfigUpdate("kea", "host_add", 1),
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
			NewConfigUpdate("kea", "host_update", 1),
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
