package agent

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fsnotify/fsnotify"
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

var (
	ErrHeaders              = errors.New("cannot process column headers as a Lease structure")
	ErrUnexpectedV6         = errors.New("found an IPv6 address where an IPv4 address was expected")
	ErrUnexpectedV4         = errors.New("found an IPv4 address where an IPv6 address was expected")
	ErrCLTTTooOld           = errors.New("the CLTT for this lease is older than the limit; refusing to parse")
	ErrInvalidExpOrLifetime = errors.New("unable to parse the expires or valid_lifetime columns as numbers")
	ErrInvalidSubnetID      = errors.New("the subnet ID is not valid")
	ErrInvalidLeaseState    = errors.New("the lease state is not valid")
	ErrInvalidPrefixLen     = errors.New("the prefix length not valid")
)

// How does this module work?
// Users of this module are expected to create a RowSource, then feed its output one-at-a-time into ParseRowAsLease4 or ParseRowAsLease6.
// This creates a pipeline of goroutines connected with channels which looks like this:
//
// +----------+   +-----------+   +----------+
// | FsNotify +-->| RowSource +-->| (caller) |
// +----------+   +-----------+   +----------+

// The subset of data about a DHCPv4 lease that lease tracking is interested in.
type Lease4 struct {
	// IPv4 address in dotted-decimal notation.
	IPAddr string
	// MAC address in hexadecimal notation with colons between each byte.
	HWAddr string
	// Expiration date of the lease in seconds since the Unix epoch (1 Jan 1970, UTC)
	Expire uint64
	// "Client Last Transaction Time", the last time this client talked to us, in seconds as above.
	CLTT uint64
	// Number of seconds for which the lease is valid.
	ValidLifetime uint32
	// ID of the subnet from the Kea configuration file.
	SubnetID int
	// Actually an enum, documented here: https://reports.kea.isc.org/dev_guide/d6/dbd/dhcpDatabaseBackends.html#lease4-csv
	State int
}

// Create a new Lease4 from a CSV row, and the already-parsed values.
func newLease4(record []string, expire uint64, cltt uint64, lifetime uint32) (*Lease4, error) {
	subnet, err := strconv.Atoi(record[v4Subnet])
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidSubnetID, err)
	}
	state, err := strconv.Atoi(record[v4State])
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidLeaseState, err)
	}
	lease := Lease4{
		record[v4IPAddr],
		record[v4HWAddr],
		expire,
		cltt,
		lifetime,
		subnet,
		state,
	}
	return &lease, nil
}

// The subset of data about a DHCPv6 lease that lease tracking is interested in.
type Lease6 struct {
	// IPv6 address in hexadecimal with colons between every two bytes, and with a
	// double-colon indicating a span of zeroes.
	IPAddr string
	// "Device Unique IDentifier", a unique ID that may or may not be the MAC address (up to the client).
	DUID string
	// Expiration date of the lease in seconds since the Unix epoch (1 Jan 1970, UTC)
	Expire uint64
	// "Client Last Transaction Time", the last time this client talked to us, in
	// seconds as above.
	CLTT uint64
	// Number of seconds for which the lease is valid.
	ValidLifetime uint32
	// ID of the subnet from the Kea configuration file.
	SubnetID int
	// Actually an enum, documented here:
	// https://reports.kea.isc.org/dev_guide/d6/dbd/dhcpDatabaseBackends.html#lease4-csv
	State int
	// Length of the CIDR prefix for this address. 128 means that it's just the one
	// address, other values mean this is a delegated prefix.
	PrefixLen int
}

// Create a new Lease6 from a CSV row, and the already-parsed values.
func newLease6(record []string, expire uint64, cltt uint64, lifetime uint32) (Lease6, error) {
	subnet, err := strconv.Atoi(record[v6Subnet])
	if err != nil {
		return Lease6{}, fmt.Errorf("%w: %w", ErrInvalidSubnetID, err)
	}
	state, err := strconv.Atoi(record[v6State])
	if err != nil {
		return Lease6{}, fmt.Errorf("%w: %w", ErrInvalidLeaseState, err)
	}
	prefixLen, err := strconv.Atoi(record[v6Prefix])
	if err != nil {
		return Lease6{}, fmt.Errorf("%w: %w", ErrInvalidPrefixLen, err)
	}
	lease := Lease6{
		record[v6IPAddr],
		record[v6DUID],
		expire,
		cltt,
		lifetime,
		subnet,
		state,
		prefixLen,
	}
	return lease, nil
}

