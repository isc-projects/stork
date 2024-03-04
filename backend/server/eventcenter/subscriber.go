package eventcenter

import (
	"net/url"
	"strconv"

	errors "github.com/pkg/errors"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
)

// Holds the filters specified by an SSE subscriber connecting to
// the server. They are all possible values that are accepted for
// the supported types of streams.
type subscriberFilters struct {
	dbmodel.Relations
	level      dbmodel.EventLevel
	SSEStreams []dbmodel.SSEStream
}

// Structure describing SSE subscriber. Subscriber connects to the
// server via an URL which may optionally include filtering parameters
// for events. Filtering parameters are stored in filters structure
// and they are populated by parsing the URL used to connect to the
// server. Finally, the useFilter boolean value is set to true when
// it is detected that no filtering rules have been set. If this
// value is set to false (which is a default), the server sends all
// events to the subscriber.
type Subscriber struct {
	serverURL         *url.URL
	subscriberAddress string
	useFilter         bool
	filters           subscriberFilters
	done              chan struct{}
}

// Attempts to retrieve a named parameter from the subscriber's query
// and convert it to a numeric value. If such parameter does not
// exist in an URL, a value of 0 is returned. If the parameter can't
// be converted to a numeric value an error is returned.
func getQueryValueAsInt64(name string, values url.Values) (int64, error) {
	value, ok := values[name]
	if !ok || len(value) == 0 {
		return 0, nil
	}
	// In theory there may be multiple parameters with the same name specified,
	// but in our use cases we expect one. Let's take the first one.
	numericValue, err := strconv.ParseInt(value[0], 10, 64)
	if err != nil {
		err = errors.Errorf("sse query parameter %s=%s is not a valid numeric value", name, value[0])
		return 0, err
	}
	return numericValue, nil
}

// Checks if the event should be emitted based on the filtering criteria.
func (sf subscriberFilters) isInFiter(event *dbmodel.Event) bool {
	return (sf.level == 0 || sf.level <= event.Level) &&
		(sf.MachineID == 0 || event.Relations.MachineID == sf.MachineID) &&
		(sf.AppID == 0 || event.Relations.AppID == sf.AppID) &&
		(sf.SubnetID == 0 || event.Relations.SubnetID == sf.SubnetID) &&
		(sf.DaemonID == 0 || event.Relations.DaemonID == sf.DaemonID) &&
		(sf.UserID == 0 || event.Relations.UserID == sf.UserID)
}

// Creates a new instance of the subscriber using URL. It doesn't populate filters.
func newSubscriber(serverURL *url.URL, subscriberAddress string) *Subscriber {
	subscriber := &Subscriber{
		serverURL:         serverURL,
		useFilter:         false,
		subscriberAddress: subscriberAddress,
		done:              make(chan struct{}),
	}
	return subscriber
}

