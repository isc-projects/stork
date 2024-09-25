package dbmodel

import (
	"sync"
	"testing"
	"time"

	"github.com/go-pg/pg/v10"
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

	// Remember the creation time so it can be compared after the update.
	createdAt := daemon.CreatedAt
	require.NotZero(t, createdAt)

	// Reset the creation time to ensure it is not modified during the update.
	daemon.CreatedAt = time.Time{}
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

	err = UpdateDaemon(db, daemon)
	require.NoError(t, err)

	app, err = GetAppByID(db, app.ID)
	require.NoError(t, err)
	require.NotNil(t, app)
	require.Len(t, app.Daemons, 1)
	daemon = app.Daemons[0]

	require.Equal(t, createdAt, daemon.CreatedAt)
	require.EqualValues(t, 123, daemon.Pid)
	require.Equal(t, DaemonNameDHCPv6, daemon.Name)
	require.False(t, daemon.Active)
	require.Equal(t, "2.0.0", daemon.Version)
	require.NotNil(t, daemon.KeaDaemon)
	require.NotNil(t, daemon.KeaDaemon.Config)
	require.NotNil(t, daemon.KeaDaemon.KeaDHCPDaemon)
	require.EqualValues(t, 1000, daemon.KeaDaemon.KeaDHCPDaemon.Stats.RPS1)
	require.EqualValues(t, 2000, daemon.KeaDaemon.KeaDHCPDaemon.Stats.RPS2)
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

	accessPoints := []*AccessPoint{}
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "", "", 1234, false)
	app := &App{
		ID:        0,
		MachineID: m.ID,
		Type:      AppTypeKea,
		Daemons: []*Daemon{
			daemonEntry,
		},
		AccessPoints: accessPoints,
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
	require.NotNil(t, dmn.App)
	require.Len(t, dmn.App.AccessPoints, 1)
}

// Test getting multiple daemons by IDs.
func TestGetDaemonsByIDs(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Get non-existing daemons
	returnedDaemons, err := GetDaemonsByIDs(db, []int64{123, 234})
	require.NoError(t, err)
	require.Empty(t, returnedDaemons)

	// Create machine and then app with daemon.
	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = AddMachine(db, m)
	require.NoError(t, err)
	require.NotZero(t, m.ID)

	// add app but without machine, error should be raised
	var daemons []*Daemon
	for _, daemonName := range []string{DaemonNameDHCPv4, DaemonNameDHCPv6, DaemonNameD2, DaemonNameCA} {
		daemons = append(daemons, NewKeaDaemon(daemonName, true))
	}

	accessPoints := []*AccessPoint{}
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "", "", 1234, false)
	app := &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeKea,
		Daemons:      daemons,
		AccessPoints: accessPoints,
	}
	_, err = AddApp(db, app)
	require.NoError(t, err)
	require.NotNil(t, app)
	require.Len(t, app.Daemons, 4)

	// Get selected daemons.
	selectedDaemons := []int64{daemons[0].ID, daemons[1].ID}
	returnedDaemons, err = GetDaemonsByIDs(db, selectedDaemons)
	require.NoError(t, err)
	require.Len(t, returnedDaemons, 2)

	var ids []int64
	for _, rd := range returnedDaemons {
		ids = append(ids, rd.ID)
		require.NotNil(t, rd.App)
		require.EqualValues(t, app.ID, rd.App.ID)
		require.NotNil(t, rd.App.AccessPoints)
		require.Len(t, rd.App.AccessPoints, 1)
		require.Equal(t, accessPoints[0].Port, rd.App.AccessPoints[0].Port)
		require.NotNil(t, rd.App.Machine)
		require.EqualValues(t, m.ID, rd.App.Machine.ID)
		require.NotNil(t, rd.KeaDaemon)
		require.NotNil(t, rd.KeaDaemon.KeaDHCPDaemon)
	}
	require.ElementsMatch(t, ids, selectedDaemons)
}

