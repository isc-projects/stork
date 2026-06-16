package dbmodel

import (
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"isc.org/stork/daemondata/bind9xfr"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/testutil"
)

// Test getting the zone transfer states by page from the database.
func TestGetZoneTransferStatesByPage(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &Machine{
		Address:   "127.0.0.1",
		AgentPort: 8080,
	}
	err := AddMachine(db, machine)
	require.NoError(t, err)

	daemon := &Daemon{
		MachineID: machine.ID,
		AccessPoints: []*AccessPoint{
			{
				Type:    AccessPointControl,
				Address: "127.0.0.1",
				Port:    8080,
			},
		},
	}
	err = AddDaemon(db, daemon)
	require.NoError(t, err)

	testZoneTransfers := testutil.GetTestZoneTransfers()
	for _, zoneTransfer := range testZoneTransfers {
		zoneTransfer := &ZoneTransferState{
			DaemonID:       daemon.ID,
			ViewName:       zoneTransfer.ViewName,
			ZoneName:       zoneTransfer.ZoneName,
			Serial:         zoneTransfer.Serial,
			Client:         zoneTransfer.Client,
			Server:         zoneTransfer.Server,
			MessagesCount:  zoneTransfer.MessagesCount,
			RecordsCount:   zoneTransfer.RecordsCount,
			BytesCount:     zoneTransfer.BytesCount,
			Duration:       zoneTransfer.Duration,
			Status:         zoneTransfer.Status,
			StartTime:      zoneTransfer.StartTime,
			CompletionTime: zoneTransfer.CompletionTime,
			Message:        zoneTransfer.Message,
		}
		err = AddZoneTransferState(db, zoneTransfer)
		require.NoError(t, err)
	}

	zoneTransfers, total, err := GetZoneTransferStatesByPage(db, 0, 10)
	require.NoError(t, err)
	require.Len(t, zoneTransfers, len(testZoneTransfers))
	require.EqualValues(t, total, len(testZoneTransfers))

	// Validate the returned zone transfer states.
	for i, zoneTransfer := range zoneTransfers {
		// Find the corresponding test zone transfer.
		index := slices.IndexFunc(testZoneTransfers, func(testZoneTransfer *bind9xfr.State) bool {
			return testZoneTransfer.ViewName == zoneTransfer.ViewName && testZoneTransfer.ZoneName == zoneTransfer.ZoneName
		})
		require.GreaterOrEqual(t, index, 0)
		require.Equal(t, testZoneTransfers[index].ViewName, zoneTransfer.ViewName)
		require.Equal(t, testZoneTransfers[index].ZoneName, zoneTransfer.ZoneName)
		require.Equal(t, testZoneTransfers[index].Serial, zoneTransfer.Serial)
		require.Equal(t, testZoneTransfers[index].Client, zoneTransfer.Client)
		require.Equal(t, testZoneTransfers[index].Server, zoneTransfer.Server)
		require.Equal(t, testZoneTransfers[index].MessagesCount, zoneTransfer.MessagesCount)
		require.Equal(t, testZoneTransfers[index].RecordsCount, zoneTransfer.RecordsCount)
		require.Equal(t, testZoneTransfers[index].BytesCount, zoneTransfer.BytesCount)
		require.Equal(t, testZoneTransfers[index].Duration, zoneTransfer.Duration)
		require.Equal(t, testZoneTransfers[index].Status, zoneTransfer.Status)
		require.Equal(t, testZoneTransfers[index].StartTime, zoneTransfer.StartTime)
		require.Equal(t, testZoneTransfers[index].CompletionTime, zoneTransfer.CompletionTime)
		require.Equal(t, testZoneTransfers[index].Message, zoneTransfer.Message)

		if i > 0 {
			// Ensure correct sorting order.
			require.LessOrEqual(t, zoneTransfer.CreatedAt, zoneTransfers[i-1].CreatedAt)
		}
	}
}

