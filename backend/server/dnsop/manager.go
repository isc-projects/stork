package dnsop

import (
	"context"
	"fmt"
	"hash/fnv"
	"iter"
	"runtime"
	"slices"
	"sync"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"isc.org/stork/appcfg/dnsconfig"
	agentcomm "isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
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

// An error returned upon concurrent attempts to transfer the same zone
// for the same view and daemon.
type ManagerRRsAlreadyRequestedError struct {
	viewName string
	zoneName string
}

// Instantiates a new ManagerRRsAlreadyRequestedError.
func NewManagerRRsAlreadyRequestedError(viewName string, zoneName string) *ManagerRRsAlreadyRequestedError {
	return &ManagerRRsAlreadyRequestedError{
		viewName: viewName,
		zoneName: zoneName,
	}
}

// Returns the error as text.
func (error *ManagerRRsAlreadyRequestedError) Error() string {
	return fmt.Sprintf("zone transfer for view %s, zone %s has been already requested by another user", error.viewName, error.zoneName)
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
	GetZoneRRs(zoneID int64, daemonID int64, viewName string, options ...GetZoneRRsOption) iter.Seq[*RRResponse]
	Shutdown()
}

// Type of options for GetZoneRRs function.
type GetZoneRRsOption int

const (
	// Force zone transfer even when RRs are cached.
	GetZoneRRsOptionForceZoneTransfer GetZoneRRsOption = iota
)

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

// A state of requests sent to the agents to fetch RRs.
type rrsRequestingState struct {
	pool        *storkutil.PausablePool
	cancel      context.CancelFunc
	requestChan chan *rrsRequest
	requests    map[uint64]bool
	mutex       sync.Mutex
}

// A structure holding a request to fetch RRs for a zone.
type rrsRequest struct {
	ctx       context.Context
	app       *dbmodel.App
	zoneName  string
	viewName  string
	respChan  chan *RRResponse
	closeOnce sync.Once
	key       uint64
}

// Instantiates a new RRs request.
func newRRsRequest(ctx context.Context, key uint64, app *dbmodel.App, zoneName string, viewName string) *rrsRequest {
	return &rrsRequest{
		ctx:       ctx,
		app:       app,
		zoneName:  zoneName,
		viewName:  viewName,
		respChan:  make(chan *RRResponse),
		closeOnce: sync.Once{},
		key:       key,
	}
}

// A structure encapsulating a set of RRs returned by the agent, a database,
// or an error.
type RRResponse struct {
	Cached         bool
	ZoneTransferAt time.Time
	RRs            []*dnsconfig.RR
	Err            error
}

// Creates a new RRResponse with an error.
func NewErrorRRResponse(err error) *RRResponse {
	return &RRResponse{
		Err: err,
	}
}

// Creates a new non-cached RRResponse with RRs. The zone transfer time is
// set to current time.
func NewZoneTransferRRResponse(rrs []*dnsconfig.RR) *RRResponse {
	return &RRResponse{
		Cached:         false,
		ZoneTransferAt: time.Now().UTC(),
		RRs:            rrs,
	}
}

// Creates a new cached RRResponse with RRs.
func NewCacheRRResponse(rrs []*dnsconfig.RR, transferAt time.Time) *RRResponse {
	return &RRResponse{
		Cached:         true,
		ZoneTransferAt: transferAt,
		RRs:            rrs,
	}
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
	// A state of RRs requests.
	rrsReqsState *rrsRequestingState
}

// A structure returned over the channel when Manager completes asynchronous task.
type ManagerDoneNotify struct {
	// A map holding zone inventory state for different servers. The map key is
	// a daemon ID to which the state pertains.
	results map[int64]*dbmodel.ZoneInventoryStateDetails
}

// Instantiates DNS Manager.
func NewManager(owner ManagerAccessors) (Manager, error) {
	ctx, cancel := context.WithCancel(context.Background())
	impl := &managerImpl{
		db:            owner.GetDB(),
		agents:        owner.GetConnectedAgents(),
		fetchingState: &fetchingState{},
		rrsReqsState: &rrsRequestingState{
			requestChan: make(chan *rrsRequest),
			requests:    make(map[uint64]bool),
			cancel:      cancel,
		},
	}
	impl.startRRsRequestWorkers(ctx)
	return impl, nil
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
	apps, err := dbmodel.GetAppsByType(manager.db, dbmodel.AppTypeBind9, dbmodel.AppTypePDNS)
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
				// Read next app from the channel.
				for app := range appsChan {
					manager.fetchZonesFromDNSServer(&app, batchSize, &wg, &mutex, results)
				}
			}(appsChan)
		}
		// Wait for all the go-routines to complete.
		wg.Wait()

		// Delete orphaned zones in the database.
		if _, err := dbmodel.DeleteOrphanedZones(manager.db); err != nil {
			log.WithError(err).Error("Failed to delete orphaned zones in the database")
		}

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

