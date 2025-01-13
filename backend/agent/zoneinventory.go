package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"iter"
	"os"
	"path"
	"slices"
	"sync"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"isc.org/stork/appdata/bind9stats"
	storkutil "isc.org/stork/util"
)

var (
	_ zoneFetcher                     = (*bind9StatsClient)(nil)
	_ zoneInventoryStorage            = (*zoneInventoryStorageDisk)(nil)
	_ zoneInventoryStorage            = (*zoneInventoryStorageMemory)(nil)
	_ zoneInventoryStorage            = (*zoneInventoryStorageMemoryDisk)(nil)
	_ bind9stats.ZoneIteratorAccessor = (*viewIO)(nil)
	_ bind9stats.NameAccessor         = (os.DirEntry)(nil)
)

// A file name holding zone inventory meta data.
const zoneInventoryMetaFileName = "zone-inventory.json"

// An interface to a REST client communicating with a DNS server and returning
// configured zones encapsulated in views. The bind9StatsClient implements this
// interface. In the future, a BIND9 REST client and other clients communicating
// with non-ISC DNS implementations should also implement this interface.
//
// This interface returns *bind9stats.Views for simplicity. It groups zones into
// views which are BIND9-specific concept. For other DNS implementations we can use
// the same structure, and create an artificial (default) view but we also take into
// account that this structure may be replaced with a generic structure or interface
// if the bind9stats.Views is not a good fit for the integrations with other DNS
// servers.
type zoneFetcher interface {
	// Returns a list of zones configured in a DNS server grouped into views.
	getViews(host string, port int64) (httpResponse, *bind9stats.Views, error)
}

// An error indicating that the zone inventory is busy and the requested transition
// cannot be invoked.
//
// The zone inventory is busy when it is running a long lasting operation in
// background. For example, this error can be returned when the agent is fetching
// the zones from a DNS server and populating the zone inventory, and there is
// another call from the Stork server to get the zones discovered by the agent.
type zoneInventoryBusyError struct {
	// Inventory current state.
	currState *zoneInventoryState
	// Inventory intended state.
	newState *zoneInventoryState
}

// Instantiates the error. The function parameters specify the current inventory
// state and the intended state.
func newZoneInventoryBusyError(currState, newState *zoneInventoryState) error {
	return &zoneInventoryBusyError{
		currState,
		newState,
	}
}

// Returns error string. It indicates the current and the intended inventory state
// informing that such a transition is not allowed.
func (e zoneInventoryBusyError) Error() string {
	return fmt.Sprintf("cannot transition to the %s state while the zone inventory is in %s state", e.newState.name, e.currState.name)
}

// An error indicating that the inventory hasn't been populated yet. It means that
// the agent has not yet contacted the DNS server to fetch the list of zones.
type zoneInventoryNotInitedError struct{}

// Instantiates the error.
func newZoneInventoryNotInitedError() error {
	return &zoneInventoryNotInitedError{}
}

// Returns error string.
func (e zoneInventoryNotInitedError) Error() string {
	return "zone inventory has not been initialized yet"
}

// An error indicating that no persistent storage is configured for the inventory.
// This is the case when the zoneInventoryStorageMemory is used as a storage. In
// that case the zones are stored in memory only.
type zoneInventoryNoDiskStorageError struct{}

// Instantiates the error.
func newZoneInventoryNoDiskStorageError() error {
	return &zoneInventoryNoDiskStorageError{}
}

// Returns error string.
func (e zoneInventoryNoDiskStorageError) Error() string {
	return "zone inventory has no persistent storage"
}

// An interface implemented for all supported storage types:
// - a storage that holds the zones in memory and disk,
// - a storage that holds the zones on disk only,
// - a storage that holds the zones in memory only.
//
// This interface is intended to return storage capabilities so the inventory
// implementation can decide how to access the requested information. For example:
// when getting a zone from a storage that holds zone information in memory and on
// disk the inventory will return the zone information held in memory because it
// is faster.
type zoneInventoryStorage interface {
	// Returns a path to the views and zones stored on disk.
	getStorageLocation() string
	// Returns true if the storage holds zone information in memory.
	hasMemoryStorage() bool
	// Returns true if the storage holds zone information on disk.
	hasDiskStorage() bool
}

