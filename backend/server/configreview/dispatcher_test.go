package configreview

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
)

// Test hasher.
type testHasher struct{}

// Test hashing function returning predictable value.
func (h testHasher) Hash(input any) string {
	return "test"
}

// Tests creating new dispatcher instance.
func TestNewDispatcher(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	dispatcher := NewDispatcher(db).(*dispatcherImpl)
	require.NotNil(t, dispatcher)
	require.Equal(t, db, dispatcher.db)
	require.NotNil(t, dispatcher.groups)
	require.NotNil(t, dispatcher.shutdownWg)
	require.NotNil(t, dispatcher.reviewWg)
	require.NotNil(t, dispatcher.mutex)
	require.NotNil(t, dispatcher.reviewDoneChan)
	require.NotNil(t, dispatcher.dispatchCtx)
	require.NotNil(t, dispatcher.cancelDispatch)
	require.NotNil(t, dispatcher.state)
	require.NotNil(t, dispatcher.checkerController)
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
	daemonNames := []string{"dhcp4", "dhcp6", "ca", "d2", "named"}
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
	reports := &[]*Report{}
	mutex := &sync.Mutex{}

	// Register different checkers for different daemon types.
	for i := 0; i < len(selectors); i++ {
		continueChan := make(chan bool)
		channels[i] = continueChan
		dispatcher.RegisterChecker(selectors[i], "test_checker", GetDefaultTriggers(), func(ctx *ReviewContext) (*Report, error) {
			// The checker waits here until the test gives it a green light
			// to proceed. It allows for controlling the concurrency of the
			// reviews.
			<-continueChan
			report, err := NewReport(ctx, "test output").create()
			mutex.Lock()
			defer mutex.Unlock()
			*reports = append(*reports, report)
			return report, err
		})
	}

	// Start the dispatcher's worker goroutine.
	dispatcher.Start()

	// Schedule reviews for multiple daemons/daemon types. The checkers should
	// now block reading from the continueChan.
	for i := 0; i < len(daemonNames); i++ {
		daemon := &dbmodel.Daemon{
			ID:   int64(i),
			Name: daemonNames[i],
		}
		ok := dispatcher.BeginReview(daemon, Triggers{ConfigModified}, nil)
		require.True(t, ok)
	}

	// Unblock first three checkers. The remaining ones should still wait.
	// That way we cause the situation that 3 reviews are done, and the
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

	// Register a different checker for each daemon.
	dispatcher.RegisterChecker(KeaDHCPv4Daemon, "dhcp4_test_checker", GetDefaultTriggers(), func(ctx *ReviewContext) (*Report, error) {
		report, err := NewReport(ctx, "DHCPv4 test output").create()
		return report, err
	})

	dispatcher.RegisterChecker(KeaDHCPv6Daemon, "dhcp6_test_checker", GetDefaultTriggers(), func(ctx *ReviewContext) (*Report, error) {
		report, err := NewReport(ctx, "DHCPv6 test output").create()
		return report, err
	})
	dispatcher.RegisterChecker(KeaDHCPDaemon, "dhcp_test_checker", GetDefaultTriggers(), func(rc *ReviewContext) (*Report, error) {
		return nil, nil
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
	ok := dispatcher.BeginReview(daemons[0], Triggers{ConfigModified}, func(daemonID int64, err error) {
		defer wg.Done()
		innerErrors[0] = err
	})
	require.True(t, ok)

	ok = dispatcher.BeginReview(daemons[1], Triggers{ConfigModified}, func(daemonID int64, err error) {
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
	reports, total, err := dbmodel.GetConfigReportsByDaemonID(db, 0, 0, daemons[0].ID, false)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, reports, 2)
	require.Equal(t, "dhcp_test_checker", reports[0].CheckerName)
	require.Nil(t, reports[0].Content)
	require.Equal(t, "dhcp4_test_checker", reports[1].CheckerName)
	require.Equal(t, "DHCPv4 test output", *reports[1].Content)

	review, err := dbmodel.GetConfigReviewByDaemonID(db, daemons[0].ID)
	require.NoError(t, err)
	require.NotNil(t, review)
	require.WithinDuration(t, time.Now(), review.CreatedAt, 5*time.Second)
	require.NotEmpty(t, review.ConfigHash)
	require.NotEmpty(t, review.Signature)

	// Filter out the reports without issues.
	reports, total, err = dbmodel.GetConfigReportsByDaemonID(db, 0, 0, daemons[0].ID, true)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, reports, 1)
	require.Equal(t, "dhcp4_test_checker", reports[0].CheckerName)
	require.Equal(t, "DHCPv4 test output", *reports[0].Content)

	// Ensure that the reports for the second daemon have not been inserted.
	reports, total, err = dbmodel.GetConfigReportsByDaemonID(db, 0, 0, daemons[1].ID, false)
	require.NoError(t, err)
	require.Zero(t, total)
	require.Empty(t, reports)

	review, err = dbmodel.GetConfigReviewByDaemonID(db, daemons[1].ID)
	require.NoError(t, err)
	require.Nil(t, review)
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
				Name:   "named",
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

	// Register a test checker for the BIND9 daemon.
	dispatcher.RegisterChecker(Bind9Daemon, "test_checker", GetDefaultTriggers(), func(ctx *ReviewContext) (*Report, error) {
		report, err := NewReport(ctx, "Bind9 test output").create()
		return report, err
	})

	// Start the dispatcher worker.
	dispatcher.Start()
	defer dispatcher.Shutdown()

	var innerError error
	wg := &sync.WaitGroup{}
	wg.Add(1)

	// Begin the review.
	ok := dispatcher.BeginReview(daemons[0], Triggers{ConfigModified}, func(daemonID int64, err error) {
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
	reports, total, err := dbmodel.GetConfigReportsByDaemonID(db, 0, 0, daemons[0].ID, false)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, reports, 1)
	require.Equal(t, "test_checker", reports[0].CheckerName)
	require.Equal(t, "Bind9 test output", *reports[0].Content)
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
				Name:   "named",
				Active: true,
			},
		},
	}
	daemons, err := dbmodel.AddApp(db, app)
	require.NoError(t, err)
	require.Len(t, daemons, 1)

	// Create new dispatcher.
	dispatcher := NewDispatcher(db).(*dispatcherImpl)
	require.NotNil(t, dispatcher)

	// Register the checker which blocks until it receives a value
	// over the continueChan or when doneCtx is cancelled.
	continueChan := make(chan bool)
	doneCtx, cancel := context.WithCancel(context.Background())
	dispatcher.RegisterChecker(Bind9Daemon, "test_checker", GetDefaultTriggers(), func(ctx *ReviewContext) (*Report, error) {
		report, err := NewReport(ctx, "Bind9 test output").create()
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
	ok := dispatcher.BeginReview(daemons[0], Triggers{ConfigModified}, func(daemonID int64, err error) {
		defer wg.Done()
	})
	require.True(t, ok)

	// Ensure that the fact that the review is in progress has been recorded.
	state, ok := dispatcher.state[daemons[0].ID]
	require.True(t, ok)
	require.True(t, state)
	require.True(t, dispatcher.ReviewInProgress(daemons[0].ID))

	// Try to begin another review for the same daemon.
	ok = dispatcher.BeginReview(daemons[0], Triggers{ConfigModified}, nil)

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
	require.False(t, dispatcher.ReviewInProgress(daemons[0].ID))
}

// Tests the case when a checker requires reviewing another daemon's
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

	// Register a checker for the first daemon. It fetches the configuration of
	// the other daemon besides the reviewed configuration.
	dispatcher.RegisterChecker(KeaDHCPv4Daemon, "dhcp4_test_checker", GetDefaultTriggers(), func(ctx *ReviewContext) (*Report, error) {
		ctx.refDaemons = append(ctx.refDaemons, daemons[1])
		report, err := NewReport(ctx, "DHCPv4 test output").
			referencingDaemon(ctx.refDaemons[0]).
			referencingDaemon(ctx.subjectDaemon).
			create()
		return report, err
	})

	// Register a checker for the second daemon. It fetches the configuration of
	// the other daemon besides the reviewed configuration.
	dispatcher.RegisterChecker(KeaCADaemon, "ca_test_checker", GetDefaultTriggers(), func(ctx *ReviewContext) (*Report, error) {
		ctx.refDaemons = append(ctx.refDaemons, daemons[0])
		report, _ := NewReport(ctx, "CA test output").
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
	ok := dispatcher.BeginReview(daemons[0], Triggers{ConfigModified}, func(daemonID int64, err error) {
		defer wg.Done()
	})
	require.True(t, ok)

	// Wait until it completes.
	wg.Wait()

	reports, total, err := dbmodel.GetConfigReportsByDaemonID(db, 0, 0, daemons[0].ID, false)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, reports, 1)
	require.Equal(t, "dhcp4_test_checker", reports[0].CheckerName)
	require.Equal(t, "DHCPv4 test output", *reports[0].Content)

	// The first daemon's checker references the second daemon. Therefore,
	// this review should cause the review of the second daemon's
	// configuration. Ensure that it has been performed.
	reports, total, err = dbmodel.GetConfigReportsByDaemonID(db, 0, 0, daemons[1].ID, false)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, reports, 1)
	require.Equal(t, "ca_test_checker", reports[0].CheckerName)
	require.Equal(t, "CA test output", *reports[0].Content)

	// Now, start the review for the second daemon. It should result in the
	// cascaded review as well.
	wg.Add(2)
	ok = dispatcher.BeginReview(daemons[1], Triggers{ConfigModified}, func(daemonID int64, err error) {
		defer wg.Done()
	})
	require.True(t, ok)

	wg.Wait()

	reports, total, err = dbmodel.GetConfigReportsByDaemonID(db, 0, 0, daemons[0].ID, false)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, reports, 1)
	require.Equal(t, "DHCPv4 test output", *reports[0].Content)

	reports, total, err = dbmodel.GetConfigReportsByDaemonID(db, 0, 0, daemons[1].ID, false)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, reports, 1)
	require.Equal(t, "CA test output", *reports[0].Content)
}

