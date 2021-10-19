package configreview

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
)

// Tests creating new dispatcher instance.
func TestNewDispatcher(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	dispatcher := NewDispatcher(db)
	require.NotNil(t, dispatcher)
	require.Equal(t, db, dispatcher.db)
	require.NotNil(t, dispatcher.groups)
	require.NotNil(t, dispatcher.wg)
	require.NotNil(t, dispatcher.wg2)
	require.NotNil(t, dispatcher.mutex)
	require.NotNil(t, dispatcher.reviewDoneChan)
	require.NotNil(t, dispatcher.dispatchCtx)
	require.NotNil(t, dispatcher.cancelDispatch)
	require.NotNil(t, dispatcher.state)
}

// Tests the whole lifecycle of the dispatcher. In particular, it verifies that
// scheduled reviews are completed after stopping the dispatcher, and that the
// stop function waits for them.
func TestGracefulShutdown(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Create new dispatcher.
	dispatcher := NewDispatcher(db)
	require.NotNil(t, dispatcher)

	// We will simulate reviews for all daemon types.
	daemonNames := []string{"dhcp4", "dhcp6", "ca", "d2", "bind9"}
	// Selectors must correspond to the daemons above.
	selectors := []DispatchGroupSelector{
		KeaDHCPv4Daemon,
		KeaDHCPv6Daemon,
		KeaCADaemon,
		KeaD2Daemon,
		Bind9Daemon,
	}

	// Each review/daemon is assigned a dedicated communication channel, so
	// we can control the review process from the test.
	channels := make([]chan bool, len(daemonNames))
	reports := &[]*report{}
	mutex := &sync.Mutex{}

	// Register different producers for different daemon types.
	for i := 0; i < len(selectors); i++ {
		continueChan := make(chan bool)
		channels[i] = continueChan
		dispatcher.RegisterProducer(selectors[i], "test_producer", func(ctx *reviewContext) (*report, error) {
			// The producer waits here until the test gives it a green light
			// to proceed. It allows for controlling the concurency of the
			// reviews.
			<-continueChan
			report, err := newReport(ctx, "test output").create()
			mutex.Lock()
			defer mutex.Unlock()
			*reports = append(*reports, report)
			return report, err
		})
	}

	// Start the dispatcher's worker goroutine.
	dispatcher.Start()

	// Schedule reviews for multiple daemons/daemon types. The producers should
	// now block reading from the continueChan.
	for i := 0; i < len(daemonNames); i++ {
		daemon := &dbmodel.Daemon{
			ID:   int64(i),
			Name: daemonNames[i],
		}
		ok := dispatcher.BeginReview(daemon, nil)
		require.True(t, ok)
	}

	// Unblock first three producers. The remaining ones should still wait.
	// That way we cause the situtation that 3 reviews are done, and the
	// other ones are still in progress.
	for i := 0; i < 3; i++ {
		channels[i] <- true
	}

	// Stop the dispatcher while some reviews are still in progress. We
	// need to do it in the goroutine because the dispatcher.Shutdown()
	// is expected to block.
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		dispatcher.Shutdown()
	}()

	// Unblock remaining reviews.
	for i := 3; i < 5; i++ {
		channels[i] <- true
	}

	// Wait for the dispatcher to stop.
	wg.Wait()

	// We should have generated all reports.
	require.Len(t, *reports, 5)
}

// Tests that generated review reports are populated into the database.
func TestPopulateKeaReports(t *testing.T) {
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

	// Add an app with two daemons into the database.
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

	// Create review dispatcher.
	dispatcher := NewDispatcher(db)
	require.NotNil(t, dispatcher)

	// Register a different producer for each daemon.
	dispatcher.RegisterProducer(KeaDHCPv4Daemon, "dhcp4_test_producer", func(ctx *reviewContext) (*report, error) {
		report, err := newReport(ctx, "DHCPv4 test output").create()
		return report, err
	})

	dispatcher.RegisterProducer(KeaDHCPv6Daemon, "dhcp6_test_producer", func(ctx *reviewContext) (*report, error) {
		report, err := newReport(ctx, "DHCPv6 test output").create()
		return report, err
	})

	// Start the dispatcher worker.
	dispatcher.Start()
	defer dispatcher.Shutdown()

	// Simulate the case for the DHCPv6 daemon when its configuration (has) has
	// changed while we were performing the review. The dispatcher should detect
	// the mismatch and cancel the review before inserting the reports into the
	// database.
	daemons[1].KeaDaemon.ConfigHash = "3456"

	// Review errors will be recorded in this slice.
	innerErrors := make([]error, 2)
	wg := &sync.WaitGroup{}
	wg.Add(2)

	// Begin the reviews for both daemons.
	ok := dispatcher.BeginReview(daemons[0], func(daemonID int64, err error) {
		defer wg.Done()
		innerErrors[0] = err
	})
	require.True(t, ok)

	ok = dispatcher.BeginReview(daemons[1], func(daemonID int64, err error) {
		defer wg.Done()
		innerErrors[1] = err
	})
	require.True(t, ok)
	wg.Wait()

	// The review for the first daemon should be successful.
	require.NoError(t, innerErrors[0])
	// The review for the second daemon should be unsuccessful because
	// the dispatcher should detect the configuration mismatch.
	require.Error(t, innerErrors[1])

	// Ensure that the reports for the first daemon have been inserted.
	reports, err := dbmodel.GetConfigReportsByDaemonID(db, daemons[0].ID)
	require.NoError(t, err)
	require.Len(t, reports, 1)
	require.Equal(t, "dhcp4_test_producer", reports[0].ProducerName)
	require.Equal(t, "DHCPv4 test output", reports[0].Contents)

	// Ensure that the reports for the second daemon have not been inserted.
	reports, err = dbmodel.GetConfigReportsByDaemonID(db, daemons[1].ID)
	require.NoError(t, err)
	require.Empty(t, reports)
}