// A storage holding the zone information in memory and on disk.
type zoneInventoryStorageMemoryDisk struct {
	// Path to the disk storage.
	location string
}

// Instantiates the storage. The parameter specifies the disk storage path.
func newZoneInventoryStorageMemoryDisk(location string) *zoneInventoryStorageMemoryDisk {
	return &zoneInventoryStorageMemoryDisk{
		location,
	}
}

// Returns a path to the views and zones stored on disk.
func (storage zoneInventoryStorageMemoryDisk) getStorageLocation() string {
	return storage.location
}

// Indicates that the storage holds zone information in memory.
func (storage zoneInventoryStorageMemoryDisk) hasMemoryStorage() bool {
	return true
}

// Indicates that the storage holds zone information on disk.
func (storage zoneInventoryStorageMemoryDisk) hasDiskStorage() bool {
	return true
}

// A storage holding the zone information in on disk.
type zoneInventoryStorageDisk struct {
	// Path to the disk storage.
	location string
}

// Instantiates the storage. The parameter specifies the disk storage path.
func newZoneInventoryStorageDisk(location string) *zoneInventoryStorageDisk {
	return &zoneInventoryStorageDisk{
		location,
	}
}

// Returns a path to the views and zones stored on disk.
func (storage zoneInventoryStorageDisk) getStorageLocation() string {
	return storage.location
}

// Indicates that the storage does not hold the zone information in memory.
func (storage zoneInventoryStorageDisk) hasMemoryStorage() bool {
	return false
}

// Indicates that the storage holds zone information on disk.
func (storage zoneInventoryStorageDisk) hasDiskStorage() bool {
	return true
}

// A storage holding the zone information in memory only.
type zoneInventoryStorageMemory struct{}

// Instantiates the storage.
func newZoneInventoryStorageMemory() *zoneInventoryStorageMemory {
	return &zoneInventoryStorageMemory{}
}

// Stub function returning empty storage location.
func (storage zoneInventoryStorageMemory) getStorageLocation() string {
	return ""
}

// Indicates that the storage holds zone information in memory.
func (storage zoneInventoryStorageMemory) hasMemoryStorage() bool {
	return true
}

// Indicates that the storage does not hold zone information on disk.
func (storage zoneInventoryStorageMemory) hasDiskStorage() bool {
	return false
}

// Zone inventory state name type. This type is used instead of a string
// to ensure that the caller uses state names from the pool defined below.
// It ensures that the correct state name is used (e.g., prevents typos etc.).
type zoneInventoryStateName string

const (
	// Initial state: the zone information was never fetched from DNS server.
	zoneInventoryStateInitial zoneInventoryStateName = "INITIAL"
	// The inventory is reading the zones from disk into memory.
	zoneInventoryStateLoading zoneInventoryStateName = "LOADING"
	// The inventory finished reading the zones from disk.
	zoneInventoryStateLoaded zoneInventoryStateName = "LOADED"
	// Loading the inventory failed.
	zoneInventoryStateLoadingErred zoneInventoryStateName = "LOADING_ERRED"
	// The inventory is fetching the zones from a DNS server.
	zoneInventoryStatePopulating zoneInventoryStateName = "POPULATING"
	// The inventory finished fetching the zones from the DNS server.
	zoneInventoryStatePopulated zoneInventoryStateName = "POPULATED"
	// Fetching the zones from the DNS server and/or saving them failed.
	zoneInventoryStatePopulatingErred zoneInventoryStateName = "POPULATING_ERRED"
	// A caller is receiving zones from the inventory.
	zoneInventoryStateReceivingZones zoneInventoryStateName = "RECEIVING_ZONES"
	// A caller finished receiving the zones from the inventory.
	zoneInventoryStateReceivedZones zoneInventoryStateName = "RECEIVED_ZONES"
)

// Zone inventory state.
//
// The inventory is a state machine which internally keeps track
// of the operations it performs. It is important to track long
// lasting operations performed by the inventory to avoid collisions
// between the calls (e.g., populating the zones and returning the
// zones to the Stork server). By recognizing the current state the
// inventory may signal to the caller that selected operation may
// not be performed at the given time and should be retried later.
type zoneInventoryState struct {
	// State name.
	name zoneInventoryStateName
	// State creation time.
	createdAt time.Time
	// An error set for the states occurring after a failure.
	err error
}