// Populates filters from URL. In a simplest case, a caller provides ids of the
// objects to filter by, e.g. machine=1, indicating that only events associated
// with machine id of 1 should be returned. However, there are also other
// parameters, such as appType or daemonName, which can't be directly used to
// filter events. In order to map these parameters to the event relations this
// function needs to query the database. In particular, machine id and app type
// map to a specific app id. When also a daemon name is specified, this maps
// to a specific daemon id etc.
func (s *Subscriber) applyFiltersFromQuery(db *dbops.PgDB) (err error) {
	f := &s.filters
	queryValues := s.serverURL.Query()

	// Level is also specified as numeric value. Possible values are 0, 1, 2.
	level, err := getQueryValueAsInt64("level", queryValues)
	if err != nil {
		return err
	}
	s.filters.level = dbmodel.EventLevel(level)

	messageStreamEnabled := false
	if streams, ok := queryValues["stream"]; ok {
		for _, stream := range streams {
			f.SSEStreams = append(f.SSEStreams, dbmodel.SSEStream(stream))
			if stream == string(dbmodel.SSERegularMessage) {
				messageStreamEnabled = true
			}
		}
	}

	// The reminder of this function applies filters for the main SEE stream.
	// If the stream is not enabled, there is nothing to do.
	if !messageStreamEnabled {
		return nil
	}

	// Check if direct event relations are specified in the URL. All of them
	// are IDs pointing to some specific objects in the database.
	if f.MachineID, err = getQueryValueAsInt64("machine", queryValues); err != nil {
		return err
	}
	if f.AppID, err = getQueryValueAsInt64("app", queryValues); err != nil {
		return err
	}
	if f.SubnetID, err = getQueryValueAsInt64("subnet", queryValues); err != nil {
		return err
	}
	if f.DaemonID, err = getQueryValueAsInt64("daemon", queryValues); err != nil {
		return err
	}
	if f.UserID, err = getQueryValueAsInt64("user", queryValues); err != nil {
		return err
	}

	// There are additional query parameters supported by the server: appType and
	// daemonName. They are mutually exclusive with app and daemon parmameters.
	// Also, daemonName require appType to be specified. Let's get those parameters
	// and perform appropriate sanity checks.
	appType := queryValues["appType"]
	daemonName := queryValues["daemonName"]

	// app must not be specified together with appType.
	if len(appType) > 0 && f.AppID != 0 {
		return errors.Errorf("appType and app query parameters are mutually exclusive: %s", s.serverURL)
	}

	// daemon must not be specified with daemonName.
	if len(daemonName) > 0 && f.DaemonID != 0 {
		return errors.Errorf("daemonName and daemon query parameters are mutually exclusive: %s", s.serverURL)
	}

	// daemonName requires appType.
	if len(daemonName) > 0 && len(appType) == 0 {
		return errors.Errorf("app or appType parameter is required when daemonName is specified: %s",
			s.serverURL)
	}

	if len(appType) > 0 {
		// App type and daemon name are ambiguous without saying to which machine
		// the app and/or daemon belong.
		if f.MachineID == 0 {
			return errors.Errorf("machine is required when appType or daemonName is specified: %s",
				s.serverURL)
		}
		machineApps, err := dbmodel.GetAppsByMachine(db, f.MachineID)
		if err != nil {
			return errors.WithMessagef(err, "problem getting machine by ID %d while applying sse filters: %s",
				f.MachineID, s.serverURL)
		}

		// Find an app matching the type.
		var app *dbmodel.App
		for i := range machineApps {
			if machineApps[i].Type.String() == appType[0] {
				f.AppID = machineApps[i].ID
				app = machineApps[i]
				break
			}
		}

		if app == nil {
			return errors.Errorf("app with type %s does not exist on machine %d", appType[0], f.MachineID)
		}

		if len(daemonName) > 0 {
			daemon := app.GetDaemonByName(daemonName[0])
			if daemon == nil {
				return errors.Errorf("daemon %s does not exist in app %d", daemonName, app.ID)
			}
			f.DaemonID = daemon.ID
		}
	}

	// In order to avoid iterating over all the filters every time we have a new
	// event we should check if everything we have done above resulted in setting
	// any of these values. If all of them happen to be zero we leave the useFilter
	// value as false reducing the number of checks to be performed to only this
	// value. Otherwise, we need to do the matching for each event.
	for _, id := range []int64{f.MachineID, f.AppID, f.SubnetID, f.DaemonID, f.UserID, level} {
		if id != 0 {
			s.useFilter = true
			break
		}
	}
	return nil
}

// Returns a list of SSE streams in which this event should be sent. The event is not
// sent when the returned list is empty.
func (s *Subscriber) findMatchingEventStreams(event *dbmodel.Event) (streams []dbmodel.SSEStream) {
	for _, stream := range s.filters.SSEStreams {
		if stream == dbmodel.SSERegularMessage && (!s.useFilter || s.filters.isInFiter(event)) {
			streams = append(streams, dbmodel.SSERegularMessage)
		} else {
			for _, eventStream := range event.SSEStreams {
				if eventStream == stream {
					streams = append(streams, stream)
					break
				}
			}
		}
	}
	return streams
}
