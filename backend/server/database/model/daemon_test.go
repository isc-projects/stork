package dbmodel

import (
	"testing"
	"time"

	require "github.com/stretchr/testify/require"
	keaconfig "isc.org/stork/appcfg/kea"
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

// Test getting daemon by ID.
func TestGetDaemonByID(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// get non-existing daemon
	dmn, err := GetDaemonByID(db, 123)
	require.NoError(t, err)
	require.Nil(t, dmn)

	// create machine and then app with daemon
	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = AddMachine(db, m)
	require.NoError(t, err)
	require.NotZero(t, m.ID)

	// add app but without machine, error should be raised
	daemonEntry := NewKeaDaemon("kea-dhcp4", true)

	// Set initial configuration with one logger.
	err = daemonEntry.SetConfigFromJSON(`{
        "Dhcp4": {
            "loggers": [
                {
                    "name": "kea-dhcp4",
                    "severity": "debug",
                    "output_options": [
                        {
                            "output": "/tmp/kea-dhcp4.log"
                        }
                    ]
                }
            ]
        }
    }`)
	require.NoError(t, err)

	app := &App{
		ID:        0,
		MachineID: m.ID,
		Type:      AppTypeKea,
		Daemons: []*Daemon{
			daemonEntry,
		},
	}
	_, err = AddApp(db, app)
	require.NoError(t, err)
	require.NotNil(t, app)
	require.Len(t, app.Daemons, 1)
	daemon := app.Daemons[0]
	require.NotZero(t, daemon.ID)

	// get daemon, now it should be there
	dmn, err = GetDaemonByID(db, daemon.ID)
	require.NoError(t, err)
	require.NotNil(t, dmn)
	require.EqualValues(t, daemon.ID, dmn.ID)
	require.EqualValues(t, daemon.Active, dmn.Active)
	require.NotNil(t, dmn.KeaDaemon)
	require.NotNil(t, dmn.KeaDaemon.Config)
}

// Tests that Kea loggin configuration information is correctly populated within
// the daemon structures.
func TestSetKeaLoggingConfig(t *testing.T) {
	daemon := NewKeaDaemon("kea-dhcp4", true)

	// Set initial configuration with one logger.
	err := daemon.SetConfigFromJSON(`{
        "Dhcp4": {
            "loggers": [
                {
                    "name": "kea-dhcp4",
                    "severity": "debug",
                    "output_options": [
                        {
                            "output": "/tmp/kea-dhcp4.log"
                        }
                    ]
                }
            ]
        }
    }`)
	require.NoError(t, err)

	require.Len(t, daemon.LogTargets, 1)
	require.Equal(t, "kea-dhcp4", daemon.LogTargets[0].Name)
	require.Equal(t, "debug", daemon.LogTargets[0].Severity)
	require.Equal(t, "/tmp/kea-dhcp4.log", daemon.LogTargets[0].Output)

	// Simulate adding this to the database and set some identifiers.
	daemon.ID = 1
	daemon.LogTargets[0].ID = 2
	daemon.LogTargets[0].DaemonID = 1

	// Set new configuration with one new logger and one logger with modified
	// data.
	err = daemon.SetConfigFromJSON(`{
        "Dhcp4": {
            "loggers": [
                {
                    "name": "kea-dhcp4",
                    "severity": "error",
                    "output_options": [
                        {
                            "output": "/tmp/kea-dhcp4.log"
                        }
                    ]
                },
                {
                    "name": "kea-dhcp4.packets",
                    "severity": "debug",
                    "output_options": [
                        {
                            "output": "/tmp/kea-dhcp4-packets.log"
                        }
                    ]
                }
            ]
        }
    }`)
	require.NoError(t, err)

	require.Len(t, daemon.LogTargets, 2)
	require.Equal(t, "kea-dhcp4", daemon.LogTargets[0].Name)
	require.Equal(t, "error", daemon.LogTargets[0].Severity)
	require.Equal(t, "/tmp/kea-dhcp4.log", daemon.LogTargets[0].Output)
	// The first logger should inherit old ids.
	require.EqualValues(t, 2, daemon.LogTargets[0].ID)
	require.EqualValues(t, 1, daemon.LogTargets[0].DaemonID)

	require.Equal(t, "kea-dhcp4.packets", daemon.LogTargets[1].Name)
	require.Equal(t, "debug", daemon.LogTargets[1].Severity)
	require.Equal(t, "/tmp/kea-dhcp4-packets.log", daemon.LogTargets[1].Output)
	// The new logger has no id set until added to the db.
	require.Zero(t, daemon.LogTargets[1].ID)
	require.Zero(t, daemon.LogTargets[1].DaemonID)

	// Set the second logger's ids.
	daemon.LogTargets[1].ID = 3
	daemon.LogTargets[1].DaemonID = 1

	// CHeck that the same data can be refreshed.
	err = daemon.SetConfigFromJSON(`{
        "Dhcp4": {
            "loggers": [
                {
                    "name": "kea-dhcp4",
                    "severity": "error",
                    "output_options": [
                        {
                            "output": "/tmp/kea-dhcp4.log"
                        }
                    ]
                },
                {
                    "name": "kea-dhcp4.packets",
                    "severity": "debug",
                    "output_options": [
                        {
                            "output": "/tmp/kea-dhcp4-packets.log"
                        }
                    ]
                }
            ]
        }
    }`)
	require.NoError(t, err)

	require.Len(t, daemon.LogTargets, 2)

	// Check that the number of loggers can be reduced.
	err = daemon.SetConfigFromJSON(`{
        "Dhcp4": {
            "loggers": [
                {
                    "name": "kea-dhcp4.packets",
                    "severity": "debug",
                    "output_options": [
                        {
                            "output": "/tmp/kea-dhcp4-packets.log"
                        }
                    ]
                }
            ]
        }
    }`)
	require.NoError(t, err)

	require.Len(t, daemon.LogTargets, 1)
}

// This test verifies that the config hash is setting configuration as string.
func TestSetConfigFromJSONWithHash(t *testing.T) {
	daemon := NewKeaDaemon("kea-dhcp4", true)

	// Set initial configuration with one logger.
	err := daemon.SetConfigFromJSON(`{
        "Dhcp4": {
            "loggers": [
                {
                    "name": "kea-dhcp4",
                    "severity": "debug",
                    "output_options": [
                        {
                            "output": "/tmp/kea-dhcp4.log"
                        }
                    ]
                }
            ]
        }
    }`)
	require.NoError(t, err)
	require.NotNil(t, daemon.KeaDaemon)
	require.NotNil(t, daemon.KeaDaemon.Config)
	require.Equal(t, "f1c994d55b6f4edba9568d89ce2a804a", daemon.KeaDaemon.ConfigHash)
}

// Test that SetConfig does not set hash for the config.
func TestSetConfig(t *testing.T) {
	daemon := NewKeaDaemon("kea-dhcp4", true)

	config := `{
        "Dhcp4": {}
    }`
	parsedConfig, err := keaconfig.NewFromJSON(config)
	require.NoError(t, err)

	err = daemon.SetConfig(parsedConfig)
	require.NoError(t, err)

	require.NotNil(t, daemon.KeaDaemon)
	require.NotNil(t, daemon.KeaDaemon.Config)
	require.Empty(t, daemon.KeaDaemon.ConfigHash)
}

// Test that shallow copy of a Kea daemon can be created.
func TestShallowCopyKeaDaemon(t *testing.T) {
	// Create Daemon instance with not nil KeaDaemon.
	daemon := NewKeaDaemon("kea-dhcp4", true)
	copy := ShallowCopyKeaDaemon(daemon)
	require.NotNil(t, copy)
	require.NotNil(t, copy.KeaDaemon)
	require.NotSame(t, daemon, copy)
	require.NotSame(t, daemon.KeaDaemon, copy.KeaDaemon)

	// Repeat the same test but this time the KeaDaemon is nil.
	daemon = &Daemon{}
	require.NotPanics(t, func() {
		copy = ShallowCopyKeaDaemon(daemon)
	})
	require.NotNil(t, copy)
	require.NotSame(t, daemon, copy)
}
