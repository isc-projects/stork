package dbmodel

import (
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/go-pg/pg/v10"
	require "github.com/stretchr/testify/require"
	"isc.org/stork/datamodel/daemonname"
	"isc.org/stork/datamodel/protocoltype"
	dbtest "isc.org/stork/server/database/test"
)

// Test that new instance of the generic Kea daemon can be created.
func TestNewKeaDaemon(t *testing.T) {
	// Create machine first
	machine := &Machine{
		ID:        1,
		Address:   "localhost",
		AgentPort: 8080,
	}

	// Create the daemon with active flag set to true.
	daemon := NewDaemon(machine, daemonname.DHCPv4, true, []*AccessPoint{})
	require.NotNil(t, daemon)
	require.NotNil(t, daemon.KeaDaemon)
	require.NotNil(t, daemon.KeaDaemon.KeaDHCPDaemon)
	require.Nil(t, daemon.Bind9Daemon)
	require.Equal(t, daemonname.DHCPv4, daemon.Name)
	require.True(t, daemon.Active)

	// Create the non DHCP daemon.
	daemon = NewDaemon(machine, daemonname.CA, false, []*AccessPoint{})
	require.NotNil(t, daemon)
	require.NotNil(t, daemon.KeaDaemon)
	require.Nil(t, daemon.KeaDaemon.KeaDHCPDaemon)
	require.Nil(t, daemon.Bind9Daemon)
	require.Equal(t, daemonname.CA, daemon.Name)
	require.False(t, daemon.Active)
}

// Test that new instance of the Bind9 daemon can be created.
func TestNewBind9Daemon(t *testing.T) {
	// Create machine first
	machine := &Machine{
		ID:        1,
		Address:   "localhost",
		AgentPort: 8080,
	}

	daemon := NewDaemon(machine, daemonname.Bind9, true, []*AccessPoint{})
	require.NotNil(t, daemon)
	require.NotNil(t, daemon.Bind9Daemon)
	require.Nil(t, daemon.KeaDaemon)
	require.Equal(t, daemonname.Bind9, daemon.Name)
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

	// Create and add daemon
	daemon := NewDaemon(m, daemonname.DHCPv4, true, []*AccessPoint{})
	err = AddDaemon(db, daemon)
	require.NoError(t, err)
	require.NotZero(t, daemon.ID)

	// Remember the creation time so it can be compared after the update.
	createdAt := daemon.CreatedAt
	require.NotZero(t, createdAt)

	// Reset the creation time to ensure it is not modified during the update.
	daemon.CreatedAt = time.Time{}
	daemon.Pid = 123
	daemon.Name = daemonname.DHCPv6
	daemon.Active = false
	daemon.Version = "2.0.0"
	err = daemon.SetKeaConfigFromJSON([]byte(`{
        "Dhcp4": {
            "valid-lifetime": 1234
        }
    }`))
	require.NoError(t, err)

	daemon.KeaDaemon.KeaDHCPDaemon.Stats.RPS1 = 1000
	daemon.KeaDaemon.KeaDHCPDaemon.Stats.RPS2 = 2000

	err = UpdateDaemon(db, daemon)
	require.NoError(t, err)

	updatedDaemon, err := GetDaemonByID(db, daemon.ID)
	require.NoError(t, err)
	require.NotNil(t, updatedDaemon)

	require.Equal(t, createdAt, updatedDaemon.CreatedAt)
	require.EqualValues(t, 123, updatedDaemon.Pid)
	require.Equal(t, daemonname.DHCPv6, updatedDaemon.Name)
	require.False(t, updatedDaemon.Active)
	require.Equal(t, "2.0.0", updatedDaemon.Version)
	require.NotNil(t, updatedDaemon.KeaDaemon)
	require.NotNil(t, updatedDaemon.KeaDaemon.Config)
	require.NotNil(t, updatedDaemon.KeaDaemon.KeaDHCPDaemon)
	require.EqualValues(t, 1000, updatedDaemon.KeaDaemon.KeaDHCPDaemon.Stats.RPS1)
	require.EqualValues(t, 2000, updatedDaemon.KeaDaemon.KeaDHCPDaemon.Stats.RPS2)
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

	// Create and add daemon
	daemon := NewDaemon(m, daemonname.Bind9, true, []*AccessPoint{})
	err = AddDaemon(db, daemon)
	require.NoError(t, err)
	require.NotZero(t, daemon.ID)

	daemon.Pid = 123
	daemon.Active = false
	daemon.Version = "9.20"

	daemon.Bind9Daemon.Stats.ZoneCount = 123

	err = UpdateDaemon(db, daemon)
	require.NoError(t, err)

	updatedDaemon, err := GetDaemonByID(db, daemon.ID)
	require.NoError(t, err)
	require.NotNil(t, updatedDaemon)

	require.EqualValues(t, 123, updatedDaemon.Pid)
	require.Equal(t, daemonname.Bind9, updatedDaemon.Name)
	require.False(t, updatedDaemon.Active)
	require.Equal(t, "9.20", updatedDaemon.Version)
	require.NotNil(t, updatedDaemon.Bind9Daemon)
	require.EqualValues(t, 123, updatedDaemon.Bind9Daemon.Stats.ZoneCount)
}

