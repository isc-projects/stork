package eventcenter

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
)

// Tests that new subscriber instance cab be created and that default
// values are set correctly.
func TestNewSubscriber(t *testing.T) {
	// Use no filtering.
	url, err := url.Parse("http://example.org/sse?stream=message")
	require.NoError(t, err)

	subscriber := newSubscriber(url, "localhost:8080")
	require.NotNil(t, subscriber)

	// Make sure the done channel is created.
	require.NotNil(t, subscriber.done)

	// Make sure no filters are set.
	require.False(t, subscriber.useFilter)
	require.Equal(t, dbmodel.EvInfo, subscriber.filters.level)
	require.Zero(t, subscriber.filters.MachineID)
	require.Zero(t, subscriber.filters.AppID)
	require.Zero(t, subscriber.filters.SubnetID)
	require.Zero(t, subscriber.filters.DaemonID)
	require.Zero(t, subscriber.filters.UserID)
}

// Test that filters are set when present in the URL.
func TestSetFilterValues(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Use an URL with all parameters set.
	url, err := url.Parse("http://example.org/sse?stream=connectivity&stream=message&machine=1&app=2&subnet=3&daemon=4&user=5&level=1")
	require.NoError(t, err)

	subscriber := newSubscriber(url, "localhost:8080")
	require.NotNil(t, subscriber)

	// Apply the filters from the query.
	err = subscriber.applyFiltersFromQuery(db)
	require.NoError(t, err)

	// Filtering used.
	require.True(t, subscriber.useFilter)

	// Verify that the values were parsed correctly.
	require.Len(t, subscriber.filters.SSEStreams, 2)
	require.EqualValues(t, 1, subscriber.filters.MachineID)
	require.EqualValues(t, 2, subscriber.filters.AppID)
	require.EqualValues(t, 3, subscriber.filters.SubnetID)
	require.EqualValues(t, 4, subscriber.filters.DaemonID)
	require.EqualValues(t, 5, subscriber.filters.UserID)
	require.EqualValues(t, 1, subscriber.filters.level)
}

// A set of tests verifying that each ID type is supported as a filter.
func TestAcceptEventsSingleFilter(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	testCases := []string{"machine", "app", "subnet", "daemon", "user"}
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc, func(t *testing.T) {
			// Create an URL with a test case specific parameter.
			rawURL := fmt.Sprintf("http://example.org/sse?stream=message&%s=123", tc)
			url, err := url.Parse(rawURL)
			require.NoError(t, err)

			subscriber := newSubscriber(url, "localhost:8080")
			require.NotNil(t, subscriber)

			err = subscriber.applyFiltersFromQuery(db)
			require.NoError(t, err)

			require.True(t, subscriber.useFilter)

			// Create an event matching the query.
			ev := &dbmodel.Event{
				Relations: &dbmodel.Relations{},
			}
			switch tc {
			case "machine":
				ev.Relations.MachineID = 123
			case "app":
				ev.Relations.AppID = 123
			case "subnet":
				ev.Relations.SubnetID = 123
			case "daemon":
				ev.Relations.DaemonID = 123
			case "user":
				ev.Relations.UserID = 123
			}
			// Event should be accepted.
			require.Contains(t, subscriber.findMatchingEventStreams(ev), dbmodel.SSERegularMessage)

			// When event relations are cleared this event is no longer accepted.
			ev.Relations = &dbmodel.Relations{}
			require.Empty(t, subscriber.findMatchingEventStreams(ev))
		})
	}
}

// Test verifying that complex filter can be applied and the event
// must match all of the filtering rules.
func TestAcceptEventsMultipleFilters(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Create a filtering rule by machine ID, app ID and warning event level.
	url, err := url.Parse("http://example.org/sse?stream=message&machine=1&app=2&level=1")
	require.NoError(t, err)

	subscriber := newSubscriber(url, "localhost:8080")
	require.NotNil(t, subscriber)

	err = subscriber.applyFiltersFromQuery(db)
	require.NoError(t, err)

	require.True(t, subscriber.useFilter)

	// This event lacks app id so it should not be accepted.
	ev := &dbmodel.Event{
		Level: dbmodel.EvError,
		Relations: &dbmodel.Relations{
			MachineID: 1,
		},
	}
	require.Empty(t, subscriber.findMatchingEventStreams(ev))

	// This is similar to previous case but this time app id is lacking.
	ev = &dbmodel.Event{
		Level: dbmodel.EvError,
		Relations: &dbmodel.Relations{
			AppID: 2,
		},
	}
	require.Empty(t, subscriber.findMatchingEventStreams(ev))

	// The first parameter is not matching.
	ev = &dbmodel.Event{
		Level: dbmodel.EvError,
		Relations: &dbmodel.Relations{
			UserID: 1,
			AppID:  2,
		},
	}
	require.Empty(t, subscriber.findMatchingEventStreams(ev))

	// Both IDs are matching but it doesn't match the event level.
	ev = &dbmodel.Event{
		Level: dbmodel.EvInfo,
		Relations: &dbmodel.Relations{
			MachineID: 1,
			AppID:     2,
		},
	}
	require.Empty(t, subscriber.findMatchingEventStreams(ev))

	// Everything is matching.
	ev = &dbmodel.Event{
		Level: dbmodel.EvError,
		Relations: &dbmodel.Relations{
			MachineID: 1,
			AppID:     2,
		},
	}
	require.Contains(t, subscriber.findMatchingEventStreams(ev), dbmodel.SSERegularMessage)

	// More parameters is also fine as long as the first two are matching.
	ev = &dbmodel.Event{
		Level: dbmodel.EvWarning,
		Relations: &dbmodel.Relations{
			MachineID: 1,
			AppID:     2,
			UserID:    5,
		},
	}
	require.Contains(t, subscriber.findMatchingEventStreams(ev), dbmodel.SSERegularMessage)
}