// Test that the dispatcher accepts different trigger types and schedules
// the reviews depending on whether appropriate config checkers have been
// registered.
func TestTriggers(t *testing.T) {
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

	dispatcher := NewDispatcher(db).(*dispatcherImpl)
	require.NotNil(t, dispatcher)
	dispatcher.Start()
	defer dispatcher.Shutdown()

	// Register two test checkers setting the two boolean values declared
	// below to true.
	var dhcp4CheckComplete, dhcp6CheckComplete atomic.Value
	dhcp4CheckComplete.Store(false)
	dhcp6CheckComplete.Store(false)
	dispatcher.RegisterChecker(KeaDHCPv4Daemon, "dhcp4_test_checker", GetDefaultTriggers(), func(ctx *ReviewContext) (*Report, error) {
		dhcp4CheckComplete.Store(true)
		return nil, nil
	})
	dispatcher.RegisterChecker(KeaDHCPv6Daemon, "dhcp6_test_checker", Triggers{ManualRun}, func(ctx *ReviewContext) (*Report, error) {
		dhcp6CheckComplete.Store(true)
		return nil, nil
	})

	// Scheduling the review using the internalRun trigger is not allowed.
	ok := dispatcher.BeginReview(daemons[0], Triggers{internalRun}, nil)
	require.False(t, ok)

	// Schedule a review for the first daemon using the trigger associated
	// with the first checker. Wait until the review is completed.
	dhcp4CheckComplete.Store(false)
	ok = dispatcher.BeginReview(daemons[0], Triggers{ConfigModified}, nil)
	require.True(t, ok)
	require.Eventually(t, func() bool {
		return dhcp4CheckComplete.Load().(bool) && !dispatcher.ReviewInProgress(daemons[0].ID)
	}, 5*time.Second, 100*time.Millisecond)

	// A review using the same trigger for the second daemon should not
	// be launched because the checker has been registered for the
	// ManualRun only.
	ok = dispatcher.BeginReview(daemons[1], Triggers{ConfigModified}, nil)
	require.False(t, ok)

	// Finally, try the review for the second daemon using the ManualRun
	// trigger. It should be launched. Wait for the review to complete.
	dhcp6CheckComplete.Store(false)
	ok = dispatcher.BeginReview(daemons[1], Triggers{ManualRun}, nil)
	require.True(t, ok)
	require.Eventually(t, func() bool {
		return dhcp6CheckComplete.Load().(bool) && !dispatcher.ReviewInProgress(daemons[1].ID)
	}, 5*time.Second, 100*time.Millisecond)

	// The review shouldn't start if the multiple unrelated triggers are provided.
	ok = dispatcher.BeginReview(daemons[1], Triggers{DBHostsModified, StorkAgentConfigModified}, nil)
	require.False(t, ok)

	// The review should be launched if at least one of the provided triggers
	// corresponds to the registered checker.
	dhcp6CheckComplete.Store(false)
	ok = dispatcher.BeginReview(daemons[1], Triggers{DBHostsModified, StorkAgentConfigModified, ManualRun}, nil)
	require.True(t, ok)
	require.Eventually(t, func() bool {
		return dhcp6CheckComplete.Load().(bool) && !dispatcher.ReviewInProgress(daemons[1].ID)
	}, 5*time.Second, 100*time.Millisecond)
}