// Test that the daemon statistics are properly updated.
func TestUpdateDaemonStatistics(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &Machine{Address: "localhost", AgentPort: 8080}
	_ = AddMachine(db, m)

	t.Run("KeaDHCPDaemon", func(t *testing.T) {
		daemon := NewDaemon(m, daemonname.DHCPv4, true, []*AccessPoint{})
		_ = AddDaemon(db, daemon)
		daemon.KeaDaemon.KeaDHCPDaemon.Stats.RPS1 = 42
		daemon.KeaDaemon.KeaDHCPDaemon.Stats.RPS2 = 24
		daemon.Active = false

		// Act
		err := UpdateDaemonStatistics(db, daemon)

		// Assert
		require.NoError(t, err)
		daemon, _ = GetDaemonByID(db, daemon.ID)
		require.EqualValues(t, 42, daemon.KeaDaemon.KeaDHCPDaemon.Stats.RPS1)
		require.EqualValues(t, 24, daemon.KeaDaemon.KeaDHCPDaemon.Stats.RPS2)
		require.True(t, daemon.Active)
	})

	t.Run("KeaDaemon", func(t *testing.T) {
		daemon := NewDaemon(m, daemonname.CA, true, []*AccessPoint{})
		_ = AddDaemon(db, daemon)
		daemon.Active = false

		// Act
		err := UpdateDaemonStatistics(db, daemon)

		// Assert
		require.NoError(t, err)
		daemon, _ = GetDaemonByID(db, daemon.ID)
		require.Nil(t, daemon.KeaDaemon.KeaDHCPDaemon)
		require.True(t, daemon.Active)
	})

	t.Run("Bind9Daemon", func(t *testing.T) {
		daemon := NewDaemon(m, daemonname.Bind9, true, []*AccessPoint{})
		_ = AddDaemon(db, daemon)
		daemon.Bind9Daemon.Stats.ZoneCount = 42
		daemon.Bind9Daemon.Stats.NamedStats.BootTime = "2024-01-01T12:00:00Z"
		daemon.Active = false

		// Act
		err := UpdateDaemonStatistics(db, daemon)

		// Assert
		require.NoError(t, err)
		daemon, _ = GetDaemonByID(db, daemon.ID)
		require.EqualValues(t, 42, daemon.Bind9Daemon.Stats.ZoneCount)
		require.Equal(t, "2024-01-01T12:00:00Z", daemon.Bind9Daemon.Stats.NamedStats.BootTime)
		require.True(t, daemon.Active)
	})

	t.Run("PDNSDaemon", func(t *testing.T) {
		daemon := NewDaemon(m, daemonname.PDNS, true, []*AccessPoint{})
		_ = AddDaemon(db, daemon)
		daemon.Active = false

		// Act
		err := UpdateDaemonStatistics(db, daemon)

		// Assert
		require.NoError(t, err)
		daemon, _ = GetDaemonByID(db, daemon.ID)
		require.True(t, daemon.Active)
	})
}

// Returns all HA state names to which the daemon belongs and the
// failure times.
func TestGetHAOverview(t *testing.T) {
	failoverAt := time.Date(2020, 6, 4, 11, 32, 0, 0, time.UTC)
	machine := &Machine{
		ID:        1,
		Address:   "localhost",
		AgentPort: 8080,
	}
	daemon := NewDaemon(machine, daemonname.DHCPv4, true, []*AccessPoint{})
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

	// create machine and then daemon
	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = AddMachine(db, m)
	require.NoError(t, err)
	require.NotZero(t, m.ID)

	// create daemon with initial configuration
	accessPoints := []*AccessPoint{
		{
			Type:    AccessPointControl,
			Address: "",
			Port:    1234,
			Key:     "",
		},
	}
	daemon := NewDaemon(m, daemonname.DHCPv4, true, accessPoints)

	// Set initial configuration with one logger.
	err = daemon.SetKeaConfigFromJSON([]byte(`{
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
    }`))
	require.NoError(t, err)

	err = AddDaemon(db, daemon)
	require.NoError(t, err)
	require.NotZero(t, daemon.ID)

	// get daemon, now it should be there
	dmn, err = GetDaemonByID(db, daemon.ID)
	require.NoError(t, err)
	require.NotNil(t, dmn)
	require.EqualValues(t, daemon.ID, dmn.ID)
	require.EqualValues(t, daemon.Active, dmn.Active)
	require.NotNil(t, dmn.KeaDaemon)
	require.NotNil(t, dmn.KeaDaemon.Config)
	require.NotNil(t, dmn.Machine)
	require.Len(t, dmn.AccessPoints, 1)
}

