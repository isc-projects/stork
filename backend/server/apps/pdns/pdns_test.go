package pdns

import (
	context "context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	pdnsdata "isc.org/stork/appdata/pdns"
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
func TestGetAppState(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	agents := NewMockConnectedAgents(ctrl)

	agents.EXPECT().GetPowerDNSServerInfo(gomock.Any(), gomock.Any()).Return(&pdnsdata.ServerInfo{
		Version: "4.7.0",
	}, nil)

	dbApp := dbmodel.App{
		AccessPoints: []*dbmodel.AccessPoint{
			{
				Type:    dbmodel.AccessPointControl,
				Address: "127.0.0.1",
				Port:    53,
			},
		},
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 1111,
		},
	}
	GetAppState(context.Background(), agents, &dbApp, nil)

	// Make sure that the daemon is added and contains the returned info.
	require.Len(t, dbApp.Daemons, 1)
	require.Equal(t, "4.7.0", dbApp.Daemons[0].Version)
	require.Equal(t, "4.7.0", dbApp.Meta.Version)
	require.Equal(t, "4.7.0", dbApp.Meta.ExtendedVersion)
}

// Test successfully getting state from existing PowerDNS server.
func TestGetAppStateUpdateDaemon(t *testing.T) {
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

	dbApp := dbmodel.App{
		AccessPoints: []*dbmodel.AccessPoint{
			{
				Type:    dbmodel.AccessPointControl,
				Address: "127.0.0.1",
				Port:    53,
			},
		},
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 1111,
		},
		Daemons: []*dbmodel.Daemon{
			{
				ID:              123,
				Active:          true,
				Version:         "4.5.2",
				ExtendedVersion: "4.5.2",
				Uptime:          1234,
				PDNSDaemon: &dbmodel.PDNSDaemon{
					ID:       234,
					DaemonID: 123,
					Details: dbmodel.PDNSDaemonDetails{
						URL:              "http://127.0.0.1:8081",
						ConfigURL:        "http://127.0.0.1:8081/config",
						ZonesURL:         "http://127.0.0.1:8081/zones",
						AutoprimariesURL: "http://127.0.0.1:8081/autoprimaries",
					},
				},
			},
		},
	}
	GetAppState(context.Background(), agents, &dbApp, nil)

	// The existing daemon information should be updated.
	require.Len(t, dbApp.Daemons, 1)
	require.EqualValues(t, 123, dbApp.Daemons[0].ID)
	require.True(t, dbApp.Daemons[0].Active)
	require.Equal(t, "4.7.0", dbApp.Daemons[0].Version)
	require.Equal(t, "4.7.0", dbApp.Meta.Version)
	require.Equal(t, "4.7.0", dbApp.Meta.ExtendedVersion)
	require.NotNil(t, dbApp.Daemons[0].PDNSDaemon)
	require.EqualValues(t, 234, dbApp.Daemons[0].PDNSDaemon.ID)
	require.EqualValues(t, 123, dbApp.Daemons[0].PDNSDaemon.DaemonID)
	require.EqualValues(t, 1234, dbApp.Daemons[0].Uptime)
	require.Equal(t, "http://127.0.0.1:8081", dbApp.Daemons[0].PDNSDaemon.Details.URL)
	require.Equal(t, "http://127.0.0.1:8081/config", dbApp.Daemons[0].PDNSDaemon.Details.ConfigURL)
	require.Equal(t, "http://127.0.0.1:8081/zones", dbApp.Daemons[0].PDNSDaemon.Details.ZonesURL)
	require.Equal(t, "http://127.0.0.1:8081/autoprimaries", dbApp.Daemons[0].PDNSDaemon.Details.AutoprimariesURL)
}

// Test the case when an attempt to get state fails.
func TestGetAppStateError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	agents := NewMockConnectedAgents(ctrl)

	agents.EXPECT().GetPowerDNSServerInfo(gomock.Any(), gomock.Any()).Return(nil, &testError{})

	dbApp := dbmodel.App{
		AccessPoints: []*dbmodel.AccessPoint{
			{
				Type:    dbmodel.AccessPointControl,
				Address: "127.0.0.1",
				Port:    53,
			},
		},
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 1111,
		},
		Daemons: []*dbmodel.Daemon{
			{
				Version: "4.5.2",
			},
		},
	}
	GetAppState(context.Background(), agents, &dbApp, nil)

	// The existing daemon info should not be updated.
	require.Len(t, dbApp.Daemons, 1)
	require.Equal(t, "4.5.2", dbApp.Daemons[0].Version)
}

// Test inserting PowerDNS app into the database.
func TestCommitAppIntoDB(t *testing.T) {
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

	var accessPoints []*dbmodel.AccessPoint
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "", "", 1234, false)
	app := &dbmodel.App{
		ID:           0,
		MachineID:    machine.ID,
		Machine:      machine,
		Type:         dbmodel.AppTypePDNS,
		Active:       true,
		AccessPoints: accessPoints,
	}

	// Make sure an event is emitted with appropriate parameters when the
	// app is first added to the database.
	eventCenterMock := NewMockEventCenter(ctrl)
	eventCenterMock.EXPECT().AddInfoEvent("added {app}", app.Machine, app).Times(1)

	// Insert the app into the database.
	err = CommitAppIntoDB(db, app, eventCenterMock)
	require.NoError(t, err)

	// Update the app in the database. The event should not be emitted.
	accessPoints = []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "", "", 2345, false)
	app.AccessPoints = accessPoints
	err = CommitAppIntoDB(db, app, eventCenterMock)
	require.NoError(t, err)

	// Check that the app is stored in the database.
	returned, err := dbmodel.GetAppByID(db, app.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)
	require.Len(t, returned.AccessPoints, 1)
	require.EqualValues(t, 2345, returned.AccessPoints[0].Port)
	require.NotZero(t, returned.ID)
}
