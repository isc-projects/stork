package restservice

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/server/gen/restapi/operations/events"
	storktest "isc.org/stork/server/test/dbmodel"
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
	rapi, err := NewRestAPI(dbSettings, db)
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

func TestDeleteEvents(t *testing.T) {
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
	ctx := context.Background()
	fec := &storktest.FakeEventCenter{}
	rapi, err := NewRestAPI(dbSettings, db, fec)
	require.NoError(t, err)

	// Create session manager.
	ctx, err = rapi.SessionManager.Load(ctx, "")
	require.NoError(t, err)

	// Create testing user in the database.
	user := &dbmodel.SystemUser{
		Email:    "jan@example.org",
		Lastname: "Kowalski",
		Name:     "Jan",
	}
	con, err := dbmodel.CreateUserWithPassword(db, user, "pass")
	require.False(t, con)
	require.NoError(t, err)

	// delete all (fails because no user in session)
	deleteParams := events.DeleteEventsParams{}
	rsp := rapi.DeleteEvents(ctx, deleteParams)
	require.IsType(t, &events.DeleteEventsDefault{}, rsp)

	// Log in the test user
	err = rapi.SessionManager.LoginHandler(ctx, user)
	require.NoError(t, err)

	// delete all (succeeds)
	deleteParams = events.DeleteEventsParams{}
	rsp = rapi.DeleteEvents(ctx, deleteParams)
	require.IsType(t, &events.DeleteEventsNoContent{}, rsp)
}
