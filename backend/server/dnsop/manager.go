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
	agentapi "isc.org/stork/api"
	bind9config "isc.org/stork/daemoncfg/bind9"
	"isc.org/stork/daemoncfg/dnsconfig"
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

// An error returned upon concurrent attempts to transfer the same zone
// for the same view and daemon.
type ManagerBind9FormattedConfigAlreadyRequestedError struct{}

// Instantiates a new ManagerRRsAlreadyRequestedError.
func NewManagerBind9FormattedConfigAlreadyRequestedError() *ManagerBind9FormattedConfigAlreadyRequestedError {
	return &ManagerBind9FormattedConfigAlreadyRequestedError{}
}

// Returns the error as text.
func (error *ManagerBind9FormattedConfigAlreadyRequestedError) Error() string {
	return "BIND 9 configuration for the specified daemon has been already requested by another user"
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
	// parameters indicate the number of daemons from which the zones are fetched and
	// the number of daemons from which the zones have been fetched already.
	GetFetchZonesProgress() (bool, int, int)
	// Returns the RRs for the specified zone, daemon and view name. The optional
	// filter allows for filtering the RRs by type and/or text matching the name or
	// rdata. It also allows for paging the long results. If the filter is nil,
	// all RRs are returned for the zoneID, daemonID, viewName. If the filter is
	// specified but the RRs have not been cached yet, all RRs are fetched (and cached)
	// from the agent using the zone transfer, and only a subset is returned to the
	// caller. Using the options, the caller can request to force zone transfer even
	// when RRs are cached in the database.
	GetZoneRRs(zoneID int64, daemonID int64, viewName string, filter *dbmodel.GetZoneRRsFilter, options ...GetZoneRRsOption) iter.Seq[*RRResponse]
	// Returns the BIND 9 configuration for the specified daemon with filtering
	// and file type selection. If the filter is nil, all configuration elements are
	// returned. Otherwise,only the configuration elements explicitly enabled in the
	// filter are returned. Similarly, if the file selector is nil, all configuration
	// files are returned. Otherwise, only the configuration files explicitly enabled
	// in the file selector are returned.
	GetBind9FormattedConfig(ctx context.Context, daemonID int64, fileSelector *bind9config.FileTypeSelector, filter *bind9config.Filter) iter.Seq[*Bind9FormattedConfigResponse]
	Shutdown()
}

// Type of options for GetZoneRRs function.
type GetZoneRRsOption int

const (
	// Force zone transfer even when RRs are cached.
	GetZoneRRsOptionForceZoneTransfer GetZoneRRsOption = iota
	// Exclude the trailing SOA RR from the response.
	GetZoneRRsOptionExcludeTrailingSOA
)

// A zones fetching state including the flag whether or not the fetch
// is in progress, and other data useful to track fetch progress.
type fetchingState struct {
	fetching              bool
	completedDaemonsCount int
	daemonsCount          int
	mutex                 sync.RWMutex
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
	state.daemonsCount = 0
	state.completedDaemonsCount = 0
	return true
}

// Sets the flag indicating that fetching was stopped.
func (state *fetchingState) stopFetching() {
	state.mutex.Lock()
	defer state.mutex.Unlock()
	state.fetching = false
}

// Returns the underlying state values: a flag indicating whether or not
// fetch is in progress, the number of daemons from which the zones are being
// fetched, and the number of daemons from which the zones have been fetched
// already.
func (state *fetchingState) getFetchZonesProgress() (bool, int, int) {
	state.mutex.RLock()
	defer state.mutex.RUnlock()
	return state.fetching, state.daemonsCount, state.completedDaemonsCount
}

// Sets the total number of daemons from which the zones are to be fetched.
func (state *fetchingState) setDaemonsCount(daemonsCount int) {
	state.mutex.Lock()
	defer state.mutex.Unlock()
	state.daemonsCount = daemonsCount
}

// Increases a counter of daemons from which the zones have been fetched.
func (state *fetchingState) increaseCompletedDaemonsCount() {
	state.mutex.Lock()
	defer state.mutex.Unlock()
	state.completedDaemonsCount += 1
}

