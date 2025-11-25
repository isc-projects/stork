package bind9

import (
	"testing"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	agentapi "isc.org/stork/api"
	"isc.org/stork/datamodel/daemonname"
	"isc.org/stork/datamodel/protocoltype"
	"isc.org/stork/server/agentcomm"
	agentcommtest "isc.org/stork/server/agentcomm/test"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	storktest "isc.org/stork/server/test/dbmodel"
)

// Check creating and shutting down StatsPuller.
func TestStatsPullerBasic(t *testing.T) {
	// prepare db
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// set one setting that is needed by puller
	setting := dbmodel.Setting{
		Name:    "bind9_stats_puller_interval",
		ValType: dbmodel.SettingValTypeInt,
		Value:   "60",
	}
	_, err := db.Model(&setting).Insert()
	require.NoError(t, err)

	// prepare fake agents and eventcenter
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}

	sp, _ := NewStatsPuller(db, fa, fec)
	sp.Shutdown()
}

// Check if pulling stats works.
func TestStatsPullerPullStats(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Set named stats response.
	response := NamedStatsGetResponse{
		Views: map[string]*ViewStatsData{
			"trusted": {
				Resolver: ResolverData{
					CacheStats: CacheStatsData{
						CacheHits:   60,
						CacheMisses: 40,
						QueryHits:   10,
						QueryMisses: 90,
					},
				},
			},
			"guest": {
				Resolver: ResolverData{
					CacheStats: CacheStatsData{
						CacheHits:   100,
						CacheMisses: 200,
						QueryHits:   56,
						QueryMisses: 75,
					},
				},
			},
			"_bind": {
				Resolver: ResolverData{
					CacheStats: CacheStatsData{
						CacheHits:   30,
						CacheMisses: 70,
						QueryHits:   20,
						QueryMisses: 80,
					},
				},
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnectedAgents := NewMockConnectedAgents(ctrl)
	mockConnectedAgents.EXPECT().
		ForwardToNamedStats(gomock.Any(), gomock.Any(), agentapi.ForwardToNamedStatsReq_SERVER, gomock.Any()).
		AnyTimes().
		SetArg(3, response).
		Return(nil)

	fec := &storktest.FakeEventCenter{}

	// prepare bind9 daemons
	var err error

	machine1 := &dbmodel.Machine{
		ID:        0,
		Address:   "192.0.1.0",
		AgentPort: 1111,
	}
	err = dbmodel.AddMachine(db, machine1)
	require.NoError(t, err)
	require.NotZero(t, machine1.ID)

	daemon1 := dbmodel.NewDaemon(machine1, daemonname.Bind9, true, []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "127.0.0.1",
			Port:     953,
			Key:      "abcd",
			Protocol: protocoltype.RNDC,
		},
		{
			Type:     dbmodel.AccessPointStatistics,
			Address:  "127.0.0.1",
			Port:     8000,
			Key:      "abcd",
			Protocol: protocoltype.HTTP,
		},
	})
	err = CommitDaemonIntoDB(db, daemon1, fec)
	require.NoError(t, err)

	machine2 := &dbmodel.Machine{
		ID:        0,
		Address:   "192.0.2.0",
		AgentPort: 2222,
	}
	err = dbmodel.AddMachine(db, machine2)
	require.NoError(t, err)
	require.NotZero(t, machine2.ID)

	daemon2 := dbmodel.NewDaemon(machine2, daemonname.Bind9, true, []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "127.0.0.1",
			Port:     953,
			Key:      "abcd",
			Protocol: protocoltype.RNDC,
		},
		{
			Type:     dbmodel.AccessPointStatistics,
			Address:  "127.0.0.1",
			Port:     8000,
			Key:      "abcd",
			Protocol: protocoltype.HTTP,
		},
	})
	err = CommitDaemonIntoDB(db, daemon2, fec)
	require.NoError(t, err)

	// set one setting that is needed by puller
	setting := dbmodel.Setting{
		Name:    "bind9_stats_puller_interval",
		ValType: dbmodel.SettingValTypeInt,
		Value:   "60",
	}
	_, err = db.Model(&setting).Insert()
	require.NoError(t, err)

	// prepare stats puller
	sp, err := NewStatsPuller(db, mockConnectedAgents, fec)
	require.NoError(t, err)
	// shutdown stats puller at the end
	defer sp.Shutdown()

	// invoke pulling stats
	err = sp.pullStats()
	require.NoError(t, err)

	// check collected stats
	daemon1Retrieved, err := dbmodel.GetDaemonByID(db, daemon1.ID)
	require.NoError(t, err)

	require.NotNil(t, daemon1Retrieved.Bind9Daemon)
	require.EqualValues(t, 60, daemon1Retrieved.Bind9Daemon.Stats.NamedStats.Views["trusted"].Resolver.CacheStats["CacheHits"])
	require.EqualValues(t, 40, daemon1Retrieved.Bind9Daemon.Stats.NamedStats.Views["trusted"].Resolver.CacheStats["CacheMisses"])
	require.EqualValues(t, 10, daemon1Retrieved.Bind9Daemon.Stats.NamedStats.Views["trusted"].Resolver.CacheStats["QueryHits"])
	require.EqualValues(t, 90, daemon1Retrieved.Bind9Daemon.Stats.NamedStats.Views["trusted"].Resolver.CacheStats["QueryMisses"])

	// Add assertions for "guest" view
	require.EqualValues(t, 100, daemon1Retrieved.Bind9Daemon.Stats.NamedStats.Views["guest"].Resolver.CacheStats["CacheHits"])
	require.EqualValues(t, 200, daemon1Retrieved.Bind9Daemon.Stats.NamedStats.Views["guest"].Resolver.CacheStats["CacheMisses"])
	require.EqualValues(t, 56, daemon1Retrieved.Bind9Daemon.Stats.NamedStats.Views["guest"].Resolver.CacheStats["QueryHits"])
	require.EqualValues(t, 75, daemon1Retrieved.Bind9Daemon.Stats.NamedStats.Views["guest"].Resolver.CacheStats["QueryMisses"])

	daemon2Retrieved, err := dbmodel.GetDaemonByID(db, daemon2.ID)
	require.NoError(t, err)
	require.NotNil(t, daemon2Retrieved.Bind9Daemon)
	require.EqualValues(t, 60, daemon2Retrieved.Bind9Daemon.Stats.NamedStats.Views["trusted"].Resolver.CacheStats["CacheHits"])
	require.EqualValues(t, 40, daemon2Retrieved.Bind9Daemon.Stats.NamedStats.Views["trusted"].Resolver.CacheStats["CacheMisses"])
	require.EqualValues(t, 10, daemon2Retrieved.Bind9Daemon.Stats.NamedStats.Views["trusted"].Resolver.CacheStats["QueryHits"])
	require.EqualValues(t, 90, daemon2Retrieved.Bind9Daemon.Stats.NamedStats.Views["trusted"].Resolver.CacheStats["QueryMisses"])

	// Add assertions for "guest" view
	require.EqualValues(t, 100, daemon2Retrieved.Bind9Daemon.Stats.NamedStats.Views["guest"].Resolver.CacheStats["CacheHits"])
	require.EqualValues(t, 200, daemon2Retrieved.Bind9Daemon.Stats.NamedStats.Views["guest"].Resolver.CacheStats["CacheMisses"])
	require.EqualValues(t, 56, daemon2Retrieved.Bind9Daemon.Stats.NamedStats.Views["guest"].Resolver.CacheStats["QueryHits"])
	require.EqualValues(t, 75, daemon2Retrieved.Bind9Daemon.Stats.NamedStats.Views["guest"].Resolver.CacheStats["QueryMisses"])

	require.NotContains(t, daemon2Retrieved.Bind9Daemon.Stats.NamedStats.Views, "_bind")
}

