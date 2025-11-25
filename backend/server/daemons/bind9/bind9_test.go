package bind9

import (
	"context"
	_ "embed"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	agentapi "isc.org/stork/api"
	"isc.org/stork/datamodel/daemonname"
	"isc.org/stork/datamodel/protocoltype"
	"isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	storktest "isc.org/stork/server/test/dbmodel"
)

//go:embed testdata/rndc-output.txt
var mockRndcOutput []byte

//go:generate mockgen -package=bind9 -destination=connectedagentsmock_test.go -source=../../agentcomm/agentcomm.go ConnectedAgents

// Test retrieving state of BIND 9 daemon.
func TestGetDaemonState(t *testing.T) {
	ctx := context.Background()

	fec := &storktest.FakeEventCenter{}

	machine := &dbmodel.Machine{
		Address:   "192.0.2.0",
		AgentPort: 1111,
	}

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

	// Set named stats response.
	response := NamedStatsGetResponse{
		Views: map[string]*ViewStatsData{
			"_default": {
				Resolver: ResolverData{
					CacheStats: CacheStatsData{
						CacheHits:   40,
						CacheMisses: 10,
						QueryHits:   70,
						QueryMisses: 30,
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
						CacheHits:   1,
						CacheMisses: 5,
						QueryHits:   4,
						QueryMisses: 6,
					},
				},
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnectedAgents := NewMockConnectedAgents(ctrl)

	// rndc response
	mockConnectedAgents.EXPECT().
		ForwardRndcCommand(gomock.Any(), daemon, "status").
		Times(3).
		Return(&agentcomm.RndcOutput{Output: string(mockRndcOutput)}, nil)

	// named stats response
	mockConnectedAgents.EXPECT().
		ForwardToNamedStats(gomock.Any(), daemon, agentapi.ForwardToNamedStatsReq_SERVER, gomock.Any()).
		Times(3).
		// Response is returned via argument pointer.
		SetArg(3, response).
		Return(nil)

	GetDaemonState(ctx, mockConnectedAgents, daemon, fec)

	require.True(t, daemon.Active)
	require.Equal(t, daemon.Version, "9.9.9")

	require.NotNil(t, daemon.Bind9Daemon)
	require.True(t, daemon.Active)
	require.Equal(t, daemonname.Bind9, daemon.Name)
	require.Equal(t, "9.9.9", daemon.Version)
	reloadedAt, _ := time.Parse(namedLongDateFormat, "Mon, 03 Feb 2020 14:39:36 GMT")
	require.Equal(t, reloadedAt, daemon.ReloadedAt)
	require.EqualValues(t, 5, daemon.Bind9Daemon.Stats.ZoneCount)
	require.EqualValues(t, 96, daemon.Bind9Daemon.Stats.AutomaticZoneCount)

	// Test statistics.
	require.EqualValues(t, 40, daemon.Bind9Daemon.Stats.NamedStats.Views["_default"].Resolver.CacheStats["CacheHits"])
	require.EqualValues(t, 10, daemon.Bind9Daemon.Stats.NamedStats.Views["_default"].Resolver.CacheStats["CacheMisses"])
	require.EqualValues(t, 70, daemon.Bind9Daemon.Stats.NamedStats.Views["_default"].Resolver.CacheStats["QueryHits"])
	require.EqualValues(t, 30, daemon.Bind9Daemon.Stats.NamedStats.Views["_default"].Resolver.CacheStats["QueryMisses"])

	require.EqualValues(t, 100, daemon.Bind9Daemon.Stats.NamedStats.Views["guest"].Resolver.CacheStats["CacheHits"])
	require.EqualValues(t, 200, daemon.Bind9Daemon.Stats.NamedStats.Views["guest"].Resolver.CacheStats["CacheMisses"])
	require.EqualValues(t, 56, daemon.Bind9Daemon.Stats.NamedStats.Views["guest"].Resolver.CacheStats["QueryHits"])
	require.EqualValues(t, 75, daemon.Bind9Daemon.Stats.NamedStats.Views["guest"].Resolver.CacheStats["QueryMisses"])

	require.NotContains(t, daemon.Bind9Daemon.Stats.NamedStats.Views, "_bind")

	// If the daemon has no ID, it means it is a new daemon that hasn't
	// been yet added to the database. In this case, the function will update
	// the same daemon instance.
	GetDaemonState(ctx, mockConnectedAgents, daemon, fec)
	require.NotNil(t, daemon.Bind9Daemon)

	// Set the ID. This time, the daemon should be preserved.
	daemon.ID = 1
	GetDaemonState(ctx, mockConnectedAgents, daemon, fec)
	require.NotNil(t, daemon.Bind9Daemon)
}

// Tests that BIND 9 can be added and then updated in the database.
func TestCommitDaemonIntoDB(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	fec := &storktest.FakeEventCenter{}

	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)
	require.NotZero(t, machine.ID)

	daemon := dbmodel.NewDaemon(machine, daemonname.Bind9, true, []*dbmodel.AccessPoint{
		{
			Type:    dbmodel.AccessPointControl,
			Address: "",
			Port:    1234,
		},
	})
	err = CommitDaemonIntoDB(db, daemon, fec)
	require.NoError(t, err)

	daemon.AccessPoints = []*dbmodel.AccessPoint{
		{
			Type:    dbmodel.AccessPointControl,
			Address: "",
			Port:    2345,
		},
	}
	err = CommitDaemonIntoDB(db, daemon, fec)
	require.NoError(t, err)

	returned, err := dbmodel.GetDaemonByID(db, daemon.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)
	require.Len(t, returned.AccessPoints, 1)
	require.EqualValues(t, 2345, returned.AccessPoints[0].Port)
}
