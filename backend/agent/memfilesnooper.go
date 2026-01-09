package agent

import (
	"encoding/csv"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"isc.org/stork/daemondata/kea"
	"isc.org/stork/datamodel/daemonname"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	// IPv4 Lease Memfile Column Indices below.
	v4IPAddr        = 0
	v4HWAddr        = 1
	v4ValidLifetime = 3
	v4Expire        = 4
	v4Subnet        = 5
	v4State         = 9
	// IPv6 Lease Memfile Column Indices below.
	v6IPAddr        = 0
	v6DUID          = 1
	v6ValidLifetime = 2
	v6Expire        = 3
	v6Subnet        = 4
	v6Prefix        = 8
	v6State         = 13
)

// How does this module work?
// Users of this module are expected to create a RowSource, then feed its output one-at-a-time into ParseRowAsLease4 or ParseRowAsLease6.
// This creates a pipeline of goroutines connected with channels which looks like this:
//
// +----------+   +-----------+   +----------+
// | FsNotify +-->| RowSource +-->| (caller) |
// +----------+   +-----------+   +----------+

// Create a new Lease4 from a CSV row, and the already-parsed values.
func newLease4(record []string, cltt uint64, lifetime uint32) (*keadata.Lease, error) {
	subnet64, err := strconv.ParseUint(record[v4Subnet], 10, 32)
	if err != nil {
		return nil, errors.Wrap(err, "the subnet ID is not valid")
	}
	subnet := uint32(subnet64)
	state, err := strconv.Atoi(record[v4State])
	if err != nil {
		return nil, errors.Wrap(err, "the lease state is not valid")
	}
	lease := keadata.NewLease4(
		record[v4IPAddr],
		record[v4HWAddr],
		cltt,
		lifetime,
		subnet,
		state,
	)
	return &lease, nil
}

// Create a new Lease6 from a CSV row, and the already-parsed values.
func newLease6(record []string, cltt uint64, lifetime uint32) (*keadata.Lease, error) {
	subnet64, err := strconv.Atoi(record[v6Subnet])
	if err != nil {
		return nil, errors.Wrap(err, "the subnet ID is not valid")
	}
	subnet := uint32(subnet64)
	state, err := strconv.Atoi(record[v6State])
	if err != nil {
		return nil, errors.Wrap(err, "the lease state is not valid")
	}
	prefixLen64, err := strconv.ParseUint(record[v6Prefix], 10, 32)
	if err != nil {
		return nil, errors.Wrap(err, "the prefix length is not valid")
	}
	prefixLen := uint32(prefixLen64)
	lease := keadata.NewLease6(
		record[v6IPAddr],
		record[v6DUID],
		cltt,
		lifetime,
		subnet,
		state,
		prefixLen,
	)
	return &lease, nil
}

type RowSource interface {
	Start() chan []string
	EnsureWatching(path string) error
	Stop()
}

// A tool to produce rows from a CSV file over time.
//
// RowSource watches a file using the `fsnotify` library and emits string slices
// as produced by the CSV parser. The slices are provided through the `results`
// channel. RowSource will pay attention to file rename and replacement events.
// When the watched file is renamed, it will be read one last time to process any
// data that may have been written prior to the rename, and then it will wait for a
// new file to appear with the original name.
type FSNotifyRowSource struct {
	path    string
	results chan []string
	memfile *os.File
	reader  *csv.Reader
	watcher *fsnotify.Watcher
	running bool
	stop    chan bool
}

// Create a new RowSource which watches the file described by `path`.
// When `error` is non-nil, the first return value is undefined.
func NewRowSource(path string) (RowSource, error) {
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, err
	}
	results := make(chan []string, 1)
	stop := make(chan bool)
	memfile, err := os.Open(realPath)
	if err != nil {
		return nil, err
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	err = watcher.Add(filepath.Dir(realPath))
	if err != nil {
		return nil, err
	}
	reader := csv.NewReader(memfile)
	rs := FSNotifyRowSource{
		realPath,
		results,
		memfile,
		reader,
		watcher,
		false,
		stop,
	}
	log.WithField("file", path).Info("Watching file")
	return &rs, nil
}

// Read the watched file and emit rows into the channel until reaching EOF.
func (rs *FSNotifyRowSource) readToEOF() {
	for {
		log.Trace("Start of readToEOF loop")
		record, err := rs.reader.Read()
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			log.WithError(err).Error("Failed to read file, stopping loop")
			return
		}
		log.WithField("record", record).Trace("Read row from watched file")
		rs.results <- record
	}
}

// Reopen the watched file (after it was renamed or deleted).
func (rs *FSNotifyRowSource) reopen() error {
	log.Trace("Trying to reopen log file")
	// If it's already closed, the error doesn't matter.
	rs.memfile.Close()
	memfile, err := os.Open(rs.path)
	if errors.Is(err, os.ErrNotExist) {
		// If the file doesn't exist, fsnotify should tell us when it is created.
		return nil
	} else if err != nil {
		return err
	}
	// According to the documentation, this isn't necessary.  However, at least
	// on macOS, kqueue doesn't seem to add watches for files newly created in an
	// already-watched directory.
	err = rs.watcher.Add(rs.path)
	if err != nil {
		return err
	}
	rs.memfile = memfile
	rs.reader = csv.NewReader(memfile)
	// Don't read the file again here, otherwise it might read the first row twice.
	return nil
}