// A tool to produce rows from a CSV file over time.
//
// RowSource watches a file using the `fsnotify` library and emits string slices
// as produced by the CSV parser. The slices are provided through the `results`
// channel. RowSource will pay attention to file rename and replacement events.
// When the watched file is renamed, it will be read one last time to process any
// data that may have been written prior to the rename, and then it will wait for a
// new file to appear with the original name.
type RowSource struct {
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
func NewRowSource(path string) (*RowSource, error) {
	results := make(chan []string, 1)
	stop := make(chan bool)
	memfile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	err = watcher.Add(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	reader := csv.NewReader(memfile)
	rs := RowSource{
		path,
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
func (rs *RowSource) readToEOF() {
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

// Reopen the watched file (after it was renamed or closed).
func (rs *RowSource) reopen() error {
	log.Trace("Trying to reopen log file")
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
	// Try to read the file again, just in case.
	rs.readToEOF()
	return nil
}

// Start a background goroutine which watches the file and puts CSV-parsed rows
// into the channel (this function's return value).
func (rs *RowSource) Start() chan []string {
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
				// * Write and create are checked after delete/remove because different and higher-priority steps must occur first in order to ensure the right data is being read.
				if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
					err := rs.reopen()
					if err != nil {
						log.WithError(err).Error("Failed to reopen log file")
						return
					}
				} else if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
					rs.readToEOF()
				}
			}
		}
	}()
	rs.running = true
	return rs.results
}

// Stop the background goroutine. After this function is called, the RowSource cannot be reused.
func (rs *RowSource) Stop() {
	if !rs.running {
		return
	}
	rs.stop <- true
	close(rs.results)
	rs.running = false
}

// Parse the "Expire" and "Lifetime" columns of the provided into uint64s.
func parseExpireLifetime(record []string, expireIdx, lifetimeIdx int) (uint64, uint64, error) {
	expire, err := strconv.ParseUint(record[expireIdx], 10, 64)
	if err != nil {
		return 0, 0, err
	}
	lifetime64, err := strconv.ParseUint(record[lifetimeIdx], 10, 32)
	return expire, lifetime64, err
}

// Parse the provided row as a [Lease4].  If the row's CLTT is less than (older
// than) minCLTT, this parser will return nil and an appropriate error.
func ParseRowAsLease4(record []string, minCLTT uint64) (*Lease4, error) {
	if record[0] == "address" {
		return nil, ErrHeaders
	}
	if strings.Contains(record[0], ":") {
		return nil, fmt.Errorf("'%s' contains a colon: %w", record[0], ErrUnexpectedV6)
	}
	expire, lifetime64, err := parseExpireLifetime(record, v4Expire, v4ValidLifetime)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidExpOrLifetime, err)
	}
	lifetime := uint32(lifetime64)
	// Infinite-lifetime leases are stored as 0xFFFFFFFF.  This will need to be
	// refactored come 2038.
	cltt := expire - lifetime64
	if cltt < minCLTT {
		return nil, fmt.Errorf("%w: %d < %d", ErrCLTTTooOld, cltt, minCLTT)
	}
	lease, err := newLease4(record, expire, cltt, lifetime)
	if err != nil {
		return nil, err
	}
	return lease, nil
}

// Parse the provided row as a [Lease6].  If the row's CLTT is less than (older
// than) minCLTT, this parser will return nil and an appropriate error.
func ParseRowAsLease6(record []string, minCLTT uint64) (*Lease6, error) {
	if record[0] == "address" {
		return nil, ErrHeaders
	}
	if strings.Contains(record[0], ".") {
		return nil, fmt.Errorf("'%s' contains a dot: %w", record[0], ErrUnexpectedV4)
	}
	expire, lifetime64, err := parseExpireLifetime(record, v6Expire, v6ValidLifetime)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidExpOrLifetime, err)
	}
	lifetime := uint32(lifetime64)
	cltt := expire - lifetime64
	if cltt < minCLTT {
		return nil, fmt.Errorf("%w: %d < %d", ErrCLTTTooOld, cltt, minCLTT)
	}
	lease, err := newLease6(record, expire, cltt, lifetime)
	if err != nil {
		return nil, err
	}
	return &lease, nil
}