// Tests that default checkers are registered.
func TestRegisterDefaultCheckers(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	dispatcher := NewDispatcher(db).(*dispatcherImpl)
	require.NotNil(t, dispatcher)

	RegisterDefaultCheckers(dispatcher)

	// KeaDHCPDaemon group.
	require.Contains(t, dispatcher.groups, KeaDHCPDaemon)
	checkerNames := []string{}
	for _, p := range dispatcher.groups[KeaDHCPDaemon].checkers {
		checkerNames = append(checkerNames, p.name)
	}
	require.Contains(t, checkerNames, "stat_cmds_presence")
	require.Contains(t, checkerNames, "lease_cmds_presence")
	require.Contains(t, checkerNames, "host_cmds_presence")
	require.Contains(t, checkerNames, "dispensable_shared_network")
	require.Contains(t, checkerNames, "dispensable_subnet")
	require.Contains(t, checkerNames, "out_of_pool_reservation")
	require.Contains(t, checkerNames, "ha_mt_presence")
	require.Contains(t, checkerNames, "ha_dedicated_ports")
	require.Contains(t, checkerNames, "address_pools_exhausted_by_reservations")
	require.Contains(t, checkerNames, "pd_pools_exhausted_by_reservations")
	require.Contains(t, checkerNames, "overlapping_subnet")
	require.Contains(t, checkerNames, "canonical_prefix")
	require.Contains(t, checkerNames, "subnet_cmds_and_cb_mutual_exclusion")
	require.Contains(t, checkerNames, "statistics_unavailable_due_to_number_overflow")

	checkerNames = []string{}
	for _, p := range dispatcher.groups[KeaCADaemon].checkers {
		checkerNames = append(checkerNames, p.name)
	}

	require.Contains(t, checkerNames, "agent_credentials_over_https")
	require.Contains(t, checkerNames, "ca_control_sockets")

	// Ensure that the appropriate triggers were registered for the
	// default checkers.
	require.Contains(t, dispatcher.groups[KeaDHCPDaemon].triggerRefCounts, ManualRun)
	require.Contains(t, dispatcher.groups[KeaDHCPDaemon].triggerRefCounts, ConfigModified)
	require.Contains(t, dispatcher.groups[KeaDHCPDaemon].triggerRefCounts, DBHostsModified)

	require.EqualValues(t, 14, dispatcher.groups[KeaDHCPDaemon].triggerRefCounts[ManualRun])
	require.EqualValues(t, 14, dispatcher.groups[KeaDHCPDaemon].triggerRefCounts[ConfigModified])
	require.EqualValues(t, 4, dispatcher.groups[KeaDHCPDaemon].triggerRefCounts[DBHostsModified])
	require.EqualValues(t, 0, dispatcher.groups[KeaDHCPDaemon].triggerRefCounts[StorkAgentConfigModified])
	require.EqualValues(t, 2, dispatcher.groups[KeaCADaemon].triggerRefCounts[ManualRun])
	require.EqualValues(t, 2, dispatcher.groups[KeaCADaemon].triggerRefCounts[ConfigModified])
	require.EqualValues(t, 0, dispatcher.groups[KeaCADaemon].triggerRefCounts[DBHostsModified])
	require.EqualValues(t, 0, dispatcher.groups[KeaCADaemon].triggerRefCounts[StorkAgentConfigModified])
}