// Start a background goroutine which watches the file and puts CSV-parsed rows
// into the channel (this function's return value).
func (rs *FSNotifyRowSource) Start() chan []string {
	// Don't start two of them.
	if rs.running {
		return rs.results
	}
	go func() {
		// Ensure that we read everything that's already in the file when this starts.
		rs.readToEOF()
		for {
			select {
			case <-rs.stop:
				return
			case err, ok := <-rs.watcher.Errors:
				if !ok {
					log.Error("Watcher error: channel closed")
					return
				}
				log.WithError(err).Error("Received error from watcher")
			case event, ok := <-rs.watcher.Events:
				log.WithField("event", event).Debug("FsNotify event received")
				if !ok {
					log.Warn("Failed to read from watcher events channel")
					return
				}
				if event.Name != rs.path {
					log.WithField("file", event.Name).Debug("Ignoring event for other file")
					continue
				}
				// These are ordered intentionally; one event can have several of these flags set:
				// * Chmod is not checked because it doesn't convey useful information for this system.
				// * Delete and rename are checked first because those indicate kea-lfc is running.
				// * Create is checked after delete/rename and before write so that the file descriptor can be opened prior to reading from the file.
				// * Write and create are checked after delete/remove because different and higher-priority steps must occur first in order to ensure the right data is being read.
				switch {
				case event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename):
					err := rs.reopen()
					if err != nil {
						log.WithError(err).Debug("Failed to reopen log file after REMOVE or RENAME; waiting for it to reappear")
					}
				case event.Has(fsnotify.Create):
					err := rs.reopen()
					if err != nil {
						log.WithError(err).Error("Failed to reopen log file after CREATE event")
						return
					}
				case event.Has(fsnotify.Write):
					rs.readToEOF()
				}
			}
		}
	}()
	rs.running = true
	return rs.results
}

// Stop the background goroutine. After this function is called, the RowSource cannot be reused.
func (rs *FSNotifyRowSource) Stop() {
	if !rs.running {
		return
	}
	rs.watcher.Close()
	rs.memfile.Close()
	// Don't send the stop signal because .Close() on the Watcher also exits the
	// loop and means the stop would block indefinitely.
	close(rs.results)
	rs.running = false
}

// Ensure that the RowSource is watching the named file.  If this is the same
// file as it was already watching, no state is changed.  If it is a different
// file, the RowSource is stopped, reset to examine the new file, and then
// started again.
func (rs *FSNotifyRowSource) EnsureWatching(path string) error {
	if !rs.running {
		return nil
	}
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	// Don't do anything if it's the same file.
	if realPath == rs.path {
		return nil
	}
	// Don't call rs.Stop() because it closes the channel, which makes things more difficult for consumers of this API.
	rs.watcher.Close()
	rs.memfile.Close()
	rs.stop <- true
	rs.running = false

	rs.path = realPath

	rs.memfile, err = os.Open(realPath)
	if err != nil {
		return err
	}
	rs.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	err = rs.watcher.Add(filepath.Dir(realPath))
	if err != nil {
		return err
	}
	rs.reader = csv.NewReader(rs.memfile)
	rs.Start()
	return nil
}

// Parse the "Expire" and "Lifetime" columns of the provided into uint64s.
func parseExpireLifetime(record []string, expireIdx, lifetimeIdx int) (uint64, uint32, error) {
	expire, err := strconv.ParseUint(record[expireIdx], 10, 64)
	if err != nil {
		return 0, 0, err
	}
	lifetime64, err := strconv.ParseUint(record[lifetimeIdx], 10, 32)
	lifetime := uint32(lifetime64)
	return expire, lifetime, err
}

// Parse the provided row as a [Lease4].  If the row's CLTT is less than (older
// than) minCLTT, this parser will return nil and also a nil error.
func ParseRowAsLease4(record []string, minCLTT uint64) (*keadata.Lease, error) {
	if len(record) == 0 {
		return nil, errors.New("cannot parse empty slice as a lease structure")
	}
	if record[0] == "address" {
		return nil, errors.New("cannot parse column headers as a lease structure")
	}
	if strings.Contains(record[0], ":") {
		return nil, errors.Errorf("'%s' contains a colon: unexpected IPv6 address", record[0])
	}
	expire, lifetime, err := parseExpireLifetime(record, v4Expire, v4ValidLifetime)
	if err != nil {
		return nil, errors.Wrap(err, "the expiry or valid_lifetime values were not valid")
	}
	// Infinite-lifetime leases are stored as 0xFFFFFFFF.  This will need to be
	// refactored come 2038.
	cltt := expire - uint64(lifetime)
	if cltt < minCLTT {
		return nil, nil
	}
	lease, err := newLease4(record, cltt, lifetime)
	if err != nil {
		return nil, err
	}
	return lease, nil
}