// Tests that the configuration reviews for the BIND9 daemon are populated
// into the database.
func TestPopulateBind9Reports(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add a machine.
	machine := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	// Add an app with a BIND9 daemon into the database.
	app := &dbmodel.App{
		Type:      dbmodel.AppTypeBind9,
		MachineID: machine.ID,
		Daemons: []*dbmodel.Daemon{
			{
				Name:   "bind9",
				Active: true,
			},
		},
	}
	daemons, err := dbmodel.AddApp(db, app)
	require.NoError(t, err)
	require.Len(t, daemons, 1)

	// Create the dispatcher instance.
	dispatcher := NewDispatcher(db)
	require.NotNil(t, dispatcher)

	// Register a test producer for the BIND9 daemon.
	dispatcher.RegisterProducer(Bind9Daemon, "test_producer", func(ctx *reviewContext) (*report, error) {
		report, err := newReport(ctx, "Bind9 test output").create()
		return report, err
	})

	// Start the dispatcher worker.
	dispatcher.Start()
	defer dispatcher.Shutdown()

	var innerError error
	wg := &sync.WaitGroup{}
	wg.Add(1)

	// Begin the review.
	ok := dispatcher.BeginReview(daemons[0], func(daemonID int64, err error) {
		defer wg.Done()
		innerError = err
	})
	require.True(t, ok)

	// Wait for the review to finish.
	wg.Wait()

	// There should be no error and the reports should be populated into the
	// database successfully.
	require.NoError(t, innerError)

	// Ensure that the reports have been populated.
	reports, err := dbmodel.GetConfigReportsByDaemonID(db, daemons[0].ID)
	require.NoError(t, err)
	require.Len(t, reports, 1)
	require.Equal(t, "test_producer", reports[0].ProducerName)
	require.Equal(t, "Bind9 test output", reports[0].Contents)
}

// Tests the scenario when another review for the same daemon is scheduled
// while the earlier review for this daemon is in progress.
func TestReviewInProgress(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add a machine.
	machine := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	// Add an app with a BIND9 daemon.
	app := &dbmodel.App{
		Type:      dbmodel.AppTypeBind9,
		MachineID: machine.ID,
		Daemons: []*dbmodel.Daemon{
			{
				Name:   "bind9",
				Active: true,
			},
		},
	}
	daemons, err := dbmodel.AddApp(db, app)
	require.NoError(t, err)
	require.Len(t, daemons, 1)

	// Create new dispatcher.
	dispatcher := NewDispatcher(db)
	require.NotNil(t, dispatcher)

	// Register the producer which blocks until it receives a value
	// over the continueChan or when doneCtx is cancelled.
	continueChan := make(chan bool)
	doneCtx, cancel := context.WithCancel(context.Background())
	dispatcher.RegisterProducer(Bind9Daemon, "test_producer", func(ctx *reviewContext) (*report, error) {
		report, err := newReport(ctx, "Bind9 test output").create()
		for {
			select {
			case <-continueChan:
				return report, err
			case <-doneCtx.Done():
				return report, err
			}
		}
	})

	// Start the dispatcher worker.
	dispatcher.Start()
	defer dispatcher.Shutdown()

	wg := &sync.WaitGroup{}
	wg.Add(1)

	// Begin first review.
	ok := dispatcher.BeginReview(daemons[0], func(daemonID int64, err error) {
		defer wg.Done()
	})
	require.True(t, ok)

	// Ensure that the fact that the review is in progress has been recorded.
	state, ok := dispatcher.state[daemons[0].ID]
	require.True(t, ok)
	require.True(t, state)

	// Try to begin another review for the same daemon.
	ok = dispatcher.BeginReview(daemons[0], nil)

	// Unblock the first review.
	continueChan <- true

	// This should be no-op, but we do it to unblock the second review in
	// case it was started. If the code is correct, the second review should
	// not start.
	cancel()

	// Make sure that the second review was not started while the first
	// one was still in progress.
	require.False(t, ok)

	// Wait for the review to complete.
	wg.Wait()

	// The state should no longer indicate that the review is in progress.
	state, ok = dispatcher.state[daemons[0].ID]
	require.True(t, ok)
	require.False(t, state)
}