// Contacts a specified DNS server and fetches zones from it. The app
// indicates the DNS server to contact. The batchSize parameter controls
// the size of the batch of zones to be inserted into the database in a single
// SQL INSERT. The wg and mutex parameters are used to synchronize the work
// between multiple goroutines. The results parameter is a map of errors for
// respective daemons. This function can merely be called from the
// FetchZones function.
func (manager *managerImpl) fetchZonesFromDNSServer(app *dbmodel.App, batchSize int, wg *sync.WaitGroup, mutex *sync.Mutex, results map[int64]*dbmodel.ZoneInventoryStateDetails) {
	defer wg.Done()
	var (
		// Track views. We need to flush the batch when the view changes.
		// Otherwise, if the new view contains the same zone name that already
		// exists in the batch, the database will return an error on the
		// ON CONFLICT DO UPDATE clause.
		view string
		// During the first iteration we need to delete the local zones.
		// This flag is used to identify the first iteration.
		isFirst = true
	)
	// Insert zones into the database in batches. It significantly improves
	// performance for large number of zones.
	batch := dbmodel.NewBatch(manager.db, batchSize, dbmodel.AddZones)
	state := dbmodel.NewZoneInventoryStateDetails()
	for zone, err := range manager.agents.ReceiveZones(context.Background(), app, nil) {
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
		if isFirst {
			// Delete the local zones.
			isFirst = false
			err = dbmodel.DeleteLocalZones(manager.db, app.Daemons[0].ID)
			if err != nil {
				state.SetStatus(dbmodel.ZoneInventoryStatusErred, err)
				break
			}
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
					RPZ:      zone.RPZ,
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
	zoneCountStats, err := dbmodel.GetZoneCountStatsByDaemon(manager.db, app.Daemons[0].ID)
	if err != nil {
		state.SetStatus(dbmodel.ZoneInventoryStatusErred, err)
	} else {
		state.SetDistinctZoneCount(zoneCountStats.DistinctZones)
		state.SetBuiltinZoneCount(zoneCountStats.BuiltinZones)
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
	storeResult(mutex, results, app.Daemons[0].ID, state)
	manager.fetchingState.increaseCompletedAppsCount()
}

// Checks if the DNS Manager is currently fetching the zones.
func (manager *managerImpl) GetFetchZonesProgress() (bool, int, int) {
	return manager.fetchingState.getFetchZonesProgress()
}

// Requests RRs for a zone from the agent's zone inventory and returns them
// over the response channel.
func (manager *managerImpl) runRRsRequest(request *rrsRequest) {
	defer func() {
		close(request.respChan)
		manager.rrsReqsState.mutex.Lock()
		defer manager.rrsReqsState.mutex.Unlock()
		delete(manager.rrsReqsState.requests, request.key)
	}()
	for rr, err := range manager.agents.ReceiveZoneRRs(context.Background(), request.app, request.zoneName, request.viewName) {
		var response *RRResponse
		if err != nil {
			response = NewErrorRRResponse(err)
		} else {
			response = NewZoneTransferRRResponse(rr)
		}
		select {
		case <-request.ctx.Done():
			// The client stopped reading the RRs.
			return
		case request.respChan <- response:
			// Get more RRs.
		}
	}
}

// Requests RRs for a zone from the agent's zone inventory and returns them
// over the response channel. The specified key should uniquely identify a
// zone, view and daemon for which the RRs are requested. If there is an
// ongoing request for the same key, the function returns an error.
func (manager *managerImpl) requestZoneRRs(ctx context.Context, key uint64, app *dbmodel.App, zoneName string, viewName string) (chan *RRResponse, error) {
	// Try to mark the request as ongoing. If the request is already present
	// under the same key, return an error.
	manager.rrsReqsState.mutex.Lock()
	if _, ok := manager.rrsReqsState.requests[key]; ok {
		manager.rrsReqsState.mutex.Unlock()
		return nil, NewManagerRRsAlreadyRequestedError(viewName, zoneName)
	}
	manager.rrsReqsState.requests[key] = true
	manager.rrsReqsState.mutex.Unlock()

	// Create a new request and send it to the channel.
	request := newRRsRequest(ctx, key, app, zoneName, viewName)
	manager.rrsReqsState.requestChan <- request
	// Return the response channel, so the caller can receive the RRs.
	return request.respChan, nil
}

// Starts a pool of workers that fetch RRs for the requested zones from the
// agents' zone inventories.
func (manager *managerImpl) startRRsRequestWorkers(ctx context.Context) {
	pool := storkutil.NewPausablePool(runtime.GOMAXPROCS(0) * 2)
	manager.rrsReqsState.pool = pool
	go func(ctx context.Context) {
		defer func() {
			// When the worker pool is stopped, we need to close the request channel.
			close(manager.rrsReqsState.requestChan)
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case request, ok := <-manager.rrsReqsState.requestChan:
				if !ok {
					return
				}
				err := pool.Submit(func() {
					manager.runRRsRequest(request)
				})
				if err != nil {
					log.WithError(err).Error("Failed to submit RRs request to the worker pool")
					request.respChan <- NewErrorRRResponse(err)
				}
			}
		}
	}(ctx)
}

// Stops the worker pool for fetching RRs.
func (manager *managerImpl) stopRRsRequestWorkers() {
	manager.rrsReqsState.pool.Stop()
	manager.rrsReqsState.cancel()
}

// Shuts down the DNS manager by stopping background tasks.
func (manager *managerImpl) Shutdown() {
	log.Info("Shutting down DNS Manager")
	manager.stopRRsRequestWorkers()
}

// Returns zone contents (RRs) for a specified view, zone and daemon.
// Depending on the options, the function may cache transferred RRs in
// the database. A caller may also request forcing zone transfer to
// override the cached RRs.
func (manager *managerImpl) GetZoneRRs(zoneID int64, daemonID int64, viewName string, options ...GetZoneRRsOption) iter.Seq[*RRResponse] {
	return func(yield func(*RRResponse) bool) {
		// We need an app associated with the daemon.
		daemon, err := dbmodel.GetDaemonByID(manager.db, daemonID)
		if err != nil {
			// This is unexpected and we can't proceed because we
			// don't have the app instance.
			_ = yield(NewErrorRRResponse(err))
			return
		}
		if daemon == nil {
			// This is also unexpected.
			_ = yield(NewErrorRRResponse(errors.Errorf("daemon with the ID of %d not found", daemonID)))
			return
		}
		app := daemon.App

		// We need a zone name, so let's get it from the database.
		zone, err := dbmodel.GetZoneByID(manager.db, zoneID)
		if err != nil {
			// Again, it should be rare, unless someone used a link to a non-existing
			// zone or tempered with the ID in the URL.
			_ = yield(NewErrorRRResponse(err))
			return
		}
		if zone == nil {
			// This is also unexpected.
			_ = yield(NewErrorRRResponse(errors.Errorf("zone with the ID of %d not found", zoneID)))
			return
		}
		// Local zone is required to fetch cached RRs.
		localZone := zone.GetLocalZone(daemonID, viewName)
		if localZone == nil {
			_ = yield(NewErrorRRResponse(errors.Errorf("local zone information for daemon ID %d and view %s not found in zone: %s", daemonID, viewName, zone.Name)))
			return
		}
		// If we don't force zone transfer the local zone should have been initialized
		// from the database.
		if !slices.Contains(options, GetZoneRRsOptionForceZoneTransfer) && localZone.ZoneTransferAt != nil {
			// Get cached RRs from the database and return to the caller.
			rrs, err := dbmodel.GetDNSConfigRRs(manager.db, localZone.ID)
			if err != nil {
				_ = yield(NewErrorRRResponse(err))
				return
			}
			// This response indicates that the RRs were fetched from the database.
			_ = yield(NewCacheRRResponse(rrs, *localZone.ZoneTransferAt))
			return
		}
		// To avoid sending multiple requests for the same zone, we should check
		// if any requests are already in progress. The FNV key is unique for the
		// daemon, zone and view.
		h := fnv.New64a()
		h.Write([]byte(fmt.Sprintf("%d:%d:%s", daemonID, zoneID, viewName)))
		key := h.Sum64()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		ch, err := manager.requestZoneRRs(ctx, key, app, zone.Name, viewName)
		if err != nil {
			// The zone inventory is most likely busy.
			_ = yield(NewErrorRRResponse(err))
			return
		}
		// Let's create transaction to cache the RRs.
		tx, err := manager.db.Begin()
		if err != nil {
			_ = yield(NewErrorRRResponse(errors.Wrap(err, "failed to begin a transaction for caching RRs")))
			return
		}
		defer func() {
			err := tx.Rollback()
			if err != nil {
				log.WithError(err).Error("Failed to rollback the transaction for caching RRs")
			}
		}()
		// Update the timestamp indicating when RRs were last cached.
		if err := dbmodel.UpdateLocalZoneRRsTransferAt(tx, localZone.ID); err != nil {
			_ = yield(NewErrorRRResponse(errors.Wrap(err, "failed to update the RRs fetched time for the local zone")))
			return
		}
		// Delete existing RRs for the local zone.
		if err := dbmodel.DeleteLocalZoneRRs(tx, localZone.ID); err != nil {
			_ = yield(NewErrorRRResponse(errors.Wrap(err, "failed to delete cached RRs for the local zone")))
			return
		}
		// Batch insert RRs into the database.
		batch := dbmodel.NewBatch(tx, 100, dbmodel.AddLocalZoneRRs)
		for r := range ch {
			if r.Err != nil {
				// There was an error reading from the channel. The channel will be closed and
				// the transaction will be rolled back.
				_ = yield(r)
				return
			}
			for _, rr := range r.RRs {
				// Insert next RR into the database.
				if err := batch.Add(&dbmodel.LocalZoneRR{
					RR:          *rr,
					LocalZoneID: localZone.ID,
				}); err != nil {
					// Something went wrong with database insertion.
					_ = yield(NewErrorRRResponse(err))
					return
				}
			}
			if !yield(r) {
				// Stop reading and caching the RRs if the caller is done reading.
				return
			}
		}
		// We're done reading and caching the RRs. Let's flush and commit the transaction.
		if err := batch.Flush(); err != nil {
			// The caller no longer expects responses.
			_ = yield(NewErrorRRResponse(errors.Wrap(err, "failed to flush the batch of RRs")))
			return
		}
		if err := tx.Commit(); err != nil {
			// The caller no longer expects responses.
			_ = yield(NewErrorRRResponse(errors.Wrap(err, "failed to commit the transaction for caching RRs")))
			return
		}
	}
}

// Convenience function storing a value in a map with mutex protection.
func storeResult[K comparable, T any](mutex *sync.Mutex, results map[K]T, key K, value T) {
	mutex.Lock()
	defer mutex.Unlock()
	results[key] = value
}
