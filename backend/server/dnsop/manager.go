package dnsop

import (
	"context"
	"errors"
	"sync"

	"github.com/go-pg/pg/v10"
	log "github.com/sirupsen/logrus"
	agentcomm "isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
)

var _ Manager = (*managerImpl)(nil)

// An error returned upon a subsequent attempt to fetch zones while
// fetching zones is still in progress. The REST API can use this
// error to return appropriate HTTP status code to indicate that
// the request was "Accepted" and it is still being processed.
type ManagerAlreadyFetchingError struct{}

// Returns the error as text.
func (error *ManagerAlreadyFetchingError) Error() string {
	return "DNS manager is already fetching zones from the agents"
}

// This interface must be implemented by the instance owning the Manager.
// The server.StorkServer instance implements this interface because it
// owns the instance of the Manager. The Manager uses it to get access to
// the common database connection and to the communication interface with
// the connected agents.
type ManagerAccessors interface {
	// Returns an instance of the database handler used by the manager.
	GetDB() *pg.DB
	// Returns an interface to the agents the manager communicates with.
	GetConnectedAgents() agentcomm.ConnectedAgents
}

// An interface to the DNS Manager used from external packages. Exposing
// an interface rather than the structure makes it convenient for unit
// testing.
type Manager interface {
	// Contacts all agents with DNS servers and fetches zones from these servers.
	// The implementation fetches the zones in the background, and the number of
	// concurrent goroutines is controlled by the poolSize. Using the poolSize of 1
	// makes the fetch sequential (next agent is contacted after fetching all zones
	// from the previous agent). The batchSize controls the maximum number of zones
	// that are collected and inserted into the database in a single SQL INSERT.
	// Finally, block boolean flag indicates if the caller is going to wait for the
	// completion of the function (if true) or not (if false). In the latter case
	// the returned ManagerDoneNotify channel is buffered, in which case there is
	// no requirement to read from this channel before the fetch is complete. Setting
	// this flag to true is helpful in the unit tests to ensure that specific sequence
	// of calls is executed.
	FetchZones(poolSize, batchSize int, block bool) (chan ManagerDoneNotify, error)
	// Checks if the DNS Manager is currently fetching the zones and returns progress.
	// The first boolean flag indicates whether or not fetch is in progress. The int
	// parameters indicate the number of apps from which the zones are fetched and the
	// number of apps from which the zones have been fetched already.
	GetFetchZonesProgress() (bool, int, int)
}

// A zones fetching state including the flag whether or not the fetch
// is in progress, and other data useful to track fetch progress.
type fetchingState struct {
	fetching           bool
	completedAppsCount int
	appsCount          int
	mutex              sync.RWMutex
}

// Attempts to start fetching the zones. If we're already in the fetching state
// this function returns false indicating that another fetch cannot be started.
// Otherwise, it resets the state and returns true.
func (state *fetchingState) startFetching() bool {
	state.mutex.Lock()
	defer state.mutex.Unlock()
	if state.fetching {
		return false
	}
	state.fetching = true
	state.appsCount = 0
	state.completedAppsCount = 0
	return true
}

// Sets the flag indicating that fetching was stopped.
func (state *fetchingState) stopFetching() {
	state.mutex.Lock()
	defer state.mutex.Unlock()
	state.fetching = false
}

// Returns the underlying state values: a flag indicating whether or not
// fetch is in progress, the number of apps from which the zones are being
// fetched, and the number of apps from which the zones have been fetched
// already.
func (state *fetchingState) getFetchZonesProgress() (bool, int, int) {
	state.mutex.RLock()
	defer state.mutex.RUnlock()
	return state.fetching, state.appsCount, state.completedAppsCount
}

// Sets the total number of apps from which the zones are to be fetched.
func (state *fetchingState) setAppsCount(appsCount int) {
	state.mutex.Lock()
	defer state.mutex.Unlock()
	state.appsCount = appsCount
}