// Creates selected inventory state.
func newZoneInventoryState(name zoneInventoryStateName, createdAt time.Time, err error) *zoneInventoryState {
	state := &zoneInventoryState{
		name:      name,
		err:       err,
		createdAt: time.Now(),
	}
	if !createdAt.IsZero() {
		state.createdAt = createdAt
	}
	return state
}

// Creates initial state.
func newZoneInventoryStateInitial() *zoneInventoryState {
	return newZoneInventoryState(zoneInventoryStateInitial, time.Time{}, nil)
}

// Creates loading state.
func newZoneInventoryStateLoading() *zoneInventoryState {
	return newZoneInventoryState(zoneInventoryStateLoading, time.Time{}, nil)
}

// Creates loaded state.
func newZoneInventoryStateLoaded(populatedAt time.Time) *zoneInventoryState {
	return newZoneInventoryState(zoneInventoryStateLoaded, populatedAt, nil)
}

// Creates loading erred state.
func newZoneInventoryStateLoadingErred(err error) *zoneInventoryState {
	return newZoneInventoryState(zoneInventoryStateLoadingErred, time.Time{}, err)
}

// Creates populating state.
func newZoneInventoryStatePopulating() *zoneInventoryState {
	return newZoneInventoryState(zoneInventoryStatePopulating, time.Time{}, nil)
}

// Creates populated state.
func newZoneInventoryStatePopulated() *zoneInventoryState {
	return newZoneInventoryState(zoneInventoryStatePopulated, time.Time{}, nil)
}

// Creates populating erred state.
func newZoneInventoryStatePopulatingErred(err error) *zoneInventoryState {
	return newZoneInventoryState(zoneInventoryStatePopulatingErred, time.Time{}, err)
}

// Creates receiving zones state.
func newZoneInventoryStateReceivingZones() *zoneInventoryState {
	return newZoneInventoryState(zoneInventoryStateReceivingZones, time.Time{}, nil)
}

// Creates received zones state.
func newZoneInventoryStateReceivedZones() *zoneInventoryState {
	return newZoneInventoryState(zoneInventoryStateReceivedZones, time.Time{}, nil)
}

// Checks if the state is INITIAL.
func (state zoneInventoryState) isInitial() bool {
	return state.name == zoneInventoryStateInitial
}

// Checks if the state is one of the long lasting operations.
func (state zoneInventoryState) isLongLasting() bool {
	switch state.name {
	case zoneInventoryStateLoading, zoneInventoryStatePopulating, zoneInventoryStateReceivingZones:
		return true
	default:
		return false
	}
}

// Checks if the state indicates that the inventory has been populated, loaded
// or received (zones).
func (state zoneInventoryState) isReady() bool {
	switch state.name {
	case zoneInventoryStateLoaded, zoneInventoryStatePopulated, zoneInventoryStateReceivedZones:
		return true
	default:
		return false
	}
}

// Checks if the state indicates that populating or loading the zones failed.
func (state zoneInventoryState) isErred() bool {
	switch state.name {
	case zoneInventoryStateLoadingErred, zoneInventoryStatePopulatingErred:
		return true
	default:
		return false
	}
}

// Metadata describing the zone inventory.
type ZoneInventoryMeta struct {
	// A timestamp when the zone inventory was populated.
	// It can be used to make a decision about updating the
	// inventory.
	PopulatedAt time.Time
}

// Zone inventory.
//
// It coordinates fetching the zone information from the monitored DNS servers,
// maintaining this information and exposing to the callers (typically Stork
// server). It runs long lasting operations in background and ensures that the
// conflicting calls cannot be invoked. Fetched zones can be stored in memory
// and/or on disk, depending on the configuration.
type zoneInventory struct {
	storage       zoneInventoryStorage
	client        zoneFetcher
	host          string
	port          int64
	state         zoneInventoryStateName
	visitedStates map[zoneInventoryStateName]*zoneInventoryState
	views         *bind9stats.Views
	mutex         sync.RWMutex
	wg            sync.WaitGroup
}

