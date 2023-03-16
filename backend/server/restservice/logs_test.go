package restservice

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
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

	a := &dbmodel.App{
		ID:        0,
		MachineID: m.ID,
		Type:      dbmodel.AppTypeKea,
		Name:      "test-app",
		Active:    true,
		Daemons: []*dbmodel.Daemon{
			{
				Name:    "kea-dhcp4",
				Version: "1.7.5",
				Active:  true,
				LogTargets: []*dbmodel.LogTarget{
					{
						Output: "/tmp/filename.log",
					},
				},
			},
		},
	}
	_, err = dbmodel.AddApp(db, a)
	require.NoError(t, err)
	require.NotZero(t, a.ID)
	require.Len(t, a.Daemons, 1)
	require.Len(t, a.Daemons[0].LogTargets, 1)
	require.NotZero(t, a.Daemons[0].LogTargets[0].ID)

	fa := agentcommtest.NewFakeAgents(nil, nil)
	rapi, err := NewRestAPI(dbSettings, db, fa)
	require.NoError(t, err)
	ctx := context.Background()

	// Try to tail the log associated with our app. The response should be ok.
	params := services.GetLogTailParams{
		ID: a.Daemons[0].LogTargets[0].ID,
	}
	rsp := rapi.GetLogTail(ctx, params)
	require.IsType(t, &services.GetLogTailOK{}, rsp)
	okRsp := rsp.(*services.GetLogTailOK).Payload

	// Make sure that all values have been set correctly.
	require.Equal(t, "localhost", okRsp.Machine.Address)
	require.EqualValues(t, m.ID, okRsp.Machine.ID)
	require.EqualValues(t, a.ID, *okRsp.AppID)
	require.Equal(t, a.Type.String(), *okRsp.AppType)
	require.Equal(t, a.Name, *okRsp.AppName)
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

	a := &dbmodel.App{
		ID:        0,
		MachineID: m.ID,
		Type:      dbmodel.AppTypeKea,
		Active:    true,
		Daemons: []*dbmodel.Daemon{
			{
				Name:    "kea-dhcp4",
				Version: "1.7.5",
				Active:  true,
				LogTargets: []*dbmodel.LogTarget{
					{
						Output: "syslog:xyz",
					},
					{
						Output: "stdout",
					},
					{
						Output: "stderr",
					},
				},
			},
		},
	}
	_, err = dbmodel.AddApp(db, a)
	require.NoError(t, err)
	require.NotZero(t, a.ID)
	require.Len(t, a.Daemons, 1)
	require.Len(t, a.Daemons[0].LogTargets, 3)
	for i := range a.Daemons[0].LogTargets {
		require.NotZero(t, a.Daemons[0].LogTargets[i].ID)
	}

	fa := agentcommtest.NewFakeAgents(nil, nil)
	rapi, err := NewRestAPI(dbSettings, db, fa)
	require.NoError(t, err)
	ctx := context.Background()

	// Specify ID of non-existing log file.
	params := services.GetLogTailParams{
		ID: a.Daemons[0].LogTargets[0].ID + 10,
	}
	rsp := rapi.GetLogTail(ctx, params)
	require.IsType(t, &services.GetLogTailDefault{}, rsp)
	defaultRsp := rsp.(*services.GetLogTailDefault)
	require.Equal(t, http.StatusNotFound, getStatusCode(*defaultRsp))
	require.Equal(t, fmt.Sprintf("Log file with ID %d does not exist", params.ID),
		*defaultRsp.Payload.Message)

	// Make sure that an attempt to view the log from the targets other than file
	// is not allowed.
	for i := range a.Daemons[0].LogTargets {
		params = services.GetLogTailParams{
			ID: a.Daemons[0].LogTargets[i].ID,
		}
		rsp = rapi.GetLogTail(ctx, params)
		require.IsType(t, &services.GetLogTailDefault{}, rsp)
		defaultRsp = rsp.(*services.GetLogTailDefault)
		require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
		require.Equal(t, fmt.Sprintf("Viewing log from %s is not supported", a.Daemons[0].LogTargets[i].Output),
			*defaultRsp.Payload.Message)
	}
}