// A state of requests sent to the agents to fetch RRs.
type rrsRequestingState struct {
	requestChan chan *rrsRequest
	requests    map[uint64]bool
	mutex       sync.Mutex
}

// A structure holding a request to fetch RRs for a zone.
type rrsRequest struct {
	ctx       context.Context
	daemon    *dbmodel.Daemon
	zoneName  string
	viewName  string
	respChan  chan *RRResponse
	closeOnce sync.Once
	key       uint64
}

// Instantiates a new RRs request.
func newRRsRequest(ctx context.Context, key uint64, daemon *dbmodel.Daemon, zoneName string, viewName string) *rrsRequest {
	return &rrsRequest{
		ctx:       ctx,
		daemon:    daemon,
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
	Total          int
	RRs            []*dnsconfig.RR
	Err            error
}

// Returns a new RRResponse with the RRs filtered according to the specified filter.
// The pos parameter tracks the position in the RRs stream, starting from 0. It is
// increased by the number of processed RRs and returned as the second return value.
// If the filter is nil, the response is returned unchanged.
func (response *RRResponse) applyFilter(filter *dbmodel.GetZoneRRsFilter, pos int) (*RRResponse, int) {
	if filter == nil {
		// If the filter is nil, there is nothing to do. Return the response unchanged.
		response.Total = pos + len(response.RRs)
		return response, response.Total
	}
	// Filter the RRs according to the filter.
	filteredRRs := []*dnsconfig.RR{}
	for _, rr := range response.RRs {
		if filter.IsTypeEnabled(rr.Type) && filter.IsTextMatches(rr) {
			filteredRRs = append(filteredRRs, rr)
		}
	}

	// Calculate the new position in the RRs stream. It is moved forward by the
	// number of filtered RRs. It must be calculate before paging is applied because
	// it will be returned as the total number of RRs.
	newPos := pos + len(filteredRRs)

	if len(filteredRRs) > 0 {
		// Apply paging if requested.
		start := filter.GetOffset() - pos
		if start < 0 {
			// Negative start means that we're passed the beginning of the
			// desired range. Set the start to 0 to consume all RRs.
			start = 0
		}
		if start >= len(filteredRRs) {
			// If the start is passed the end of the RRs, we haven't reached the
			// desired offset. Set the start to the end of the current RRs set to
			// not consume any RRs.
			start = len(filteredRRs)
		}
		limit := filter.GetLimit()
		end := start + limit
		// A limit of 0 means no limit.
		if limit == 0 || end > len(filteredRRs) {
			end = len(filteredRRs)
		}
		// It is possible that start is equal to end if we haven't reached the
		// desired offset yet. This will result in returning an empty slice.
		filteredRRs = filteredRRs[start:end]
	}
	return &RRResponse{
		Cached:         response.Cached,
		ZoneTransferAt: response.ZoneTransferAt,
		Total:          newPos,
		RRs:            filteredRRs,
		Err:            response.Err,
	}, newPos
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
func NewCacheRRResponse(rrs []*dnsconfig.RR, total int, transferAt time.Time) *RRResponse {
	return &RRResponse{
		Cached:         true,
		ZoneTransferAt: transferAt,
		RRs:            rrs,
		Total:          total,
	}
}

// A state of requests sent to the agents to receive BIND 9 configuration.
type bind9FormattedConfigRequestingState struct {
	requestChan chan *bind9FormattedConfigRequest
	requests    map[int64]bool
	mutex       sync.Mutex
}

// A structure holding a request to receive BIND 9 configuration from the agent.
type bind9FormattedConfigRequest struct {
	ctx          context.Context
	daemon       *dbmodel.Daemon
	fileSelector *bind9config.FileTypeSelector
	filter       *bind9config.Filter
	respChan     chan *Bind9FormattedConfigResponse
	closeOnce    sync.Once
	daemonID     int64
}

// Instantiates a new BIND 9 configuration request.
func newBind9FormattedConfigRequest(ctx context.Context, daemonID int64, daemon *dbmodel.Daemon, fileSelector *bind9config.FileTypeSelector, filter *bind9config.Filter) *bind9FormattedConfigRequest {
	return &bind9FormattedConfigRequest{
		ctx:          ctx,
		daemon:       daemon,
		fileSelector: fileSelector,
		filter:       filter,
		respChan:     make(chan *Bind9FormattedConfigResponse),
		closeOnce:    sync.Once{},
		daemonID:     daemonID,
	}
}

// A structure representing a chunk of the BIND 9 configuration received
// over the stream.
type Bind9FormattedConfigResponse struct {
	// The BIND 9 configuration file preamble.
	File *agentapi.ReceiveBind9ConfigFile
	// The BIND 9 configuration file contents chunk.
	Contents *string
	// The error returned when receiving the BIND 9 configuration.
	Err error
}

// Creates a new Bind9FormattedConfigResponse representing a BIND 9 configuration
// file preamble.
func NewBind9FormattedConfigResponseFile(file *agentapi.ReceiveBind9ConfigFile) *Bind9FormattedConfigResponse {
	return &Bind9FormattedConfigResponse{
		File:     file,
		Contents: nil,
		Err:      nil,
	}
}

// Creates a new Bind9FormattedConfigResponse representing a BIND 9 configuration
// contents chunk.
func NewBind9FormattedConfigResponseContents(contents *string) *Bind9FormattedConfigResponse {
	return &Bind9FormattedConfigResponse{
		File:     nil,
		Contents: contents,
		Err:      nil,
	}
}

// Creates a new Bind9FormattedConfigResponse representing an error.
func NewBind9FormattedConfigResponseError(err error) *Bind9FormattedConfigResponse {
	return &Bind9FormattedConfigResponse{
		File:     nil,
		Contents: nil,
		Err:      err,
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
	// A state of BIND 9 configuration requests.
	bind9FormattedConfigReqsState *bind9FormattedConfigRequestingState
	// A pool of workers for RRs requests and BIND 9 configuration requests.
	pool *storkutil.PausablePool
	// A cancel function for the pool.
	cancel context.CancelFunc
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
		},
		bind9FormattedConfigReqsState: &bind9FormattedConfigRequestingState{
			requestChan: make(chan *bind9FormattedConfigRequest),
			requests:    make(map[int64]bool),
		},
		cancel: cancel,
	}
	impl.startAsyncRequestWorkers(ctx)
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
	daemons, err := dbmodel.GetDNSDaemons(manager.db)
	if err != nil {
		manager.fetchingState.stopFetching()
		return nil, err
	}
	manager.fetchingState.setDaemonsCount(len(daemons))
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
		daemonsChan := make(chan dbmodel.Daemon, len(daemons))
		for _, daemon := range daemons {
			daemonsChan <- daemon
		}
		close(daemonsChan)
		// Wait group is used to ensure we wait for zone fetch from all DNS servers.
		wg := sync.WaitGroup{}
		wg.Add(len(daemons))
		// Mutex protects the results map from concurrent write. The results hold
		// a map of errors for respective daemons.
		mutex := sync.Mutex{}
		results := make(map[int64]*dbmodel.ZoneInventoryStateDetails)
		// Create worker goroutines. The goroutines will read from the common
		// channel and fetch the zones from the daemons listed in the channel.
		for i := 0; i < poolSize; i++ {
			go func(daemonsChan <-chan dbmodel.Daemon) {
				// Read next daemon from the channel.
				for daemon := range daemonsChan {
					manager.fetchZonesFromDNSServer(&daemon, batchSize, &wg, &mutex, results)
				}
			}(daemonsChan)
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
			"daemonCount": len(results),
			"zoneCount":   zoneCount,
		}).Info("Completed fetching the zones from the agents")

		// Return the result.
		notifyChannel <- ManagerDoneNotify{
			results,
		}
		manager.fetchingState.stopFetching()
	}()
	return notifyChannel, nil
}