// Verifies that registering new checkers and bumping up the
// enforceDispatchSeq affects the returned signature.
func TestGetSignature(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	dispatcher := NewDispatcher(db).(*dispatcherImpl)
	require.NotNil(t, dispatcher)

	signatures := make([]string, 9)

	// No checkers registered yet. The signature should be returned anyway.
	signatures[0] = dispatcher.GetSignature()

	// Register checkers and record the signatures.
	dispatcher.RegisterChecker(EachDaemon, "checker1", GetDefaultTriggers(), nil)
	signatures[1] = dispatcher.GetSignature()
	require.NotEqual(t, signatures[0], signatures[1])

	dispatcher.RegisterChecker(EachDaemon, "checker2", GetDefaultTriggers(), nil)
	signatures[2] = dispatcher.GetSignature()
	require.NotEqual(t, signatures[0], signatures[2])
	require.NotEqual(t, signatures[1], signatures[2])

	dispatcher.RegisterChecker(KeaDHCPDaemon, "checker3", GetDefaultTriggers(), nil)
	signatures[3] = dispatcher.GetSignature()
	require.NotEqual(t, signatures[0], signatures[3])
	require.NotEqual(t, signatures[1], signatures[3])
	require.NotEqual(t, signatures[2], signatures[3])

	// Unregister the last checker. The signature should be now
	// equal to the signature from before registering the
	// checker3.
	require.True(t, dispatcher.UnregisterChecker(KeaDHCPDaemon, "checker3"))
	signatures[4] = dispatcher.GetSignature()
	require.Equal(t, signatures[2], signatures[4])

	// Register this checker but for a different dispatch group.
	// The new signature should be different than previously.
	dispatcher.RegisterChecker(KeaDHCPv4Daemon, "checker3", GetDefaultTriggers(), nil)
	signatures[5] = dispatcher.GetSignature()
	require.NotEqual(t, signatures[0], signatures[5])
	require.NotEqual(t, signatures[1], signatures[5])
	require.NotEqual(t, signatures[2], signatures[5])
	require.NotEqual(t, signatures[3], signatures[5])
	require.NotEqual(t, signatures[4], signatures[5])

	// Unregister the checker2.
	require.True(t, dispatcher.UnregisterChecker(EachDaemon, "checker2"))
	signatures[6] = dispatcher.GetSignature()

	// Re-register it. Make sure that the signature is affected
	// and that it is equal to the signature from before
	// unregistering the checker2.
	dispatcher.RegisterChecker(EachDaemon, "checker2", GetDefaultTriggers(), nil)
	signatures[7] = dispatcher.GetSignature()
	require.Equal(t, signatures[5], signatures[7])
	require.NotEqual(t, signatures[6], signatures[7])

	// Ensure that the hasher affects the signature.
	dispatcher.hasher = &testHasher{}
	signatures[8] = dispatcher.GetSignature()
	require.Equal(t, "test", signatures[8])
}

