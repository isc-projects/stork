package dbmodel

import (
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"isc.org/stork/daemondata/bind9xfr"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/testutil"
	storkutil "isc.org/stork/util"
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

	machine2 := &Machine{
		Address:   "127.0.0.2",
		AgentPort: 8080,
	}
	err = AddMachine(db, machine2)
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
			DaemonID:        daemon.ID,
			ViewName:        zoneTransfer.ViewName,
			ZoneName:        zoneTransfer.ZoneName,
			Serial:          zoneTransfer.Serial,
			Client:          zoneTransfer.Client,
			Server:          zoneTransfer.Server,
			MessagesCount:   zoneTransfer.MessagesCount,
			RecordsCount:    zoneTransfer.RecordsCount,
			BytesCount:      zoneTransfer.BytesCount,
			Duration:        zoneTransfer.Duration,
			Status:          zoneTransfer.Status,
			StartedAt:       zoneTransfer.StartTime,
			CompletedAt:     zoneTransfer.CompletionTime,
			Message:         zoneTransfer.Message,
			Local:           false,
			ClientMachineID: machine.ID,
			ServerMachineID: machine2.ID,
		}
		if zoneTransfer.Duration > 0 {
			zoneTransfer.BytesPerSecond = zoneTransfer.BytesCount * int64(time.Second) / zoneTransfer.Duration.Nanoseconds()
		}
		err = AddOrUpdateZoneTransferState(db, zoneTransfer)
		require.NoError(t, err)
	}

	filter := &GetZoneTransferStatesFilter{}
	zoneTransfers, total, err := GetZoneTransferStatesByPage(db, filter, "status", SortDirDesc, ZoneTransferStateRelationClientMachine, ZoneTransferStateRelationServerMachine)
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
		require.Equal(t, testZoneTransfers[index].StartTime, zoneTransfer.StartedAt)
		require.Equal(t, testZoneTransfers[index].CompletionTime, zoneTransfer.CompletedAt)
		require.Equal(t, testZoneTransfers[index].Message, zoneTransfer.Message)
		require.False(t, zoneTransfer.Local)
		require.Equal(t, machine.ID, zoneTransfer.ClientMachineID)
		require.Equal(t, machine2.ID, zoneTransfer.ServerMachineID)

		switch {
		case testZoneTransfers[index].Duration > 0:
			require.Equal(t, testZoneTransfers[index].Duration, zoneTransfer.Duration)
			require.Equal(t, testZoneTransfers[index].BytesCount*int64(time.Second)/testZoneTransfers[index].Duration.Nanoseconds(), zoneTransfer.BytesPerSecond)
		case testZoneTransfers[index].Duration == 0 && !zoneTransfer.StartedAt.IsZero():
			require.InDelta(t, zoneTransfer.EffectiveDuration, time.Since(zoneTransfer.StartedAt), float64(1*time.Second))
		default:
			require.Zero(t, zoneTransfer.Duration)
			require.Zero(t, zoneTransfer.BytesPerSecond)
		}

		require.NotNil(t, zoneTransfer.ClientMachine)
		require.Equal(t, machine.ID, zoneTransfer.ClientMachine.ID)
		require.Equal(t, machine.Address, zoneTransfer.ClientMachine.Address)
		require.NotNil(t, zoneTransfer.ServerMachine)
		require.Equal(t, machine2.ID, zoneTransfer.ServerMachine.ID)
		require.Equal(t, machine2.Address, zoneTransfer.ServerMachine.Address)

		if i > 0 {
			// Ensure correct sorting order.
			require.GreaterOrEqual(t, zoneTransfers[i-1].Status, zoneTransfer.Status)
		}
	}
}

