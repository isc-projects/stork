package agent

import (
	"encoding/csv"
	"errors"
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

// How does this module work?
// Users of this module are expected to create a RowSource, then feed its output one-at-a-time into ParseRowAsLease4 or ParseRowAsLease6.
// This creates a pipeline of goroutines connected with channels which looks like this:
//
// +----------+   +-----------+   +----------+
// | FsNotify +-->| RowSource +-->| (caller) |
// +----------+   +-----------+   +----------+

type Lease4 struct {
	IPAddr        string
	HWAddr        string
	Expire        uint64
	CLTT          uint64
	ValidLifetime uint32
	SubnetID      int
	State         int
}

// Create a new Lease4 from a CSV row, and the already-parsed values.
func newLease4(record []string, expire uint64, cltt uint64, lifetime uint32) (Lease4, error) {
	subnet, err := strconv.Atoi(record[v4Subnet])
	if err != nil {
		return Lease4{}, err
	}
	state, err := strconv.Atoi(record[v4State])
	if err != nil {
		return Lease4{}, err
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
	return lease, nil
}

type Lease6 struct {
	IPAddr        string
	DUID          string
	Expire        uint64
	CLTT          uint64
	ValidLifetime uint32
	SubnetID      int
	State         int
	PrefixLen     int
}

// Create a new Lease6 from a CSV row, and the already-parsed values.
func newLease6(record []string, expire uint64, cltt uint64, lifetime uint32) (Lease6, error) {
	subnet, err := strconv.Atoi(record[v6Subnet])
	if err != nil {
		return Lease6{}, err
	}
	state, err := strconv.Atoi(record[v6State])
	if err != nil {
		return Lease6{}, err
	}
	prefixLen, err := strconv.Atoi(record[v6Prefix])
	if err != nil {
		return Lease6{}, err
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

type RowSource struct {
	path    string
	results chan []string
	memfile *os.File
	reader  *csv.Reader
	watcher *fsnotify.Watcher
	running bool
	stop    chan bool
}

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
	log.WithField("file", path).Info("watching log file")
	return &rs, nil
}

func (rs *RowSource) readToEOF() {
	for {
		log.Trace("readToEOF loop start")
		record, err := rs.reader.Read()
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			log.WithError(err).Info("failed to read file, stopping loop")
			// TODO: is this the right way to handle all other IO errors?
			return
		}
		// If we've successfully read something, but not yet found EOF again, there
		// might be more data.
		log.WithField("record", record).Debug("read row from log file")
		rs.results <- record
	}
}

func (rs *RowSource) reopen() error {
	log.Trace("trying to reopen log file")
	rs.memfile.Close()
	memfile, err := os.Open(rs.path)
	if err != nil {
		return err
	}
	// According to the documentation, this isn't necessary.  However, at least on macOS, kqueue doesn't seem to add watches for files newly created in an already-watched directory.
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

func (rs *RowSource) Start() chan []string {
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
					// TODO: log
					log.Info("watcher error channel closed")
					return
				}
				log.WithError(err).Info("received error from watcher")
			case event, ok := <-rs.watcher.Events:
				log.WithField("event", event).Info("FsNotify event received")
				if !ok {
					log.Info("failed to read from watcher events channel")
					return
				}
				if event.Name != rs.path {
					log.WithField("file", event.Name).Info("ignoring event for other file")
					continue
				}
				// These are ordered intentionally; one event can have several of these flags set:
				// * Chmod is not checked because it doesn't convey useful information for this system.
				// * Delete and rename are checked first because those indicate kea-lfc is running.
				// * Write and create are checked after delete/remove because different and higher-priority steps must occur first in order to ensure the right data is being read.
				if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
					err := rs.reopen()
					if err != nil {
						log.WithError(err).Debug("Failed to reopen log file")
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

func (rs *RowSource) Stop() {
	if !rs.running {
		return
	}
	rs.stop <- true
	rs.running = false
}

func parseExpireLifetime(record []string, expireIdx, lifetimeIdx int) (uint64, uint64, error) {
	expire, err := strconv.ParseUint(record[expireIdx], 10, 64)
	if err != nil {
		return 0, 0, err
	}
	lifetime64, err := strconv.ParseUint(record[lifetimeIdx], 10, 32)
	return expire, lifetime64, err
}

func ParseRowAsLease4(record []string, minCLTT uint64) *Lease4 {
	if record[0] == "address" {
		return nil
	}
	if strings.Contains(record[0], ":") {
		return nil
	}
	expire, lifetime64, err := parseExpireLifetime(record, v4Expire, v4ValidLifetime)
	if err != nil {
		return nil
	}
	lifetime := uint32(lifetime64)
	// Infinite-lifetime leases are stored as 0xFFFFFFFF.  This will need to be
	// refactored come 2038.
	cltt := expire - lifetime64
	if cltt < minCLTT {
		return nil
	}
	lease, err := newLease4(record, expire, cltt, lifetime)
	if err != nil {
		return nil
	}
	return &lease
}

func ParseRowAsLease6(record []string, minCLTT uint64) *Lease6 {
	if record[0] == "address" {
		return nil
	}
	if strings.Contains(record[0], ".") {
		return nil
	}
	expire, lifetime64, err := parseExpireLifetime(record, v6Expire, v6ValidLifetime)
	if err != nil {
		return nil
	}
	lifetime := uint32(lifetime64)
	cltt := expire - lifetime64
	if cltt < minCLTT {
		return nil
	}
	lease, err := newLease6(record, expire, cltt, lifetime)
	if err != nil {
		return nil
	}
	return &lease
}
