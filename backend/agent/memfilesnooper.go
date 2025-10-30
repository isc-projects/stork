package agent

import (
	"encoding/csv"
	"errors"
	"io"
	"os"
	"strconv"
	"time"
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

type FSChangeState int

const (
	FSCSUnchanged FSChangeState = iota
	FSCSModified
	FSCSTruncated
	FSCSDeleted
	FSCSMoved
)

// A tool which can be used to detect whether (and how) a file has changed since
// the last time it was checked.
type ChangeDetector struct {
	Filename string
	File     os.File
	Info     os.FileInfo
	Size     int64
}

// Construct a new ChangeDetector for the given file.
func NewChangeDetector(file os.File) (*ChangeDetector, error) {
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	cd := &ChangeDetector{
		file.Name(),
		file,
		info,
		info.Size(),
	}
	return cd, nil
}

// Determine whether the tracked file has changed since the last time this function was called.
// If error is non-nil, the meaning of the first return value is undefined.
func (cd *ChangeDetector) DidChange() (FSChangeState, error) {
	newInfo, err := os.Stat(cd.Filename)
	if err != nil {
		if os.IsNotExist(err) {
			return FSCSDeleted, nil
		}
		return FSCSUnchanged, err
	}
	if !os.SameFile(cd.Info, newInfo) {
		return FSCSMoved, nil
	}

	prevSize := cd.Size
	if prevSize > 0 && prevSize > newInfo.Size() {
		cd.Size = newInfo.Size()
		return FSCSTruncated, nil
	}

	if prevSize > 0 && prevSize < newInfo.Size() {
		cd.Size = newInfo.Size()
		return FSCSModified, nil
	}

	cd.Size = newInfo.Size()

	mtime := newInfo.ModTime()
	if mtime.Compare(cd.Info.ModTime()) != 0 {
		cd.Info = newInfo
		return FSCSModified, nil
	}
	cd.Info = newInfo

	return FSCSUnchanged, nil
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

type LeaseParser[T any] struct {
	path               string
	results            chan T
	memfile            *os.File
	reader             *csv.Reader
	cd                 *ChangeDetector
	newLease           func(record []string, expires, cltt uint64, lifetime uint32) (T, error)
	expireIdx          int
	lifetimeIdx        int
	sawAllExistingData bool
}

func NewLease4Parser(path string) (*LeaseParser[Lease4], error) {
	results := make(chan Lease4, 1)
	memfile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	cd, err := NewChangeDetector(*memfile)
	if err != nil {
		return nil, err
	}
	reader := csv.NewReader(memfile)
	p := LeaseParser[Lease4]{
		path,
		results,
		memfile,
		reader,
		cd,
		NewLease4,
		v4Expire,
		v4ValidLifetime,
		false,
	}
	return &p, nil
}

func NewLease6Parser(path string) (*LeaseParser[Lease6], error) {
	results := make(chan Lease6, 1)
	memfile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	cd, err := NewChangeDetector(*memfile)
	if err != nil {
		return nil, err
	}
	reader := csv.NewReader(memfile)
	p := LeaseParser[Lease6]{
		path,
		results,
		memfile,
		reader,
		cd,
		NewLease6,
		v6Expire,
		v6ValidLifetime,
		false,
	}
	return &p, nil
}

func (p *LeaseParser[T]) parseExpireLifetime(record []string) (uint64, uint64, error) {
	expire, err := strconv.ParseUint(record[p.expireIdx], 10, 64)
	if err != nil {
		return 0, 0, err
	}
	lifetime64, err := strconv.ParseUint(record[p.lifetimeIdx], 10, 32)
	return expire, lifetime64, err
}

func (p *LeaseParser[T]) parseLoop(minCLTT uint64) {
	defer p.memfile.Close()
	// Discard column headers.  If this read fails, keep trying for a while.
	for range 1024 {
		_, err := p.reader.Read()
		if err == nil {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	for {
		// Only check this if we've seen the whole file, otherwise it may be
		// "unchanged" and wait to read data that is already available.
		if p.sawAllExistingData {
			changeType, err := p.cd.DidChange()
			if err != nil {
				// TODO: this is probably the wrong way to handle it.
				continue
			}
			switch changeType {
			case FSCSMoved:
				fallthrough
			case FSCSDeleted:
				p.memfile.Close()
				memfile, err := os.Open(p.path)
				if err != nil {
					time.Sleep(250 * time.Millisecond)
					// TODO: probably the wrong way to handle this.
					continue
				}
				p.memfile = memfile
				p.reader = csv.NewReader(memfile)
				p.cd, err = NewChangeDetector(*memfile)
				if err != nil {
					time.Sleep(250 * time.Millisecond)
					// TODO: probably the wrong way to handle this.
					continue
				}
				p.sawAllExistingData = false
				// No continue here, read the file.
			case FSCSUnchanged:
				time.Sleep(250 * time.Millisecond)
				continue
			case FSCSTruncated:
				// No continue or fallthrough here; should do rest of the loop.
			case FSCSModified:
				// Do nothing intentionally, all other cases must restart the loop.
			}
		}
		record, err := p.reader.Read()
		if errors.Is(err, io.EOF) {
			p.sawAllExistingData = true
			continue
		}
		if err != nil {
			// TODO: is the "it was deleted" error one of these?
			close(p.results)
			return
		}
		// If we've successfully read something, but not yet found EOF again, there
		// might be more data.
		p.sawAllExistingData = false
		if record[0] == "address" {
			continue
		}
		expire, lifetime64, err := p.parseExpireLifetime(record)
		if err != nil {
			continue
		}
		lifetime := uint32(lifetime64)
		// Infinite-lifetime leases are stored as 0xFFFFFFFF.  This will need to be
		// refactored come 2038.
		cltt := expire - lifetime64
		if cltt < minCLTT {
			continue
		}
		lease, err := p.newLease(record, expire, cltt, lifetime)
		if err != nil {
			continue
		}
		p.results <- lease
	}
}

func (p *LeaseParser[T]) StartParser(minCLTT uint64) chan T {
	go p.parseLoop(minCLTT)
	return p.results
}