// Check if statistics-channel response is handled correctly when it is empty.
func TestStatsPullerEmptyResponse(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// prepare fake agents
	bind9Mock := func(callNo int, statsOutput interface{}) {
		json := `{
                    "json-stats-version":"1.2"
                }`

		agentcomm.UnmarshalNamedStatsResponse(json, statsOutput)
	}
	fa := agentcommtest.NewFakeAgents(nil, bind9Mock)
	fec := &storktest.FakeEventCenter{}

	// prepare bind9 daemon
	var err error

	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "192.0.1.0",
		AgentPort: 1111,
	}
	err = dbmodel.AddMachine(db, machine)
	require.NoError(t, err)
	require.NotZero(t, machine.ID)

	daemon := dbmodel.NewDaemon(machine, daemonname.Bind9, true, []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "127.0.0.1",
			Port:     953,
			Key:      "abcd",
			Protocol: protocoltype.RNDC,
		},
		{
			Type:     dbmodel.AccessPointStatistics,
			Address:  "127.0.0.1",
			Port:     8000,
			Key:      "abcd",
			Protocol: protocoltype.HTTP,
		},
	})
	err = CommitDaemonIntoDB(db, daemon, fec)
	require.NoError(t, err)

	// set one setting that is needed by puller
	setting := dbmodel.Setting{
		Name:    "bind9_stats_puller_interval",
		ValType: dbmodel.SettingValTypeInt,
		Value:   "60",
	}
	_, err = db.Model(&setting).Insert()
	require.NoError(t, err)

	// prepare stats puller
	sp, err := NewStatsPuller(db, fa, fec)
	require.NoError(t, err)
	// shutdown stats puller at the end
	defer sp.Shutdown()

	// invoke pulling stats
	err = sp.pullStats()
	require.NoError(t, err)

	// check collected stats
	daemonRetrieved, err := dbmodel.GetDaemonByID(db, daemon.ID)
	require.NoError(t, err)
	require.NotNil(t, daemonRetrieved.Bind9Daemon)
	require.Empty(t, daemonRetrieved.Bind9Daemon.Stats.NamedStats.Views)
}