// Test getting the zone transfer states by page without relations.
func TestGetZoneTransferStatesByPageNoRelations(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &Machine{
		Address:   "127.0.0.1",
		AgentPort: 8080,
	}
	err := AddMachine(db, machine)
	require.NoError(t, err)

	machine2 := &Machine{
		Address:   "127.0.0.2",
		AgentPort: 8080,
	}
	err = AddMachine(db, machine2)
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
			DaemonID:        daemon.ID,
			ViewName:        zoneTransfer.ViewName,
			ZoneName:        zoneTransfer.ZoneName,
			Serial:          zoneTransfer.Serial,
			Client:          zoneTransfer.Client,
			Server:          zoneTransfer.Server,
			MessagesCount:   zoneTransfer.MessagesCount,
			RecordsCount:    zoneTransfer.RecordsCount,
			BytesCount:      zoneTransfer.BytesCount,
			Duration:        zoneTransfer.Duration,
			Status:          zoneTransfer.Status,
			StartedAt:       zoneTransfer.StartTime,
			CompletedAt:     zoneTransfer.CompletionTime,
			Message:         zoneTransfer.Message,
			ClientMachineID: machine.ID,
			ServerMachineID: machine2.ID,
		}
		err = AddOrUpdateZoneTransferState(db, zoneTransfer)
		require.NoError(t, err)
	}

	zoneTransfers, total, err := GetZoneTransferStatesByPage(db, nil, "status", SortDirAsc)
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
		require.Equal(t, testZoneTransfers[index].StartTime, zoneTransfer.StartedAt)
		require.Equal(t, testZoneTransfers[index].CompletionTime, zoneTransfer.CompletedAt)
		require.Equal(t, testZoneTransfers[index].Message, zoneTransfer.Message)
		require.False(t, zoneTransfer.Local)
		require.Equal(t, machine.ID, zoneTransfer.ClientMachineID)
		require.Equal(t, machine2.ID, zoneTransfer.ServerMachineID)

		switch {
		case testZoneTransfers[index].Duration > 0:
			require.Equal(t, testZoneTransfers[index].Duration, zoneTransfer.Duration)
		case testZoneTransfers[index].Duration == 0 && !zoneTransfer.StartedAt.IsZero():
			require.InDelta(t, time.Since(zoneTransfer.StartedAt), zoneTransfer.EffectiveDuration, float64(10*time.Second))
		default:
			require.Zero(t, zoneTransfer.Duration)
		}

		require.Nil(t, zoneTransfer.ClientMachine)
		require.Nil(t, zoneTransfer.ServerMachine)

		if i > 0 {
			// Ensure correct sorting order.
			require.LessOrEqual(t, zoneTransfers[i-1].Status, zoneTransfer.Status)
		}
	}
}

// Test the case of adding a started zone transfer to the database and then
// overriding it with the completed zone transfer for the same zone and view.
func TestAddOrUpdateZoneTransfersOverrideStartedByCompleted(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &Machine{
		Address:   "127.0.0.1",
		AgentPort: 8080,
	}
	err := AddMachine(db, machine)
	require.NoError(t, err)

	machine2 := &Machine{
		Address:   "127.0.0.2",
		AgentPort: 8080,
	}
	err = AddMachine(db, machine2)
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
		StartedAt: time.Date(2026, 4, 16, 10, 41, 27, 71000, time.UTC),
	}
	// Add the corresponding completed zone transfer to the database. It has
	// the same daemon ID, start time, zone name, view name and client, so it
	// should be identified as the same zone transfer. It should override the
	// started zone transfer.
	completed := &ZoneTransferState{
		DaemonID:        daemon.ID,
		CreatedAt:       time.Date(2026, 4, 16, 10, 42, 3, 71000, time.UTC),
		ViewName:        "_default",
		ZoneName:        "good.example.org",
		Serial:          storkutil.Ptr(int64(2026041600)),
		Client:          "127.0.0.1",
		Status:          bind9xfr.StatusCompleted,
		StartedAt:       time.Date(2026, 4, 16, 10, 41, 27, 71000, time.UTC),
		CompletedAt:     time.Date(2026, 4, 16, 10, 45, 11, 124000, time.UTC),
		Duration:        4 * time.Minute,
		MessagesCount:   79,
		RecordsCount:    24872,
		BytesCount:      1320233,
		Message:         "Transfer completed: 79 messages, 24872 records, 1320233 bytes, 0.052 secs (25389096 bytes/sec) (serial 2026041600)",
		Local:           true,
		ClientMachineID: machine.ID,
		ServerMachineID: machine2.ID,
	}
	// Add them to the database sequentially.
	for _, zoneTransfer := range []ZoneTransferState{*started, *completed} {
		err = AddOrUpdateZoneTransferState(db, &zoneTransfer)
		require.NoError(t, err)
	}

	// Make sure there is only one instance in the database, and it is
	// the second one.
	returned, total, err := GetZoneTransferStatesByPage(db, nil, "", SortDirAny, ZoneTransferStateRelationClientMachine, ZoneTransferStateRelationServerMachine)
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
	require.Equal(t, completed.StartedAt, returned[0].StartedAt)
	require.Equal(t, completed.CompletedAt, returned[0].CompletedAt)
	require.Equal(t, completed.Duration, returned[0].Duration)
	require.Equal(t, completed.MessagesCount, returned[0].MessagesCount)
	require.Equal(t, completed.RecordsCount, returned[0].RecordsCount)
	require.Equal(t, completed.BytesCount, returned[0].BytesCount)
	require.Equal(t, completed.Message, returned[0].Message)
	require.True(t, returned[0].Local)
	require.Equal(t, machine.ID, returned[0].ClientMachineID)
	require.Equal(t, machine2.ID, returned[0].ServerMachineID)

	require.NotNil(t, returned[0].ClientMachine)
	require.Equal(t, machine.ID, returned[0].ClientMachine.ID)
	require.Equal(t, machine.Address, returned[0].ClientMachine.Address)
	require.NotNil(t, returned[0].ServerMachine)
	require.Equal(t, machine2.ID, returned[0].ServerMachine.ID)
	require.Equal(t, machine2.Address, returned[0].ServerMachine.Address)
}