// Test getting all Kea DHCP daemons.
func TestGetKeaDHCPDaemons(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Getting Kea daemons when none are in the database should cause
	// no error and return an empty list.
	daemons, err := GetKeaDHCPDaemons(db)
	require.NoError(t, err)
	require.Empty(t, daemons)

	// Add a machine.
	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = AddMachine(db, m)
	require.NoError(t, err)
	require.NotZero(t, m.ID)

	// Add an app with several Kea daemons of different type.
	daemonNames := []string{DaemonNameDHCPv4, DaemonNameDHCPv6, DaemonNameCA, DaemonNameD2}
	accessPoints := []*AccessPoint{}
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "", "", 1234, false)
	app := &App{
		MachineID:    m.ID,
		Type:         AppTypeKea,
		AccessPoints: accessPoints,
	}
	for _, dn := range daemonNames {
		app.Daemons = append(app.Daemons, NewKeaDaemon(dn, true))
	}
	_, err = AddApp(db, app)
	require.NoError(t, err)

	// Add named app and daemon.
	accessPoints[0].Port++
	app = &App{
		MachineID:    m.ID,
		Type:         AppTypeBind9,
		AccessPoints: accessPoints,
		Daemons: []*Daemon{
			NewBind9Daemon(true),
		},
	}
	_, err = AddApp(db, app)
	require.NoError(t, err)

	// Try to get Kea DHCP daemons only. There should be two.
	daemons, err = GetKeaDHCPDaemons(db)
	require.NoError(t, err)
	require.Len(t, daemons, 2)

	// Validate returned daemons.
	names := []string{}
	for _, d := range daemons {
		names = append(names, d.Name)
		require.NotNil(t, d.App)
		require.NotNil(t, d.KeaDaemon)
		require.Equal(t, d.ID, d.KeaDaemon.DaemonID)
		require.NotNil(t, d.KeaDaemon.KeaDHCPDaemon)
	}
	require.Contains(t, names, DaemonNameDHCPv4)
	require.Contains(t, names, DaemonNameDHCPv6)
}

// Test selecting BIND9 daemon by ID for update which should result in locking
// the daemon information until the transaction is committed or rolled back.
func TestGetBind9DaemonsForUpdate(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Get non-existing daemons.
	tx, err := db.Begin()
	require.NoError(t, err)

	daemons, err := GetDaemonsForUpdate(tx, []*Daemon{
		{
			ID: 123,
		},
		{
			ID: 567,
		},
	})
	require.NoError(t, err)
	require.Empty(t, daemons)
	err = tx.Commit()
	require.NoError(t, err)

	// Create a machine.
	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = AddMachine(db, m)
	require.NoError(t, err)
	require.NotZero(t, m.ID)

	// Create an app.
	app := &App{
		ID:        0,
		MachineID: m.ID,
		Type:      AppTypeBind9,
		Daemons: []*Daemon{
			{
				Name:   "named",
				Active: true,
			},
		},
	}
	_, err = AddApp(db, app)
	require.NoError(t, err)
	require.NotNil(t, app)
	require.Len(t, app.Daemons, 1)
	require.NotZero(t, app.Daemons[0].ID)

	// Start new transaction.
	tx, err = db.Begin()
	require.NoError(t, err)

	// An attempt to select no particular daemon for update should result
	// in an error.
	daemons, err = GetDaemonsForUpdate(tx, []*Daemon{})
	require.Error(t, err)
	require.Empty(t, daemons)

	// Select daemon for update.
	daemons, err = GetDaemonsForUpdate(tx, app.Daemons)
	require.NoError(t, err)
	require.Len(t, daemons, 1)

	// Sanity check selected data.
	require.NotZero(t, daemons[0].ID)
	require.True(t, daemons[0].Active)
	require.NotNil(t, daemons[0].App)
	require.Equal(t, app.ID, daemons[0].App.ID)
	require.NotNil(t, daemons[0].App.Machine)
	require.Equal(t, m.ID, daemons[0].App.Machine.ID)

	// When daemon is selected for update within a transaction, no other
	// transaction can modify the daemon until the current transaction is
	// committed or rolled back. We will now run a goroutine which will
	// attempt such a modification.
	var result pg.Result
	mutex := &sync.Mutex{}
	mutex.Lock()
	cond := sync.NewCond(mutex)
	wg := &sync.WaitGroup{}
	wg.Add(1)

	// Actually run the goroutine.
	go func() {
		defer wg.Done()
		// The main thread is waiting for this conditional to ensure that the
		// goroutine is started before the test continues.
		cond.Signal()
		// Attempt to delete the app while the main transaction is in progress
		// and the daemons are locked for update. This should block until the
		// main transaction is committed or rolled back.
		result, _ = db.Model(app).WherePK().Delete()
	}()

	// Wait for the goroutine to begin.
	cond.Wait()

	// We want to ensure that the goroutine executes db.Delete() before we
	// run the tx.Delete() from this transaction. If the tx.Delete() is
	// executed first, it has no effect on the test result. But, running
	// db.Delete() before tx.Delete() validates the effectiveness of the
	// locking mechanism.
	time.Sleep(100 * time.Millisecond)
	// It should take precedence over the db.Delete() invoked from the goroutine.
	// Thus, there should be no error.
	_, err = tx.Model(app).WherePK().Delete()
	require.NoError(t, err)

	// Commit the transaction which should cause the goroutine to complete.
	err = tx.Commit()
	require.NoError(t, err)

	// Wait for the goroutine to complete.
	wg.Wait()

	// Ensure that the goroutine deleted no data because the data was already
	// deleted by the main thread. It tests that the locking mechanism works
	// as expected.
	require.Zero(t, result.RowsAffected())
}