// Contacts a specified DNS server and fetches zones from it. The daemon
// indicates the DNS server to contact. The batchSize parameter controls
// the size of the batch of zones to be inserted into the database in a single
// SQL INSERT. The wg and mutex parameters are used to synchronize the work
// between multiple goroutines. The results parameter is a map of errors for
// respective daemons. This function can merely be called from the
// FetchZones function.
func (manager *managerImpl) fetchZonesFromDNSServer(daemon *dbmodel.Daemon, batchSize int, wg *sync.WaitGroup, mutex *sync.Mutex, results map[int64]*dbmodel.ZoneInventoryStateDetails) {
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
	for zone, err := range manager.agents.ReceiveZones(context.Background(), daemon, nil) {
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
			err = dbmodel.DeleteLocalZones(manager.db, daemon.ID)
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
					DaemonID: daemon.ID,
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
	zoneCountStats, err := dbmodel.GetZoneCountStatsByDaemon(manager.db, daemon.ID)
	if err != nil {
		state.SetStatus(dbmodel.ZoneInventoryStatusErred, err)
	} else {
		state.SetDistinctZoneCount(zoneCountStats.DistinctZones)
		state.SetBuiltinZoneCount(zoneCountStats.BuiltinZones)
	}
	// Add the zone inventory state into the database for this DNS server.
	if err := dbmodel.AddZoneInventoryState(manager.db, dbmodel.NewZoneInventoryState(daemon.ID, state)); err != nil {
		// This is an exceptional situation and normally shouldn't happen.
		// Let's communicate this issue to the caller and log it. The
		// zone inventory state won't be available for this server.
		state.SetStatus(dbmodel.ZoneInventoryStatusErred, err)
		log.WithFields(log.Fields{
			"daemon": daemon.Name,
		}).WithError(err).Error("Failed to save the zone inventory status in the database")
	}
	// Store the inventory state in the common map, so it can be returned
	// to a caller.
	storeResult(mutex, results, daemon.ID, state)
	manager.fetchingState.increaseCompletedDaemonsCount()
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
	for rr, err := range manager.agents.ReceiveZoneRRs(context.Background(), request.daemon, request.zoneName, request.viewName) {
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
func (manager *managerImpl) requestZoneRRs(ctx context.Context, key uint64, daemon *dbmodel.Daemon, zoneName string, viewName string) (chan *RRResponse, error) {
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
	request := newRRsRequest(ctx, key, daemon, zoneName, viewName)
	manager.rrsReqsState.requestChan <- request
	// Return the response channel, so the caller can receive the RRs.
	return request.respChan, nil
}

// Requests BIND 9 configuration from the agent and returns it over the
// response channel.
func (manager *managerImpl) runBind9FormattedConfigRequest(request *bind9FormattedConfigRequest) {
	defer func() {
		close(request.respChan)
		manager.bind9FormattedConfigReqsState.mutex.Lock()
		defer manager.bind9FormattedConfigReqsState.mutex.Unlock()
		delete(manager.bind9FormattedConfigReqsState.requests, request.daemonID)
	}()
	for rsp, err := range manager.agents.ReceiveBind9FormattedConfig(request.ctx, request.daemon, request.fileSelector, request.filter) {
		var response *Bind9FormattedConfigResponse
		switch {
		case err != nil:
			response = NewBind9FormattedConfigResponseError(err)
		case rsp == nil:
			response = NewBind9FormattedConfigResponseError(errors.Errorf("unexpected nil response while getting BIND 9 configuration from the agent"))
		default:
			// Non-error response.
			switch r := rsp.Response.(type) {
			case *agentapi.ReceiveBind9ConfigRsp_File:
				// A BIND 9 configuration file preamble.
				response = NewBind9FormattedConfigResponseFile(r.File)
			case *agentapi.ReceiveBind9ConfigRsp_Line:
				// The BIND 9 configuration file contents.
				response = NewBind9FormattedConfigResponseContents(&r.Line)
			default:
				// Unexpected response type.
				response = NewBind9FormattedConfigResponseError(errors.Errorf("unexpected response type: %T", rsp.Response))
			}
		}
		select {
		case <-request.ctx.Done():
			// The client stopped reading the BIND 9 configuration.
			return
		case request.respChan <- response:
			// Get more requests.
		}
	}
}

// Requests BIND 9 configuration from the agent and returns it over the response channel.
// This function prevents concurrent requests for the same daemon. If another
// request for the same daemon is in progress, the function returns an error.
func (manager *managerImpl) requestBind9FormattedConfig(ctx context.Context, daemonID int64, daemon *dbmodel.Daemon, fileSelector *bind9config.FileTypeSelector, filter *bind9config.Filter) (chan *Bind9FormattedConfigResponse, error) {
	// Try to mark the request as ongoing. If the request is already present
	// for the same daemon, return an error.
	manager.bind9FormattedConfigReqsState.mutex.Lock()
	if _, ok := manager.bind9FormattedConfigReqsState.requests[daemonID]; ok {
		manager.bind9FormattedConfigReqsState.mutex.Unlock()
		return nil, NewManagerBind9FormattedConfigAlreadyRequestedError()
	}
	manager.bind9FormattedConfigReqsState.requests[daemonID] = true
	manager.bind9FormattedConfigReqsState.mutex.Unlock()

	// Create a new request and send it to the channel.
	request := newBind9FormattedConfigRequest(ctx, daemonID, daemon, fileSelector, filter)
	manager.bind9FormattedConfigReqsState.requestChan <- request
	// Return the response channel, so the caller can receive the BIND 9 configuration.
	return request.respChan, nil
}

// Starts a pool of workers that fetch RRs for the requested zones and
// BIND 9 configurations from the agents. Using the worker pool limits
// the number of concurrent requests to the agents.
func (manager *managerImpl) startAsyncRequestWorkers(ctx context.Context) {
	pool := storkutil.NewPausablePool(runtime.GOMAXPROCS(0) * 2)
	manager.pool = pool
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
			case request, ok := <-manager.bind9FormattedConfigReqsState.requestChan:
				if !ok {
					return
				}
				err := pool.Submit(func() {
					manager.runBind9FormattedConfigRequest(request)
				})
				if err != nil {
					log.WithError(err).Error("Failed to submit BIND 9 configuration request to the worker pool")
					request.respChan <- NewBind9FormattedConfigResponseError(err)
				}
			}
		}
	}(ctx)
}