// Test getting multiple Kea daemons by IDs.
func TestGetKeaDaemonsByIDs(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Get non-existing daemons
	returnedDaemons, err := GetKeaDaemonsByIDs(db, []int64{123, 234})
	require.NoError(t, err)
	require.Empty(t, returnedDaemons)

	// Create machine and then daemons.
	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = AddMachine(db, m)
	require.NoError(t, err)
	require.NotZero(t, m.ID)

	// create daemons
	accessPoints := []*AccessPoint{
		{
			Type:    AccessPointControl,
			Address: "",
			Port:    1234,
			Key:     "",
		},
	}

	var daemons []*Daemon
	for _, daemonName := range []daemonname.Name{daemonname.DHCPv4, daemonname.DHCPv6, daemonname.D2, daemonname.CA} {
		daemon := NewDaemon(m, daemonName, true, accessPoints)
		err = AddDaemon(db, daemon)
		require.NoError(t, err)
		daemons = append(daemons, daemon)
	}

	// Get selected daemons.
	selectedDaemons := []int64{daemons[0].ID, daemons[1].ID}
	returnedDaemons, err = GetKeaDaemonsByIDs(db, selectedDaemons)
	require.NoError(t, err)
	require.Len(t, returnedDaemons, 2)

	var ids []int64
	for _, rd := range returnedDaemons {
		ids = append(ids, rd.ID)
		require.NotNil(t, rd.Machine)
		require.EqualValues(t, m.ID, rd.Machine.ID)
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

	// Add several Kea daemons of different type.
	accessPoints := []*AccessPoint{{
		Type:    AccessPointControl,
		Address: "",
		Port:    1234,
		Key:     "",
	}}

	daemonNames := []daemonname.Name{daemonname.DHCPv4, daemonname.DHCPv6, daemonname.CA, daemonname.D2}
	for _, dn := range daemonNames {
		daemon := NewDaemon(m, dn, true, accessPoints)
		err = AddDaemon(db, daemon)
		require.NoError(t, err)
	}

	// Add named daemon.
	daemon := NewDaemon(m, daemonname.Bind9, true, accessPoints)
	err = AddDaemon(db, daemon)
	require.NoError(t, err)

	// Try to get Kea DHCP daemons only. There should be two.
	daemons, err = GetKeaDHCPDaemons(db)
	require.NoError(t, err)
	require.Len(t, daemons, 2)

	// Validate returned daemons.
	names := []daemonname.Name{}
	for _, d := range daemons {
		names = append(names, d.Name)
		require.NotNil(t, d.Machine)
		require.NotNil(t, d.KeaDaemon)
		require.Equal(t, d.ID, d.KeaDaemon.DaemonID)
		require.NotNil(t, d.KeaDaemon.KeaDHCPDaemon)
	}
	require.Contains(t, names, daemonname.DHCPv4)
	require.Contains(t, names, daemonname.DHCPv6)
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

	// Create a daemon.
	daemon := NewDaemon(m, daemonname.Bind9, true, []*AccessPoint{})
	err = AddDaemon(db, daemon)
	require.NoError(t, err)
	require.NotZero(t, daemon.ID)

	// Start new transaction.
	tx, err = db.Begin()
	require.NoError(t, err)

	// An attempt to select no particular daemon for update should result
	// in an error.
	daemons, err = GetDaemonsForUpdate(tx, []*Daemon{})
	require.Error(t, err)
	require.Empty(t, daemons)

	// Select daemon for update.
	daemons, err = GetDaemonsForUpdate(tx, []*Daemon{daemon})
	require.NoError(t, err)
	require.Len(t, daemons, 1)

	// Sanity check selected data.
	require.NotZero(t, daemons[0].ID)
	require.True(t, daemons[0].Active)
	require.Equal(t, m.ID, daemons[0].MachineID)

	// When daemon is selected for update within a transaction, no other
	// transaction can modify the daemon until the current transaction is
	// committed or rolled back. We will now run a goroutine which will
	// attempt such a modification.
	var result pg.Result
	mutex := &sync.Mutex{}
	mutex.Lock()
	ch := make(chan struct{})
	wg := &sync.WaitGroup{}
	wg.Add(1)

	// Actually run the goroutine.
	go func() {
		defer wg.Done()
		// The main thread is waiting for this conditional to ensure that the
		// goroutine is started before the test continues.
		ch <- struct{}{}
		// Attempt to delete the daemon while the main transaction is in progress
		// and the daemons are locked for update. This should block until the
		// main transaction is committed or rolled back.
		result, _ = db.Model(daemon).WherePK().Delete()
	}()

	// Wait for the goroutine to begin.
	<-ch

	// We want to ensure that the goroutine executes db.Delete() before we
	// run the tx.Delete() from this transaction. If the tx.Delete() is
	// executed first, it has no effect on the test result. But, running
	// db.Delete() before tx.Delete() validates the effectiveness of the
	// locking mechanism.
	time.Sleep(100 * time.Millisecond)
	// It should take precedence over the db.Delete() invoked from the goroutine.
	// Thus, there should be no error.
	_, err = tx.Model(daemon).WherePK().Delete()
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

	// Create daemons.
	daemon1 := NewDaemon(m, daemonname.DHCPv4, true, []*AccessPoint{})
	daemon1.SetKeaConfigFromJSON([]byte(`{"Dhcp4": { }}`))
	err = AddDaemon(db, daemon1)
	require.NoError(t, err)

	daemon2 := NewDaemon(m, daemonname.DHCPv6, true, []*AccessPoint{})
	daemon2.SetKeaConfigFromJSON([]byte(`{"Dhcp6": { }}`))
	err = AddDaemon(db, daemon2)
	require.NoError(t, err)

	testDaemons := []*Daemon{daemon1, daemon2}

	// Start new transaction.
	tx, err = db.Begin()
	require.NoError(t, err)

	// An attempt to select no particular daemon for update should result
	// in an error.
	selectedDaemons, err := GetKeaDaemonsForUpdate(tx, []*Daemon{})
	require.Error(t, err)
	require.Empty(t, selectedDaemons)

	// Select both daemons for update.
	selectedDaemons, err = GetKeaDaemonsForUpdate(tx, testDaemons)
	require.NoError(t, err)
	require.Len(t, selectedDaemons, 2)

	// Sanity check selected data.
	for i, daemon := range selectedDaemons {
		require.NotZero(t, daemon.ID)
		require.True(t, daemon.Active)
		require.NotNil(t, daemon.KeaDaemon)
		require.NotZero(t, daemon.KeaDaemon.ID)
		require.NotNil(t, daemon.KeaDaemon.Config)
		require.Equal(t, testDaemons[i].KeaDaemon.ConfigHash, daemon.KeaDaemon.ConfigHash)
		require.NotNil(t, daemon.Machine)
		require.Equal(t, m.ID, daemon.Machine.ID)
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
		// Attempt to delete the daemon while the main transaction is in progress
		// and the daemons are locked for update. This should block until the
		// main transaction is committed or rolled back.
		result, _ = db.Model(daemon1).WherePK().Delete()
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
	_, err = tx.Model(daemon1).WherePK().Delete()
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
	machine := &Machine{
		ID:        1,
		Address:   "localhost",
		AgentPort: 8080,
	}
	daemon := NewDaemon(machine, daemonname.DHCPv4, true, []*AccessPoint{})

	// Set initial configuration with one logger.
	err := daemon.SetKeaConfigFromJSON([]byte(`{
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
    }`))
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
	err = daemon.SetKeaConfigFromJSON([]byte(`{
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
    }`))
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
	require.EqualValues(t, 1, daemon.LogTargets[1].DaemonID)

	// Set the second logger's ids.
	daemon.LogTargets[1].ID = 3
	daemon.LogTargets[1].DaemonID = 1

	// Check that the same data can be refreshed.
	err = daemon.SetKeaConfigFromJSON([]byte(`{
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
    }`))
	require.NoError(t, err)

	require.Len(t, daemon.LogTargets, 2)

	// Check that the number of loggers can be reduced.
	err = daemon.SetKeaConfigFromJSON([]byte(`{
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
    }`))
	require.NoError(t, err)

	require.Len(t, daemon.LogTargets, 1)
}

// This test verifies that the config hash is setting configuration as string.
func TestSetKeaConfigFromJSONWithHash(t *testing.T) {
	machine := &Machine{
		ID:        1,
		Address:   "localhost",
		AgentPort: 8080,
	}
	daemon := NewDaemon(machine, daemonname.DHCPv4, true, []*AccessPoint{})

	// Set initial configuration with one logger.
	err := daemon.SetKeaConfigFromJSON([]byte(`{
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
    }`))
	require.NoError(t, err)
	require.NotNil(t, daemon.KeaDaemon)
	require.NotNil(t, daemon.KeaDaemon.Config)
	require.Len(t, daemon.KeaDaemon.ConfigHash, 32)
}

// Test that SetConfig does not set hash for the config.
func TestSetConfig(t *testing.T) {
	machine := &Machine{
		ID:        1,
		Address:   "localhost",
		AgentPort: 8080,
	}
	daemon := NewDaemon(machine, daemonname.DHCPv4, true, []*AccessPoint{})

	err := daemon.SetKeaConfigFromJSON([]byte(`{"Dhcp4": {}}`))
	require.NoError(t, err)

	require.NotNil(t, daemon.KeaDaemon)
	require.NotNil(t, daemon.KeaDaemon.Config)
	require.NotEmpty(t, daemon.KeaDaemon.ConfigHash)
}

// Test that shallow copy of a Kea daemon can be created.
func TestShallowCopyKeaDaemon(t *testing.T) {
	machine := &Machine{
		ID:        1,
		Address:   "localhost",
		AgentPort: 8080,
	}
	// Create Daemon instance with not nil KeaDaemon.
	daemon := NewDaemon(machine, daemonname.DHCPv4, true, []*AccessPoint{})
	copy := ShallowCopyKeaDaemon(daemon)
	require.NotNil(t, copy)
	require.NotNil(t, copy.KeaDaemon)
	require.NotSame(t, daemon, copy)
	require.NotSame(t, daemon.KeaDaemon, copy.KeaDaemon)
	require.Equal(t, daemon, copy)
	require.Equal(t, daemon.Machine, copy.Machine)
	require.Equal(t, daemon.MachineID, copy.MachineID)

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
	machine := &Machine{
		ID:        1,
		Address:   "localhost",
		AgentPort: 8080,
	}

	// Create daemon with the given configuration.
	accessPoints := []*AccessPoint{{
		Type:     AccessPointControl,
		Address:  "",
		Port:     1234,
		Key:      "",
		Protocol: protocoltype.HTTP,
	}}
	daemon := NewDaemon(machine, daemonname.DHCPv4, true, accessPoints)
	err := daemon.SetKeaConfigFromJSON([]byte(`{
		"Dhcp4": {
            "subnet4": [
				{
					"id":     1,
					"subnet": "192.0.2.0/24"
				}
            ]
        }
    }`))
	require.NoError(t, err)

	// Try to find a non-existing subnet.
	require.Zero(t, daemon.GetLocalSubnetID("192.0.3.0/24"))
	// Next, try to find the existing subnet.
	require.EqualValues(t, 1, daemon.GetLocalSubnetID("192.0.2.0/24"))
}

// Test DaemonTag interface implementation.
func TestDaemonTag(t *testing.T) {
	machine := &Machine{
		ID:        2,
		Address:   "localhost",
		AgentPort: 8080,
	}
	daemon := Daemon{
		ID:        1,
		Name:      daemonname.DHCPv4,
		MachineID: 2,
		Machine:   machine,
	}
	require.EqualValues(t, 1, daemon.GetID())
	require.Equal(t, daemonname.DHCPv4, daemon.GetName())
	require.EqualValues(t, 2, daemon.GetMachineID())
}

// Test that GetMachineTag() returns the machine reference.
func TestDaemonTagMachineTag(t *testing.T) {
	machine := &Machine{
		ID:        42,
		Address:   "localhost",
		AgentPort: 8080,
	}
	daemon := NewDaemon(machine, daemonname.Bind9, true, []*AccessPoint{})
	daemon.ID = 24
	var daemonTag DaemonTag = daemon
	require.EqualValues(t, 24, daemonTag.GetID())
	require.EqualValues(t, 42, daemonTag.GetMachineID())
	require.Equal(t, daemonname.Bind9, daemonTag.GetName())
}

// Test that the machine ID is not nil if the machine reference is not set.
func TestDaemonTagMissingMachineID(t *testing.T) {
	// Arrange
	daemon := Daemon{
		MachineID: 42,
	}

	// Act & Assert
	require.EqualValues(t, 42, daemon.GetMachineID())
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

	daemon1 := NewDaemon(machine, daemonname.DHCPv4, true, []*AccessPoint{})
	daemon1.KeaDaemon.ConfigHash = "1234"
	err = AddDaemon(db, daemon1)
	require.NoError(t, err)

	daemon2 := NewDaemon(machine, daemonname.DHCPv6, true, []*AccessPoint{})
	daemon2.KeaDaemon.ConfigHash = "2345"
	err = AddDaemon(db, daemon2)
	require.NoError(t, err)

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

	// Create and add daemon
	daemon := NewDaemon(m, daemonname.DHCPv4, true, []*AccessPoint{})
	err = AddDaemon(db, daemon)
	require.NoError(t, err)
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
		ID:          daemon.KeaDaemon.KeaDHCPDaemon.ID,
		KeaDaemonID: daemon.KeaDaemon.KeaDHCPDaemon.KeaDaemonID,
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
	daemons, err := GetKeaDaemonsByIDs(db, []int64{daemon.ID})
	require.NoError(t, err)
	require.Len(t, daemons, 1)
	daemon = &daemons[0]
	require.NotNil(t, daemon.KeaDaemon)
	require.NotNil(t, daemon.KeaDaemon.KeaDHCPDaemon)
	require.NotNil(t, daemon.KeaDaemon.KeaDHCPDaemon.Stats)
	require.Equal(t, float32(1000), daemon.KeaDaemon.KeaDHCPDaemon.Stats.RPS1)
	require.Equal(t, float32(2000), daemon.KeaDaemon.KeaDHCPDaemon.Stats.RPS2)
}

// The benchmark checks performance of fetching a daemon by ID.
// I used it to compare performance in relation to the number of daemon
// relations included.
//
// WithAllRelations     6991    160582 ns/op    9843 B/op     123 allocs/op
// OnlyKeaRelation	    9717    112593 ns/op    7810 B/op     103 allocs/op
// OnlyBind9Relation   12734     93699 ns/op    7115 B/op      98 allocs/op
// OnlyPDNSRelation    12949     92653 ns/op    6748 B/op      97 allocs/op
// WithoutRelations    15116     78878 ns/op    5707 B/op      87 allocs/op
//
// Generated on goos: darwin, goarch: arm64, cpu: Apple M4 Pro.
func BenchmarkGetDaemonByID(b *testing.B) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(b)
	defer teardown()

	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	_ = AddMachine(db, m)

	accessPoints := []*AccessPoint{
		{
			Type:    AccessPointControl,
			Address: "",
			Port:    1234,
			Key:     "",
		},
	}

	for i := 0; i < 300; i++ {
		daemon := NewDaemon(m, daemonname.DHCPv4, true, accessPoints)
		_ = AddDaemon(db, daemon)
	}
	for i := 0; i < 300; i++ {
		daemon := NewDaemon(m, daemonname.Bind9, true, accessPoints)
		_ = AddDaemon(db, daemon)
	}
	for i := 0; i < 300; i++ {
		daemon := NewDaemon(m, daemonname.PDNS, true, accessPoints)
		_ = AddDaemon(db, daemon)
	}

	for b.Loop() {
		id := rand.Int()%900 + 1
		_, _ = GetDaemonByID(db, int64(id))
	}
}

// Test that the daemons can be fetched by their names.
func TestGetDaemonsByName(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	_ = AddMachine(db, m)

	daemonNames := []daemonname.Name{
		daemonname.CA,
		daemonname.DHCPv4,
		daemonname.DHCPv4,
		daemonname.DHCPv4,
		daemonname.DHCPv6,
		daemonname.Bind9,
		daemonname.PDNS,
	}

	for _, name := range daemonNames {
		daemon := NewDaemon(m, name, true, []*AccessPoint{})
		_ = AddDaemon(db, daemon)
	}

	t.Run("CA", func(t *testing.T) {
		// Act
		daemons, err := GetDaemonsByName(db, daemonname.CA)

		// Assert
		require.NoError(t, err)
		require.Len(t, daemons, 1)
		daemon := daemons[0]
		require.Equal(t, daemonname.CA, daemon.Name)
		require.NotNil(t, daemon.KeaDaemon)
		require.Nil(t, daemon.KeaDaemon.KeaDHCPDaemon)
		require.Nil(t, daemon.Bind9Daemon)
		require.Nil(t, daemon.PDNSDaemon)
	})

	t.Run("DHCPv4", func(t *testing.T) {
		// Act
		daemons, err := GetDaemonsByName(db, daemonname.DHCPv4)

		// Assert
		require.NoError(t, err)
		require.Len(t, daemons, 3)
		for _, daemon := range daemons {
			require.Equal(t, daemonname.DHCPv4, daemon.Name)
			require.NotNil(t, daemon.KeaDaemon)
			require.NotNil(t, daemon.KeaDaemon.KeaDHCPDaemon)
			require.Nil(t, daemon.Bind9Daemon)
			require.Nil(t, daemon.PDNSDaemon)
		}
	})

	t.Run("DHCPv6", func(t *testing.T) {
		// Act
		daemons, err := GetDaemonsByName(db, daemonname.DHCPv6)

		// Assert
		require.NoError(t, err)
		require.Len(t, daemons, 1)
		daemon := daemons[0]
		require.Equal(t, daemonname.DHCPv6, daemon.Name)
		require.NotNil(t, daemon.KeaDaemon)
		require.NotNil(t, daemon.KeaDaemon.KeaDHCPDaemon)
		require.Nil(t, daemon.Bind9Daemon)
		require.Nil(t, daemon.PDNSDaemon)
	})

	t.Run("BIND 9", func(t *testing.T) {
		// Act
		daemons, err := GetDaemonsByName(db, daemonname.Bind9)

		// Assert
		require.NoError(t, err)
		require.Len(t, daemons, 1)
		daemon := daemons[0]
		require.Equal(t, daemonname.Bind9, daemon.Name)
		require.NotNil(t, daemon.Bind9Daemon)
		require.Nil(t, daemon.KeaDaemon)
		require.Nil(t, daemon.PDNSDaemon)
	})

	t.Run("PDNS", func(t *testing.T) {
		// Act
		daemons, err := GetDaemonsByName(db, daemonname.PDNS)

		// Assert
		require.NoError(t, err)
		require.Len(t, daemons, 1)
		daemon := daemons[0]
		require.Equal(t, daemonname.PDNS, daemon.Name)
		require.NotNil(t, daemon.PDNSDaemon)
		require.Nil(t, daemon.KeaDaemon)
		require.Nil(t, daemon.Bind9Daemon)
	})
}

// Test getting machine tag from daemon.
func TestGetMachineTag(t *testing.T) {
	machine := &Machine{
		ID:        1,
		Address:   "localhost",
		AgentPort: 8080,
	}
	daemon := Daemon{
		Machine: machine,
	}
	require.Equal(t, machine, daemon.GetMachineTag())
}

// Test getting DNS daemon by ID.
func TestGetDNSDaemonByID(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &Machine{Address: "localhost", AgentPort: 8080}
	err := AddMachine(db, m)
	require.NoError(t, err)

	// Add BIND9 daemon
	bind9 := NewDaemon(m, daemonname.Bind9, true, []*AccessPoint{})
	err = AddDaemon(db, bind9)
	require.NoError(t, err)

	// Add PDNS daemon.
	pdns := NewDaemon(m, daemonname.PDNS, true, []*AccessPoint{})
	err = AddDaemon(db, pdns)
	require.NoError(t, err)

	// Add DHCP daemon.
	dhcp := NewDaemon(m, daemonname.DHCPv4, true, []*AccessPoint{})
	err = AddDaemon(db, dhcp)
	require.NoError(t, err)

	// Get BIND9.
	d, err := GetDNSDaemonByID(db, bind9.ID)
	require.NoError(t, err)
	require.NotNil(t, d)
	require.Equal(t, daemonname.Bind9, d.Name)
	require.NotNil(t, d.Bind9Daemon)

	// Get PDNS.
	d, err = GetDNSDaemonByID(db, pdns.ID)
	require.NoError(t, err)
	require.NotNil(t, d)
	require.Equal(t, daemonname.PDNS, d.Name)
	require.NotNil(t, d.PDNSDaemon)

	// Get DHCP.
	d, err = GetDNSDaemonByID(db, dhcp.ID)
	require.NoError(t, err)
	require.NotNil(t, d)
	require.Equal(t, daemonname.DHCPv4, d.Name)
	require.Nil(t, d.Bind9Daemon)
	require.Nil(t, d.PDNSDaemon)
	require.Nil(t, d.KeaDaemon)
}

// Test getting daemons by machine ID.
func TestGetDaemonsByMachine(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m1 := &Machine{Address: "machine1", AgentPort: 8080}
	err := AddMachine(db, m1)
	require.NoError(t, err)
	m2 := &Machine{Address: "machine2", AgentPort: 8080}
	err = AddMachine(db, m2)
	require.NoError(t, err)

	d1 := NewDaemon(m1, daemonname.DHCPv4, true, []*AccessPoint{})
	err = AddDaemon(db, d1)
	require.NoError(t, err)
	d2 := NewDaemon(m1, daemonname.Bind9, true, []*AccessPoint{})
	err = AddDaemon(db, d2)
	require.NoError(t, err)
	d3 := NewDaemon(m2, daemonname.PDNS, true, []*AccessPoint{})
	err = AddDaemon(db, d3)
	require.NoError(t, err)

	daemons, err := GetDaemonsByMachine(db, m1.ID)
	require.NoError(t, err)
	require.Len(t, daemons, 2)
	// Check IDs or Names
	ids := []int64{daemons[0].ID, daemons[1].ID}
	require.Contains(t, ids, d1.ID)
	require.Contains(t, ids, d2.ID)

	daemons, err = GetDaemonsByMachine(db, m2.ID)
	require.NoError(t, err)
	require.Len(t, daemons, 1)
	require.Equal(t, d3.ID, daemons[0].ID)
}

// Test getting daemons by page.
func TestGetDaemonsByPage(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &Machine{Address: "localhost", AgentPort: 8080}
	err := AddMachine(db, m)
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		d := NewDaemon(m, daemonname.DHCPv4, true, []*AccessPoint{})
		d.Version = "1.0"
		err = AddDaemon(db, d)
		require.NoError(t, err)
	}

	// Test pagination
	daemons, total, err := GetDaemonsByPage(db, 0, 5, nil, "id", SortDirAsc)
	require.NoError(t, err)
	require.Len(t, daemons, 5)
	require.EqualValues(t, 10, total)

	daemons, total, err = GetDaemonsByPage(db, 5, 5, nil, "id", SortDirAsc)
	require.NoError(t, err)
	require.Len(t, daemons, 5)
	require.EqualValues(t, 10, total)

	// Test filtering
	filter := "1.0"
	daemons, total, err = GetDaemonsByPage(db, 0, 10, &filter, "id", SortDirAsc)
	require.NoError(t, err)
	require.Len(t, daemons, 10)
	require.EqualValues(t, 10, total)

	filter = "non-existent"
	daemons, total, err = GetDaemonsByPage(db, 0, 10, &filter, "id", SortDirAsc)
	require.NoError(t, err)
	require.Len(t, daemons, 0)
	require.EqualValues(t, 0, total)

	// Test filtering by name
	daemons, total, err = GetDaemonsByPage(db, 0, 10, nil, "id", SortDirAsc, daemonname.DHCPv4)
	require.NoError(t, err)
	require.Len(t, daemons, 10)
	require.EqualValues(t, 10, total)

	daemons, total, err = GetDaemonsByPage(db, 0, 10, nil, "id", SortDirAsc, daemonname.Bind9)
	require.NoError(t, err)
	require.Len(t, daemons, 0)
	require.EqualValues(t, 0, total)
}