// Test different scenarios when started zone transfer inserted into the
// database differs with the completed zone transfer by one field.
func TestAddOrUpdateZoneTransfersOverrideDataMismatch(t *testing.T) {
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
					StartedAt: time.Date(2026, 4, 16, 10, 41, 27, 71000, time.UTC),
				},
				{
					ViewName:  "view2",
					ZoneName:  "good.example.org",
					Client:    "127.0.0.1",
					Status:    bind9xfr.StatusCompleted,
					StartedAt: time.Date(2026, 4, 16, 10, 41, 27, 71000, time.UTC),
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
					StartedAt: time.Date(2026, 4, 16, 10, 41, 27, 71000, time.UTC),
				},
				{
					ViewName:  "_default",
					ZoneName:  "zone2.example.org",
					Client:    "127.0.0.1",
					Status:    bind9xfr.StatusCompleted,
					StartedAt: time.Date(2026, 4, 16, 10, 41, 27, 71000, time.UTC),
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
					StartedAt: time.Date(2026, 4, 16, 10, 41, 27, 71000, time.UTC),
				},
				{
					ViewName:  "_default",
					ZoneName:  "good.example.org",
					Client:    "2.2.2.2",
					Status:    bind9xfr.StatusCompleted,
					StartedAt: time.Date(2026, 4, 16, 10, 41, 27, 71000, time.UTC),
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
					StartedAt: time.Date(2026, 4, 16, 10, 41, 27, 71000, time.UTC),
				},
				{
					ViewName:  "_default",
					ZoneName:  "good.example.org",
					Client:    "1.1.1.1",
					Status:    bind9xfr.StatusCompleted,
					StartedAt: time.Date(2026, 4, 16, 10, 41, 28, 71000, time.UTC),
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
				err = AddOrUpdateZoneTransferState(db, &zoneTransfer)
				require.NoError(t, err)
			}

			// Since there is a mismatch between the started and completed zone transfer states,
			// they should both be present in the database.
			returned, total, err := GetZoneTransferStatesByPage(db, nil, "", SortDirAny)
			require.NoError(t, err)
			require.Len(t, returned, 2)
			require.EqualValues(t, total, 2)
		})
	}
}

