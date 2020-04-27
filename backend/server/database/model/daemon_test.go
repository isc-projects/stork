package dbmodel

import (
	"testing"

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
	err = AddApp(db, app)
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

	daemon.KeaDaemon.KeaDHCPDaemon.Stats.LPS15min = 1000
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
	require.EqualValues(t, 1000, daemon.KeaDaemon.KeaDHCPDaemon.Stats.LPS15min)
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
	err = AddApp(db, app)
	require.NoError(t, err)
	require.NotNil(t, app)
	require.Len(t, app.Daemons, 1)
	daemon := app.Daemons[0]
	require.NotZero(t, daemon.ID)

	daemon.Pid = 123
	daemon.Active = false
	daemon.Version = "9.20"

	daemon.Bind9Daemon.Stats.ZoneCount = 123
	daemon.Bind9Daemon.Stats.CacheHits = 5

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
	require.EqualValues(t, 5, daemon.Bind9Daemon.Stats.CacheHits)
}

// Returns all HA state names to which the daemon belongs.
func TestGetHAStateNames(t *testing.T) {
	daemon := NewKeaDaemon("dhcp4", true)
	daemon.ID = 1
	daemon.Services = []*Service{
		{
			BaseService: BaseService{
				ID: 1,
			},
			HAService: &BaseHAService{
				HAType:             "dhcp4",
				PrimaryID:          1,
				SecondaryID:        2,
				BackupID:           []int64{3, 4},
				PrimaryLastState:   "load-balancing",
				SecondaryLastState: "syncing",
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
				HAType:             "dhcp4",
				PrimaryID:          1,
				SecondaryID:        5,
				PrimaryLastState:   "hot-standby",
				SecondaryLastState: "hot-standby",
			},
		},
	}
	states := daemon.GetHAStateNames()
	require.Len(t, states, 2)

	require.Contains(t, states, "load-balancing")
	require.Contains(t, states, "hot-standby")
}