// Test getting DHCP daemons.
func TestGetDHCPDaemons(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &Machine{Address: "localhost", AgentPort: 8080}
	err := AddMachine(db, m)
	require.NoError(t, err)

	err = AddDaemon(db, NewDaemon(m, daemonname.DHCPv4, true, []*AccessPoint{}))
	require.NoError(t, err)
	err = AddDaemon(db, NewDaemon(m, daemonname.DHCPv6, true, []*AccessPoint{}))
	require.NoError(t, err)
	err = AddDaemon(db, NewDaemon(m, daemonname.Bind9, true, []*AccessPoint{}))
	require.NoError(t, err)

	daemons, err := GetDHCPDaemons(db)
	require.NoError(t, err)
	require.Len(t, daemons, 2)
	for _, d := range daemons {
		require.Contains(t, []daemonname.Name{daemonname.DHCPv4, daemonname.DHCPv6}, d.Name)
	}
}

// Test getting DNS daemons.
func TestGetDNSDaemons(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &Machine{Address: "localhost", AgentPort: 8080}
	err := AddMachine(db, m)
	require.NoError(t, err)

	err = AddDaemon(db, NewDaemon(m, daemonname.DHCPv4, true, []*AccessPoint{}))
	require.NoError(t, err)
	err = AddDaemon(db, NewDaemon(m, daemonname.Bind9, true, []*AccessPoint{}))
	require.NoError(t, err)
	err = AddDaemon(db, NewDaemon(m, daemonname.PDNS, true, []*AccessPoint{}))
	require.NoError(t, err)

	daemons, err := GetDNSDaemons(db)
	require.NoError(t, err)
	require.Len(t, daemons, 2)
	for _, d := range daemons {
		require.Contains(t, []daemonname.Name{daemonname.Bind9, daemonname.PDNS}, d.Name)
	}
}