// A message sent over the channels to notify that the long lasting
// operation has completed. If the operation failed the err field
// contains the error.
type zoneInventoryAsyncNotify struct {
	err error
}

// A structure encapsulating a zone streamed by the inventory.
// It includes an optional err field which signals an error during
// the zone fetch (e.g., an IO error during zone read from disk).
type zoneInventoryReceiveZoneResult struct {
	zone *bind9stats.ExtendedZone
	err  error
}

// Instantiates the inventory. If the specified storage saves the zone information on
// disk this function prepares required directory structures. An error is returned if
// creating these structures fails.
func newZoneInventory(storage zoneInventoryStorage, client zoneFetcher, host string, port int64) (*zoneInventory, error) {
	if storage.hasDiskStorage() {
		// The inventory will store zone information on disk. Create the necessary
		// data structures.
		storageLocation := storage.getStorageLocation()
		fileInfo, err := os.Stat(storageLocation)
		switch {
		case err == nil:
			if !fileInfo.IsDir() {
				// The specified location exists but it is not a directory.
				return nil, errors.Errorf("failed to create zone inventory because %s is not a directory", storageLocation)
			}
		case errors.Is(err, os.ErrNotExist):
			// This directory does not exist. Try to create it.
			if err = os.MkdirAll(storageLocation, 0o755); err != nil {
				return nil, errors.Wrapf(err, "failed to create a zone inventory directory structure %s", storageLocation)
			}
		default:
			// Other error.
			return nil, errors.Wrapf(err, "failed to create zone inventory in %s", storageLocation)
		}
	}
	// Everything OK.
	state := newZoneInventoryStateInitial()
	return &zoneInventory{
		storage: storage,
		client:  client,
		host:    host,
		port:    port,
		state:   state.name,
		visitedStates: map[zoneInventoryStateName]*zoneInventoryState{
			zoneInventoryStateInitial: state,
		},
		mutex: sync.RWMutex{},
		wg:    sync.WaitGroup{},
		views: nil,
	}, nil
}

// Removes the inventory metadata file if it exists.
func (inventory *zoneInventory) removeMeta() error {
	metaFileName := path.Join(inventory.storage.getStorageLocation(), zoneInventoryMetaFileName)
	err := os.Remove(metaFileName)
	var pathError *fs.PathError
	if errors.As(err, &pathError) {
		// If the file doesn't exist it is not an error.
		return nil
	}
	return errors.Wrapf(err, "failed to remove the inventory metadata file %s", metaFileName)
}

// Saves the inventory metadata.
func (inventory *zoneInventory) saveMeta(meta *ZoneInventoryMeta) error {
	if !inventory.storage.hasDiskStorage() {
		return newZoneInventoryNoDiskStorageError()
	}
	metaFileName := path.Join(inventory.storage.getStorageLocation(), zoneInventoryMetaFileName)
	metaFile, err := os.OpenFile(metaFileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o640)
	if err != nil {
		return errors.Wrapf(err, "failed to save inventory metadata file %s", metaFileName)
	}
	defer metaFile.Close()
	encoder := json.NewEncoder(metaFile)
	err = encoder.Encode(meta)
	if err != nil {
		return errors.Wrapf(err, "failed to encode inventory metadata into file %s", metaFileName)
	}
	return nil
}

// Readds the inventory metadata.
func (inventory *zoneInventory) readMeta() (*ZoneInventoryMeta, error) {
	if !inventory.storage.hasDiskStorage() {
		return nil, newZoneInventoryNoDiskStorageError()
	}
	metaFileName := path.Join(inventory.storage.getStorageLocation(), zoneInventoryMetaFileName)
	content, err := os.ReadFile(metaFileName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "failed to read the inventory metadata file %s", metaFileName)
	}
	var meta ZoneInventoryMeta
	err = json.Unmarshal(content, &meta)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse the inventory metadata file %s", metaFileName)
	}
	return &meta, nil
}