// Test getting zone transfer states by page with and without filtering.
func TestGetZoneTransferStatesByPageWithFiltering(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &Machine{
		Address:   "127.0.0.1",
		AgentPort: 8080,
	}
	err := AddMachine(db, machine)
	require.NoError(t, err)

	machine2 := &Machine{
		Address:   "127.0.0.2",
		AgentPort: 8080,
	}
	err = AddMachine(db, machine2)
	require.NoError(t, err)

	daemon := &Daemon{
		ID:        1,
		MachineID: machine.ID,
	}
	err = AddDaemon(db, daemon)
	require.NoError(t, err)

	testZoneTransfers := testutil.GetTestZoneTransfers()
	for i, zoneTransfer := range testZoneTransfers {
		zoneTransfer := &ZoneTransferState{
			DaemonID:        daemon.ID,
			ViewName:        zoneTransfer.ViewName,
			ZoneName:        zoneTransfer.ZoneName,
			Serial:          zoneTransfer.Serial,
			Client:          zoneTransfer.Client,
			Server:          zoneTransfer.Server,
			MessagesCount:   zoneTransfer.MessagesCount,
			RecordsCount:    zoneTransfer.RecordsCount,
			BytesCount:      zoneTransfer.BytesCount,
			Duration:        zoneTransfer.Duration,
			Status:          zoneTransfer.Status,
			StartedAt:       zoneTransfer.StartTime,
			CompletedAt:     zoneTransfer.CompletionTime,
			Message:         zoneTransfer.Message,
			Local:           i%2 == 0,
			ClientMachineID: machine.ID,
			ServerMachineID: machine2.ID,
		}
		err = AddOrUpdateZoneTransferState(db, zoneTransfer)
		require.NoError(t, err)
	}

	t.Run("no filtering", func(t *testing.T) {
		// Without filtering we should get all zone transfer states.
		zoneTransfers, total, err := GetZoneTransferStatesByPage(db, nil, "", SortDirAny)
		require.NoError(t, err)
		require.EqualValues(t, len(testZoneTransfers), total)
		require.Len(t, zoneTransfers, len(testZoneTransfers))
	})

	t.Run("filter by serial", func(t *testing.T) {
		filter := &GetZoneTransferStatesFilter{
			Serial: storkutil.Ptr("41601"),
		}
		zoneTransfers, total, err := GetZoneTransferStatesByPage(db, filter, "", SortDirAny)
		require.NoError(t, err)
		require.EqualValues(t, 1, total)
		require.Len(t, zoneTransfers, 1)
		for _, zoneTransfer := range zoneTransfers {
			require.NotNil(t, zoneTransfer.Serial)
			require.EqualValues(t, 2026041601, *zoneTransfer.Serial)
		}
	})

	t.Run("filter by zero serial", func(t *testing.T) {
		filter := &GetZoneTransferStatesFilter{
			Serial: storkutil.Ptr("0"),
		}
		zoneTransfers, total, err := GetZoneTransferStatesByPage(db, filter, "", SortDirAny)
		require.NoError(t, err)
		require.EqualValues(t, 6, total)
		require.Len(t, zoneTransfers, 6)
		var zeroSerialIncluded bool
		for _, zoneTransfer := range zoneTransfers {
			if zoneTransfer.Serial != nil && *zoneTransfer.Serial == 0 {
				zeroSerialIncluded = true
			}
		}
		require.True(t, zeroSerialIncluded)
	})

	t.Run("filter by single status", func(t *testing.T) {
		filter := &GetZoneTransferStatesFilter{}
		filter.EnableStatus(bind9xfr.StatusStarted)
		zoneTransfers, total, err := GetZoneTransferStatesByPage(db, filter, "", SortDirAny)
		require.NoError(t, err)
		require.EqualValues(t, 1, total)
		require.Len(t, zoneTransfers, 1)
		require.Equal(t, bind9xfr.StatusStarted, zoneTransfers[0].Status)
	})

	t.Run("filter by multiple statuses", func(t *testing.T) {
		filter := &GetZoneTransferStatesFilter{}
		filter.EnableStatus(bind9xfr.StatusStarted)
		filter.EnableStatus(bind9xfr.StatusCompleted)
		zoneTransfers, total, err := GetZoneTransferStatesByPage(db, filter, "status", SortDirAsc)
		require.NoError(t, err)
		require.EqualValues(t, 3, total)
		require.Len(t, zoneTransfers, 3)
		require.Equal(t, bind9xfr.StatusCompleted, zoneTransfers[0].Status)
		require.Equal(t, bind9xfr.StatusCompleted, zoneTransfers[1].Status)
		require.Equal(t, bind9xfr.StatusStarted, zoneTransfers[2].Status)
	})

	t.Run("filter by zone transfer statuses unspecified", func(t *testing.T) {
		filter := &GetZoneTransferStatesFilter{
			Statuses: NewGetZoneTransferStatesStatuses(),
		}
		zoneTransfers, total, err := GetZoneTransferStatesByPage(db, filter, "", SortDirAny)
		require.NoError(t, err)
		require.EqualValues(t, len(testZoneTransfers), total)
		require.Len(t, zoneTransfers, len(testZoneTransfers))

		// Collect unique zone transfer statuses from the zone transfers.
		collectedStatuses := make(map[bind9xfr.Status]struct{})
		for _, zoneTransfer := range zoneTransfers {
			collectedStatuses[zoneTransfer.Status] = struct{}{}
		}
		require.Equal(t, 3, len(collectedStatuses))
		require.Contains(t, collectedStatuses, bind9xfr.StatusStarted)
		require.Contains(t, collectedStatuses, bind9xfr.StatusCompleted)
		require.Contains(t, collectedStatuses, bind9xfr.StatusMessage)
	})

	t.Run("offset and limit", func(t *testing.T) {
		// Get first 2 zone transfers ordered by status
		filter := &GetZoneTransferStatesFilter{
			Offset: storkutil.Ptr(0),
			Limit:  storkutil.Ptr(2),
		}
		zoneTransfers1, total, err := GetZoneTransferStatesByPage(db, filter, "", SortDirAny)
		require.NoError(t, err)
		require.EqualValues(t, len(testZoneTransfers), total)
		require.Len(t, zoneTransfers1, 2)

		// Use the 2nd zone transfer as a start for another fetch.
		filter.Offset = storkutil.Ptr(1)
		filter.Limit = nil
		zoneTransfers2, total, err := GetZoneTransferStatesByPage(db, filter, "", SortDirAny)
		require.NoError(t, err)
		require.EqualValues(t, len(testZoneTransfers), total)
		require.Len(t, zoneTransfers2, len(testZoneTransfers)-1)

		// The first returned zone transfer should overlap with the last zone transfer
		// returned during the first fetch.
		require.Equal(t, zoneTransfers1[1].ID, zoneTransfers2[0].ID)
	})

	t.Run("filter by machine ID", func(t *testing.T) {
		filter := &GetZoneTransferStatesFilter{
			ClientMachineID: storkutil.Ptr(machine.ID),
		}
		zoneTransfers, total, err := GetZoneTransferStatesByPage(db, filter, "", SortDirAny)
		require.NoError(t, err)
		require.EqualValues(t, len(testZoneTransfers), total)
		require.Len(t, zoneTransfers, len(testZoneTransfers))

		filter.ServerMachineID = storkutil.Ptr(machine2.ID)
		filter.ClientMachineID = nil
		zoneTransfers2, total, err := GetZoneTransferStatesByPage(db, filter, "", SortDirAny)
		require.NoError(t, err)
		require.EqualValues(t, len(testZoneTransfers), total)
		require.Len(t, zoneTransfers2, len(testZoneTransfers))

		filter.ClientMachineID = storkutil.Ptr(machine2.ID)
		filter.ServerMachineID = storkutil.Ptr(machine.ID)
		zoneTransfers3, total, err := GetZoneTransferStatesByPage(db, filter, "", SortDirAny)
		require.NoError(t, err)
		require.Zero(t, total)
		require.Empty(t, zoneTransfers3)
	})

	t.Run("filter excluding local", func(t *testing.T) {
		filter := &GetZoneTransferStatesFilter{
			ExcludeLocal: true,
		}
		zoneTransfers, total, err := GetZoneTransferStatesByPage(db, filter, "", SortDirAny)
		require.NoError(t, err)
		require.EqualValues(t, len(testZoneTransfers)/2, total)
		require.Len(t, zoneTransfers, len(testZoneTransfers)/2)

		for _, zoneTransfer := range zoneTransfers {
			require.False(t, zoneTransfer.Local)
		}
	})
}

