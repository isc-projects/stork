package restservice

import (
	http "net/http"
	"slices"
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/require"
	bind9xfr "isc.org/stork/daemondata/bind9xfr"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/server/gen/restapi/operations/dns"
	"isc.org/stork/testutil"
	storkutil "isc.org/stork/util"
)

// Test that the GetZoneTransferStates method returns the correct zone transfer states.
func TestGetZoneTransferStatesNoFiltering(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &dbmodel.Machine{
		Address:   "127.0.0.1",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	machine2 := &dbmodel.Machine{
		Address:   "127.0.0.2",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, machine2)
	require.NoError(t, err)

	daemon := &dbmodel.Daemon{
		MachineID: machine.ID,
		AccessPoints: []*dbmodel.AccessPoint{
			{
				Type:    dbmodel.AccessPointControl,
				Address: "localhost",
				Port:    5300,
			},
		},
	}
	err = dbmodel.AddDaemon(db, daemon)
	require.NoError(t, err)

	daemon2 := &dbmodel.Daemon{
		MachineID: machine2.ID,
		AccessPoints: []*dbmodel.AccessPoint{
			{
				Type:    dbmodel.AccessPointControl,
				Address: "localhost",
				Port:    5300,
			},
		},
	}
	err = dbmodel.AddDaemon(db, daemon2)
	require.NoError(t, err)

	testZoneTransfers := testutil.GetTestZoneTransfers()
	for _, xfr := range testZoneTransfers {
		xfr := &dbmodel.ZoneTransferState{
			DaemonID:        daemon.ID,
			ViewName:        xfr.ViewName,
			ZoneName:        xfr.ZoneName,
			Serial:          xfr.Serial,
			Client:          xfr.Client,
			Server:          xfr.Server,
			MessagesCount:   xfr.MessagesCount,
			RecordsCount:    xfr.RecordsCount,
			BytesCount:      xfr.BytesCount,
			BytesPerSecond:  1024,
			Duration:        xfr.Duration,
			Status:          xfr.Status,
			StartedAt:       xfr.StartTime,
			CompletedAt:     xfr.CompletionTime,
			Message:         xfr.Message,
			ClientMachineID: machine.ID,
			ServerMachineID: machine2.ID,
		}
		err = dbmodel.AddOrUpdateZoneTransferState(db, xfr)
		require.NoError(t, err)
	}

	rapi, err := NewRestAPI(&RestAPISettings{}, &dbops.DatabaseSettings{}, db)
	require.NoError(t, err)
	require.NotNil(t, rapi)

	params := dns.GetZoneTransferStatesParams{}

	rsp := rapi.GetZoneTransferStates(t.Context(), params)
	require.IsType(t, &dns.GetZoneTransferStatesOK{}, rsp)
	okResp := rsp.(*dns.GetZoneTransferStatesOK)
	require.Len(t, okResp.Payload.Items, len(testZoneTransfers))
	require.EqualValues(t, okResp.Payload.Total, len(testZoneTransfers))

	// Validate the returned zone transfer states.
	for i, zoneTransfer := range okResp.Payload.Items {
		// Find the corresponding test zone transfer.
		index := slices.IndexFunc(testZoneTransfers, func(testXFR *bind9xfr.State) bool {
			return testXFR.ViewName == zoneTransfer.ViewName && testXFR.ZoneName == zoneTransfer.ZoneName
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
		require.EqualValues(t, 1024, zoneTransfer.BytesPerSecond)
		require.EqualValues(t, testZoneTransfers[index].Status, zoneTransfer.Status)
		require.Equal(t, strfmt.DateTime(testZoneTransfers[index].StartTime), zoneTransfer.StartedAt)
		require.Equal(t, testZoneTransfers[index].Message, zoneTransfer.Message)
		require.Equal(t, machine.ID, zoneTransfer.ClientMachineID)
		require.Equal(t, machine2.ID, zoneTransfer.ServerMachineID)

		require.NotNil(t, zoneTransfer.ClientMachineID)
		require.Equal(t, machine.ID, zoneTransfer.ClientMachineID)
		require.Equal(t, machine.Address, zoneTransfer.ClientMachineAddress)
		require.Equal(t, machine.AgentPort, zoneTransfer.ClientMachineAgentPort)
		require.NotNil(t, zoneTransfer.ServerMachineID)
		require.Equal(t, machine2.ID, zoneTransfer.ServerMachineID)
		require.Equal(t, machine2.Address, zoneTransfer.ServerMachineAddress)
		require.Equal(t, machine2.AgentPort, zoneTransfer.ServerMachineAgentPort)

		switch {
		case testZoneTransfers[index].CompletionTime.IsZero():
			require.Nil(t, zoneTransfer.CompletedAt)
		default:
			require.Equal(t, strfmt.DateTime(testZoneTransfers[index].CompletionTime), *zoneTransfer.CompletedAt)
		}

		switch {
		case testZoneTransfers[index].Duration.Nanoseconds() > 0:
			require.EqualValues(t, testZoneTransfers[index].Duration, zoneTransfer.Duration)
		case testZoneTransfers[index].Duration.Nanoseconds() == 0 && !zoneTransfer.StartedAt.IsZero():
			require.InDelta(t, time.Since(testZoneTransfers[index].StartTime), time.Duration(zoneTransfer.Duration), float64(10*time.Second))
		default:
			require.Zero(t, zoneTransfer.Duration)
		}

		if i > 0 {
			// Ensure correct sorting order.
			require.GreaterOrEqual(t, okResp.Payload.Items[i-1].StartedAt, zoneTransfer.StartedAt)
		}
	}

	// Make sure that the pagination works correctly.
	params.Start = storkutil.Ptr[int64](1)
	params.Limit = storkutil.Ptr[int64](2)

	rsp2 := rapi.GetZoneTransferStates(t.Context(), params)
	require.IsType(t, &dns.GetZoneTransferStatesOK{}, rsp2)
	okResp2 := rsp2.(*dns.GetZoneTransferStatesOK)
	require.Len(t, okResp2.Payload.Items, 2)
	require.EqualValues(t, okResp2.Payload.Total, len(testZoneTransfers))

	require.Equal(t, okResp.Payload.Items[1].ID, okResp2.Payload.Items[0].ID)
	require.Equal(t, okResp.Payload.Items[2].ID, okResp2.Payload.Items[1].ID)
}

// Test that the GetZoneTransferStates method returns the correct zone transfer states
// when the client and server machine IDs are not set.
func TestGetZoneTransferStatesNoMachineIDs(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &dbmodel.Machine{
		Address:   "127.0.0.1",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	daemon := &dbmodel.Daemon{
		MachineID: machine.ID,
		AccessPoints: []*dbmodel.AccessPoint{
			{
				Type:    dbmodel.AccessPointControl,
				Address: "localhost",
				Port:    5300,
			},
		},
	}
	err = dbmodel.AddDaemon(db, daemon)
	require.NoError(t, err)

	testZoneTransfers := testutil.GetTestZoneTransfers()
	for _, xfr := range testZoneTransfers {
		xfr := &dbmodel.ZoneTransferState{
			DaemonID:      daemon.ID,
			ViewName:      xfr.ViewName,
			ZoneName:      xfr.ZoneName,
			Serial:        xfr.Serial,
			Client:        xfr.Client,
			Server:        xfr.Server,
			MessagesCount: xfr.MessagesCount,
			RecordsCount:  xfr.RecordsCount,
			BytesCount:    xfr.BytesCount,
			Duration:      xfr.Duration,
			Status:        xfr.Status,
			StartedAt:     xfr.StartTime,
			CompletedAt:   xfr.CompletionTime,
			Message:       xfr.Message,
		}
		err = dbmodel.AddOrUpdateZoneTransferState(db, xfr)
		require.NoError(t, err)
	}

	rapi, err := NewRestAPI(&RestAPISettings{}, &dbops.DatabaseSettings{}, db)
	require.NoError(t, err)
	require.NotNil(t, rapi)

	params := dns.GetZoneTransferStatesParams{}

	rsp := rapi.GetZoneTransferStates(t.Context(), params)
	require.IsType(t, &dns.GetZoneTransferStatesOK{}, rsp)
	okResp := rsp.(*dns.GetZoneTransferStatesOK)
	require.Len(t, okResp.Payload.Items, len(testZoneTransfers))
	require.EqualValues(t, okResp.Payload.Total, len(testZoneTransfers))

	// Validate the returned zone transfer states.
	for _, zoneTransfer := range okResp.Payload.Items {
		// Find the corresponding test zone transfer.
		index := slices.IndexFunc(testZoneTransfers, func(testXFR *bind9xfr.State) bool {
			return testXFR.ViewName == zoneTransfer.ViewName && testXFR.ZoneName == zoneTransfer.ZoneName
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
		require.Zero(t, zoneTransfer.BytesPerSecond)
		require.EqualValues(t, testZoneTransfers[index].Status, zoneTransfer.Status)
		require.Equal(t, strfmt.DateTime(testZoneTransfers[index].StartTime), zoneTransfer.StartedAt)
		require.Equal(t, testZoneTransfers[index].Message, zoneTransfer.Message)

		switch {
		case testZoneTransfers[index].CompletionTime.IsZero():
			require.Nil(t, zoneTransfer.CompletedAt)
		default:
			require.Equal(t, strfmt.DateTime(testZoneTransfers[index].CompletionTime), *zoneTransfer.CompletedAt)
		}

		switch {
		case testZoneTransfers[index].Duration > 0:
			require.EqualValues(t, testZoneTransfers[index].Duration, zoneTransfer.Duration)
		case testZoneTransfers[index].Duration == 0 && !zoneTransfer.StartedAt.IsZero():
			require.InDelta(t, time.Since(testZoneTransfers[index].StartTime), time.Duration(zoneTransfer.Duration), float64(10*time.Second))
		default:
			require.Zero(t, zoneTransfer.Duration)
		}

		// Machine relations are not set.
		require.Zero(t, zoneTransfer.ClientMachineID)
		require.Empty(t, zoneTransfer.ClientMachineAddress)
		require.Zero(t, zoneTransfer.ClientMachineAgentPort)
		require.Zero(t, zoneTransfer.ServerMachineID)
		require.Empty(t, zoneTransfer.ServerMachineAddress)
		require.Zero(t, zoneTransfer.ServerMachineAgentPort)
	}
}

// Test that filtering using different fields works correctly while
// getting the zone transfer states.
func TestGetZoneTransferStatesWithFiltering(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &dbmodel.Machine{
		Address:   "127.0.0.1",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	machine2 := &dbmodel.Machine{
		Address:   "127.0.0.2",
		AgentPort: 8080,
	}
	err = dbmodel.AddMachine(db, machine2)
	require.NoError(t, err)

	daemon := &dbmodel.Daemon{
		MachineID: machine.ID,
		AccessPoints: []*dbmodel.AccessPoint{
			{
				Type:    dbmodel.AccessPointControl,
				Address: "localhost",
				Port:    5300,
			},
		},
	}
	err = dbmodel.AddDaemon(db, daemon)
	require.NoError(t, err)

	daemon2 := &dbmodel.Daemon{
		MachineID: machine2.ID,
		AccessPoints: []*dbmodel.AccessPoint{
			{
				Type:    dbmodel.AccessPointControl,
				Address: "localhost",
				Port:    5300,
			},
		},
	}
	err = dbmodel.AddDaemon(db, daemon2)
	require.NoError(t, err)

	testZoneTransfers := testutil.GetTestZoneTransfers()
	testZoneTransfers = append(testZoneTransfers, testutil.GetTestLocalZoneTransfers()...)
	for i, xfr := range testZoneTransfers {
		xfr := &dbmodel.ZoneTransferState{
			DaemonID:      daemon.ID,
			ViewName:      xfr.ViewName,
			ZoneName:      xfr.ZoneName,
			Serial:        xfr.Serial,
			Client:        xfr.Client,
			Server:        xfr.Server,
			MessagesCount: xfr.MessagesCount,
			RecordsCount:  xfr.RecordsCount,
			BytesCount:    xfr.BytesCount,
			Duration:      xfr.Duration,
			Status:        xfr.Status,
			StartedAt:     xfr.StartTime,
			CompletedAt:   xfr.CompletionTime,
			Message:       xfr.Message,
			Local:         xfr.Client == "127.0.0.1" && xfr.Server == "127.0.0.1",
		}
		switch i {
		case 0:
			xfr.ClientMachineID = machine.ID
		case 1:
			xfr.ServerMachineID = machine2.ID
		case 2:
			xfr.ClientMachineID = machine.ID
			xfr.ServerMachineID = machine2.ID
		}
		err = dbmodel.AddOrUpdateZoneTransferState(db, xfr)
		require.NoError(t, err)
	}

	rapi, err := NewRestAPI(&RestAPISettings{}, &dbops.DatabaseSettings{}, db)
	require.NoError(t, err)
	require.NotNil(t, rapi)

	t.Run("filter by serial", func(t *testing.T) {
		params := dns.GetZoneTransferStatesParams{
			Serial: storkutil.Ptr("41601"),
		}
		rapi.GetZoneTransferStates(t.Context(), params)
		rsp := rapi.GetZoneTransferStates(t.Context(), params)
		require.IsType(t, &dns.GetZoneTransferStatesOK{}, rsp)
		okResp := rsp.(*dns.GetZoneTransferStatesOK)
		require.Len(t, okResp.Payload.Items, 1)
		require.EqualValues(t, okResp.Payload.Total, 1)
		require.EqualValues(t, 2026041601, okResp.Payload.Items[0].Serial)
	})

	t.Run("filter by multiple statuses", func(t *testing.T) {
		params := dns.GetZoneTransferStatesParams{
			Status: []string{"started", "completed"},
		}
		rapi.GetZoneTransferStates(t.Context(), params)
		rsp := rapi.GetZoneTransferStates(t.Context(), params)
		require.IsType(t, &dns.GetZoneTransferStatesOK{}, rsp)
		okResp := rsp.(*dns.GetZoneTransferStatesOK)
		require.Len(t, okResp.Payload.Items, 2)
		require.EqualValues(t, okResp.Payload.Total, 2)
	})

	t.Run("filter by zone transfer statuses unspecified", func(t *testing.T) {
		params := dns.GetZoneTransferStatesParams{
			Status: []string{},
		}
		rapi.GetZoneTransferStates(t.Context(), params)
		rsp := rapi.GetZoneTransferStates(t.Context(), params)
		require.IsType(t, &dns.GetZoneTransferStatesOK{}, rsp)
		okResp := rsp.(*dns.GetZoneTransferStatesOK)
		require.Len(t, okResp.Payload.Items, 6)
		require.EqualValues(t, okResp.Payload.Total, 6)
	})

	t.Run("offset", func(t *testing.T) {
		// Get first 2 zone transfers ordered by status
		params := dns.GetZoneTransferStatesParams{
			Start: storkutil.Ptr[int64](0),
			Limit: storkutil.Ptr[int64](2),
		}
		rapi.GetZoneTransferStates(t.Context(), params)
		rsp := rapi.GetZoneTransferStates(t.Context(), params)
		require.IsType(t, &dns.GetZoneTransferStatesOK{}, rsp)
		okResp := rsp.(*dns.GetZoneTransferStatesOK)
		require.Len(t, okResp.Payload.Items, 2)
		require.EqualValues(t, okResp.Payload.Total, 6)

		// Use the 2nd zone transfer as a start for another fetch.
		params.Start = storkutil.Ptr[int64](1)
		params.Limit = nil
		rsp = rapi.GetZoneTransferStates(t.Context(), params)
		require.IsType(t, &dns.GetZoneTransferStatesOK{}, rsp)
		okResp2 := rsp.(*dns.GetZoneTransferStatesOK)
		require.Len(t, okResp2.Payload.Items, 5)
		require.EqualValues(t, okResp2.Payload.Total, 6)

		// The first returned zone transfer should overlap with the last zone transfer
		// returned during the first fetch.
		require.Equal(t, okResp.Payload.Items[1].ID, okResp2.Payload.Items[0].ID)
	})

	t.Run("filter by machine ID", func(t *testing.T) {
		// Match client machine ID.
		params := dns.GetZoneTransferStatesParams{
			ClientMachineID: storkutil.Ptr(machine.ID),
		}
		rapi.GetZoneTransferStates(t.Context(), params)
		rsp := rapi.GetZoneTransferStates(t.Context(), params)
		require.IsType(t, &dns.GetZoneTransferStatesOK{}, rsp)
		okResp := rsp.(*dns.GetZoneTransferStatesOK)
		require.Len(t, okResp.Payload.Items, 2)
		require.EqualValues(t, okResp.Payload.Total, 2)
		for _, zoneTransfer := range okResp.Payload.Items {
			require.Equal(t, machine.ID, zoneTransfer.ClientMachineID)
		}

		// Match server machine ID.
		params.ClientMachineID = nil
		params.ServerMachineID = storkutil.Ptr(machine2.ID)
		rapi.GetZoneTransferStates(t.Context(), params)
		rsp2 := rapi.GetZoneTransferStates(t.Context(), params)
		require.IsType(t, &dns.GetZoneTransferStatesOK{}, rsp)
		okResp2 := rsp2.(*dns.GetZoneTransferStatesOK)
		require.Len(t, okResp.Payload.Items, 2)
		require.EqualValues(t, okResp2.Payload.Total, 2)
		for _, zoneTransfer := range okResp2.Payload.Items {
			require.Equal(t, machine2.ID, zoneTransfer.ServerMachineID)
		}

		// Match both client and server machine IDs.
		params.ClientMachineID = storkutil.Ptr(machine.ID)
		rsp3 := rapi.GetZoneTransferStates(t.Context(), params)
		require.IsType(t, &dns.GetZoneTransferStatesOK{}, rsp3)
		okResp3 := rsp3.(*dns.GetZoneTransferStatesOK)
		require.Len(t, okResp3.Payload.Items, 1)
		require.EqualValues(t, 1, okResp3.Payload.Total)

		// No matches.
		params.ClientMachineID = storkutil.Ptr(machine2.ID)
		params.ServerMachineID = storkutil.Ptr(machine.ID)
		rsp4 := rapi.GetZoneTransferStates(t.Context(), params)
		require.IsType(t, &dns.GetZoneTransferStatesOK{}, rsp4)
		okResp4 := rsp4.(*dns.GetZoneTransferStatesOK)
		require.Len(t, okResp4.Payload.Items, 0)
		require.Zero(t, okResp4.Payload.Total)
	})

	t.Run("filter by text", func(t *testing.T) {
		params := dns.GetZoneTransferStatesParams{
			Text: storkutil.Ptr("good.example"),
		}
		rapi.GetZoneTransferStates(t.Context(), params)
		rsp := rapi.GetZoneTransferStates(t.Context(), params)
		require.IsType(t, &dns.GetZoneTransferStatesOK{}, rsp)
		okResp := rsp.(*dns.GetZoneTransferStatesOK)
		require.Len(t, okResp.Payload.Items, 1)
		require.EqualValues(t, okResp.Payload.Total, 1)
		require.Equal(t, "good.example.org", okResp.Payload.Items[0].ZoneName)
	})

	t.Run("filter with including local zone transfers", func(t *testing.T) {
		params := dns.GetZoneTransferStatesParams{
			IncludeLocal: storkutil.Ptr(true),
		}
		rapi.GetZoneTransferStates(t.Context(), params)
		rsp := rapi.GetZoneTransferStates(t.Context(), params)
		require.IsType(t, &dns.GetZoneTransferStatesOK{}, rsp)
		okResp := rsp.(*dns.GetZoneTransferStatesOK)
		require.Len(t, okResp.Payload.Items, 7)
		require.EqualValues(t, okResp.Payload.Total, 7)
		for _, zoneTransfer := range okResp.Payload.Items {
			if zoneTransfer.Client == "127.0.0.1" && zoneTransfer.Server == "127.0.0.1" {
				require.True(t, zoneTransfer.Local)
			} else {
				require.False(t, zoneTransfer.Local)
			}
		}
	})

	t.Run("sort by effective duration", func(t *testing.T) {
		params := dns.GetZoneTransferStatesParams{
			SortField: storkutil.Ptr("effective_duration"),
			SortDir:   storkutil.Ptr(string(dbmodel.SortDirDesc)),
		}
		rapi.GetZoneTransferStates(t.Context(), params)
		rsp := rapi.GetZoneTransferStates(t.Context(), params)
		require.IsType(t, &dns.GetZoneTransferStatesOK{}, rsp)
		okResp := rsp.(*dns.GetZoneTransferStatesOK)
		require.Len(t, okResp.Payload.Items, 6)
		require.EqualValues(t, okResp.Payload.Total, 6)
		for i := range okResp.Payload.Items {
			if i > 0 {
				require.Less(t, okResp.Payload.Items[i].Duration, okResp.Payload.Items[i-1].Duration)
			}
		}
	})

	t.Run("default sort", func(t *testing.T) {
		params := dns.GetZoneTransferStatesParams{}
		rapi.GetZoneTransferStates(t.Context(), params)
		rsp := rapi.GetZoneTransferStates(t.Context(), params)
		require.IsType(t, &dns.GetZoneTransferStatesOK{}, rsp)
		okResp := rsp.(*dns.GetZoneTransferStatesOK)
		require.Len(t, okResp.Payload.Items, 6)
		require.EqualValues(t, okResp.Payload.Total, 6)
		for i := range okResp.Payload.Items {
			if i > 0 {
				require.Greater(t, okResp.Payload.Items[i-1].StartedAt, okResp.Payload.Items[i].StartedAt)
			}
		}
	})
}

// Test that the GetZoneTransferStates method returns an error when the database operation fails.
func TestGetZoneTransferStatesDatabaseError(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	// Teardown the database immediately to cause an error while using
	// the REST API.
	teardown()

	rapi, err := NewRestAPI(&RestAPISettings{}, &dbops.DatabaseSettings{}, db)
	require.NoError(t, err)
	require.NotNil(t, rapi)

	params := dns.GetZoneTransferStatesParams{}
	rsp := rapi.GetZoneTransferStates(t.Context(), params)
	require.IsType(t, &dns.GetZoneTransferStatesDefault{}, rsp)
	defaultRsp := rsp.(*dns.GetZoneTransferStatesDefault)
	require.Equal(t, http.StatusInternalServerError, getStatusCode(*defaultRsp))
	require.Equal(t, "Failed to get zone transfer states from the database", *defaultRsp.Payload.Message)
}