// Increases a counter of apps from which the zones have been fetched.
func (state *fetchingState) increaseCompletedAppsCount() {
	state.mutex.Lock()
	defer state.mutex.Unlock()
	state.completedAppsCount += 1
}

// DNS Manager implementation. The Manager is responsible for coordinating all
// operations pertaining to DNS in Stork server.
type managerImpl struct {
	// Common database instance.
	db *pg.DB
	// Interface to the connected agents.
	agents agentcomm.ConnectedAgents
	// A state of fetching zones from the DNS servers by the manager.
	fetchingState *fetchingState
}

// A structure returned over the channel when Manager completes asynchronous task.
type ManagerDoneNotify struct {
	// A map holding zone inventory state for different servers. The map key is
	// a daemon ID to which the state pertains.
	results map[int64]*dbmodel.ZoneInventoryStateDetails
}

// Instantiates DNS Manager.
func NewManager(owner ManagerAccessors) Manager {
	return &managerImpl{
		db:            owner.GetDB(),
		agents:        owner.GetConnectedAgents(),
		fetchingState: &fetchingState{},
	}
}

// Contacts all agents with DNS servers and fetches zones from these servers.
// It implements the Manager interface.
func (manager *managerImpl) FetchZones(poolSize, batchSize int, block bool) (chan ManagerDoneNotify, error) {
	// Only start fetching if there is no other fetch in progress.
	if !manager.fetchingState.startFetching() {
		return nil, &ManagerAlreadyFetchingError{}
	}
	// Get the list of monitored DNS servers. We're going to communicate
	// with the zone inventories created for these servers to fetch the
	// list of zones.
	apps, err := dbmodel.GetAppsByType(manager.db, dbmodel.AppTypeBind9)
	if err != nil {
		manager.fetchingState.stopFetching()
		return nil, err
	}
	manager.fetchingState.setAppsCount(len(apps))
	// Use the channel to communicate when the fetch has finished.
	// By default the channel is non-blocking in case the caller doesn't
	// want to wait for the completion. The buffer length of 1 ensures
	// that writing to the channel doesn't block. Otherwise, set the
	// length of 0 to make the channel blocking.
	bufLen := 1
	if block {
		bufLen = 0
	}
	notifyChannel := make(chan ManagerDoneNotify, bufLen)
	// Fetch is a background task. It may take significant amount of
	// time, depending on the number of zones and the number of DNS
	// servers.
	go func() {
		// Create a channel to distribute work to workers. Each worker
		// communicates with one machine. Concurrently fetching from
		// multiple machines should significantly decrease the time to
		// populate all zones.
		appsChan := make(chan dbmodel.App, len(apps))
		for _, app := range apps {
			appsChan <- app
		}
		close(appsChan)
		// Wait group is used to ensure we wait for zone fetch from all DNS servers.
		wg := sync.WaitGroup{}
		wg.Add(len(apps))
		// Mutex protects the results map from concurrent write. The results hold
		// a map of errors for respective daemons.
		mutex := sync.Mutex{}
		results := make(map[int64]*dbmodel.ZoneInventoryStateDetails)
		// Create worker goroutines. The goroutines will read from the common
		// channel and fetch the zones from the apps listed in the channel.
		for i := 0; i < poolSize; i++ {
			go func(appsChan <-chan dbmodel.App) {
				state := dbmodel.NewZoneInventoryStateDetails()
				// Read next app from the channel.
				for app := range appsChan {
					defer wg.Done()
					// Track views. We need to flush the batch when the view changes.
					// Otherwise, if the new view contains the same zone name that already
					// exists in the batch, the database will return an error on the
					// ON CONFLICT DO UPDATE clause.
					var view string
					// Insert zones into the database in batches. It significantly improves
					// performance for large number of zones.
					batch := dbmodel.NewBatch(manager.db, batchSize, dbmodel.AddZones)
					for zone, err := range manager.agents.ReceiveZones(context.Background(), &app, nil) {
						if err != nil {
							// Returned status depends on the returned error type. Some
							// errors require special handling.
							var (
								busyError      *agentcomm.ZoneInventoryBusyError
								notInitedError *agentcomm.ZoneInventoryNotInitedError
							)
							switch {
							case errors.As(err, &busyError):
								// Unable to fetch from the inventory because the inventory on
								// the agent is busy running some long lasting operation.
								state.SetStatus(dbmodel.ZoneInventoryStatusBusy, err)
							case errors.As(err, &notInitedError):
								// Unable to fetch from the inventory because the inventory has
								// not been initialized yet.
								state.SetStatus(dbmodel.ZoneInventoryStatusUninitialized, err)
							default:
								// Some other error.
								state.SetStatus(dbmodel.ZoneInventoryStatusErred, err)
							}
							break
						}
						// Successfully received the zone from the agent. Let's queue
						// it in the database for insertion.
						dbZone := dbmodel.Zone{
							Name: zone.Name(),
							LocalZones: []*dbmodel.LocalZone{
								{
									DaemonID: app.Daemons[0].ID,
									View:     zone.ViewName,
									Class:    zone.Class,
									Serial:   zone.Serial,
									Type:     zone.Type,
									LoadedAt: zone.Loaded,
								},
							},
						}
						// The zone also carries the total number of zones in the inventory.
						state.SetTotalZones(zone.TotalZoneCount)
						if view != zone.ViewName {
							// Flush the batch to complete the view insertion. Note that
							// this is ok even when the view is empty (first zone). In
							// this case the FlushAndAdd will skip the flush.
							err = batch.FlushAndAdd(&dbZone)
							view = zone.ViewName
						} else {
							err = batch.Add(&dbZone)
						}
						if err != nil {
							state.SetStatus(dbmodel.ZoneInventoryStatusErred, err)
							break
						}
					}
					if state.Error == nil {
						// If we successfully added zones to the database so far. There is
						// one more batch to add with a lower number of zones than the
						// specified batchSize.
						if err := batch.Flush(); err != nil {
							state.SetStatus(dbmodel.ZoneInventoryStatusErred, err)
						}
					}
					// Add the zone inventory state into the database for this DNS server.
					if err := dbmodel.AddZoneInventoryState(manager.db, dbmodel.NewZoneInventoryState(app.Daemons[0].ID, state)); err != nil {
						// This is an exceptional situation and normally shouldn't happen.
						// Let's communicate this issue to the caller and log it. The
						// zone inventory state won't be available for this server.
						state.SetStatus(dbmodel.ZoneInventoryStatusErred, err)
						log.WithFields(log.Fields{
							"app": app.Name,
						}).WithError(err).Error("Failed to save the zone inventory status in the database")
					}
					// Store the inventory state in the common map, so it can be returned
					// to a caller.
					storeResult(&mutex, results, app.Daemons[0].ID, state)
					manager.fetchingState.increaseCompletedAppsCount()
				}
			}(appsChan)
		}
		// Wait for all the go-routines to complete.
		wg.Wait()

		// Log the successful completion.
		var zoneCount int64
		for _, result := range results {
			if result.ZoneCount != nil {
				zoneCount += *result.ZoneCount
			}
		}
		log.WithFields(log.Fields{
			"appCount":  len(results),
			"zoneCount": zoneCount,
		}).Info("Completed fetching the zones from the agents")

		// Return the result.
		notifyChannel <- ManagerDoneNotify{
			results,
		}
		manager.fetchingState.stopFetching()
	}()
	return notifyChannel, nil
}

// Checks if the DNS Manager is currently fetching the zones.
func (manager *managerImpl) GetFetchZonesProgress() (bool, int, int) {
	return manager.fetchingState.getFetchZonesProgress()
}

// Convenience function storing a value in a map with mutex protection.
func storeResult[K comparable, T any](mutex *sync.Mutex, results map[K]T, key K, value T) {
	mutex.Lock()
	defer mutex.Unlock()
	results[key] = value
}