// Test getting zone transfer states by page with text filtering.
func TestGetZoneTransferStatesByPageWithTextFilter(t *testing.T) {
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

	testZoneTransfers := testutil.GetTestZoneTransfers()
	for _, zoneTransfer := range testZoneTransfers {
		zoneTransfer := &ZoneTransferState{
			DaemonID:      daemon.ID,
			ViewName:      zoneTransfer.ViewName,
			ZoneName:      zoneTransfer.ZoneName,
			Serial:        zoneTransfer.Serial,
			Client:        zoneTransfer.Client,
			Server:        zoneTransfer.Server,
			MessagesCount: zoneTransfer.MessagesCount,
			RecordsCount:  zoneTransfer.RecordsCount,
			BytesCount:    zoneTransfer.BytesCount,
			Duration:      zoneTransfer.Duration,
			Status:        zoneTransfer.Status,
			StartedAt:     zoneTransfer.StartTime,
			CompletedAt:   zoneTransfer.CompletionTime,
			Message:       zoneTransfer.Message,
		}
		err = AddOrUpdateZoneTransferState(db, zoneTransfer)
		require.NoError(t, err)
	}

	t.Run("filter by zone name", func(t *testing.T) {
		filter := &GetZoneTransferStatesFilter{
			Text: storkutil.Ptr("good.example"),
		}
		zoneTransfers, total, err := GetZoneTransferStatesByPage(db, filter, "", SortDirAny)
		require.NoError(t, err)
		require.EqualValues(t, 1, total)
		require.Len(t, zoneTransfers, 1)
	})

	t.Run("filter by view name", func(t *testing.T) {
		filter := &GetZoneTransferStatesFilter{
			Text: storkutil.Ptr("_default"),
		}
		zoneTransfers, total, err := GetZoneTransferStatesByPage(db, filter, "", SortDirAny)
		require.NoError(t, err)
		require.EqualValues(t, 3, total)
		require.Len(t, zoneTransfers, 3)
		for _, zoneTransfer := range zoneTransfers {
			require.Equal(t, "_default", zoneTransfer.ViewName)
		}
	})

	t.Run("filter by root zone name", func(t *testing.T) {
		searchKeys := []string{"roo", "root", "(root", "(root)", "Root", "(rooT"}
		for _, searchKey := range searchKeys {
			filter := &GetZoneTransferStatesFilter{
				Text: storkutil.Ptr(searchKey),
			}
			zoneTransfers, total, err := GetZoneTransferStatesByPage(db, filter, "", SortDirAny)
			require.NoError(t, err)
			require.EqualValuesf(t, 1, total, "failed for search key: %s", searchKey)
			require.Lenf(t, zoneTransfers, 1, "failed for search key: %s", searchKey)
		}
	})

	t.Run("filter by client name", func(t *testing.T) {
		filter := &GetZoneTransferStatesFilter{
			Text: storkutil.Ptr("127.0.0.1"),
		}
		zoneTransfers, total, err := GetZoneTransferStatesByPage(db, filter, "", SortDirAny)
		require.NoError(t, err)
		require.EqualValues(t, 2, total)
		require.Len(t, zoneTransfers, 2)
		for _, zoneTransfer := range zoneTransfers {
			require.Equal(t, "127.0.0.1", zoneTransfer.Client)
		}
	})

	t.Run("filter by server name", func(t *testing.T) {
		filter := &GetZoneTransferStatesFilter{
			Text: storkutil.Ptr("192.5.5.241"),
		}
		zoneTransfers, total, err := GetZoneTransferStatesByPage(db, filter, "", SortDirAny)
		require.NoError(t, err)
		require.EqualValues(t, 2, total)
		require.Len(t, zoneTransfers, 2)
		for _, zoneTransfer := range zoneTransfers {
			require.Equal(t, "192.5.5.241", zoneTransfer.Server)
		}
	})

	t.Run("filter by message", func(t *testing.T) {
		filter := &GetZoneTransferStatesFilter{
			Text: storkutil.Ptr("AXFR timed out after 0"),
		}
		zoneTransfers, total, err := GetZoneTransferStatesByPage(db, filter, "", SortDirAny)
		require.NoError(t, err)
		require.EqualValues(t, 1, total)
		require.Len(t, zoneTransfers, 1)
		for _, zoneTransfer := range zoneTransfers {
			require.Equal(t, "Transfer failed: AXFR timed out after 0 seconds (serial 0)", zoneTransfer.Message)
		}
	})
}