// Test selecting Kea daemons by IDs for update which should result in locking
// the daemon information until the transaction is committed or rolled back.
func TestGetKeaDaemonsForUpdate(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Get non-existing daemons.
	tx, err := db.Begin()
	require.NoError(t, err)

	daemons, err := GetKeaDaemonsForUpdate(tx, []*Daemon{
		{
			ID: 123,
		},
		{
			ID: 567,
		},
	})
	require.NoError(t, err)
	require.Empty(t, daemons)
	err = tx.Commit()
	require.NoError(t, err)

	// Create a machine.
	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = AddMachine(db, m)
	require.NoError(t, err)
	require.NotZero(t, m.ID)

	// Create the configs for daemons.
	config1, err := NewKeaConfigFromJSON(`{"Dhcp4": { }}`)
	require.NoError(t, err)
	config2, err := NewKeaConfigFromJSON(`{"Dhcp6": { }}`)
	require.NoError(t, err)

	// Create an app.
	app := &App{
		ID:        0,
		MachineID: m.ID,
		Type:      AppTypeKea,
		Daemons: []*Daemon{
			{
				Name:   "dhcp4",
				Active: true,
				KeaDaemon: &KeaDaemon{
					Config:     config1,
					ConfigHash: "1234",
				},
			},
			{
				Name:   "dhcp6",
				Active: true,
				KeaDaemon: &KeaDaemon{
					Config:     config2,
					ConfigHash: "2345",
				},
			},
		},
	}
	_, err = AddApp(db, app)
	require.NoError(t, err)
	require.NotNil(t, app)
	require.Len(t, app.Daemons, 2)
	require.NotZero(t, app.Daemons[0].ID)
	require.NotZero(t, app.Daemons[1].ID)

	// Start new transaction.
	tx, err = db.Begin()
	require.NoError(t, err)

	// An attempt to select no particular daemon for update should result
	// in an error.
	daemons, err = GetKeaDaemonsForUpdate(tx, []*Daemon{})
	require.Error(t, err)
	require.Empty(t, daemons)

	// Select both daemons for update.
	daemons, err = GetKeaDaemonsForUpdate(tx, app.Daemons)
	require.NoError(t, err)
	require.Len(t, daemons, 2)

	// Sanity check selected data.
	for i, daemon := range daemons {
		require.NotZero(t, daemon.ID)
		require.True(t, daemon.Active)
		require.NotNil(t, daemon.KeaDaemon)
		require.NotZero(t, daemon.KeaDaemon.ID)
		require.NotNil(t, daemon.KeaDaemon.Config)
		require.Equal(t, app.Daemons[i].KeaDaemon.ConfigHash, daemon.KeaDaemon.ConfigHash)
		require.NotNil(t, daemon.App)
		require.Equal(t, app.ID, daemon.App.ID)
		require.NotNil(t, daemon.App.Machine)
		require.Equal(t, m.ID, daemon.App.Machine.ID)
	}

	// When daemons are selected for update within a transaction, no other
	// transaction can modify the daemons until the current transaction is
	// committed or rolled back. We will now run a goroutine which will
	// attempt such a modification.
	var result pg.Result
	mutex := &sync.Mutex{}
	mutex.Lock()
	cond := sync.NewCond(mutex)
	wg := &sync.WaitGroup{}
	wg.Add(1)

	// Actually run the goroutine.
	go func() {
		defer wg.Done()
		// The main thread is waiting for this conditional to ensure that the
		// goroutine is started before the test continues.
		cond.Signal()
		// Attempt to delete the app while the main transaction is in progress
		// and the daemons are locked for update. This should block until the
		// main transaction is committed or rolled back.
		result, _ = db.Model(app).WherePK().Delete()
	}()

	// Wait for the goroutine to begin.
	cond.Wait()

	// We want to ensure that the goroutine executes db.Delete() before we
	// run the tx.Delete() from this transaction. If the tx.Delete() is
	// executed first, it has no effect on the test result. But, running
	// db.Delete() before tx.Delete() validates the effectiveness of the
	// locking mechanism.
	time.Sleep(100 * time.Millisecond)
	// It should take precedence over the db.Delete() invoked from the goroutine.
	// Thus, there should be no error.
	_, err = tx.Model(app).WherePK().Delete()
	require.NoError(t, err)

	// Commit the transaction which should cause the goroutine to complete.
	err = tx.Commit()
	require.NoError(t, err)

	// Wait for the goroutine to complete.
	wg.Wait()

	// Ensure that the goroutine deleted no data because the data was already
	// deleted by the main thread. It tests that the locking mechanism works
	// as expected.
	require.Zero(t, result.RowsAffected())
}

