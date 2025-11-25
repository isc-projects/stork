package pdns

import (
	context "context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	pdnsdata "isc.org/stork/daemondata/pdns"
	"isc.org/stork/datamodel/daemonname"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
)

//go:generate mockgen -package=pdns -destination=connectedagentsmock_test.go -source=../../agentcomm/agentcomm.go ConnectedAgents
//go:generate mockgen -package=pdns -destination=eventcentermock_test.go -source=../../eventcenter/eventcenter.go EventCenter

// Error type used in tests.
type testError struct{}

// Returns a string representation of the error.
func (e *testError) Error() string {
	return "test error"
}

// Test successfully getting state from new PowerDNS server.
func TestGetDaemonState(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	agents := NewMockConnectedAgents(ctrl)

	agents.EXPECT().GetPowerDNSServerInfo(gomock.Any(), gomock.Any()).Return(&pdnsdata.ServerInfo{
		Version: "4.7.0",
	}, nil)

	machine := &dbmodel.Machine{
		Address:   "127.0.0.1",
		AgentPort: 1111,
	}

	daemon := dbmodel.NewDaemon(machine, daemonname.PDNS, true, []*dbmodel.AccessPoint{
		{
			Type:    dbmodel.AccessPointControl,
			Address: "127.0.0.1",
			Port:    53,
		},
	})

	GetDaemonState(context.Background(), agents, daemon, nil)

	// Make sure that the daemon contains the returned info.
	require.Equal(t, "4.7.0", daemon.Version)
	require.Equal(t, "4.7.0", daemon.ExtendedVersion)
}

// Test successfully getting state from existing PowerDNS server.
func TestGetDaemonStateUpdateDaemon(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	agents := NewMockConnectedAgents(ctrl)

	agents.EXPECT().GetPowerDNSServerInfo(gomock.Any(), gomock.Any()).Return(&pdnsdata.ServerInfo{
		Version:          "4.7.0",
		Uptime:           1234,
		URL:              "http://127.0.0.1:8081",
		ConfigURL:        "http://127.0.0.1:8081/config",
		ZonesURL:         "http://127.0.0.1:8081/zones",
		AutoprimariesURL: "http://127.0.0.1:8081/autoprimaries",
	}, nil)

	machine := &dbmodel.Machine{
		Address:   "127.0.0.1",
		AgentPort: 1111,
	}

	daemon := dbmodel.NewDaemon(machine, daemonname.PDNS, true, []*dbmodel.AccessPoint{
		{
			Type:    dbmodel.AccessPointControl,
			Address: "127.0.0.1",
			Port:    53,
		},
	})
	daemon.ID = 123
	daemon.Version = "4.5.2"
	daemon.ExtendedVersion = "4.5.2"
	daemon.Uptime = 1234
	daemon.PDNSDaemon.ID = 234
	daemon.PDNSDaemon.DaemonID = 123
	daemon.PDNSDaemon.Details = dbmodel.PDNSDaemonDetails{
		URL:              "http://127.0.0.1:8081",
		ConfigURL:        "http://127.0.0.1:8081/config",
		ZonesURL:         "http://127.0.0.1:8081/zones",
		AutoprimariesURL: "http://127.0.0.1:8081/autoprimaries",
	}

	GetDaemonState(context.Background(), agents, daemon, nil)

	// The existing daemon information should be updated.
	require.EqualValues(t, 123, daemon.ID)
	require.True(t, daemon.Active)
	require.Equal(t, "4.7.0", daemon.Version)
	require.Equal(t, "4.7.0", daemon.ExtendedVersion)
	require.NotNil(t, daemon.PDNSDaemon)
	require.EqualValues(t, 234, daemon.PDNSDaemon.ID)
	require.EqualValues(t, 123, daemon.PDNSDaemon.DaemonID)
	require.EqualValues(t, 1234, daemon.Uptime)
	require.Equal(t, "http://127.0.0.1:8081", daemon.PDNSDaemon.Details.URL)
	require.Equal(t, "http://127.0.0.1:8081/config", daemon.PDNSDaemon.Details.ConfigURL)
	require.Equal(t, "http://127.0.0.1:8081/zones", daemon.PDNSDaemon.Details.ZonesURL)
	require.Equal(t, "http://127.0.0.1:8081/autoprimaries", daemon.PDNSDaemon.Details.AutoprimariesURL)
}

// Test the case when an attempt to get state fails.
func TestGetDaemonStateError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	agents := NewMockConnectedAgents(ctrl)

	agents.EXPECT().GetPowerDNSServerInfo(gomock.Any(), gomock.Any()).Return(nil, &testError{})

	machine := &dbmodel.Machine{
		Address:   "127.0.0.1",
		AgentPort: 1111,
	}

	daemon := dbmodel.NewDaemon(machine, daemonname.PDNS, true, []*dbmodel.AccessPoint{
		{
			Type:    dbmodel.AccessPointControl,
			Address: "127.0.0.1",
			Port:    53,
		},
	})
	daemon.Version = "4.5.2"

	GetDaemonState(context.Background(), agents, daemon, nil)

	// The existing daemon info should not be updated.
	require.Equal(t, "4.5.2", daemon.Version)
}

// Test inserting PowerDNS daemon into the database.
func TestCommitDaemonIntoDB(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)
	require.NotZero(t, machine.ID)

	daemon := dbmodel.NewDaemon(machine, daemonname.PDNS, true, []*dbmodel.AccessPoint{
		{
			Type:    dbmodel.AccessPointControl,
			Address: "",
			Port:    1234,
			Key:     "",
		},
	})

	// Make sure an event is emitted with appropriate parameters when the
	// daemon is first added to the database.
	eventCenterMock := NewMockEventCenter(ctrl)
	eventCenterMock.EXPECT().AddInfoEvent("added {daemon} to {machine}", daemon, daemon.Machine).Times(1)

	// Insert the daemon into the database.
	err = CommitDaemonIntoDB(db, daemon, eventCenterMock)
	require.NoError(t, err)

	// Update the daemon in the database. The event should not be emitted.
	daemon.AccessPoints = []*dbmodel.AccessPoint{
		{
			Type:    dbmodel.AccessPointControl,
			Address: "",
			Port:    2345,
			Key:     "",
		},
	}
	err = CommitDaemonIntoDB(db, daemon, eventCenterMock)
	require.NoError(t, err)

	// Check that the daemon is stored in the database.
	returned, err := dbmodel.GetDaemonByID(db, daemon.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)
	require.Len(t, returned.AccessPoints, 1)
	require.EqualValues(t, 2345, returned.AccessPoints[0].Port)
	require.NotZero(t, returned.ID)
}
