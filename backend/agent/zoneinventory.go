package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"iter"
	"os"
	"path"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
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
	_ zoneInventoryAXFRExecutor       = (*zoneInventoryAXFRExecutorImpl)(nil)
	_ bind9stats.ZoneIteratorAccessor = (*viewIO)(nil)
	_ bind9stats.NameAccessor         = (os.DirEntry)(nil)
)

// A file name holding zone inventory meta data.
const zoneInventoryMetaFileName = "zone-inventory.json"

// An interface to the DNS server configuration. It is DNS implementation agnostic.
// It returns the configuration elements required by the zone inventory. For example,
// it returns the elements required to perform zone transfers.
type dnsConfigAccessor interface {
	GetAXFRCredentials(viewName string, zoneName string) (address *string, keyName *string, algorithm *string, secret *string, err error)
	GetAPIKey() string
}

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
	getViews(apiKey string, host string, port int64) (httpResponse, *bind9stats.Views, error)
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

// An error indicating that the zone transfer is currently not possible because
// the zone inventory is performing a long lasting operation.
type zoneInventoryAXFRBusyError struct {
	currState *zoneInventoryState
}

// Instantiates the error. The function parameters specify the current inventory
// state.
func newZoneInventoryAXFRBusyError(currState *zoneInventoryState) error {
	return &zoneInventoryAXFRBusyError{currState}
}

