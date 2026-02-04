package eventcenter

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
	dbmodeltest "isc.org/stork/server/database/model/test"
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
	require.Zero(t, subscriber.filters.SubnetID)
	require.Zero(t, subscriber.filters.DaemonID)
	require.Zero(t, subscriber.filters.UserID)
}

// Test that filters are set when present in the URL.
func TestSetFilterValues(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	server, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	daemon, err := server.GetDaemon()
	require.NoError(t, err)

	// Use an URL with all parameters set.
	url, err := url.Parse(fmt.Sprintf(
		"http://example.org/sse?stream=connectivity&stream=message&machineId=1&&subnetId=3&daemonId=%d&userId=5&level=1",
		daemon.ID,
	))
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
	require.EqualValues(t, 3, subscriber.filters.SubnetID)
	require.EqualValues(t, daemon.ID, subscriber.filters.DaemonID)
	require.EqualValues(t, 5, subscriber.filters.UserID)
	require.EqualValues(t, 1, subscriber.filters.level)
}

// A set of tests verifying that each ID type is supported as a filter.
func TestAcceptEventsSingleFilter(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	testCases := []string{"machineId", "subnetId", "daemonId", "userId"}
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc, func(t *testing.T) {
			id := int64(123)
			// Create an URL with a test case specific parameter.
			rawURL := fmt.Sprintf("http://example.org/sse?stream=message&%s=%d", tc, id)
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
			case "machineId":
				ev.Relations.MachineID = 123
			case "subnetId":
				ev.Relations.SubnetID = 123
			case "daemonId":
				ev.Relations.DaemonID = 123
			case "userId":
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

	server, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	daemon, err := server.GetDaemon()
	require.NoError(t, err)

	// Create a filtering rule by machine ID, daemon ID and warning event level.
	url, err := url.Parse(fmt.Sprintf(
		"http://example.org/sse?stream=message&machineId=%d&daemonId=%d&level=%d",
		daemon.MachineID, daemon.ID, dbmodel.EvWarning,
	))
	require.NoError(t, err)

	subscriber := newSubscriber(url, "localhost:8080")
	require.NotNil(t, subscriber)

	err = subscriber.applyFiltersFromQuery(db)
	require.NoError(t, err)

	require.True(t, subscriber.useFilter)

	// This event lacks daemon id so it should not be accepted.
	ev := &dbmodel.Event{
		Level: dbmodel.EvError,
		Relations: &dbmodel.Relations{
			MachineID: 1,
		},
	}
	require.Empty(t, subscriber.findMatchingEventStreams(ev))

	// The first parameter is not matching.
	ev = &dbmodel.Event{
		Level: dbmodel.EvError,
		Relations: &dbmodel.Relations{
			UserID:    1,
			MachineID: daemon.MachineID,
		},
	}
	require.Empty(t, subscriber.findMatchingEventStreams(ev))

	// ID is matching but it doesn't match the event level.
	ev = &dbmodel.Event{
		Level: dbmodel.EvInfo,
		Relations: &dbmodel.Relations{
			MachineID: daemon.MachineID,
		},
	}
	require.Empty(t, subscriber.findMatchingEventStreams(ev))

	// Everything is matching.
	ev = &dbmodel.Event{
		Level: dbmodel.EvWarning,
		Relations: &dbmodel.Relations{
			MachineID: daemon.MachineID,
			DaemonID:  daemon.ID,
		},
	}
	require.Contains(t, subscriber.findMatchingEventStreams(ev), dbmodel.SSERegularMessage)

	// More parameters is also fine as long as the first is matching.
	ev = &dbmodel.Event{
		Level: dbmodel.EvWarning,
		Relations: &dbmodel.Relations{
			MachineID: daemon.MachineID,
			DaemonID:  daemon.ID,
			UserID:    5,
		},
	}
	require.Contains(t, subscriber.findMatchingEventStreams(ev), dbmodel.SSERegularMessage)
}

// Test that daemonName can be specified instead of daemon ID when machine ID
// is also provided.
func TestIndirectRelationsDaemonName(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	server, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	machine, err := server.GetMachine()
	require.NoError(t, err)
	daemon, err := server.GetDaemon()
	require.NoError(t, err)

	url, err := url.Parse("http://example.org/sse?stream=message&machineId=1&daemonName=dhcp4&level=1")
	require.NoError(t, err)

	subscriber := newSubscriber(url, "localhost:8080")
	require.NotNil(t, subscriber)

	err = subscriber.applyFiltersFromQuery(db)
	require.NoError(t, err)

	require.True(t, subscriber.useFilter)
	require.EqualValues(t, 1, subscriber.filters.MachineID)
	require.EqualValues(t, machine.ID, subscriber.filters.MachineID)
	require.EqualValues(t, daemon.ID, subscriber.filters.DaemonID)
	require.EqualValues(t, dbmodel.EvWarning, subscriber.filters.level)
	require.Equal(t, "localhost:8080", subscriber.subscriberAddress)
}

// Test that invalid combination of daemonName parameters with other
// parameters yields an error.
func TestIndirectRelationsWrongParams(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	server, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	daemon, err := server.GetDaemon()
	require.NoError(t, err)

	t.Run("NoMachineID", func(t *testing.T) {
		// Daemon name requires machine ID.
		url, err := url.Parse("http://example.org/sse?stream=message&daemonName=dhcp4")
		require.NoError(t, err)

		subscriber := newSubscriber(url, "localhost:8080")
		require.NotNil(t, subscriber)

		err = subscriber.applyFiltersFromQuery(db)
		require.Error(t, err)
	})

	t.Run("missing machine ID", func(t *testing.T) {
		// Daemon name requires machine ID.
		url, err := url.Parse("http://example.org/sse?stream=message&daemonName=dhcp4")
		require.NoError(t, err)

		subscriber := newSubscriber(url, "localhost:8080")
		require.NotNil(t, subscriber)

		err = subscriber.applyFiltersFromQuery(db)
		require.Error(t, err)
	})

	t.Run("daemon ID and daemon name", func(t *testing.T) {
		// Daemon ID with daemon name are mutually exclusive.
		rawURL := fmt.Sprintf("http://example.org/sse?stream=message&machineId=1&daemonId=%d&daemonName=dhcp4",
			daemon.ID)
		url, err := url.Parse(rawURL)
		require.NoError(t, err)

		subscriber := newSubscriber(url, "localhost:8080")
		require.NotNil(t, subscriber)

		err = subscriber.applyFiltersFromQuery(db)
		require.Error(t, err)
	})
}
