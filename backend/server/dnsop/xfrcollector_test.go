package dnsop

import (
	context "context"
	iter "iter"
	"math"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	bind9xfr "isc.org/stork/daemondata/bind9xfr"
	agentcomm "isc.org/stork/server/agentcomm"
	daemonstest "isc.org/stork/server/daemons/test"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/testutil"
)

// Test that the collect function of the xfrCollector receives the zone transfer
// states from the agent and inserts them into the database.
func TestXFRCollector(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &dbmodel.Machine{
		Address:   "127.0.0.1",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	daemon := &dbmodel.Daemon{
		MachineID: machine.ID,
		AccessPoints: []*dbmodel.AccessPoint{
			{
				Type:    dbmodel.AccessPointControl,
				Address: "localhost",
				Port:    5300,
			},
		},
	}
	err = dbmodel.AddDaemon(db, daemon)
	require.NoError(t, err)

	controller := gomock.NewController(t)
	defer controller.Finish()

	// Generate some test zone transfers to be returned over the stream.
	testXFRs := testutil.GetTestZoneTransfers()

	agents := NewMockConnectedAgents(controller)
	agents.EXPECT().ReceiveZoneTransfers(gomock.Any(), gomock.Cond(func(d any) bool {
		return d.(*dbmodel.Daemon).ID == daemon.ID
	}), true).DoAndReturn(func(context.Context, *dbmodel.Daemon, bool) iter.Seq2[*bind9xfr.State, error] {
		return func(yield func(*bind9xfr.State, error) bool) {
			for _, xfr := range testXFRs {
				if !yield(xfr, nil) {
					return
				}
			}
		}
	})

	// Create the collector instance.
	xfrCollector := newXFRCollector(daemonstest.ManagerAccessorsWrapper{
		DB:     db,
		Agents: agents,
	}, daemon)

	// Collect the zone transfer states from the agent.
	xfrCollector.collect(context.Background())

	// Validate the inserted zone transfer states.
	xfrs, _, err := dbmodel.GetZoneTransferStatesByPage(db, 0, 100)
	require.NoError(t, err)
	require.Len(t, xfrs, len(testXFRs))

	for _, xfr := range xfrs {
		for _, testXFR := range testXFRs {
			if xfr.ViewName == testXFR.ViewName && xfr.ZoneName == testXFR.ZoneName {
				require.Equal(t, testXFR.Serial, xfr.Serial)
				require.Equal(t, testXFR.Client, xfr.Client)
				require.Equal(t, testXFR.Server, xfr.Server)
				require.Equal(t, testXFR.MessagesCount, xfr.MessagesCount)
				require.Equal(t, testXFR.RecordsCount, xfr.RecordsCount)
				require.Equal(t, testXFR.BytesCount, xfr.BytesCount)
				require.Equal(t, testXFR.Duration, xfr.Duration)
				require.Equal(t, testXFR.Status, xfr.Status)
				require.Equal(t, testXFR.StartTime, xfr.StartTime)
				require.Equal(t, testXFR.CompletionTime, xfr.CompletionTime)
				require.Equal(t, testXFR.Message, xfr.Message)
				break
			}
		}
	}
}

// Test starting and stopping the xfrCollector in a goroutine during a reconnect.
func TestXFRCollectorStartStopDuringReconnect(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &dbmodel.Machine{
		Address:   "127.0.0.1",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	daemon := &dbmodel.Daemon{
		MachineID: machine.ID,
		AccessPoints: []*dbmodel.AccessPoint{
			{
				Type:    dbmodel.AccessPointControl,
				Address: "localhost",
				Port:    5300,
			},
		},
	}
	err = dbmodel.AddDaemon(db, daemon)
	require.NoError(t, err)

	synctest.Test(t, func(t *testing.T) {
		controller := gomock.NewController(t)
		defer controller.Finish()

		agents := NewMockConnectedAgents(controller)

		// To simulate stopping the collector, let's return an error from the agent, so the
		// collector enters the reconnect loop. When we stop the collector during the reconnect,
		// it should hit the context cancellation handling code, and return.
		agents.EXPECT().ReceiveZoneTransfers(gomock.Any(), gomock.Any(), true).
			Return(func(yield func(*bind9xfr.State, error) bool) {
				_ = yield(nil, &testError{})
			})

		// Create the collector instance.
		xfrCollector := newXFRCollector(daemonstest.ManagerAccessorsWrapper{
			DB:     db,
			Agents: agents,
		}, daemon)
		// The collector should be initially inactive.
		require.False(t, xfrCollector.isActive())
		// Start the collector.
		xfrCollector.start()
		// Wait for the collector to start.
		synctest.Wait()
		// The collector should be now active.
		require.True(t, xfrCollector.isActive())
		// Make sure that another start does not hurt.
		require.NotPanics(t, func() {
			xfrCollector.start()
		})
		// Stop the collector.
		xfrCollector.stop()
		// The collector should be inactive again.
		require.False(t, xfrCollector.isActive())
		// Make sure that another stop does not hurt.
		require.NotPanics(t, func() {
			xfrCollector.stop()
		})
	})
}

// Test the reconnect logic (backoff) of the xfrCollector. This test cannot use
// the synctest package because it would make it impossible to measure the time
// between the reconnect attempts - synctest uses fake time.
func TestXFRCollectorReconnect(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &dbmodel.Machine{
		Address:   "127.0.0.1",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	daemon := &dbmodel.Daemon{
		MachineID: machine.ID,
		AccessPoints: []*dbmodel.AccessPoint{
			{
				Type:    dbmodel.AccessPointControl,
				Address: "localhost",
				Port:    5300,
			},
		},
	}
	err = dbmodel.AddDaemon(db, daemon)
	require.NoError(t, err)

	controller := gomock.NewController(t)
	defer controller.Finish()

	agents := NewMockConnectedAgents(controller)

	var (
		ts    []time.Time
		mutex sync.Mutex
	)
	for i := 0; i < 7; i++ {
		// Simulate an error from the agent, so the collector enters the reconnect loop.
		// Record the timestamps of the reconnect attempts, so we can  ensure
		// that the correct intervals are used.
		agents.EXPECT().ReceiveZoneTransfers(gomock.Any(), gomock.Any(), true).
			Return(func(yield func(*bind9xfr.State, error) bool) {
				mutex.Lock()
				ts = append(ts, time.Now())
				mutex.Unlock()
				_ = yield(nil, &testError{})
			})
	}

	// Last attempt should not return an error to cause the goroutine to exit.
	waitChan := make(chan struct{})
	agents.EXPECT().ReceiveZoneTransfers(gomock.Any(), gomock.Any(), true).
		Return(func(yield func(*bind9xfr.State, error) bool) {
			close(waitChan)
		})

	// Create the collector instance.
	xfrCollector := newXFRCollector(daemonstest.ManagerAccessorsWrapper{
		DB:     db,
		Agents: agents,
	}, daemon)

	// Change the backoff factor to make sure the test runs faster.
	xfrCollector.backoffFactor = time.Millisecond * 1

	// The collector should be initially inactive.
	require.False(t, xfrCollector.isActive())
	// Start the collector and make sure it is active.
	xfrCollector.start()
	require.Eventually(t, xfrCollector.isActive, 5*time.Second, 100*time.Millisecond)
	// Make sure that the collector has made 7 attempts to reconnect. The 8-th attempt
	// should return with no error.
	require.Eventually(t, func() bool {
		mutex.Lock()
		defer mutex.Unlock()
		return len(ts) == 7
	}, 30*time.Second, 100*time.Millisecond)
	// Make sure that the collector ends gracefully.
	<-waitChan

	// Suppose the backoff factor is 1s. The durations between the consecutive attempts
	// should be: 1s, 2s, 4s, 8s, 16s, 30s, 30s. That's because the maximum duration
	// is 30 times the backoff factor.

	// Let's test the ones growing.
	for i := 1; i < 6; i++ {
		sub := ts[i].Sub(ts[i-1])
		require.GreaterOrEqual(t, sub, xfrCollector.backoffFactor*time.Duration(math.Pow(2, float64(i-1))))
	}

	// Let's now test the last two that should stabilize at the maximum duration.
	// However, we expect that the actual duration may be slightly longer because
	// of the time the code needs to execute the mock. Therefore, we merely test
	// that the duration is now shorter than the one calculated using the exponential
	// formula above.
	for i := 6; i < 7; i++ {
		sub := ts[i].Sub(ts[i-1])
		require.Less(t, sub, xfrCollector.backoffFactor*time.Duration(math.Pow(2, float64(i-1))))
	}
}

// Test that the XFR collector stops monitoring a daemon that doesn't exist in the database.
func TestXFRCollectorNonExistingDaemon(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Create a daemon but do not add it to the database.
	daemon := &dbmodel.Daemon{
		ID: 1,
		AccessPoints: []*dbmodel.AccessPoint{
			{
				Type:    dbmodel.AccessPointControl,
				Address: "localhost",
				Port:    5300,
			},
		},
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

	// Generate some test zone transfers to be returned over the stream.
	testXFRs := testutil.GetTestZoneTransfers()

	agents := NewMockConnectedAgents(controller)

	// Expect the collector to receive only one zone transfer state. After receiving
	// this state and trying to insert it to the database, it should stop monitoring
	// the daemon because the daemon does not exist in the database. yieldCount counts
	// how many times the collector received a zone transfer state. It should be 1.
	var yieldCount int
	agents.EXPECT().ReceiveZoneTransfers(gomock.Any(), gomock.Any(), true).DoAndReturn(func(context.Context, *dbmodel.Daemon, bool) iter.Seq2[*bind9xfr.State, error] {
		return func(yield func(*bind9xfr.State, error) bool) {
			for _, xfr := range testXFRs {
				yieldCount++
				if !yield(xfr, nil) {
					return
				}
			}
		}
	})

	// Create the collector instance.
	xfrCollector := newXFRCollector(daemonstest.ManagerAccessorsWrapper{
		DB:     db,
		Agents: agents,
	}, daemon)

	// Collect the zone transfer states from the agent.
	xfrCollector.collect(t.Context())

	// Make sure that the collector was stopped after receiving the first zone transfer state.
	require.Equal(t, 1, yieldCount)

	// Make sure that nothing was inserted into the database.
	xfrs, _, err := dbmodel.GetZoneTransferStatesByPage(db, 0, 100)
	require.NoError(t, err)
	require.Empty(t, xfrs)
}

// Test that the XFR collector stops monitoring a daemon for which the
// zone transfer trackingis disabled.
func TestXFRCollectorZoneTransferTrackingDisabledOnAgent(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Create a daemon but do not add it to the database.
	daemon := &dbmodel.Daemon{
		ID: 1,
		AccessPoints: []*dbmodel.AccessPoint{
			{
				Type:    dbmodel.AccessPointControl,
				Address: "localhost",
				Port:    5300,
			},
		},
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

	agents := NewMockConnectedAgents(controller)

	// Expect the collector to return after receiving the error indicating that
	// the zone transfer tracking is disabled on the agent. Any other error would
	// cause the collector to continue trying to collect the zone transfer states.
	// It would result in a single ReceiveZoneTransfers call expectation failure.
	agents.EXPECT().ReceiveZoneTransfers(gomock.Any(), gomock.Any(), true).DoAndReturn(func(context.Context, *dbmodel.Daemon, bool) iter.Seq2[*bind9xfr.State, error] {
		return func(yield func(*bind9xfr.State, error) bool) {
			_ = yield(nil, agentcomm.NewZoneTransferTrackingDisabledOnAgentError("localhost:5300"))
		}
	})

	// Create the collector instance.
	xfrCollector := newXFRCollector(daemonstest.ManagerAccessorsWrapper{
		DB:     db,
		Agents: agents,
	}, daemon)

	// Collect the zone transfer states from the agent.
	xfrCollector.collect(t.Context())

	// Make sure that nothing was inserted into the database.
	xfrs, _, err := dbmodel.GetZoneTransferStatesByPage(db, 0, 100)
	require.NoError(t, err)
	require.Empty(t, xfrs)
}