// Test the case of adding a started zone transfer to the database and then
// overriding it with the completed zone transfer for the same zone and view.
func TestAddZoneTransfersOverrideStartedByCompleted(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &Machine{
		Address:   "127.0.0.1",
		AgentPort: 8080,
	}
	err := AddMachine(db, machine)
	require.NoError(t, err)

	daemon := &Daemon{
		ID:        1,
		MachineID: machine.ID,
	}
	err = AddDaemon(db, daemon)
	require.NoError(t, err)

	// Add the started zone transfer to the database.
	started := &ZoneTransferState{
		DaemonID:  daemon.ID,
		CreatedAt: time.Date(2026, 4, 16, 10, 41, 29, 71000, time.UTC),
		ViewName:  "_default",
		ZoneName:  "good.example.org",
		Client:    "127.0.0.1",
		Status:    bind9xfr.StatusStarted,
		StartTime: time.Date(2026, 4, 16, 10, 41, 27, 71000, time.UTC),
	}
	// Add the corresponding completed zone transfer to the database. It has
	// the same daemon ID, start time, zone name, view name and client, so it
	// should be identified as the same zone transfer. It should override the
	// started zone transfer.
	completed := &ZoneTransferState{
		DaemonID:       daemon.ID,
		CreatedAt:      time.Date(2026, 4, 16, 10, 42, 3, 71000, time.UTC),
		ViewName:       "_default",
		ZoneName:       "good.example.org",
		Serial:         2026041600,
		Client:         "127.0.0.1",
		Status:         bind9xfr.StatusCompleted,
		StartTime:      time.Date(2026, 4, 16, 10, 41, 27, 71000, time.UTC),
		CompletionTime: time.Date(2026, 4, 16, 10, 45, 11, 124000, time.UTC),
		Duration:       4 * time.Minute,
		MessagesCount:  79,
		RecordsCount:   24872,
		BytesCount:     1320233,
		Message:        "Transfer completed: 79 messages, 24872 records, 1320233 bytes, 0.052 secs (25389096 bytes/sec) (serial 2026041600)",
	}
	// Add them to the database sequentially.
	for _, zoneTransfer := range []ZoneTransferState{*started, *completed} {
		err = AddZoneTransferState(db, &zoneTransfer)
		require.NoError(t, err)
	}

	// Make sure there is only one instance in the database, and it is
	// the second one.
	returned, total, err := GetZoneTransferStatesByPage(db, 0, 10)
	require.NoError(t, err)
	require.Len(t, returned, 1)
	require.EqualValues(t, total, 1)

	require.Equal(t, completed.DaemonID, returned[0].DaemonID)
	// It is important that the created_at time was not changed.
	require.Equal(t, started.CreatedAt, returned[0].CreatedAt)
	require.Equal(t, completed.ViewName, returned[0].ViewName)
	require.Equal(t, completed.ZoneName, returned[0].ZoneName)
	require.Equal(t, completed.Serial, returned[0].Serial)
	require.Equal(t, completed.Client, returned[0].Client)
	require.Equal(t, completed.Status, returned[0].Status)
	require.Equal(t, completed.StartTime, returned[0].StartTime)
	require.Equal(t, completed.CompletionTime, returned[0].CompletionTime)
	require.Equal(t, completed.Duration, returned[0].Duration)
	require.Equal(t, completed.MessagesCount, returned[0].MessagesCount)
	require.Equal(t, completed.RecordsCount, returned[0].RecordsCount)
	require.Equal(t, completed.BytesCount, returned[0].BytesCount)
	require.Equal(t, completed.Message, returned[0].Message)
}

