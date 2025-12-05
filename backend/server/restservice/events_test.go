package restservice

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/server/gen/restapi/operations/events"
	storktest "isc.org/stork/server/test/dbmodel"
	storkutil "isc.org/stork/util"
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
			DaemonID: 2,
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

// Test if GetEvents sorting works as expected.
func TestGetEventsSorting(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add test events
	ev := &dbmodel.Event{
		Text:      "text a",
		Level:     dbmodel.EvError,
		Details:   "detail a",
		CreatedAt: time.Now().UTC(),
	}

	err := dbmodel.AddEvent(db, ev)
	require.NoError(t, err)

	ev = &dbmodel.Event{
		Text:      "text b",
		Level:     dbmodel.EvInfo,
		Details:   "detail b",
		CreatedAt: time.Now().UTC().Add(time.Second),
	}

	err = dbmodel.AddEvent(db, ev)
	require.NoError(t, err)

	// prepare RestAPI
	rapi, err := NewRestAPI(dbSettings, db)
	require.NoError(t, err)

	t.Run("sort by text asc", func(t *testing.T) {
		ctx := context.Background()
		params := events.GetEventsParams{
			SortField: storkutil.Ptr("text"),
			SortDir:   storkutil.Ptr(string(dbmodel.SortDirAsc)),
		}
		rsp := rapi.GetEvents(ctx, params)
		require.IsType(t, &events.GetEventsOK{}, rsp)
		okRsp := rsp.(*events.GetEventsOK)
		require.Len(t, okRsp.Payload.Items, 2)
		require.EqualValues(t, 2, okRsp.Payload.Total)
		require.EqualValues(t, "text a", okRsp.Payload.Items[0].Text)
		require.EqualValues(t, "text b", okRsp.Payload.Items[1].Text)
	})

	t.Run("sort by text desc", func(t *testing.T) {
		ctx := context.Background()
		params := events.GetEventsParams{
			SortField: storkutil.Ptr("text"),
			SortDir:   storkutil.Ptr(string(dbmodel.SortDirDesc)),
		}
		rsp := rapi.GetEvents(ctx, params)
		require.IsType(t, &events.GetEventsOK{}, rsp)
		okRsp := rsp.(*events.GetEventsOK)
		require.Len(t, okRsp.Payload.Items, 2)
		require.EqualValues(t, 2, okRsp.Payload.Total)
		require.EqualValues(t, "text b", okRsp.Payload.Items[0].Text)
		require.EqualValues(t, "text a", okRsp.Payload.Items[1].Text)
	})

	t.Run("sort by level asc", func(t *testing.T) {
		ctx := context.Background()
		params := events.GetEventsParams{
			SortField: storkutil.Ptr("level"),
			SortDir:   storkutil.Ptr(string(dbmodel.SortDirAsc)),
		}
		rsp := rapi.GetEvents(ctx, params)
		require.IsType(t, &events.GetEventsOK{}, rsp)
		okRsp := rsp.(*events.GetEventsOK)
		require.Len(t, okRsp.Payload.Items, 2)
		require.EqualValues(t, 2, okRsp.Payload.Total)
		require.Less(t, okRsp.Payload.Items[0].Level, okRsp.Payload.Items[1].Level)
	})

	t.Run("sort by level desc", func(t *testing.T) {
		ctx := context.Background()
		params := events.GetEventsParams{
			SortField: storkutil.Ptr("level"),
			SortDir:   storkutil.Ptr(string(dbmodel.SortDirDesc)),
		}
		rsp := rapi.GetEvents(ctx, params)
		require.IsType(t, &events.GetEventsOK{}, rsp)
		okRsp := rsp.(*events.GetEventsOK)
		require.Len(t, okRsp.Payload.Items, 2)
		require.EqualValues(t, 2, okRsp.Payload.Total)
		require.Greater(t, okRsp.Payload.Items[0].Level, okRsp.Payload.Items[1].Level)
	})

	t.Run("sort by details asc", func(t *testing.T) {
		ctx := context.Background()
		params := events.GetEventsParams{
			SortField: storkutil.Ptr("details"),
			SortDir:   storkutil.Ptr(string(dbmodel.SortDirAsc)),
		}
		rsp := rapi.GetEvents(ctx, params)
		require.IsType(t, &events.GetEventsOK{}, rsp)
		okRsp := rsp.(*events.GetEventsOK)
		require.Len(t, okRsp.Payload.Items, 2)
		require.EqualValues(t, 2, okRsp.Payload.Total)
		require.EqualValues(t, "detail a", okRsp.Payload.Items[0].Details)
		require.EqualValues(t, "detail b", okRsp.Payload.Items[1].Details)
	})

	t.Run("sort by details desc", func(t *testing.T) {
		ctx := context.Background()
		params := events.GetEventsParams{
			SortField: storkutil.Ptr("details"),
			SortDir:   storkutil.Ptr(string(dbmodel.SortDirDesc)),
		}
		rsp := rapi.GetEvents(ctx, params)
		require.IsType(t, &events.GetEventsOK{}, rsp)
		okRsp := rsp.(*events.GetEventsOK)
		require.Len(t, okRsp.Payload.Items, 2)
		require.EqualValues(t, 2, okRsp.Payload.Total)
		require.EqualValues(t, "detail b", okRsp.Payload.Items[0].Details)
		require.EqualValues(t, "detail a", okRsp.Payload.Items[1].Details)
	})

	t.Run("sort by created_at asc", func(t *testing.T) {
		ctx := context.Background()
		params := events.GetEventsParams{
			SortField: storkutil.Ptr("created_at"),
			SortDir:   storkutil.Ptr(string(dbmodel.SortDirAsc)),
		}
		rsp := rapi.GetEvents(ctx, params)
		require.IsType(t, &events.GetEventsOK{}, rsp)
		okRsp := rsp.(*events.GetEventsOK)
		require.Len(t, okRsp.Payload.Items, 2)
		require.EqualValues(t, 2, okRsp.Payload.Total)
		require.Less(t, okRsp.Payload.Items[0].CreatedAt, okRsp.Payload.Items[1].CreatedAt)
	})

	t.Run("sort by created_at desc", func(t *testing.T) {
		ctx := context.Background()
		params := events.GetEventsParams{
			SortField: storkutil.Ptr("created_at"),
			SortDir:   storkutil.Ptr(string(dbmodel.SortDirDesc)),
		}
		rsp := rapi.GetEvents(ctx, params)
		require.IsType(t, &events.GetEventsOK{}, rsp)
		okRsp := rsp.(*events.GetEventsOK)
		require.Len(t, okRsp.Payload.Items, 2)
		require.EqualValues(t, 2, okRsp.Payload.Total)
		require.Greater(t, okRsp.Payload.Items[0].CreatedAt, okRsp.Payload.Items[1].CreatedAt)
	})
}

