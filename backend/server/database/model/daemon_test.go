package dbmodel

import (
	"testing"
	"time"

	require "github.com/stretchr/testify/require"

	dbtest "isc.org/stork/server/database/test"
)

// Test that new instance of the generic Kea daemon can be created.
func TestNewKeaDaemon(t *testing.T) {
	// Create the daemon with active flag set to true.
	daemon := NewKeaDaemon(DaemonNameDHCPv4, true)
	require.NotNil(t, daemon)
	require.NotNil(t, daemon.KeaDaemon)
	require.NotNil(t, daemon.KeaDaemon.KeaDHCPDaemon)
	require.Nil(t, daemon.Bind9Daemon)
	require.Equal(t, DaemonNameDHCPv4, daemon.Name)
	require.True(t, daemon.Active)

	// Create the non DHCP daemon.
	daemon = NewKeaDaemon("ca", false)
	require.NotNil(t, daemon)
	require.NotNil(t, daemon.KeaDaemon)
	require.Nil(t, daemon.KeaDaemon.KeaDHCPDaemon)
	require.Nil(t, daemon.Bind9Daemon)
	require.Equal(t, "ca", daemon.Name)
	require.False(t, daemon.Active)
}

// Test that new instance of the Bind9 daemon can be created.
func TestNewBind9Daemon(t *testing.T) {
	daemon := NewBind9Daemon(true)
	require.NotNil(t, daemon)
	require.NotNil(t, daemon.Bind9Daemon)
	require.Nil(t, daemon.KeaDaemon)
	require.Equal(t, DaemonNameBind9, daemon.Name)
	require.True(t, daemon.Active)
}

// Test that Kea DHCP daemon is properly updated.
func TestUpdateKeaDHCPDaemon(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	require.NoError(t, err)
	require.NotZero(t, m.ID)

	// add app but without machine, error should be raised
	app := &App{
		ID:        0,
		MachineID: m.ID,
		Type:      AppTypeKea,
		Daemons: []*Daemon{
			NewKeaDaemon(DaemonNameDHCPv4, true),
		},
	}
	_, err = AddApp(db, app)
	require.NoError(t, err)
	require.NotNil(t, app)
	require.Len(t, app.Daemons, 1)
	daemon := app.Daemons[0]
	require.NotZero(t, daemon.ID)

	daemon.Pid = 123
	daemon.Name = DaemonNameDHCPv6
	daemon.Active = false
	daemon.Version = "2.0.0"
	daemon.KeaDaemon.Config, err = NewKeaConfigFromJSON(`{
        "Dhcp4": {
            "valid-lifetime": 1234
        }
    }`)
	require.NoError(t, err)

	daemon.KeaDaemon.KeaDHCPDaemon.Stats.RPS1 = 1000
	daemon.KeaDaemon.KeaDHCPDaemon.Stats.RPS2 = 2000
	daemon.KeaDaemon.KeaDHCPDaemon.Stats.AddrUtilization = 90

	err = UpdateDaemon(db, daemon)
	require.NoError(t, err)

	app, err = GetAppByID(db, app.ID)
	require.NoError(t, err)
	require.NotNil(t, app)
	require.Len(t, app.Daemons, 1)
	daemon = app.Daemons[0]

	require.EqualValues(t, 123, daemon.Pid)
	require.Equal(t, DaemonNameDHCPv6, daemon.Name)
	require.False(t, daemon.Active)
	require.Equal(t, "2.0.0", daemon.Version)
	require.NotNil(t, daemon.KeaDaemon)
	require.NotNil(t, daemon.KeaDaemon.Config)
	require.NotNil(t, daemon.KeaDaemon.KeaDHCPDaemon)
	require.EqualValues(t, 1000, daemon.KeaDaemon.KeaDHCPDaemon.Stats.RPS1)
	require.EqualValues(t, 2000, daemon.KeaDaemon.KeaDHCPDaemon.Stats.RPS2)
	require.EqualValues(t, 90, daemon.KeaDaemon.KeaDHCPDaemon.Stats.AddrUtilization)
}

// Test that Bind9 daemon is properly updated.
func TestUpdateBind9Daemon(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	require.NoError(t, err)
	require.NotZero(t, m.ID)

	// add app but without machine, error should be raised
	app := &App{
		ID:        0,
		MachineID: m.ID,
		Type:      AppTypeBind9,
		Daemons: []*Daemon{
			NewBind9Daemon(true),
		},
	}
	_, err = AddApp(db, app)
	require.NoError(t, err)
	require.NotNil(t, app)
	require.Len(t, app.Daemons, 1)
	daemon := app.Daemons[0]
	require.NotZero(t, daemon.ID)

	daemon.Pid = 123
	daemon.Active = false
	daemon.Version = "9.20"

	daemon.Bind9Daemon.Stats.ZoneCount = 123

	err = UpdateDaemon(db, daemon)
	require.NoError(t, err)

	app, err = GetAppByID(db, app.ID)
	require.NoError(t, err)
	require.NotNil(t, app)
	require.Len(t, app.Daemons, 1)
	daemon = app.Daemons[0]

	require.EqualValues(t, 123, daemon.Pid)
	require.Equal(t, "named", daemon.Name)
	require.False(t, daemon.Active)
	require.Equal(t, "9.20", daemon.Version)
	require.NotNil(t, daemon.Bind9Daemon)
	require.EqualValues(t, 123, daemon.Bind9Daemon.Stats.ZoneCount)
}

// Returns all HA state names to which the daemon belongs and the
// failure times.
func TestGetHAOverview(t *testing.T) {
	failoverAt := time.Date(2020, 6, 4, 11, 32, 0, 0, time.UTC)
	daemon := NewKeaDaemon("dhcp4", true)
	daemon.ID = 1
	daemon.Services = []*Service{
		{
			BaseService: BaseService{
				ID: 1,
			},
			HAService: &BaseHAService{
				HAType:                "dhcp4",
				PrimaryID:             1,
				SecondaryID:           2,
				BackupID:              []int64{3, 4},
				PrimaryLastState:      "load-balancing",
				SecondaryLastState:    "syncing",
				PrimaryLastFailoverAt: failoverAt,
			},
		},
		{
			BaseService: BaseService{
				ID: 2,
			},
		},
		{
			BaseService: BaseService{
				ID: 3,
			},
			HAService: &BaseHAService{
				HAType:                  "dhcp4",
				PrimaryID:               1,
				SecondaryID:             5,
				PrimaryLastState:        "hot-standby",
				SecondaryLastState:      "hot-standby",
				SecondaryLastFailoverAt: failoverAt,
			},
		},
	}
	overviews := daemon.GetHAOverview()
	require.Len(t, overviews, 2)

	require.Equal(t, "load-balancing", overviews[0].State)
	require.Zero(t, overviews[0].LastFailureAt)

	require.Equal(t, "hot-standby", overviews[1].State)
	require.Equal(t, failoverAt, overviews[1].LastFailureAt)
}