// Test that the stats puller doesn't crash if the BIND 9 process has been
// detected but the communication with the named instance cannot be
// established. In this case, the daemon reference in the daemon is nil.
func TestStatsPullerPullStatsForPartiallyDetectedDaemon(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Prepare fake agents.
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}

	// Prepare bind9 daemon.
	var err error

	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "192.0.1.0",
		AgentPort: 1111,
	}
	err = dbmodel.AddMachine(db, machine)
	require.NoError(t, err)
	require.NotZero(t, machine.ID)

	daemon := dbmodel.NewDaemon(machine, daemonname.Bind9, false, []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "127.0.0.1",
			Port:     953,
			Key:      "abcd",
			Protocol: protocoltype.RNDC,
		},
		{
			Type:     dbmodel.AccessPointStatistics,
			Address:  "127.0.0.1",
			Port:     8000,
			Key:      "abcd",
			Protocol: protocoltype.HTTP,
		},
	})
	// Don't set the Bind9Daemon to simulate a partially detected daemon
	daemon.Bind9Daemon = nil
	err = dbmodel.AddDaemon(db, daemon)
	require.NoError(t, err)

	// Set one setting that is needed by puller.
	setting := dbmodel.Setting{
		Name:    "bind9_stats_puller_interval",
		ValType: dbmodel.SettingValTypeInt,
		Value:   "60",
	}
	_, err = db.Model(&setting).Insert()
	require.NoError(t, err)

	// Prepare stats puller.
	sp, err := NewStatsPuller(db, fa, fec)
	require.NoError(t, err)
	// Shutdown stats puller at the end.
	defer sp.Shutdown()

	// Act & Assert
	require.NotPanics(t, func() {
		err = sp.pullStats()
	})
	require.NoError(t, err)
}