// Test that the dispatch group selector is serialized to string properly.
func TestDispatchGroupSelectorToString(t *testing.T) {
	require.EqualValues(t, "each-daemon", EachDaemon.String())
	require.EqualValues(t, "kea-dhcp-daemon", KeaDHCPDaemon.String())
	require.EqualValues(t, "kea-ca-daemon", KeaCADaemon.String())
	require.EqualValues(t, "kea-dhcp-v4-daemon", KeaDHCPv4Daemon.String())
	require.EqualValues(t, "kea-dhcp-v6-daemon", KeaDHCPv6Daemon.String())
	require.EqualValues(t, "kea-d2-daemon", KeaD2Daemon.String())
	require.EqualValues(t, "bind9-daemon", Bind9Daemon.String())
	require.EqualValues(t, "unknown", DispatchGroupSelector(42).String())
}

// Test that the config checkers metadata are returned properly..
func TestGetCheckersMetadata(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemon1 := &dbmodel.Daemon{ID: 1, Name: dbmodel.DaemonNameDHCPv4}
	daemon2 := &dbmodel.Daemon{ID: 2, Name: dbmodel.DaemonNameBind9}
	daemon3 := &dbmodel.Daemon{ID: 3, Name: "unknown"}
	dispatcher := NewDispatcher(db)
	dispatcher.RegisterChecker(KeaDHCPDaemon, "foo", Triggers{ManualRun, ConfigModified}, nil)
	dispatcher.RegisterChecker(KeaDHCPDaemon, "bar", Triggers{ManualRun, DBHostsModified}, nil)
	dispatcher.RegisterChecker(KeaDHCPDaemon, "baz", Triggers{ConfigModified, DBHostsModified}, nil)
	dispatcher.RegisterChecker(Bind9Daemon, "boz", Triggers{ManualRun}, nil)
	dispatcher.SetCheckerState(daemon1, "bar", CheckerStateDisabled)
	dispatcher.SetCheckerState(nil, "baz", CheckerStateDisabled)

	// Act
	metadataKea, errKea := dispatcher.GetCheckersMetadata(daemon1)
	metadataBind9, errBind9 := dispatcher.GetCheckersMetadata(daemon2)
	metadataGlobal, errGlobal := dispatcher.GetCheckersMetadata(nil)
	metadataUnknown, errUnknown := dispatcher.GetCheckersMetadata(daemon3)

	// Assert
	require.Len(t, metadataKea, 3)
	require.NoError(t, errKea)

	require.EqualValues(t, "bar", metadataKea[0].Name)
	require.True(t, metadataKea[0].GloballyEnabled)
	require.Contains(t, metadataKea[0].Selectors, KeaDHCPDaemon)
	require.EqualValues(t, CheckerStateDisabled, metadataKea[0].State)
	require.Contains(t, metadataKea[0].Triggers, ManualRun)
	require.Contains(t, metadataKea[0].Triggers, DBHostsModified)

	require.EqualValues(t, "baz", metadataKea[1].Name)
	require.False(t, metadataKea[1].GloballyEnabled)
	require.Contains(t, metadataKea[1].Selectors, KeaDHCPDaemon)
	require.EqualValues(t, CheckerStateInherit, metadataKea[1].State)
	require.Contains(t, metadataKea[1].Triggers, ConfigModified)
	require.Contains(t, metadataKea[1].Triggers, DBHostsModified)

	require.EqualValues(t, "foo", metadataKea[2].Name)
	require.True(t, metadataKea[2].GloballyEnabled)
	require.Contains(t, metadataKea[2].Selectors, KeaDHCPDaemon)
	require.EqualValues(t, CheckerStateInherit, metadataKea[2].State)
	require.Contains(t, metadataKea[2].Triggers, ManualRun)
	require.Contains(t, metadataKea[2].Triggers, ConfigModified)

	require.Len(t, metadataBind9, 1)
	require.NoError(t, errBind9)

	require.EqualValues(t, "boz", metadataBind9[0].Name)
	require.True(t, metadataBind9[0].GloballyEnabled)
	require.Contains(t, metadataBind9[0].Selectors, Bind9Daemon)
	require.EqualValues(t, CheckerStateInherit, metadataBind9[0].State)
	require.Contains(t, metadataBind9[0].Triggers, ManualRun)

	require.Len(t, metadataGlobal, 4)
	require.NoError(t, errGlobal)

	require.EqualValues(t, "bar", metadataGlobal[0].Name)
	require.EqualValues(t, CheckerStateEnabled, metadataGlobal[0].State)
	require.EqualValues(t, "baz", metadataGlobal[1].Name)
	require.EqualValues(t, CheckerStateDisabled, metadataGlobal[1].State)
	require.EqualValues(t, "boz", metadataGlobal[2].Name)
	require.EqualValues(t, CheckerStateEnabled, metadataGlobal[2].State)
	require.EqualValues(t, "foo", metadataGlobal[3].Name)
	require.EqualValues(t, CheckerStateEnabled, metadataGlobal[3].State)

	require.Error(t, errUnknown)
	require.Nil(t, metadataUnknown)
}