// Stops the worker pool for receiving RRs and BIND 9 configuration.
func (manager *managerImpl) stopRRsRequestWorkers() {
	manager.pool.Stop()
	manager.cancel()
}

// Shuts down the DNS manager by stopping background tasks.
func (manager *managerImpl) Shutdown() {
	log.Info("Shutting down DNS Manager")
	manager.stopRRsRequestWorkers()
}

// Returns zone contents (RRs) for a specified view, zone and daemon.
// Depending on the options, the function may cache transferred RRs in
// the database. A caller may also request forcing zone transfer to
// override the cached RRs. The GetZoneRRsOptionExcludeTrailingSOA option
// can be specified to exclude the trailing SOA RR from the response.
// The trailing SOA marks the end of the zone transfer. However, the
// REST API handler typically excludes this last record as it is not
// interesting to a user viewing the zone contents.
func (manager *managerImpl) GetZoneRRs(zoneID int64, daemonID int64, viewName string, filter *dbmodel.GetZoneRRsFilter, options ...GetZoneRRsOption) iter.Seq[*RRResponse] {
	return func(yield func(*RRResponse) bool) {
		// We need a daemon associated with the zone.
		daemon, err := dbmodel.GetDNSDaemonByID(manager.db, daemonID)
		if err != nil {
			// This is unexpected and we can't proceed because we
			// don't have the daemon instance.
			_ = yield(NewErrorRRResponse(err))
			return
		}
		if daemon == nil {
			// This is also unexpected.
			_ = yield(NewErrorRRResponse(errors.Errorf("daemon with the ID of %d not found", daemonID)))
			return
		}

		// We need a zone name, so let's get it from the database.
		zone, err := dbmodel.GetZoneByID(manager.db, zoneID, dbmodel.ZoneRelationLocalZones)
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
			rrs, total, err := dbmodel.GetDNSConfigRRs(manager.db, localZone.ID, filter)
			if err != nil {
				_ = yield(NewErrorRRResponse(err))
				return
			}
			// This response indicates that the RRs were fetched from the database.
			_ = yield(NewCacheRRResponse(rrs, total, *localZone.ZoneTransferAt))
			return
		}
		// To avoid sending multiple requests for the same zone, we should check
		// if any requests are already in progress. The FNV key is unique for the
		// daemon, zone and view.
		h := fnv.New64a()
		fmt.Fprintf(h, "%d:%d:%s", daemonID, zoneID, viewName)
		key := h.Sum64()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		ch, err := manager.requestZoneRRs(ctx, key, daemon, zone.Name, viewName)
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
			if err != nil && !errors.Is(err, pg.ErrTxDone) {
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
		// Tracks position in the RRs stream. It points to the index of the next
		// RR to be received from the channel, starting from 0. It is required to
		// extract a subset of RRs to be returned according to the paging filter.
		pos := 0
		// Check if this is the first chunk of RRs.
		isFirst := true
		for r := range ch {
			if r.Err != nil {
				// There was an error reading from the channel. The channel will be closed and
				// the transaction will be rolled back.
				_ = yield(r)
				return
			}
			if slices.Contains(options, GetZoneRRsOptionExcludeTrailingSOA) {
				// Check if this is not a trailing SOA RR. If this is the first
				// chunk of RRs, the first record should be SOA and we must not
				// remove it. Therefore, we only remove the last record from the
				// first chunk, assuming this last record is a SOA and is not the
				// first record (single record case). For subsequent chunks we remove
				// the last record if it is a SOA record. Note that we have not means
				// to check if this is truly the last record in the stream because
				// we don't know if we have reached the end of the stream. If there
				// are any mid-stream SOA records they may not be removed. Also,
				// some mid-stream SOA records may be removed if they appear at the
				// end of the chunk of data.
				if isFirst && len(r.RRs) > 1 && r.RRs[len(r.RRs)-1].Type == "SOA" ||
					!isFirst && len(r.RRs) > 0 && r.RRs[len(r.RRs)-1].Type == "SOA" {
					r.RRs = r.RRs[:len(r.RRs)-1]
				}
				isFirst = false
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
			// Select only the records matching the filter.
			var filteredResponse *RRResponse
			filteredResponse, pos = r.applyFilter(filter, pos)
			if !yield(filteredResponse) {
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

// Returns the BIND 9 configuration for the specified daemon with filtering.
// If the filter is nil, all configuration elements are returned. Otherwise,
// only the configuration elements explicitly enabled in the filter are returned.
// The specified daemon ID must point to a BIND 9 daemon. If it points to a different
// daemon type, the function returns an error without attempting to contact the agent.
// The returned configuration may contain multiple files. Typically, it contains the
// main configuration file and the rndc.key file.
func (manager *managerImpl) GetBind9FormattedConfig(ctx context.Context, daemonID int64, fileSelector *bind9config.FileTypeSelector, filter *bind9config.Filter) iter.Seq[*Bind9FormattedConfigResponse] {
	return func(yield func(*Bind9FormattedConfigResponse) bool) {
		// The daemon must be present in the database. We're going to use the
		// connection parameters associated with the daemon to contact the agent.
		daemon, err := dbmodel.GetDaemonByID(manager.db, daemonID)
		if err != nil {
			_ = yield(NewBind9FormattedConfigResponseError(err))
			return
		}
		if daemon == nil {
			_ = yield(NewBind9FormattedConfigResponseError(errors.Errorf("unable to get BIND 9 configuration from non-existent daemon with the ID %d", daemonID)))
			return
		}
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		ch, err := manager.requestBind9FormattedConfig(ctx, daemonID, daemon, fileSelector, filter)
		if err != nil {
			// The zone inventory is most likely busy.
			_ = yield(NewBind9FormattedConfigResponseError(err))
			return
		}
		for r := range ch {
			if !yield(r) || r.Err != nil {
				// Stop reading the BIND 9 configuration if the caller is done reading
				// or there was an error reading from the channel.
				return
			}
		}
	}
}

// Convenience function storing a value in a map with mutex protection.
func storeResult[K comparable, T any](mutex *sync.Mutex, results map[K]T, key K, value T) {
	mutex.Lock()
	defer mutex.Unlock()
	results[key] = value
}
