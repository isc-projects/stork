package agent

import (
	"encoding/csv"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

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
// +----------+   +----------------+   +-----------+   +----------+
// | FsNotify +-->| ChangeDetector +-->| RowSource +-->| (caller) |
// +----------+   +----------------+   +-----------+   +----------+
//
// ChangeDetector exists to abstract away the mechanism which does the filesystem monitoring from the rest of this module's code.

type FSChangeState int

const (
	FSCSUnchanged FSChangeState = iota
	FSCSModified
	FSCSTruncated
	FSCSDeleted
	FSCSMoved
)

type ChangeDetector interface {
	RegisterListener(func(FSChangeState))
	UnregisterAllListeners()
	Start()
	Stop()
}

type FsNotifyChangeDetector struct {
	file      string
	running   bool
	watcher   *fsnotify.Watcher
	listeners []func(FSChangeState)
	stop      chan bool
	mutex     sync.Mutex
}

func NewFsNotifyChangeDetector(path string) (ChangeDetector, error) {
	channel := make(chan bool, 1)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	err = watcher.Add(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	return &FsNotifyChangeDetector{
		path,
		false,
		watcher,
		[]func(FSChangeState){},
		channel,
		sync.Mutex{},
	}, nil
}

func (cd *FsNotifyChangeDetector) RegisterListener(fn func(FSChangeState)) {
	cd.mutex.Lock()
	defer cd.mutex.Unlock()
	cd.listeners = append(cd.listeners, fn)
}

func (cd *FsNotifyChangeDetector) UnregisterAllListeners() {
	cd.mutex.Lock()
	defer cd.mutex.Unlock()
	cd.listeners = []func(FSChangeState){}
}

func (cd *FsNotifyChangeDetector) send(state FSChangeState) {
	cd.mutex.Lock()
	defer cd.mutex.Unlock()
	for _, listener := range cd.listeners {
		if listener != nil {
			listener(state)
		}
	}
}

func (cd *FsNotifyChangeDetector) Start() {
	if cd.running {
		return
	}
	cd.running = true
	go func() {
		// Send one modified event at the start to ensure any listeners read whatever's
		// already in the file.
		cd.send(FSCSModified)
		for {
			select {
			case <-cd.stop:
				log.Info("received stop message")
				return
			case err, ok := <-cd.watcher.Errors:
				if !ok {
					// TODO: log
					log.WithError(err).Info("received error from watcher")
					return
				}
			case event, ok := <-cd.watcher.Events:
				if !ok {
					log.Info("failed to read from watcher events channel")
					return
				}
				if event.Name != cd.file {
					continue
				}
				// These are ordered intentionally; one event can have several of these flags set:
				// * Chmod and create are not checked because those don't convey useful information for this system.
				// * Delete and rename are checked first because those indicate kea-lfc is running.
				// * Write is checked after delete/remove because different and higher-priority steps must occur first in order to ensure the right data is being read.
				if event.Has(fsnotify.Remove) {
					cd.send(FSCSDeleted)
				}
				if event.Has(fsnotify.Rename) {
					cd.send(FSCSMoved)
				}
				if event.Has(fsnotify.Write) {
					cd.send(FSCSModified)
				}
			}
		}
	}()
}

func (cd *FsNotifyChangeDetector) Stop() {
	if !cd.running {
		return
	}
	cd.stop <- true
	cd.running = false
}

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
func NewLease4(record []string, expire uint64, cltt uint64, lifetime uint32) (Lease4, error) {
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
func NewLease6(record []string, expire uint64, cltt uint64, lifetime uint32) (Lease6, error) {
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
	cd      ChangeDetector
	running bool
}

func NewRowSource(path string) (*RowSource, error) {
	results := make(chan []string, 1)
	memfile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	cd, err := NewFsNotifyChangeDetector(path)
	if err != nil {
		return nil, err
	}
	reader := csv.NewReader(memfile)
	rs := RowSource{
		path,
		results,
		memfile,
		reader,
		cd,
		false,
	}
	cd.RegisterListener(rs.eventHandler)
	log.WithField("file", path).Info("watching log file")
	return &rs, nil
}

func (rs *RowSource) eventHandler(state FSChangeState) {
	log.WithField("state", state).Trace("filesystem change event handler called")
	switch state {
	case FSCSMoved:
		fallthrough
	case FSCSDeleted:
		log.Trace("trying to reopen log file")
		rs.memfile.Close()
		memfile, err := os.Open(rs.path)
		if err != nil {
			log.WithError(err).Debug("Failed to reopen log file")
			return
		}
		rs.memfile = memfile
		rs.reader = csv.NewReader(memfile)
		// Try to read the file again, just in case.
		rs.readToEOF()
	case FSCSUnchanged:
		return
	case FSCSTruncated:
		fallthrough
	case FSCSModified:
		rs.readToEOF()
	}
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

func (rs *RowSource) Start() chan []string {
	if rs.running {
		return rs.results
	}
	rs.cd.Start()
	rs.running = true
	return rs.results
}

func (rs *RowSource) Stop() {
	if !rs.running {
		return
	}
	rs.cd.Stop()
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
	lease, err := NewLease4(record, expire, cltt, lifetime)
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
	lease, err := NewLease6(record, expire, cltt, lifetime)
	if err != nil {
		return nil
	}
	return &lease
}