func TestDeleteEvents(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add event
	ev := &dbmodel.Event{
		Text:  "some event",
		Level: dbmodel.EvInfo,
		Relations: &dbmodel.Relations{
			DaemonID: 2,
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

	// Create testing users in the database.
	user := &dbmodel.SystemUser{
		Email:    "jan@example.org",
		Lastname: "Kowalski",
		Name:     "Jan",
	}
	con, err := dbmodel.CreateUserWithPassword(db, user, "pass")
	require.False(t, con)
	require.NoError(t, err)
	superAdminUser := &dbmodel.SystemUser{
		Email:    "admin1@example.org",
		Lastname: "1",
		Name:     "Admin",
		Groups: []*dbmodel.SystemGroup{
			{ID: dbmodel.SuperAdminGroupID},
		},
	}
	con, err = dbmodel.CreateUser(db, superAdminUser)
	require.NoError(t, err)
	require.False(t, con)

	// delete all (fails because no user in session)
	deleteParams := events.DeleteEventsParams{}
	rsp := rapi.DeleteEvents(ctx, deleteParams)
	require.IsType(t, &events.DeleteEventsDefault{}, rsp)

	// Log in the test user
	err = rapi.SessionManager.LoginHandler(ctx, user)
	require.NoError(t, err)

	// delete all (fails without authorization)
	deleteParams = events.DeleteEventsParams{}
	rsp = rapi.DeleteEvents(ctx, deleteParams)
	require.IsType(t, &events.DeleteEventsDefault{}, rsp)

	// Log out the test user
	err = rapi.SessionManager.LogoutUser(ctx, user)
	require.NoError(t, err)

	// Log in the test super-admin user
	err = rapi.SessionManager.LoginHandler(ctx, superAdminUser)
	require.NoError(t, err)

	// delete all (succeeds)
	deleteParams = events.DeleteEventsParams{}
	rsp = rapi.DeleteEvents(ctx, deleteParams)
	require.IsType(t, &events.DeleteEventsNoContent{}, rsp)
}
