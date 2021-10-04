package configreview

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
)

func TestNewDispatcher(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	dispatcher := NewDispatcher(db)
	require.NotNil(t, dispatcher)
}

func TestDispatcherStartStop(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	dispatcher := NewDispatcher(db)
	require.NotNil(t, dispatcher)

	dispatcher.Start()
	dispatcher.Stop()
}

func TestGracefulStop(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	dispatcher := NewDispatcher(db)
	require.NotNil(t, dispatcher)

	daemonNames := []string{"dhcp4", "dhcp6", "ca", "d2", "bind9"}
	selectors := []DispatchGroupSelector{
		KeaDHCPv4Daemon,
		KeaDHCPv6Daemon,
		KeaCADaemon,
		KeaD2Daemon,
		Bind9Daemon,
	}

	channels := make([]chan bool, len(daemonNames))
	reports := &[]*report{}
	mutex := &sync.Mutex{}

	for i := 0; i < len(selectors); i++ {
		continueChan := make(chan bool)
		channels[i] = continueChan
		dispatcher.RegisterProducer(selectors[i], "test_producer", func(*reviewContext) *report {
			<-continueChan
			report := &report{
				issue: "test output",
			}
			mutex.Lock()
			defer mutex.Unlock()
			*reports = append(*reports, report)
			return report
		})
	}

	dispatcher.Start()

	for i := 0; i < len(daemonNames); i++ {
		daemon := &dbmodel.Daemon{
			ID:   int64(i),
			Name: daemonNames[i],
		}
		err := dispatcher.BeginForDaemon(daemon)
		require.NoError(t, err)
	}

	for i := 0; i < 3; i++ {
		channels[i] <- true
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		dispatcher.Stop()
	}()

	for i := 3; i < 5; i++ {
		channels[i] <- true
	}

	wg.Wait()

	require.Len(t, *reports, 5)
}

func TestPopulateReports(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add a machine.
	machine := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	// Create the configs for daemons.
	config1, err := dbmodel.NewKeaConfigFromJSON(`{"Dhcp4": { }}`)
	require.NoError(t, err)
	config2, err := dbmodel.NewKeaConfigFromJSON(`{"Dhcp6": { }}`)
	require.NoError(t, err)

	// Add an app with two daemons.
	app := &dbmodel.App{
		Type:      dbmodel.AppTypeKea,
		MachineID: machine.ID,
		Daemons: []*dbmodel.Daemon{
			{
				Name:   "dhcp4",
				Active: true,
				KeaDaemon: &dbmodel.KeaDaemon{
					Config:     config1,
					ConfigHash: "1234",
				},
			},
			{
				Name:   "dhcp6",
				Active: true,
				KeaDaemon: &dbmodel.KeaDaemon{
					Config:     config2,
					ConfigHash: "2345",
				},
			},
		},
	}
	daemons, err := dbmodel.AddApp(db, app)
	require.NoError(t, err)
	require.Len(t, daemons, 2)

	dispatcher := NewDispatcher(db)
	require.NotNil(t, dispatcher)

	dispatcher.RegisterProducer(KeaDHCPv4Daemon, "test_producer", func(*reviewContext) *report {
		report := &report{
			issue: "test output",
		}
		return report
	})

	dispatcher.Start()
	defer dispatcher.Stop()

	err = dispatcher.BeginForDaemon(daemons[0])
	require.NoError(t, err)
}