// Test getting zone transfer states by page with sorting.
func TestGetZoneTransferStatesByPageWithSorting(t *testing.T) {
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

	testZoneTransfers := testutil.GetTestZoneTransfers()
	for _, zoneTransfer := range testZoneTransfers {
		zoneTransfer := &ZoneTransferState{
			DaemonID:      daemon.ID,
			ViewName:      zoneTransfer.ViewName,
			ZoneName:      zoneTransfer.ZoneName,
			Serial:        zoneTransfer.Serial,
			Client:        zoneTransfer.Client,
			Server:        zoneTransfer.Server,
			MessagesCount: zoneTransfer.MessagesCount,
			RecordsCount:  zoneTransfer.RecordsCount,
			BytesCount:    zoneTransfer.BytesCount,
			Duration:      zoneTransfer.Duration,
			Status:        zoneTransfer.Status,
			StartedAt:     zoneTransfer.StartTime,
			CompletedAt:   zoneTransfer.CompletionTime,
			Message:       zoneTransfer.Message,
		}
		err = AddOrUpdateZoneTransferState(db, zoneTransfer)
		require.NoError(t, err)
	}

	type testCase struct {
		name      string
		sortField string
		sortDir   SortDirEnum
		compareFn func(t *testing.T, next, previous *ZoneTransferState)
	}

	testCases := []testCase{
		{
			name:      "sort by effective duration descending",
			sortField: "effective_duration",
			sortDir:   SortDirDesc,
			compareFn: func(t *testing.T, next, previous *ZoneTransferState) {
				require.Less(t, next.EffectiveDuration, previous.EffectiveDuration)
			},
		},
		{
			name:      "sort by effective duration ascending",
			sortField: "effective_duration",
			sortDir:   SortDirAsc,
			compareFn: func(t *testing.T, next, previous *ZoneTransferState) {
				require.Greater(t, next.EffectiveDuration, previous.EffectiveDuration)
			},
		},
		{
			name:      "sort by status descending",
			sortField: "status",
			sortDir:   SortDirDesc,
			compareFn: func(t *testing.T, next, previous *ZoneTransferState) {
				require.LessOrEqual(t, next.Status, previous.Status)
			},
		},
		{
			name:      "sort by status ascending",
			sortField: "status",
			sortDir:   SortDirAsc,
			compareFn: func(t *testing.T, next, previous *ZoneTransferState) {
				require.GreaterOrEqual(t, next.Status, previous.Status)
			},
		},
		{
			name:      "sort by created at descending",
			sortField: "created_at",
			sortDir:   SortDirDesc,
			compareFn: func(t *testing.T, next, previous *ZoneTransferState) {
				require.Less(t, next.CreatedAt, previous.CreatedAt)
			},
		},
		{
			name:      "sort by created at ascending",
			sortField: "created_at",
			sortDir:   SortDirAsc,
			compareFn: func(t *testing.T, next, previous *ZoneTransferState) {
				require.Greater(t, next.CreatedAt, previous.CreatedAt)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			filter := &GetZoneTransferStatesFilter{}
			zoneTransfers, total, err := GetZoneTransferStatesByPage(db, filter, testCase.sortField, testCase.sortDir)
			require.NoError(t, err)
			require.EqualValues(t, len(testZoneTransfers), total)
			require.Len(t, zoneTransfers, len(testZoneTransfers))
			for i := range zoneTransfers {
				if i > 0 {
					testCase.compareFn(t, zoneTransfers[i], zoneTransfers[i-1])
				}
			}
		})
	}
}

// Test that zone transfer status string is validated on the database side
// against permitted values.
func TestAddOrUpdateZoneTransferStateInvalidStatus(t *testing.T) {
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

	// Add a zone transfer state with an invalid status.
	zoneTransfer := &ZoneTransferState{
		DaemonID:  daemon.ID,
		ViewName:  "_default",
		ZoneName:  "good.example.org",
		Client:    "127.0.0.1",
		Status:    "invalid",
		StartedAt: time.Date(2026, 4, 16, 10, 41, 27, 71000, time.UTC),
	}

	// It should fail with a constraint violation error.
	err = AddOrUpdateZoneTransferState(db, zoneTransfer)
	require.ErrorContains(t, err, "zone_transfer_state_status_check")
}