// Tests that Kea logging configuration information is correctly populated within
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
	require.Len(t, daemon.KeaDaemon.ConfigHash, 32)
}

// Test that SetConfig does not set hash for the config.
func TestSetConfig(t *testing.T) {
	daemon := NewKeaDaemon("kea-dhcp4", true)

	config := `{
        "Dhcp4": {}
    }`
	parsedConfig, err := NewKeaConfigFromJSON(config)
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

// Test that local subnet id of the Kea subnet can be extracted.
func TestGetLocalSubnetID(t *testing.T) {
	config, err := NewKeaConfigFromJSON(`{
		"Dhcp4": {
            "subnet4": [
				{
					"id":     1,
					"subnet": "192.0.2.0/24"
				}
            ]
        }
    }`)
	require.NotNil(t, config)
	require.NoError(t, err)

	// Create an app with the given configuration.
	accessPoints := []*AccessPoint{}
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "", "", 1234, false)
	app := &App{
		ID:           0,
		MachineID:    0,
		Type:         AppTypeKea,
		Active:       true,
		AccessPoints: accessPoints,
		Daemons: []*Daemon{
			{
				KeaDaemon: &KeaDaemon{
					Config: config,
				},
			},
		},
	}

	// Try to find a non-existing subnet.
	require.Zero(t, app.Daemons[0].GetLocalSubnetID("192.0.3.0/24"))
	// Next, try to find the existing subnet.
	require.EqualValues(t, 1, app.Daemons[0].GetLocalSubnetID("192.0.2.0/24"))
}

// Test DaemonTag interface implementation.
func TestDaemonTag(t *testing.T) {
	daemon := Daemon{
		ID:    1,
		Name:  "dhcp4",
		AppID: 2,
		App: &App{
			Type: AppTypeKea,
		},
	}
	require.EqualValues(t, 1, daemon.GetID())
	require.Equal(t, "dhcp4", daemon.GetName())
	require.EqualValues(t, 2, daemon.GetAppID())
	require.Equal(t, AppTypeKea, daemon.GetAppType())
}

// Test that GetAppType() returns "kea" when KeaDaemon present.
func TestDaemonTagKeaAppType(t *testing.T) {
	daemon := Daemon{
		KeaDaemon: &KeaDaemon{},
	}
	require.Equal(t, AppTypeKea, daemon.GetAppType())
}

