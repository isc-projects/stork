package restservice

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	agentcommtest "isc.org/stork/server/agentcomm/test"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/server/gen/restapi/operations/events"
	storktest "isc.org/stork/server/test"
)

// Check searching via rest api functions.
func TestEvents(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add event
	ev := &dbmodel.Event{
		Text:  "some event",
		Level: dbmodel.EvInfo,
		Relations: &dbmodel.Relations{
			AppID: 2,
		},
	}

	err := dbmodel.AddEvent(db, ev)
	require.NoError(t, err)

	// prepare RestAPI
	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec)
	require.NoError(t, err)
	ctx := context.Background()

	// search with empty text
	params := events.GetEventsParams{}
	rsp := rapi.GetEvents(ctx, params)
	require.IsType(t, &events.GetEventsOK{}, rsp)
	okRsp := rsp.(*events.GetEventsOK)
	require.Len(t, okRsp.Payload.Items, 1)
	require.EqualValues(t, 1, okRsp.Payload.Total)
	ev2 := okRsp.Payload.Items[0]
	require.EqualValues(t, "some event", ev2.Text)
	require.EqualValues(t, dbmodel.EvInfo, ev2.Level)
}