// Transitions the inventory to a new state. It returns a zoneInventoryBusyError if the
// current state does not permit such a transition. A call to this function must be
// protected by mutex.
func (inventory *zoneInventory) transitionUnsafe(newState *zoneInventoryState) error {
	state := inventory.getCurrentStateUnsafe()
	if inventory.getCurrentStateUnsafe().isLongLasting() && newState.isLongLasting() {
		return newZoneInventoryBusyError(state, newState)
	}
	inventory.state = newState.name
	inventory.visitedStates[newState.name] = newState
	return nil
}

// Transitions the inventory to a new state. It is safe for concurrent use.
func (inventory *zoneInventory) transition(newState *zoneInventoryState) error {
	inventory.mutex.Lock()
	defer inventory.mutex.Unlock()
	return inventory.transitionUnsafe(newState)
}

// Transitions the inventory to a new state and sets views. A call to this
// function must be protected by mutex.
func (inventory *zoneInventory) transitionWithViewsUnsafe(newState *zoneInventoryState, views *bind9stats.Views) error {
	if err := inventory.transitionUnsafe(newState); err != nil {
		return err
	}
	inventory.views = views
	return nil
}

// Transitions the inventory to a new state, sets views and clears last
// error. It is safe for concurrent use.
func (inventory *zoneInventory) transitionWithViews(newState *zoneInventoryState, views *bind9stats.Views) error {
	inventory.mutex.Lock()
	defer inventory.mutex.Unlock()
	return inventory.transitionWithViewsUnsafe(newState, views)
}

// Returns current inventory state. A cal to this function must be protected
// by a mutex.
func (inventory *zoneInventory) getCurrentStateUnsafe() *zoneInventoryState {
	return inventory.visitedStates[inventory.state]
}

// Returns current inventory state. It is safe for concurrent use.
func (inventory *zoneInventory) getCurrentState() *zoneInventoryState {
	inventory.mutex.RLock()
	defer inventory.mutex.RUnlock()
	return inventory.getCurrentStateUnsafe()
}

// Returns visited inventory state by name. It returns nil if the state
// with this name has not been visited.
func (inventory *zoneInventory) getVisitedState(name zoneInventoryStateName) *zoneInventoryState {
	inventory.mutex.RLock()
	defer inventory.mutex.RUnlock()
	return inventory.visitedStates[name]
}

// Returns an iterator to the views. Depending on the type of the
// storage the views are returned from memory or disk.
func (inventory *zoneInventory) getViewsIterator(filter *bind9stats.ZoneFilter) iter.Seq2[bind9stats.ZoneIteratorAccessor, error] {
	return func(yield func(bind9stats.ZoneIteratorAccessor, error) bool) {
		switch {
		case inventory.storage.hasMemoryStorage():
			// If we have in-memory storage it is way more efficient
			// to use it than the disk storage.
			for _, view := range inventory.views.Views {
				if filter != nil && filter.View != nil && *filter.View != view.Name {
					continue
				}
				if !yield(view, nil) {
					return
				}
			}
		case inventory.storage.hasDiskStorage():
			// If there is no memory storage we have no other way but use
			// the disk storage.
			files, err := os.ReadDir(inventory.storage.getStorageLocation())
			if err != nil {
				err = errors.Wrapf(err, "failed to read view directory %s", inventory.storage.getStorageLocation())
				if !yield(nil, err) {
					return
				}
			}
			for _, file := range files {
				if !file.IsDir() || filter != nil && filter.View != nil && *filter.View != file.Name() {
					continue
				}
				vio := newViewIO(inventory.storage.getStorageLocation(), file.Name())
				if !yield(vio, nil) {
					return
				}
			}
		default:
			// An impossible condition.
			return
		}
	}
}