// Test that the checker state are loaded and validated properly.
func TestLoadAndValidateCheckerState(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	_ = dbmodel.AddMachine(db, machine)
	app := &dbmodel.App{
		Type:      dbmodel.AppTypeKea,
		MachineID: machine.ID,
		Daemons: []*dbmodel.Daemon{
			dbmodel.NewKeaDaemon("dhcp4", true),
		},
	}
	daemons, _ := dbmodel.AddApp(db, app)
	daemon := daemons[0]

	dispatcher := NewDispatcher(db)
	dispatcher.RegisterChecker(KeaDHCPDaemon, "foo", Triggers{ManualRun, ConfigModified}, nil)
	dispatcher.RegisterChecker(KeaDHCPDaemon, "bar", Triggers{ManualRun, DBHostsModified}, nil)
	dispatcher.RegisterChecker(KeaDHCPDaemon, "baz", Triggers{ManualRun}, nil)

	_ = dbmodel.CommitCheckerPreferences(db, []*dbmodel.ConfigCheckerPreference{
		dbmodel.NewGlobalConfigCheckerPreference("foo"),
		// Unknown global config checker.
		dbmodel.NewGlobalConfigCheckerPreference("ofo"),
		// Override the global preference.
		dbmodel.NewDaemonConfigCheckerPreference(daemon.ID, "foo", true),
		dbmodel.NewDaemonConfigCheckerPreference(daemon.ID, "bar", false),
		// Unknown daemon config checker.
		dbmodel.NewDaemonConfigCheckerPreference(daemon.ID, "oof", false),
	}, nil)

	// Act
	err := LoadAndValidateCheckerPreferences(db, dispatcher)

	// Assert
	require.NoError(t, err)
	checkers, _ := dispatcher.GetCheckersMetadata(daemon)
	require.Len(t, checkers, 3)

	require.EqualValues(t, "bar", checkers[0].Name)
	require.True(t, checkers[0].GloballyEnabled)
	require.EqualValues(t, CheckerStateDisabled, checkers[0].State)

	require.EqualValues(t, "baz", checkers[1].Name)
	require.True(t, checkers[1].GloballyEnabled)
	require.EqualValues(t, CheckerStateInherit, checkers[1].State)

	require.EqualValues(t, "foo", checkers[2].Name)
	require.False(t, checkers[2].GloballyEnabled)
	require.EqualValues(t, CheckerStateEnabled, checkers[2].State)
}