// Parse the provided row as a [Lease6].  If the row's CLTT is less than (older
// than) minCLTT, this parser will return nil and also a nil error.
func ParseRowAsLease6(record []string, minCLTT uint64) (*keadata.Lease, error) {
	if len(record) == 0 {
		return nil, errors.New("cannot parse empty sljice as a lease structure")
	}
	if record[0] == "address" {
		return nil, errors.New("cannot parse column headers as a lease structure")
	}
	if strings.Contains(record[0], ".") {
		return nil, errors.Errorf("'%s' contains a dot: unexpected IPv4 address", record[0])
	}
	expire, lifetime, err := parseExpireLifetime(record, v6Expire, v6ValidLifetime)
	if err != nil {
		return nil, errors.Wrap(err, "the expiry or valid_lifetime values were not valid")
	}
	cltt := expire - uint64(lifetime)
	if cltt < minCLTT {
		return nil, nil
	}
	lease, err := newLease6(record, cltt, lifetime)
	if err != nil {
		return nil, err
	}
	return lease, nil
}

type MemfileSnooper interface {
	Start()
	Stop()
	EnsureWatching(path string) error
	GetSnapshot() []*keadata.Lease
}

type RealMemfileSnooper struct {
	kind         daemonname.Name
	rs           RowSource
	lastCLTT     uint64
	leaseUpdates []*keadata.Lease
	running      bool
	stop         chan bool
	mutex        sync.Mutex
	parser       func([]string, uint64) (*keadata.Lease, error)
}

func NewMemfileSnooper(kind daemonname.Name, rs RowSource) (MemfileSnooper, error) {
	ms := RealMemfileSnooper{
		kind:         kind,
		rs:           rs,
		leaseUpdates: make([]*keadata.Lease, 0),
		stop:         make(chan bool),
	}
	switch kind {
	case daemonname.DHCPv4:
		ms.parser = ParseRowAsLease4
	case daemonname.DHCPv6:
		ms.parser = ParseRowAsLease6
	default:
		return nil, errors.New("MemfileSnooper cannot snoop lease memfiles for daemons other than DHCPv4 and DHCPv6.")
	}
	return &ms, nil
}

// A structure to use as the key for identifying duplicate leases.
type leaseKey struct {
	// The IPv[46] address that was leased.
	IP string
	// The MAC address or DUID of the client which requested the lease.
	ID string
}

func (ms *RealMemfileSnooper) GetSnapshot() []*keadata.Lease {
	snapshot := make([]*keadata.Lease, 0)
	index := map[leaseKey]int{}
	var getID func(*keadata.Lease) string
	switch ms.kind {
	case daemonname.DHCPv4:
		getID = func(lease *keadata.Lease) string {
			return lease.HWAddress
		}
	case daemonname.DHCPv6:
		getID = func(lease *keadata.Lease) string {
			return lease.DUID
		}
	default:
		// This should be impossible because the constructor function returns an error if you set a daemonname other than the two above, but let's do something approximately correct rather than calling panic().
		log.Warn("GetSnapshot was called on a MemfileSnooper with an invalid daemonname. This is a programming error that must be corrected.")
		return []*keadata.Lease{}
	}
	ms.mutex.Lock()
	for _, lease := range ms.leaseUpdates {
		key := leaseKey{
			IP: lease.IPAddress,
			ID: getID(lease),
		}
		if snapIdx, exists := index[key]; exists {
			snapLease := snapshot[snapIdx]
			if snapLease.CLTT < lease.CLTT {
				snapshot[snapIdx] = lease
			}
		} else {
			snapshot = append(snapshot, lease)
			index[key] = len(snapshot) - 1
		}
	}
	ms.mutex.Unlock()
	return snapshot
}

func (ms *RealMemfileSnooper) EnsureWatching(path string) error {
	return ms.rs.EnsureWatching(path)
}

func (ms *RealMemfileSnooper) Stop() {
	if !ms.running {
		return
	}
	close(ms.stop)
	ms.rs.Stop()
	ms.running = false
}

func (ms *RealMemfileSnooper) Start() {
	if ms.running {
		return
	}
	channel := ms.rs.Start()
	go func() {
		for {
			select {
			case <-ms.stop:
				return
			case row, ok := <-channel:
				if !ok {
					return
				}
				parsed, err := ms.parser(row, ms.lastCLTT)
				if err != nil {
					log.WithError(err).Warn("Unable to parse this lease record")
					continue
				}
				if parsed == nil {
					// This is normal; it happens when the row is older than the most recent CLTT.
					continue
				}
				ms.mutex.Lock()
				ms.leaseUpdates = append(ms.leaseUpdates, parsed)
				ms.lastCLTT = parsed.CLTT
				ms.mutex.Unlock()
			}
		}
	}()
	ms.running = true
}