// Contacts a DNS server to fetch views and zones. Then, it processes the received
// data to group them into collections that are stored in memory and/or on disk.
// It returns a channel to which the caller can subscribe to receive a notification
// about completion of populating the inventory. The block parameter indicates whether
// or not the caller will wait for the completion notification.
func (inventory *zoneInventory) populate(block bool) (chan zoneInventoryAsyncNotify, error) {
	// Start populating the zones. This transition may fail if
	// there is another long lasting operation running.
	if err := inventory.transition(newZoneInventoryStatePopulating()); err != nil {
		return nil, err
	}
	var bufLen int
	if !block {
		// The channel must be buffered to not block write when nobody listens.
		bufLen = 1
	}
	notifyChannel := make(chan zoneInventoryAsyncNotify, bufLen)
	inventory.wg.Add(1)
	go func() {
		defer inventory.wg.Done()
		// Fetch views and zones from the DNS server.
		response, views, err := inventory.client.getViews(inventory.host, inventory.port)
		if err == nil {
			if response.IsError() {
				err = errors.Errorf("DNS server returned error status code %d with message: %s", response.StatusCode(), response.String())
			} else if inventory.storage.hasDiskStorage() {
				// The inventory has a persistent storage. Let's go over the
				// views and zones and store them.
				err = inventory.removeMeta()
				if err == nil {
					for _, view := range views.Views {
						vio := newViewIO(inventory.storage.getStorageLocation(), view.Name)
						err = vio.recreateView(view)
						if err != nil {
							break
						}
					}
					err = inventory.saveMeta(&ZoneInventoryMeta{
						PopulatedAt: time.Now().UTC(),
					})
				}
			}
		}
		if err == nil {
			log.WithFields(log.Fields{
				"zones": views.GetZoneCount(),
				"views": len(views.Views),
			}).Info("Populated DNS zones for indicated number views")
			if inventory.storage.hasMemoryStorage() {
				err = inventory.transitionWithViews(newZoneInventoryStatePopulated(), views)
			} else {
				err = inventory.transitionWithViews(newZoneInventoryStatePopulated(), nil)
			}
		} else {
			err = inventory.transition(newZoneInventoryStatePopulatingErred(err))
		}
		// We are done populating the views. Send notification and close the channel.
		notifyChannel <- zoneInventoryAsyncNotify{
			err,
		}
		close(notifyChannel)
	}()
	return notifyChannel, nil
}

// Loads zone inventory from view and zone information files. It returns a channel
// to which the caller can subscribe to receive a notification about completion of
// loading the inventory. The block parameter indicates whether or not the caller
// will wait for the completion notification.
func (inventory *zoneInventory) load(block bool) (chan zoneInventoryAsyncNotify, error) {
	if !inventory.storage.hasDiskStorage() {
		// Disk storage is required because we're going to load the data
		// from disk to memory.
		return nil, newZoneInventoryNoDiskStorageError()
	}
	// Start loading the zones. This transition may fail if there is
	// another long lasting operation running.
	if err := inventory.transition(newZoneInventoryStateLoading()); err != nil {
		return nil, err
	}
	// This flag will be needed in the goroutine to decide whether or
	// not to attempt to read the metadata file from the storage.
	hasMemoryStorage := inventory.storage.hasMemoryStorage()
	var bufLen int
	if !block {
		// The channel must be buffered to not block write when nobody listens.
		bufLen = 1
	}
	notifyChannel := make(chan zoneInventoryAsyncNotify, bufLen)
	inventory.wg.Add(1)
	go func() {
		defer inventory.wg.Done()
		// If there is no memory storage we merely need to check if the inventory
		// has been populated and saved on disk.
		if !hasMemoryStorage {
			meta, err := inventory.readMeta()
			if err != nil {
				_ = inventory.transition(newZoneInventoryStateLoadingErred(err))
			} else {
				_ = inventory.transition(newZoneInventoryStateLoaded(meta.PopulatedAt))
			}
			notifyChannel <- zoneInventoryAsyncNotify{
				err,
			}
			close(notifyChannel)
			return
		}
		// Store the viewList here.
		var viewList []*bind9stats.View
		// Get the list of files/directories.
		files, err := os.ReadDir(inventory.storage.getStorageLocation())
		if err == nil {
			for _, file := range files {
				if file.IsDir() {
					// If it is a directory it should hold the view information.
					vio := newViewIO(inventory.storage.getStorageLocation(), file.Name())
					view, err := vio.loadView()
					if err != nil {
						break
					}
					// View and its zones loaded from file. Let's store it in memory.
					viewList = append(viewList, view)
				}
			}
		}
		if err == nil {
			views := bind9stats.NewViews(viewList)
			log.WithFields(log.Fields{
				"zones": views.GetZoneCount(),
				"views": len(views.Views),
			}).Info("Loaded DNS zones for indicated number views")
			_ = inventory.transitionWithViews(newZoneInventoryStateLoaded(time.Now().UTC()), bind9stats.NewViews(viewList))
		} else {
			_ = inventory.transition(newZoneInventoryStateLoadingErred(err))
		}
		// We are done loading the views. Send notification and close the channel.
		notifyChannel <- zoneInventoryAsyncNotify{
			err,
		}
		close(notifyChannel)
	}()
	return notifyChannel, nil
}