// Test that the review don't run if all config checkers were disabled for a given
// daemon.
func TestBeginReviewForDaemonWithAllCheckersDisabled(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	_ = dbmodel.AddMachine(db, machine)
	app := &dbmodel.App{
		Type:      dbmodel.AppTypeKea,
		MachineID: machine.ID,
		Daemons: []*dbmodel.Daemon{
			dbmodel.NewKeaDaemon("dhcp4", true),
		},
	}
	daemons, _ := dbmodel.AddApp(db, app)
	daemon := daemons[0]

	dispatcher := NewDispatcher(db)
	dispatcher.RegisterChecker(KeaDHCPDaemon, "foo", Triggers{ManualRun, ConfigModified}, func(rc *ReviewContext) (*Report, error) {
		require.Fail(t, "checker function shouldn't be called")
		return nil, nil
	})
	dispatcher.RegisterChecker(KeaDHCPDaemon, "bar", Triggers{ManualRun, DBHostsModified}, func(rc *ReviewContext) (*Report, error) {
		require.Fail(t, "checker function shouldn't be called")
		return nil, nil
	})
	dispatcher.SetCheckerState(nil, "foo", CheckerStateDisabled)
	dispatcher.SetCheckerState(daemon, "bar", CheckerStateDisabled)

	// Act
	ok := dispatcher.BeginReview(daemon, Triggers{ManualRun}, func(i int64, err error) {
		require.Fail(t, "callback shouldn't be called")
	})

	// Assert
	require.False(t, ok)
}