// Test getting Kea daemon by ID.
func TestGetKeaDaemonByID(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &Machine{Address: "localhost", AgentPort: 8080}
	err := AddMachine(db, m)
	require.NoError(t, err)

	// Add Kea DHCPv4 daemon
	kea4 := NewDaemon(m, daemonname.DHCPv4, true, []*AccessPoint{})
	err = AddDaemon(db, kea4)
	require.NoError(t, err)

	// Add BIND9 daemon
	bind9 := NewDaemon(m, daemonname.Bind9, true, []*AccessPoint{})
	err = AddDaemon(db, bind9)
	require.NoError(t, err)

	// Get Kea DHCPv4
	d, err := GetKeaDaemonByID(db, kea4.ID)
	require.NoError(t, err)
	require.NotNil(t, d)
	require.Equal(t, daemonname.DHCPv4, d.Name)
	require.NotNil(t, d.KeaDaemon)
	require.NotNil(t, d.KeaDaemon.KeaDHCPDaemon)

	// Get BIND9 (should return daemon but without this daemon specific fields
	// populated).
	d, err = GetKeaDaemonByID(db, bind9.ID)
	require.NoError(t, err)
	require.NotNil(t, d)
	require.Equal(t, daemonname.Bind9, d.Name)
	require.Nil(t, d.KeaDaemon)
	require.Nil(t, d.Bind9Daemon)

	// Get non-existing daemon
	d, err = GetKeaDaemonByID(db, 12345)
	require.NoError(t, err)
	require.Nil(t, d)
}
