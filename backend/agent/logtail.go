package agent

import (
	"bufio"
	"io"
	"os"
	"sync"

	"github.com/pkg/errors"
)

// Log tailer provides means for viewing log files. It maintains the list of
// unique files which can be viewed. If the file is not on the list of the allowed
// files, an error is returned upon an attempt to view it.
type logTailer struct {
	allowedPaths map[string]bool
	mutex        *sync.Mutex
}

// Creates new instance of the log tailer.
func newLogTailer() *logTailer {
	lt := &logTailer{
		allowedPaths: make(map[string]bool),
		mutex:        new(sync.Mutex),
	}
	return lt
}

// Adds a specified path to the list of files which can be viewed.
func (lt *logTailer) allow(path string) {
	lt.mutex.Lock()
	defer lt.mutex.Unlock()
	lt.allowedPaths[path] = true
}

// Checks if the given file can be viewed.
func (lt *logTailer) allowed(path string) bool {
	lt.mutex.Lock()
	defer lt.mutex.Unlock()
	_, ok := lt.allowedPaths[path]
	return ok
}

// Returns the tail of the specified log file. The path specifies the absolute
// location of the log file. The offset specifies the location relative to an
// end of the file from which the tail should be returned. The offset must be
// a positive value. If the file is not allowed, it does not exist an error
// is returned. An error is also returned if an attempt to read the file fails.
func (lt *logTailer) tail(path string, offset int64) (lines []string, err error) {
	// Check if it is allowed to tail this file.
	if !lt.allowed(path) {
		err = errors.Errorf("access forbidden to the %s", path)
		return lines, err
	}

	f, err := os.Open(path)
	if err != nil {
		err = errors.WithMessagef(err, "failed to open file for tailing: %s", path)
		return lines, err
	}
	defer func() {
		_ = f.Close()
	}()

	stat, err := f.Stat()
	if err != nil {
		err = errors.WithMessagef(err, "failed to stat the file opened for tailing: %s", path)
		return lines, err
	}

	// Can't go beyond the file size.
	if offset > stat.Size() {
		offset = stat.Size()
	}

	_, err = f.Seek(-offset, io.SeekEnd)
	if err != nil {
		err = errors.WithMessagef(err, "failed to seek in the file opened for tailing: %s", path)
		return lines, err
	}
	s := bufio.NewScanner(f)
	for s.Scan() {
		lines = append(lines, s.Text())
	}
	if err = s.Err(); err != nil {
		err = errors.WithMessagef(err, "failed to read the tailed file: %s", path)
	}
	return lines, err
}