// Test that the review doesn't execute the disabled config checkers.
func TestBeginReviewForDaemonWithSomeCheckersDisabled(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	_ = dbmodel.AddMachine(db, machine)
	app := &dbmodel.App{
		Type:      dbmodel.AppTypeKea,
		MachineID: machine.ID,
		Daemons: []*dbmodel.Daemon{
			dbmodel.NewKeaDaemon("dhcp4", true),
		},
	}
	daemons, _ := dbmodel.AddApp(db, app)
	daemon := daemons[0]
	checkerCallCount := 0

	dispatcher := NewDispatcher(db)
	dispatcher.RegisterChecker(KeaDHCPDaemon, "foo", Triggers{ManualRun, ConfigModified}, func(rc *ReviewContext) (*Report, error) {
		require.Fail(t, "checker function shouldn't be called")
		return nil, nil
	})
	dispatcher.RegisterChecker(KeaDHCPDaemon, "bar", Triggers{ManualRun, DBHostsModified}, func(rc *ReviewContext) (*Report, error) {
		checkerCallCount++
		return nil, nil
	})

	dispatcher.SetCheckerState(nil, "foo", CheckerStateDisabled)
	dispatcher.Start()

	var wg sync.WaitGroup

	// Act
	wg.Add(1)
	ok := dispatcher.BeginReview(daemon, Triggers{ManualRun}, func(i int64, err error) {
		wg.Done()
	})
	wg.Wait()

	// Assert
	require.True(t, ok)
	require.EqualValues(t, 1, checkerCallCount)
}

// Test that the checker state is verified before changing.
func TestSetCheckerStateToInvalidValue(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemon := &dbmodel.Daemon{ID: 1, Name: dbmodel.DaemonNameDHCPv4}
	dispatcher := NewDispatcher(db)
	dispatcher.RegisterChecker(KeaDHCPDaemon, "foo", Triggers{ManualRun, ConfigModified}, nil)

	// Act
	err1 := dispatcher.SetCheckerState(daemon, "bar", CheckerStateDisabled)
	err2 := dispatcher.SetCheckerState(nil, "foo", CheckerStateInherit)

	// Assert
	require.Error(t, err1)
	require.Error(t, err2)
}

// Test that the reports count is returned properly.
func TestReviewContextCounters(t *testing.T) {
	// Arrange
	ctx := newReviewContext(nil, &dbmodel.Daemon{ID: 42}, Triggers{ConfigModified}, nil)

	report1, _ := NewReport(ctx, "foo").create()
	report2, _ := NewReport(ctx, "bar").create()
	report3, _ := NewReport(ctx, "baz").create()
	report4, _ := newEmptyReport(ctx)
	report5, _ := newEmptyReport(ctx)

	ctx.reports = []taggedReport{
		{
			checkerName: "foo",
			report:      report1,
		},
		{
			checkerName: "bar",
			report:      report2,
		},
		{
			checkerName: "baz",
			report:      report3,
		},
		{
			checkerName: "boz",
			report:      report4,
		},
		{
			checkerName: "biz",
			report:      report5,
		},
	}

	// Act & Assert
	require.EqualValues(t, 5, ctx.getReportsCount())
	require.EqualValues(t, 3, ctx.getIssuesCount())
}

// Test that the internal run is recognized properly.
func TestTriggersIsInternalRun(t *testing.T) {
	t.Run("only internalRun trigger", func(t *testing.T) {
		require.True(t, Triggers{internalRun}.isInternalRun())
	})

	t.Run("missing internalRun trigger", func(t *testing.T) {
		require.False(t, Triggers{ManualRun, ConfigModified}.isInternalRun())
	})

	t.Run("internalRun combined with another trigger", func(t *testing.T) {
		require.False(t, Triggers{internalRun, ConfigModified}.isInternalRun())
	})

	t.Run("empty triggers", func(t *testing.T) {
		require.False(t, Triggers{}.isInternalRun())
	})
}