// Returns the channel used to receive all zones and the corresponding
// views. This function is typically called in the implementation of the
// streaming gRPC response used to transfer the zone information to the
// server.
func (inventory *zoneInventory) receiveZones(ctx context.Context, filter *bind9stats.ZoneFilter) (chan zoneInventoryReceiveZoneResult, error) {
	var err error
	inventory.mutex.Lock()
	state := inventory.getCurrentStateUnsafe()
	if state.isInitial() || state.isErred() {
		err = newZoneInventoryNotInitedError()
	} else {
		// Make an unsafe transition because the mutex is already locked.
		err = inventory.transitionUnsafe(newZoneInventoryStateReceivingZones())
	}
	inventory.mutex.Unlock()
	if err != nil {
		return nil, err
	}
	channel := make(chan zoneInventoryReceiveZoneResult)
	go func() {
	OUTER_LOOP:
		for view, err := range inventory.getViewsIterator(filter) {
			if err != nil {
				result := zoneInventoryReceiveZoneResult{
					zone: nil,
					err:  err,
				}
				channel <- result
				// The caller can try another view.
				continue
			}
			zones := view.GetZoneIterator(filter)
			for zone, err := range zones {
				select {
				case <-ctx.Done():
					break OUTER_LOOP
				default:
					result := zoneInventoryReceiveZoneResult{}
					if err != nil {
						result.err = err
					} else {
						result.zone = &bind9stats.ExtendedZone{
							Zone:     *zone,
							ViewName: view.GetViewName(),
						}
					}
					channel <- result
				}
			}
		}
		_ = inventory.transition(newZoneInventoryStateReceivedZones())
		close(channel)
	}()
	return channel, nil
}

// Attempts to find zone information in the specified view. Depending on the
// inventory storage it finds the zone information in memory or reads it from
// disk.
func (inventory *zoneInventory) getZoneInView(viewName, zoneName string) (*bind9stats.Zone, error) {
	state := inventory.getCurrentState()
	if state.isInitial() || state.isErred() {
		return nil, newZoneInventoryNotInitedError()
	}
	if inventory.storage.hasMemoryStorage() {
		// If the inventory has a memory storage, let's use it to get the zone info.
		inventory.mutex.RLock()
		views := inventory.views
		inventory.mutex.RUnlock()
		if views != nil {
			view := views.GetView(viewName)
			if view != nil {
				zone := view.GetZone(zoneName)
				return zone, nil
			}
		}
	} else if inventory.storage.hasDiskStorage() {
		// If disk storage available, read the zone information from file.
		vio := newViewIO(inventory.storage.getStorageLocation(), viewName)
		zone, err := vio.loadZone(zoneName)
		if err != nil {
			return nil, err
		}
		return zone, nil
	}
	return nil, nil
}

// This function waits for the asynchronous operations to complete.
func (inventory *zoneInventory) shutdown() {
	inventory.wg.Wait()
}

// Exposes IO operations to manipulate the zone views.
type viewIO struct {
	// View name.
	viewName string
	// Holds the path to the view directory.
	viewLocation string
}

// Instantiates new IO. The first parameter is a path to the inventory
// storage. The second parameter is a directory where view-specific files
// are held.
func newViewIO(storageLocation string, viewName string) *viewIO {
	return &viewIO{
		viewName:     viewName,
		viewLocation: path.Join(storageLocation, viewName),
	}
}

// Returns view name.
func (vio *viewIO) GetViewName() string {
	return vio.viewName
}

