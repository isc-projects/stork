package restservice

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/datamodel/daemonname"
	"isc.org/stork/datamodel/protocoltype"
	agentcommtest "isc.org/stork/server/agentcomm/test"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/server/gen/restapi/operations/services"
)

// This test verifies that the tail of the log file can be fetched via
// the REST API.
func TestGetLogTail(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, m)
	require.NoError(t, err)
	require.NotZero(t, m.ID)

	accessPoint := &dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "localhost",
		Port:     1234,
		Key:      "",
		Protocol: protocoltype.HTTP,
	}

	daemon := dbmodel.NewDaemon(m, daemonname.DHCPv4, true, []*dbmodel.AccessPoint{accessPoint})
	daemon.Version = "1.7.5"
	daemon.LogTargets = []*dbmodel.LogTarget{
		{
			Output: "/tmp/filename.log",
		},
	}
	err = dbmodel.AddDaemon(db, daemon)
	require.NoError(t, err)
	require.NotZero(t, daemon.ID)
	require.Len(t, daemon.LogTargets, 1)
	require.NotZero(t, daemon.LogTargets[0].ID)

	fa := agentcommtest.NewFakeAgents(nil, nil)
	rapi, err := NewRestAPI(dbSettings, db, fa)
	require.NoError(t, err)
	ctx := context.Background()

	// Try to tail the log associated with our daemon. The response should be ok.
	params := services.GetLogTailParams{
		ID: daemon.LogTargets[0].ID,
	}
	rsp := rapi.GetLogTail(ctx, params)
	require.IsType(t, &services.GetLogTailOK{}, rsp)
	okRsp := rsp.(*services.GetLogTailOK).Payload

	// Make sure that all values have been set correctly.
	require.Equal(t, "localhost", okRsp.Machine.Address)
	require.EqualValues(t, m.ID, okRsp.Machine.ID)
	require.EqualValues(t, daemon.GetVirtualApp().ID, *okRsp.AppID)
	require.Equal(t, string(daemon.GetVirtualApp().Type), *okRsp.AppType)
	require.Equal(t, daemon.GetVirtualApp().Name, *okRsp.AppName)
	require.Equal(t, "/tmp/filename.log", *okRsp.LogTargetOutput)
	require.Len(t, okRsp.Contents, 1)
	require.Equal(t, "lorem ipsum", okRsp.Contents[0])
}

// Test that error is returned when invalid parameters are specified while
// tailing the log file.
func TestLogTailBadParams(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, m)
	require.NoError(t, err)
	require.NotZero(t, m.ID)

	accessPoint := &dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "localhost",
		Port:     1234,
		Key:      "",
		Protocol: protocoltype.HTTP,
	}

	daemon := dbmodel.NewDaemon(m, daemonname.DHCPv4, true, []*dbmodel.AccessPoint{accessPoint})
	daemon.Version = "1.7.5"
	daemon.LogTargets = []*dbmodel.LogTarget{
		{
			Output: "syslog:xyz",
		},
		{
			Output: "stdout",
		},
		{
			Output: "stderr",
		},
	}
	err = dbmodel.AddDaemon(db, daemon)
	require.NoError(t, err)
	require.NotZero(t, daemon.ID)
	require.Len(t, daemon.LogTargets, 3)
	for i := range daemon.LogTargets {
		require.NotZero(t, daemon.LogTargets[i].ID)
	}

	fa := agentcommtest.NewFakeAgents(nil, nil)
	rapi, err := NewRestAPI(dbSettings, db, fa)
	require.NoError(t, err)
	ctx := context.Background()

	// Specify ID of non-existing log file.
	params := services.GetLogTailParams{
		ID: daemon.LogTargets[0].ID + 10,
	}
	rsp := rapi.GetLogTail(ctx, params)
	require.IsType(t, &services.GetLogTailDefault{}, rsp)
	defaultRsp := rsp.(*services.GetLogTailDefault)
	require.Equal(t, http.StatusNotFound, getStatusCode(*defaultRsp))
	require.Equal(t, fmt.Sprintf("Log file with ID %d does not exist", params.ID),
		*defaultRsp.Payload.Message)

	// Make sure that an attempt to view the log from the targets other than file
	// is not allowed.
	for i := range daemon.LogTargets {
		params = services.GetLogTailParams{
			ID: daemon.LogTargets[i].ID,
		}
		rsp = rapi.GetLogTail(ctx, params)
		require.IsType(t, &services.GetLogTailDefault{}, rsp)
		defaultRsp = rsp.(*services.GetLogTailDefault)
		require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
		require.Equal(t, fmt.Sprintf("Viewing log from %s is not supported", daemon.LogTargets[i].Output),
			*defaultRsp.Payload.Message)
	}
}