// Returns error string. It indicates the current inventory state.
func (e zoneInventoryAXFRBusyError) Error() string {
	return fmt.Sprintf("zone transfer is not possible because the zone inventory is in %s state", e.currState.name)
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
// This interface is a layer between the zone inventory and the actual storage
// for fetching views and zones. Concrete implementations save and read the zones
// from memory, persistent storage or both.
type zoneInventoryStorage interface {
	// Returns an iterator to the views in the storage.
	getViewsIterator(filter *bind9stats.ZoneFilter) iter.Seq2[bind9stats.ZoneIteratorAccessor, error]
	// Returns a zone within a specified view.
	getZoneInView(viewName, zoneName string) (*bind9stats.Zone, error)
	// Loads views and zones from the storage making them accessible for reading.
	loadViews() (time.Time, error)
	// Saves views and zones in the storage.
	saveViews(views *bind9stats.Views) error
}

// A storage holding the zone information in memory and on disk.
// This storage uses two other storage types internally (i.e., in-memory
// and disk storage).
type zoneInventoryStorageMemoryDisk struct {
	disk   *zoneInventoryStorageDisk
	memory *zoneInventoryStorageMemory
}

// Instantiates the storage. The parameter specifies the disk storage path.
// Instantiating this storage may fail and return an error if instantiating
// the underlying disk storage fails.
func newZoneInventoryStorageMemoryDisk(location string) (*zoneInventoryStorageMemoryDisk, error) {
	disk, err := newZoneInventoryStorageDisk(location)
	if err != nil {
		return nil, err
	}
	return &zoneInventoryStorageMemoryDisk{
		disk:   disk,
		memory: newZoneInventoryStorageMemory(),
	}, nil
}

// Returns an iterator to the views in the storage. This access to the
// views is fast because they are read from the in-memory storage.
func (storage *zoneInventoryStorageMemoryDisk) getViewsIterator(filter *bind9stats.ZoneFilter) iter.Seq2[bind9stats.ZoneIteratorAccessor, error] {
	return storage.memory.getViewsIterator(filter)
}

// Returns a selected zone from a view. The access to the data is fast
// because it is read from the in-memory storage.
func (storage *zoneInventoryStorageMemoryDisk) getZoneInView(viewName, zoneName string) (*bind9stats.Zone, error) {
	return storage.memory.getZoneInView(viewName, zoneName)
}

// Reads the views and zones from disk into memory. If loading is successful
// the views and zones can be efficiently accessed from the in-memory storage.
func (storage *zoneInventoryStorageMemoryDisk) loadViews() (time.Time, error) {
	// Begin with loading the inventory population time from the disk storage.
	populatedAt, err := storage.disk.loadViews()
	if err != nil {
		return populatedAt, err
	}
	var viewList []*bind9stats.View
	// Get the list of files/directories.
	files, err := os.ReadDir(storage.disk.location)
	if err != nil {
		return time.Time{}, err
	}
	for _, file := range files {
		if file.IsDir() {
			// If it is a directory it should hold the view information.
			vio := newViewIO(storage.disk.location, file.Name())
			view, err := vio.loadView()
			if err != nil {
				return time.Time{}, err
			}
			// View and its zones loaded from file. Let's store it in memory.
			viewList = append(viewList, view)
		}
	}
	views := bind9stats.NewViews(viewList)

	log.WithFields(log.Fields{
		"zones": views.GetZoneCount(),
		"views": len(views.Views),
	}).Info("Loaded DNS zones for indicated number views")

	storage.memory.mutex.Lock()
	defer storage.memory.mutex.Unlock()
	storage.memory.views = views
	return populatedAt, nil
}

// Saves specified views on disk and in-memory effectively replacing the
// entire inventory.
func (storage *zoneInventoryStorageMemoryDisk) saveViews(views *bind9stats.Views) error {
	for _, s := range []zoneInventoryStorage{storage.disk, storage.memory} {
		if err := s.saveViews(views); err != nil {
			return err
		}
	}
	return nil
}

// A storage holding the zone information on disk.
type zoneInventoryStorageDisk struct {
	// Path to the disk storage.
	location string
}

// Instantiates the storage. The parameter specifies the disk storage path.
// Instantiating the storage  return an error when disk IO operations fail.
// This is the case when the specified location is not a directory, doesn't
// exist, or creating the inventory directory structure fails for any other
// reason.
func newZoneInventoryStorageDisk(location string) (*zoneInventoryStorageDisk, error) {
	// The inventory will store zone information on disk. Create the necessary
	// data structures.
	fileInfo, err := os.Stat(location)
	switch {
	case err == nil:
		if !fileInfo.IsDir() {
			// The specified location exists but it is not a directory.
			return nil, errors.Errorf("failed to create zone inventory because %s is not a directory", location)
		}
	case errors.Is(err, os.ErrNotExist):
		// This directory does not exist. Try to create it.
		if err = os.MkdirAll(location, 0o755); err != nil {
			return nil, errors.Wrapf(err, "failed to create a zone inventory directory structure %s", location)
		}
	default:
		// Other error.
		return nil, errors.Wrapf(err, "failed to create zone inventory in %s", location)
	}

	return &zoneInventoryStorageDisk{
		location,
	}, nil
}

// Returns an iterator to the views in the storage. This access to the
// views is slower than in case of the zoneInventoryStorageMemoryDisk
// storage because the iterator reads the views and zones from disk
// while the caller iterates over the returned views.
func (storage *zoneInventoryStorageDisk) getViewsIterator(filter *bind9stats.ZoneFilter) iter.Seq2[bind9stats.ZoneIteratorAccessor, error] {
	return func(yield func(bind9stats.ZoneIteratorAccessor, error) bool) {
		files, err := os.ReadDir(storage.location)
		if err != nil {
			err = errors.Wrapf(err, "failed to read view directory %s", storage.location)
			if !yield(nil, err) {
				return
			}
		}
		for _, file := range files {
			if !file.IsDir() || filter != nil && filter.View != nil && *filter.View != file.Name() {
				continue
			}
			vio := newViewIO(storage.location, file.Name())
			if !yield(vio, nil) {
				return
			}
		}
	}
}

// Returns a selected zone from a view. The access to the data is slower
// than for other storage types because it is searched and read from disk.
func (storage *zoneInventoryStorageDisk) getZoneInView(viewName, zoneName string) (*bind9stats.Zone, error) {
	vio := newViewIO(storage.location, viewName)
	zone, err := vio.loadZone(getViewIOZoneFileName(zoneName))
	if err != nil {
		return nil, err
	}
	return zone, nil
}

// Reads the time when the inventory was populated and returns. Unlike other
// storages it doesn't read the zones from disk because it lacks in-memory
// storage for zones. The views and zones are read from disk on demand using
// iterators.
func (storage *zoneInventoryStorageDisk) loadViews() (time.Time, error) {
	meta, err := storage.readMeta()
	if err != nil {
		return time.Time{}, err
	}
	return meta.PopulatedAt, nil
}

// Saves specified views on disk replacing the entire inventory.
func (storage *zoneInventoryStorageDisk) saveViews(views *bind9stats.Views) error {
	err := storage.removeMeta()
	if err != nil {
		return err
	}
	for _, view := range views.Views {
		vio := newViewIO(storage.location, view.Name)
		err = vio.recreateView(view)
		if err != nil {
			break
		}
	}
	err = storage.saveMeta(&ZoneInventoryMeta{
		PopulatedAt: time.Now().UTC(),
	})
	return err
}

// Removes the inventory metadata file if it exists.
func (storage *zoneInventoryStorageDisk) removeMeta() error {
	metaFileName := path.Join(storage.location, zoneInventoryMetaFileName)
	err := os.Remove(metaFileName)
	var pathError *fs.PathError
	if errors.As(err, &pathError) {
		// If the file doesn't exist it is not an error.
		return nil
	}
	return errors.Wrapf(err, "failed to remove the inventory metadata file %s", metaFileName)
}

// Saves the inventory metadata.
func (storage *zoneInventoryStorageDisk) saveMeta(meta *ZoneInventoryMeta) error {
	metaFileName := path.Join(storage.location, zoneInventoryMetaFileName)
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

// Reads the inventory metadata.
func (storage *zoneInventoryStorageDisk) readMeta() (*ZoneInventoryMeta, error) {
	metaFileName := path.Join(storage.location, zoneInventoryMetaFileName)
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

// A storage holding the zone information in memory only.
type zoneInventoryStorageMemory struct {
	mutex sync.RWMutex
	views *bind9stats.Views
}

// Instantiates the storage.
func newZoneInventoryStorageMemory() *zoneInventoryStorageMemory {
	return &zoneInventoryStorageMemory{
		mutex: sync.RWMutex{},
	}
}

// Returns an iterator to the views in the storage. This access to the
// views is fast because they are read from the in-memory storage.
func (storage *zoneInventoryStorageMemory) getViewsIterator(filter *bind9stats.ZoneFilter) iter.Seq2[bind9stats.ZoneIteratorAccessor, error] {
	return func(yield func(bind9stats.ZoneIteratorAccessor, error) bool) {
		if storage.views == nil {
			return
		}
		for _, view := range storage.views.Views {
			if filter != nil && filter.View != nil && *filter.View != view.Name {
				continue
			}
			if !yield(view, nil) {
				return
			}
		}
	}
}

// Returns a selected zone from a view. The access to the data is fast
// because it is read from the in-memory storage.
func (storage *zoneInventoryStorageMemory) getZoneInView(viewName, zoneName string) (*bind9stats.Zone, error) {
	storage.mutex.RLock()
	views := storage.views
	storage.mutex.RUnlock()
	if views != nil {
		view := views.GetView(viewName)
		if view != nil {
			zone := view.GetZone(zoneName)
			return zone, nil
		}
	}
	return nil, nil
}

// Always returns zoneInventoryNoDiskStorageError because there is no persistent
// storage to read the views from.
func (storage *zoneInventoryStorageMemory) loadViews() (time.Time, error) {
	return time.Time{}, newZoneInventoryNoDiskStorageError()
}

// Replaces the inventory with a new collection of views.
func (storage *zoneInventoryStorageMemory) saveViews(views *bind9stats.Views) error {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()
	storage.views = views
	return nil
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

// A structure encapsulating a request to perform AXFR (zone transfer)
// from the DNS server.
type zoneInventoryAXFRRequest struct {
	zoneName  string
	keyName   *string
	algorithm *string
	secret    *string
	address   string
	respChan  chan *zoneInventoryAXFRResponse
	closeOnce sync.Once
}

// Instantiates a new AXFR request. It ensures that the zone name, key name
// and algorithm are fully qualified, as required by the dns package.
func newZoneInventoryAXFRRequest(zoneName string, keyName *string, algorithm *string, secret *string, address string) *zoneInventoryAXFRRequest {
	return &zoneInventoryAXFRRequest{
		zoneName:  zoneName,
		keyName:   keyName,
		algorithm: algorithm,
		secret:    secret,
		address:   address,
		respChan:  make(chan *zoneInventoryAXFRResponse),
		closeOnce: sync.Once{},
	}
}

// Closes the response channel. Normally, the channel is closed by the
// zone inventory. However, it is safe to call this function multiple
// times, so the caller can also attempt to close the channel.
func (request *zoneInventoryAXFRRequest) safeClose() {
	request.closeOnce.Do(func() {
		close(request.respChan)
	})
}

// A structure encapsulating a response to AXFR request. It contains
// an envelope holding a set of RRs. If it contains RRs the error field
// is nil. Otherwise, if there is an error during the transfer, the err
// field contains the error. In this case, the envelope is nil.
type zoneInventoryAXFRResponse struct {
	envelope *dns.Envelope
	err      error
}

// An interface allowing for mocking the zone transfer execution.
// The zoneInventoryAXFRExecutorImpl is a default implementation
// using the dns library. The unit tests can provide a custom
// implementation for testing purposes.
type zoneInventoryAXFRExecutor interface {
	run(transfer *dns.Transfer, message *dns.Msg, address string) (chan *dns.Envelope, error)
}

// The default implementation of the zoneInventoryAXFRExecutor interface.
// It uses the dns library to execute the zone transfer.
type zoneInventoryAXFRExecutorImpl struct{}

// Executes the zone transfer using the dns library.
func (impl *zoneInventoryAXFRExecutorImpl) run(transfer *dns.Transfer, message *dns.Msg, address string) (chan *dns.Envelope, error) {
	return transfer.In(message, address)
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
//
// Please consult https://gitlab.isc.org/isc-projects/stork/-/wikis/designs/bind-zone-view#zone-inventory
// for the state diagram.
//
// The zone inventory is responsible for fetching the zone contents from the
// DNS server using AXFR. It makes the zone contents available to the callers.
// Specifically, it returns the RRs to the Stork server via the gRPC stream.
// The zone contents may be cached in the zone inventory to avoid excessive
// DNS requests.
type zoneInventory struct {
	// The storage used to save the zone information.
	storage zoneInventoryStorage
	// Common interface to access DNS configuration from different DNS implementations.
	config dnsConfigAccessor
	// Common interface used to fetch the zone lists from different DNS implementations.
	client zoneFetcher
	// The host name of the DNS server.
	host string
	// The DNS server port.
	port int64
	// The current state of the zone inventory.
	state zoneInventoryStateName
	// The map of visited states holding errors that occurred in these states.
	visitedStates map[zoneInventoryStateName]*zoneInventoryState
	// Mutex protecting the inventory state.
	mutex sync.RWMutex
	// Wait group used to wait for the background tasks to complete.
	wg sync.WaitGroup
	// Pool of workers used to run AXFR requests. It guarantees that
	// only a limited number of AXFR requests are executed concurrently.
	axfrPool *storkutil.PausablePool
	// Channel used to schedule and run AXFR requests in order.
	axfrReqChan chan *zoneInventoryAXFRRequest
	// Cancel function used to cancel pending AXFR requests.
	axfrReqCancel context.CancelFunc
	// A wrapper receiving a transfer, message and address and executing
	// the zone transfer. By default, it uses the dns library but can be
	// replaced with a custom implementation for mocking purposes.
	axfrExecutor zoneInventoryAXFRExecutor
	// A flag indicating if the AXFR workers are active.
	axfrWorkersActive bool
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
func newZoneInventory(storage zoneInventoryStorage, config dnsConfigAccessor, client zoneFetcher, host string, port int64) *zoneInventory {
	ctx, cancel := context.WithCancel(context.Background())
	state := newZoneInventoryStateInitial()
	inventory := &zoneInventory{
		storage: storage,
		config:  config,
		client:  client,
		host:    host,
		port:    port,
		state:   state.name,
		visitedStates: map[zoneInventoryStateName]*zoneInventoryState{
			zoneInventoryStateInitial: state,
		},
		mutex:             sync.RWMutex{},
		wg:                sync.WaitGroup{},
		axfrReqChan:       make(chan *zoneInventoryAXFRRequest),
		axfrReqCancel:     cancel,
		axfrExecutor:      &zoneInventoryAXFRExecutorImpl{},
		axfrPool:          storkutil.NewPausablePool(runtime.GOMAXPROCS(0) * 2),
		axfrWorkersActive: false,
	}
	// Start the workers performing AXFR requests.
	inventory.startAXFRWorkers(ctx)
	return inventory
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
	// No zone transfers during zones population.
	if err := inventory.axfrPool.Pause(); err != nil {
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
		response, views, err := inventory.client.getViews(inventory.config.GetAPIKey(), inventory.host, inventory.port)
		if err == nil {
			if response.IsError() {
				err = errors.Errorf("DNS server returned error status code %d with message: %s", response.StatusCode(), response.String())
			} else {
				err = inventory.storage.saveViews(views)
			}
		}
		if err == nil {
			log.WithFields(log.Fields{
				"zones": views.GetZoneCount(),
				"views": len(views.Views),
			}).Info("Populated DNS zones for indicated number views")
			err = inventory.transition(newZoneInventoryStatePopulated())
		} else {
			err = inventory.transition(newZoneInventoryStatePopulatingErred(err))
		}
		// Resume paused workers.
		if errResume := inventory.axfrPool.Resume(); err != nil {
			log.WithError(errResume).Warnf("Failed to resume AXFR worker pool")
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
	if _, ok := inventory.storage.(*zoneInventoryStorageMemory); ok {
		// Disk storage is required because we're going to load the data
		// from disk to memory.
		return nil, newZoneInventoryNoDiskStorageError()
	}
	// Start loading the zones. This transition may fail if there is
	// another long lasting operation running.
	if err := inventory.transition(newZoneInventoryStateLoading()); err != nil {
		return nil, err
	}
	// No zone transfers during zones loading.
	if err := inventory.axfrPool.Pause(); err != nil {
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
		defer func() {
			inventory.wg.Done()
		}()
		populatedAt, err := inventory.storage.loadViews()
		if err != nil {
			_ = inventory.transition(newZoneInventoryStateLoadingErred(err))
		} else {
			_ = inventory.transition(newZoneInventoryStateLoaded(populatedAt))
		}
		// Resume paused workers.
		if errResume := inventory.axfrPool.Resume(); err != nil {
			log.WithError(errResume).Warnf("Failed to resume AXFR worker pool")
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
	var totalZoneCount int64
	for view, err := range inventory.storage.getViewsIterator(filter) {
		if err != nil {
			return nil, err
		}
		zoneCount, err := view.GetZoneCount()
		if err != nil {
			return nil, err
		}
		totalZoneCount += zoneCount
	}
	channel := make(chan zoneInventoryReceiveZoneResult)
	// No zone transfers during zones receiving.
	if err := inventory.axfrPool.Pause(); err != nil {
		return nil, err
	}
	go func() {
	OUTER_LOOP:
		for view, err := range inventory.storage.getViewsIterator(filter) {
			if err != nil {
				channel <- zoneInventoryReceiveZoneResult{
					err: err,
				}
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
							Zone:           *zone,
							ViewName:       view.GetViewName(),
							TotalZoneCount: totalZoneCount,
						}
					}
					channel <- result
				}
			}
		}
		// Resume paused workers.
		if errResume := inventory.axfrPool.Resume(); errResume != nil {
			log.WithError(errResume).Warnf("Failed to resume AXFR worker pool")
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
	return inventory.storage.getZoneInView(viewName, zoneName)
}

// Requests an AXFR (zone transfer) for the specified zone and view.
// It returns a channel to which the caller must subscribe to receive
// the AXFR results. The channel is closed by the zone inventory when
// the transfer is complete.
func (inventory *zoneInventory) requestAXFR(zoneName, viewName string) (chan *zoneInventoryAXFRResponse, error) {
	address, keyName, algorithm, secret, err := inventory.config.GetAXFRCredentials(viewName, zoneName)
	if err != nil {
		return nil, err
	}
	// Create and queue the request. The request is picked by one of the
	// workers and executed.
	request := newZoneInventoryAXFRRequest(zoneName, keyName, algorithm, secret, *address)
	inventory.axfrReqChan <- request
	return request.respChan, nil
}

// Performs the zone transfer from the DNS server. The request contains
// the response channel to which the results are sent. The caller must read
// from the channel until it is closed.
func (inventory *zoneInventory) runAXFR(request *zoneInventoryAXFRRequest) {
	// Close the response channel when transfer done.
	defer request.safeClose()

	// Check if the zone inventory is not busy doing a long lasting operation.
	// We're checking it here because this function is called in a worker
	// goroutine where it is guaranteed there is no race condition between
	// the task and the possibly new request to perform a long lasting operation.
	// Such a request will wait for the current tasks to complete.
	state := inventory.getCurrentState()
	if state.isLongLasting() {
		request.respChan <- &zoneInventoryAXFRResponse{
			err: newZoneInventoryAXFRBusyError(state),
		}
		return
	}
	// If the zone inventory is in an erred state, we can't perform the zone
	// transfer.
	if state.isInitial() || state.isErred() {
		request.respChan <- &zoneInventoryAXFRResponse{
			err: newZoneInventoryNotInitedError(),
		}
		return
	}

	transfer := new(dns.Transfer)
	message := new(dns.Msg)

	// Set TSIG if provided. In some cases the TSIG is not used. Typically when
	// there is only a default view.
	if request.keyName != nil && request.secret != nil {
		transfer.TsigSecret = map[string]string{
			storkutil.FullyQualifyName(*request.keyName): *request.secret,
		}
	}
	message.SetAxfr(storkutil.FullyQualifyName(request.zoneName))

	// Again, set TSIG if provided.
	if request.keyName != nil && request.algorithm != nil {
		message.SetTsig(storkutil.FullyQualifyName(*request.keyName), storkutil.FullyQualifyName(*request.algorithm), 300, time.Now().Unix())
	}

	// Perform the zone transfer.
	channel, err := inventory.axfrExecutor.run(transfer, message, request.address)
	if err != nil {
		request.respChan <- &zoneInventoryAXFRResponse{
			err: errors.WithMessagef(err, "Failed to transfer DNS zone %s from %s", request.zoneName, request.address),
		}
		return
	}
	// Send the RRs to the caller over the channel.
	for envelope := range channel {
		request.respChan <- &zoneInventoryAXFRResponse{
			envelope: envelope,
		}
	}
}

// Starts a pool of workers for running AXFR requests. This function must be
// called only once. It is called internally during the zone inventory
// initialization.
func (inventory *zoneInventory) startAXFRWorkers(ctx context.Context) {
	inventory.axfrWorkersActive = true
	go func() {
		defer func() {
			// Ensure that the channel is closed.
			close(inventory.axfrReqChan)
		}()
		for {
			select {
			case <-ctx.Done():
				// The context is cancelled when the zone inventory is destroyed.
				return
			case request, ok := <-inventory.axfrReqChan:
				if !ok {
					return
				}
				err := inventory.axfrPool.Submit(func() {
					// Schedule the AXFR request for execution.
					inventory.runAXFR(request)
				})
				if err != nil {
					// We want to be more specific with the error that the zone inventory
					// is busy performing a long lasting operation.
					var pausablePoolPausedError *storkutil.PausablePoolPausedError
					if errors.As(err, &pausablePoolPausedError) {
						err = newZoneInventoryAXFRBusyError(inventory.getCurrentState())
					}
					// Failed to submit the AXFR request to the worker pool.
					// The pool is possibly exhausted.
					log.WithError(err).
						WithField("zone", request.zoneName).
						Warnf("Failed to submit AXFR request to the worker pool")

					if request.respChan != nil {
						request.respChan <- &zoneInventoryAXFRResponse{
							err: errors.WithMessage(err, "failed to submit AXFR request to the worker pool"),
						}
						// Close the response channel to unblock the caller.
						request.safeClose()
					}
				}
			}
		}
	}()
}

// Stops the AXFR workers and waits for them to complete their tasks.
func (inventory *zoneInventory) stopAXFRWorkers() {
	// Stop the pool first to ensure that no new tasks are accepted.
	inventory.axfrPool.Stop()
	// Cancel the context to unblock the pending tasks.
	inventory.axfrReqCancel()
	// Set the flag to false to indicate that the workers are not active anymore.
	inventory.axfrWorkersActive = false
}

// Returns a flag indicating if the AXFR workers are active.
func (inventory *zoneInventory) isAXFRWorkersActive() bool {
	return inventory.axfrWorkersActive
}

// This function waits for the asynchronous operations to complete.
func (inventory *zoneInventory) awaitBackgroundTasks() {
	inventory.wg.Wait()
}

// Stops the zone inventory and waits for the background tasks to complete.
func (inventory *zoneInventory) stop() {
	inventory.stopAXFRWorkers()
	inventory.awaitBackgroundTasks()
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
			_ = yield(nil, errors.Wrapf(err, "failed to read the DNS view directory %s", vio.viewLocation))
			return
		}
		// The files must be sorted in DNS order.
		slices.SortFunc(files, func(file1, file2 os.DirEntry) int {
			return storkutil.CompareNames(file1.Name(), file2.Name())
		})
		files = bind9stats.ApplyZoneLowerBoundFilter(files, filter)
		var count int
		for _, file := range files {
			filePath := path.Join(vio.viewLocation, file.Name())
			// Read the JSON file with zone information.
			content, err := os.ReadFile(filePath)
			if err != nil {
				if !yield(nil, errors.Wrapf(err, "failed to read file containing zone information %s", filePath)) {
					return
				}
				continue
			}
			// Parse the JSON file with zone information.
			var zone bind9stats.Zone
			err = json.Unmarshal(content, &zone)
			if err != nil {
				if !yield(nil, errors.Wrapf(err, "failed to parse file containing zone information %s", filePath)) {
					return
				}
				continue
			}
			// Filter out unwanted zones.
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

// Returns number of zones for a given view name.
func (vio *viewIO) GetZoneCount() (int64, error) {
	files, err := os.ReadDir(vio.viewLocation)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to read the DNS view directory %s", vio.viewLocation)
	}
	return int64(len(files)), nil
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
	zoneDataFilePath := path.Join(vio.viewLocation, getViewIOZoneFileName(zone.Name()))
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

// Returns the file name for the zone information file.
func getViewIOZoneFileName(zoneName string) string {
	zoneName = strings.TrimSpace(zoneName)
	if zoneName == "." {
		zoneName = "root"
	}
	return storkutil.FullyQualifyName(zoneName) + "zone"
}