// Test different scenarios when started zone transfer inserted into the
// database differs with the completed zone transfer by one field.
func TestAddZoneTransfersOverrideDataMismatch(t *testing.T) {
	t.Parallel()

	// Each test case contains two zone transfer states. The first is the
	// started one, the second is the completed one. The started and completed
	// should differ in view name, zone name, client or start time.
	type testCase struct {
		name          string
		zoneTransfers []ZoneTransferState
	}
	testCases := []testCase{
		{
			name: "different view name",
			zoneTransfers: []ZoneTransferState{
				{
					ViewName:  "view1",
					ZoneName:  "good.example.org",
					Client:    "127.0.0.1",
					Status:    bind9xfr.StatusStarted,
					StartTime: time.Date(2026, 4, 16, 10, 41, 27, 71000, time.UTC),
				},
				{
					ViewName:  "view2",
					ZoneName:  "good.example.org",
					Client:    "127.0.0.1",
					Status:    bind9xfr.StatusCompleted,
					StartTime: time.Date(2026, 4, 16, 10, 41, 27, 71000, time.UTC),
				},
			},
		},
		{
			name: "different zone name",
			zoneTransfers: []ZoneTransferState{
				{
					ViewName:  "_default",
					ZoneName:  "zone1.example.org",
					Client:    "127.0.0.1",
					Status:    bind9xfr.StatusStarted,
					StartTime: time.Date(2026, 4, 16, 10, 41, 27, 71000, time.UTC),
				},
				{
					ViewName:  "_default",
					ZoneName:  "zone2.example.org",
					Client:    "127.0.0.1",
					Status:    bind9xfr.StatusCompleted,
					StartTime: time.Date(2026, 4, 16, 10, 41, 27, 71000, time.UTC),
				},
			},
		},
		{
			name: "different client",
			zoneTransfers: []ZoneTransferState{
				{
					ViewName:  "_default",
					ZoneName:  "good.example.org",
					Client:    "1.1.1.1",
					Status:    bind9xfr.StatusStarted,
					StartTime: time.Date(2026, 4, 16, 10, 41, 27, 71000, time.UTC),
				},
				{
					ViewName:  "_default",
					ZoneName:  "good.example.org",
					Client:    "2.2.2.2",
					Status:    bind9xfr.StatusCompleted,
					StartTime: time.Date(2026, 4, 16, 10, 41, 27, 71000, time.UTC),
				},
			},
		},
		{
			name: "different start time",
			zoneTransfers: []ZoneTransferState{
				{
					ViewName:  "_default",
					ZoneName:  "good.example.org",
					Client:    "1.1.1.1",
					Status:    bind9xfr.StatusStarted,
					StartTime: time.Date(2026, 4, 16, 10, 41, 27, 71000, time.UTC),
				},
				{
					ViewName:  "_default",
					ZoneName:  "good.example.org",
					Client:    "1.1.1.1",
					Status:    bind9xfr.StatusCompleted,
					StartTime: time.Date(2026, 4, 16, 10, 41, 28, 71000, time.UTC),
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			db, _, teardown := dbtest.SetupDatabaseTestCase(t)
			defer teardown()

			machine := &Machine{
				Address:   "127.0.0.1",
				AgentPort: 8080,
			}
			err := AddMachine(db, machine)
			require.NoError(t, err)

			daemon := &Daemon{
				MachineID: machine.ID,
			}
			err = AddDaemon(db, daemon)
			require.NoError(t, err)

			// Zone transfer states must point to a valid daemon ID.
			started := testCase.zoneTransfers[0]
			started.DaemonID = daemon.ID
			completed := testCase.zoneTransfers[1]
			completed.DaemonID = daemon.ID

			for _, zoneTransfer := range []ZoneTransferState{started, completed} {
				// Add started and completed zone transfer state sequentially.
				err = AddZoneTransferState(db, &zoneTransfer)
				require.NoError(t, err)
			}

			// Since there is a mismatch between the started and completed zone transfer states,
			// they should both be present in the database.
			returned, total, err := GetZoneTransferStatesByPage(db, 0, 10)
			require.NoError(t, err)
			require.Len(t, returned, 2)
			require.EqualValues(t, total, 2)
		})
	}
}