// Test that appType and daemonName can be specified instead of app and daemon
// parameters when machine id is also provided.
func TestIndirectRelationsAppTypeDaemonName(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	app := &dbmodel.App{
		MachineID: machine.ID,
		Type:      dbmodel.AppTypeKea,
		Active:    true,
		Daemons: []*dbmodel.Daemon{
			{
				Name: "dhcp4",
			},
		},
	}
	_, err = dbmodel.AddApp(db, app)
	require.NoError(t, err)

	url, err := url.Parse("http://example.org/sse?stream=message&machine=1&appType=kea&daemonName=dhcp4&level=1")
	require.NoError(t, err)

	subscriber := newSubscriber(url, "localhost:8080")
	require.NotNil(t, subscriber)

	err = subscriber.applyFiltersFromQuery(db)
	require.NoError(t, err)

	require.True(t, subscriber.useFilter)
	require.EqualValues(t, 1, subscriber.filters.MachineID)
	require.EqualValues(t, app.ID, subscriber.filters.AppID)
	require.EqualValues(t, app.Daemons[0].ID, subscriber.filters.DaemonID)
	require.EqualValues(t, dbmodel.EvWarning, subscriber.filters.level)
	require.Equal(t, "localhost:8080", subscriber.subscriberAddress)
}

// Test that invalid combination of appType and daemonName parameters with other
// parameters yields an error.
func TestIndirectRelationsWrongParams(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)

	app := &dbmodel.App{
		MachineID: machine.ID,
		Type:      dbmodel.AppTypeKea,
		Active:    true,
		Daemons: []*dbmodel.Daemon{
			{
				Name: "dhcp4",
			},
		},
	}
	_, err = dbmodel.AddApp(db, app)
	require.NoError(t, err)

	t.Run("NoMachineID", func(t *testing.T) {
		// App type and daemon name require machine id.
		url, err := url.Parse("http://example.org/sse?stream=message&appType=kea&daemonName=dhcp4")
		require.NoError(t, err)

		subscriber := newSubscriber(url, "localhost:8080")
		require.NotNil(t, subscriber)

		err = subscriber.applyFiltersFromQuery(db)
		require.Error(t, err)
	})

	t.Run("NoAppType", func(t *testing.T) {
		// Daemon name requires app type.
		url, err := url.Parse("http://example.org/sse?stream=message&machine=1&daemonName=dhcp4")
		require.NoError(t, err)

		subscriber := newSubscriber(url, "localhost:8080")
		require.NotNil(t, subscriber)

		err = subscriber.applyFiltersFromQuery(db)
		require.Error(t, err)
	})

	t.Run("AppAndAppType", func(t *testing.T) {
		// App ID with app type are mutually exclusive.
		rawURL := fmt.Sprintf("http://example.org/sse?stream=message&machine=1&app=%d&appType=kea", app.ID)
		url, err := url.Parse(rawURL)
		require.NoError(t, err)

		subscriber := newSubscriber(url, "localhost:8080")
		require.NotNil(t, subscriber)

		err = subscriber.applyFiltersFromQuery(db)
		require.Error(t, err)
	})

	t.Run("DaemonAndDaemonID", func(t *testing.T) {
		// Daemon ID with daemon name are mutually exclusive.
		rawURL := fmt.Sprintf("http://example.org/sse?stream=message&machine=1&appType=kea&daemon=%d&daemonName=dhcp4",
			app.Daemons[0].ID)
		url, err := url.Parse(rawURL)
		require.NoError(t, err)

		subscriber := newSubscriber(url, "localhost:8080")
		require.NotNil(t, subscriber)

		err = subscriber.applyFiltersFromQuery(db)
		require.Error(t, err)
	})
}