// Returns iterator to zones in the view.
func (vio *viewIO) GetZoneIterator(filter *bind9stats.ZoneFilter) iter.Seq2[*bind9stats.Zone, error] {
	return func(yield func(*bind9stats.Zone, error) bool) {
		files, err := os.ReadDir(vio.viewLocation)
		if err != nil {
			_ = yield(nil, err)
			return
		}
		slices.SortFunc(files, func(file1, file2 os.DirEntry) int {
			return storkutil.CompareNames(file1.Name(), file2.Name())
		})
		files = bind9stats.ApplyZoneLowerBoundFilter(files, filter)

		var count int
		for _, file := range files {
			content, err := os.ReadFile(path.Join(vio.viewLocation, file.Name()))
			if err != nil {
				if !yield(nil, err) {
					return
				}
				continue
			}
			var zone bind9stats.Zone
			err = json.Unmarshal(content, &zone)
			if err != nil {
				if !yield(nil, err) {
					return
				}
				continue
			}
			if filter != nil {
				if filter.LoadedAfter != nil && !zone.Loaded.After(*filter.LoadedAfter) {
					continue
				}
				if filter.Limit != nil {
					count++
					if count > *filter.Limit {
						return
					}
				}
			}
			if !yield(&zone, nil) {
				return
			}
		}
	}
}

// Removes the view directory.
func (vio *viewIO) removeView() error {
	err := os.RemoveAll(vio.viewLocation)
	return errors.Wrapf(err, "failed to remove view directory %s", vio.viewLocation)
}

// Creates the view directory and writes zone information files in this
// directory.
func (vio *viewIO) createView(view *bind9stats.View) error {
	fileInfo, err := os.Stat(vio.viewLocation)
	switch {
	case errors.Is(err, os.ErrNotExist):
		if err = os.Mkdir(vio.viewLocation, 0o700); err != nil {
			return errors.Wrapf(err, "failed to create zone view directory %s", vio.viewLocation)
		}
	case err == nil:
		if !fileInfo.IsDir() {
			return errors.Wrapf(err, "failed to save the zone view %s because %s is not a directory", view.Name, vio.viewLocation)
		}
	default:
		return errors.Wrap(err, "failed to save the zone view %s")
	}
	zones := view.GetZones()
	var count int
	for _, zone := range zones {
		err = vio.createZone(zone)
		if err != nil {
			return err
		}
		count++
	}
	return nil
}

// Recreates the view directory and creates the zone information files in it.
func (vio *viewIO) recreateView(view *bind9stats.View) (err error) {
	err = vio.removeView()
	if err == nil {
		err = vio.createView(view)
	}
	return
}

// Creates a zone information file in the view directory.
func (vio *viewIO) createZone(zone *bind9stats.Zone) error {
	zoneDataFilePath := path.Join(vio.viewLocation, zone.Name())
	zoneDataFile, err := os.OpenFile(zoneDataFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o640)
	if err != nil {
		return errors.Wrapf(err, "failed to save data for zone %s due to an error while opening the file %s", zone.Name(), zoneDataFilePath)
	}
	defer zoneDataFile.Close()
	encoder := json.NewEncoder(zoneDataFile)
	err = encoder.Encode(zone)
	if err != nil {
		return errors.Wrapf(err, "failed to encode the zone data into file %s", zoneDataFilePath)
	}
	return nil
}

// Reads the view and the corresponding zones from a file.
func (vio *viewIO) loadView() (*bind9stats.View, error) {
	files, err := os.ReadDir(vio.viewLocation)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read files while loading the zone view inventory in %s", vio.viewLocation)
	}
	var zones []*bind9stats.Zone
	for _, file := range files {
		if !file.IsDir() {
			content, err := os.ReadFile(path.Join(vio.viewLocation, file.Name()))
			if err != nil {
				continue
			}
			var zone bind9stats.Zone
			err = json.Unmarshal(content, &zone)
			if err != nil {
				continue
			}
			zones = append(zones, &zone)
		}
	}
	view := bind9stats.NewView(vio.viewName, zones)
	return view, nil
}

// Reads information about the specified zone from a file.
func (vio *viewIO) loadZone(zoneName string) (*bind9stats.Zone, error) {
	content, err := os.ReadFile(path.Join(vio.viewLocation, zoneName))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "failed to read the inventory file for zone %s", zoneName)
	}
	var zone bind9stats.Zone
	err = json.Unmarshal(content, &zone)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse the inventory file for zone %s", zoneName)
	}
	return &zone, nil
}