// Test that GetAppType() returns "bind9" when Bind9Daemon present.
func TestDaemonTagBind9AppType(t *testing.T) {
	daemon := Daemon{
		Bind9Daemon: &Bind9Daemon{},
	}
	require.Equal(t, AppTypeBind9, daemon.GetAppType())
}

// Test that the machine ID is returned if the app reference is set.
func TestDaemonTagMachineID(t *testing.T) {
	// Arrange
	daemon := Daemon{
		App: &App{
			MachineID: 42,
		},
	}

	// Assert & Assert
	require.EqualValues(t, 42, *daemon.GetMachineID())
}

// Test that the machine ID is nil if the app reference is not set.
func TestDaemonTagMissingMachineID(t *testing.T) {
	// Arrange
	daemon := Daemon{}

	// Act & Assert
	require.Nil(t, daemon.GetMachineID())
}

// Tests that Kea daemon config hashes can be wiped.
func TestDeleteKeaDaemonConfigHashes(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, machine)
	require.NoError(t, err)
	require.NotZero(t, machine.ID)

	app := &App{
		ID:        0,
		MachineID: machine.ID,
		Type:      AppTypeKea,
		Daemons: []*Daemon{
			{
				Name: DaemonNameDHCPv4,
				KeaDaemon: &KeaDaemon{
					ConfigHash: "1234",
				},
			},
			{
				Name: DaemonNameDHCPv6,
				KeaDaemon: &KeaDaemon{
					ConfigHash: "2345",
				},
			},
		},
	}
	_, err = AddApp(db, app)
	require.NoError(t, err)
	require.NotNil(t, app)
	require.Len(t, app.Daemons, 2)
	require.NotZero(t, app.Daemons[0].ID)
	require.NotZero(t, app.Daemons[1].ID)

	err = DeleteKeaDaemonConfigHashes(db)
	require.NoError(t, err)

	daemons, err := GetKeaDHCPDaemons(db)
	require.NoError(t, err)
	require.Len(t, daemons, 2)

	require.Equal(t, "", daemons[0].KeaDaemon.ConfigHash)
	require.Equal(t, "", daemons[1].KeaDaemon.ConfigHash)
}

// Test RPS statistics inserted as integers can be later
// fetched as floats. It tests the data type change in the
// RPS stats from int to float32.
func TestGetRpsStatsAsFloats(t *testing.T) {
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

	// This is the statistics structure we used to have for the
	// RPS. The RPS were stored as int.
	type testKeaDHCPDaemonStats struct {
		RPS1 int `pg:"rps1"`
		RPS2 int `pg:"rps2"`
	}

	// This structure ties the old RPS formats to the Kea DHCP daemon.
	type testKeaDHCPDaemon struct {
		tableName   struct{} `pg:"kea_dhcp_daemon"` //nolint:unused
		ID          int64    `pg:",pk"`
		KeaDaemonID int64
		Stats       testKeaDHCPDaemonStats
	}

	// Update the Kea daemon with RPS values stored as int.
	keaDaemon := testKeaDHCPDaemon{
		ID:          app.Daemons[0].KeaDaemon.KeaDHCPDaemon.ID,
		KeaDaemonID: app.Daemons[0].KeaDaemon.KeaDHCPDaemon.KeaDaemonID,
		Stats: testKeaDHCPDaemonStats{
			RPS1: 1000,
			RPS2: 2000,
		},
	}
	_, err = db.Model(&keaDaemon).WherePK().Update()
	require.NoError(t, err)

	// Get the daemon. It uses float32 data types for RPS.
	// Let's make sure it is fetched without errors and the
	// values are correctly cast to float32.
	daemons, err := GetDaemonsByIDs(db, []int64{daemon.ID})
	require.NoError(t, err)
	require.Len(t, daemons, 1)
	daemon = &daemons[0]
	require.NotNil(t, daemon.KeaDaemon)
	require.NotNil(t, daemon.KeaDaemon.KeaDHCPDaemon)
	require.NotNil(t, daemon.KeaDaemon.KeaDHCPDaemon.Stats)
	require.Equal(t, float32(1000), daemon.KeaDaemon.KeaDHCPDaemon.Stats.RPS1)
	require.Equal(t, float32(2000), daemon.KeaDaemon.KeaDHCPDaemon.Stats.RPS2)
}
