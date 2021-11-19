package configreview

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
)

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
	reports := &[]*Report{}
	mutex := &sync.Mutex{}

	// Register different checkers for different daemon types.
	for i := 0; i < len(selectors); i++ {
		continueChan := make(chan bool)
		channels[i] = continueChan
		dispatcher.RegisterChecker(selectors[i], "test_checker", func(ctx *ReviewContext) (*Report, error) {
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
		ok := dispatcher.BeginReview(daemon, nil)
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
	dispatcher.RegisterChecker(KeaDHCPv4Daemon, "dhcp4_test_checker", func(ctx *ReviewContext) (*Report, error) {
		report, err := NewReport(ctx, "DHCPv4 test output").create()
		return report, err
	})

	dispatcher.RegisterChecker(KeaDHCPv6Daemon, "dhcp6_test_checker", func(ctx *ReviewContext) (*Report, error) {
		report, err := NewReport(ctx, "DHCPv6 test output").create()
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
	reports, total, err := dbmodel.GetConfigReportsByDaemonID(db, 0, 0, daemons[0].ID)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, reports, 1)
	require.Equal(t, "dhcp4_test_checker", reports[0].CheckerName)
	require.Equal(t, "DHCPv4 test output", reports[0].Content)

	review, err := dbmodel.GetConfigReviewByDaemonID(db, daemons[0].ID)
	require.NoError(t, err)
	require.NotNil(t, review)
	require.WithinDuration(t, time.Now(), review.CreatedAt, 5*time.Second)
	require.NotEmpty(t, review.ConfigHash)
	require.NotEmpty(t, review.Signature)

	// Ensure that the reports for the second daemon have not been inserted.
	reports, total, err = dbmodel.GetConfigReportsByDaemonID(db, 0, 0, daemons[1].ID)
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

	// Register a test checker for the BIND9 daemon.
	dispatcher.RegisterChecker(Bind9Daemon, "test_checker", func(ctx *ReviewContext) (*Report, error) {
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
	reports, total, err := dbmodel.GetConfigReportsByDaemonID(db, 0, 0, daemons[0].ID)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, reports, 1)
	require.Equal(t, "test_checker", reports[0].CheckerName)
	require.Equal(t, "Bind9 test output", reports[0].Content)
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
	dispatcher := NewDispatcher(db).(*dispatcherImpl)
	require.NotNil(t, dispatcher)

	// Register the checker which blocks until it receives a value
	// over the continueChan or when doneCtx is cancelled.
	continueChan := make(chan bool)
	doneCtx, cancel := context.WithCancel(context.Background())
	dispatcher.RegisterChecker(Bind9Daemon, "test_checker", func(ctx *ReviewContext) (*Report, error) {
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
	ok := dispatcher.BeginReview(daemons[0], func(daemonID int64, err error) {
		defer wg.Done()
	})
	require.True(t, ok)

	// Ensure that the fact that the review is in progress has been recorded.
	state, ok := dispatcher.state[daemons[0].ID]
	require.True(t, ok)
	require.True(t, state)
	require.True(t, dispatcher.ReviewInProgress(daemons[0].ID))

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
	dispatcher.RegisterChecker(KeaDHCPv4Daemon, "dhcp4_test_checker", func(ctx *ReviewContext) (*Report, error) {
		ctx.refDaemons = append(ctx.refDaemons, daemons[1])
		report, err := NewReport(ctx, "DHCPv4 test output").
			referencingDaemon(ctx.refDaemons[0]).
			referencingDaemon(ctx.subjectDaemon).
			create()
		return report, err
	})

	// Register a checker for the second daemon. It fetches the configuration of
	// the other daemon besides the reviewed configuration.
	dispatcher.RegisterChecker(KeaCADaemon, "ca_test_checker", func(ctx *ReviewContext) (*Report, error) {
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
	ok := dispatcher.BeginReview(daemons[0], func(daemonID int64, err error) {
		defer wg.Done()
	})
	require.True(t, ok)

	// Wait until it completes.
	wg.Wait()

	reports, total, err := dbmodel.GetConfigReportsByDaemonID(db, 0, 0, daemons[0].ID)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, reports, 1)
	require.Equal(t, "dhcp4_test_checker", reports[0].CheckerName)
	require.Equal(t, "DHCPv4 test output", reports[0].Content)

	// The first daemon's checker references the second daemon. Therefore,
	// this review should cause the review of the second daemon's
	// configuration. Ensure that it has been performed.
	reports, total, err = dbmodel.GetConfigReportsByDaemonID(db, 0, 0, daemons[1].ID)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, reports, 1)
	require.Equal(t, "ca_test_checker", reports[0].CheckerName)
	require.Equal(t, "CA test output", reports[0].Content)

	// Now, start the review for the second daemon. It should result in the
	// cascaded review as well.
	wg.Add(2)
	ok = dispatcher.BeginReview(daemons[1], func(daemonID int64, err error) {
		defer wg.Done()
	})
	require.True(t, ok)

	wg.Wait()

	reports, total, err = dbmodel.GetConfigReportsByDaemonID(db, 0, 0, daemons[0].ID)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, reports, 1)
	require.Equal(t, "DHCPv4 test output", reports[0].Content)

	reports, total, err = dbmodel.GetConfigReportsByDaemonID(db, 0, 0, daemons[1].ID)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, reports, 1)
	require.Equal(t, "CA test output", reports[0].Content)
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
	require.Contains(t, checkerNames, "host_cmds_presence")
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
	dispatcher.RegisterChecker(EachDaemon, "checker1", nil)
	signatures[1] = dispatcher.GetSignature()
	require.NotEqual(t, signatures[0], signatures[1])

	dispatcher.RegisterChecker(EachDaemon, "checker2", nil)
	signatures[2] = dispatcher.GetSignature()
	require.NotEqual(t, signatures[0], signatures[2])
	require.NotEqual(t, signatures[1], signatures[2])

	dispatcher.RegisterChecker(KeaDHCPDaemon, "checker3", nil)
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
	dispatcher.RegisterChecker(KeaDHCPv4Daemon, "checker3", nil)
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
	dispatcher.RegisterChecker(EachDaemon, "checker2", nil)
	signatures[7] = dispatcher.GetSignature()
	require.Equal(t, signatures[5], signatures[7])
	require.NotEqual(t, signatures[6], signatures[7])

	// Ensure that bumping up the sequence number also affects
	// the signature.
	dispatcher.enforceSeq++
	signatures[8] = dispatcher.GetSignature()
	require.NotEqual(t, signatures[8], signatures[7])
}