// Tests the case when a producers requires reviewing another daemon's
// configuration, beside the subject daemon's configuration. It should
// cause deletion of both daemons' config reports and schedule a review
// of the other daemon's configuration internally.
func TestCascadeReview(t *testing.T) {
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
				Name:   "ca",
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

	// Create new dispatcher.
	dispatcher := NewDispatcher(db)
	require.NotNil(t, dispatcher)

	// Register a producer for the first daemon. It fetches the configuration of
	// the other daemon besides the reviewed configuration.
	dispatcher.RegisterProducer(KeaDHCPv4Daemon, "dhcp4_test_producer", func(ctx *reviewContext) (*report, error) {
		ctx.refDaemons = append(ctx.refDaemons, daemons[1])
		report, err := newReport(ctx, "DHCPv4 test output").
			referencingDaemon(ctx.refDaemons[0]).
			referencingDaemon(ctx.subjectDaemon).
			create()
		return report, err
	})

	// Register a producer for the second daemon. It fetches the configuration of
	// the other daemon besides the reviewed configuration.
	dispatcher.RegisterProducer(KeaCADaemon, "ca_test_producer", func(ctx *reviewContext) (*report, error) {
		ctx.refDaemons = append(ctx.refDaemons, daemons[0])
		report, _ := newReport(ctx, "CA test output").
			referencingDaemon(ctx.refDaemons[0]).
			referencingDaemon(ctx.subjectDaemon).
			create()
		return report, err
	})

	// Start the dispatcher worker.
	dispatcher.Start()
	defer dispatcher.Shutdown()

	wg := &sync.WaitGroup{}
	wg.Add(2)

	// Begin first the review for the first daemon.
	ok := dispatcher.BeginReview(daemons[0], func(daemonID int64, err error) {
		defer wg.Done()
	})
	require.True(t, ok)

	// Wait until it completes.
	wg.Wait()

	reports, err := dbmodel.GetConfigReportsByDaemonID(db, daemons[0].ID)
	require.NoError(t, err)
	require.Len(t, reports, 1)
	require.Equal(t, "dhcp4_test_producer", reports[0].ProducerName)
	require.Equal(t, "DHCPv4 test output", reports[0].Contents)

	// The first daemon's producer references the second daemon. Therefore,
	// this review should cause the review of the second daemon's
	// configuration. Ensure that it has been performed.
	reports, err = dbmodel.GetConfigReportsByDaemonID(db, daemons[1].ID)
	require.NoError(t, err)
	require.Len(t, reports, 1)
	require.Equal(t, "ca_test_producer", reports[0].ProducerName)
	require.Equal(t, "CA test output", reports[0].Contents)

	// Now, start the review for the second daemon. It should result in the
	// cascaded review as well.
	wg.Add(2)
	ok = dispatcher.BeginReview(daemons[1], func(daemonID int64, err error) {
		defer wg.Done()
	})
	require.True(t, ok)

	wg.Wait()

	reports, err = dbmodel.GetConfigReportsByDaemonID(db, daemons[0].ID)
	require.NoError(t, err)
	require.Len(t, reports, 1)
	require.Equal(t, "DHCPv4 test output", reports[0].Contents)

	reports, err = dbmodel.GetConfigReportsByDaemonID(db, daemons[1].ID)
	require.NoError(t, err)
	require.Len(t, reports, 1)
	require.Equal(t, "CA test output", reports[0].Contents)
}

// Tests that default producers are registered.
func TestRegisterDefaultProducers(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	dispatcher := NewDispatcher(db)
	require.NotNil(t, dispatcher)

	dispatcher.RegisterDefaultProducers()

	// KeaDHCPDaemon group.
	require.Contains(t, dispatcher.groups, KeaDHCPDaemon)
	producerNames := []string{}
	for _, p := range dispatcher.groups[KeaDHCPDaemon].producers {
		producerNames = append(producerNames, p.name)
	}
	require.Contains(t, producerNames, "stat_cmds_presence")
}
