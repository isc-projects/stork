package agent

import (
	"context"
	"io"
	"slices"

	"github.com/nxadm/tail"
	"github.com/pkg/errors"
)

var _ logReader = (*textFileLogReader)(nil)

// Common interface implemented by different log readers.
type logReader interface {
	// Returns true if the log reader implementation is supported on the current platform.
	isSupported() bool
	// Captures the log contents from the log source. By default, it returns the contents of the log
	// file from the start without following. The options can be used to instruct the capture to follow
	// the new lines or read from the end of the file.
	capture(context.Context, ...logReaderCaptureOption) (<-chan logReaderLine, error)
}

// A line of the log contents returned by logReader or an error.
type logReaderLine struct {
	text string
	err  error
}

// Configuration for the text file log reader.
type textLogReaderConfig struct {
	// Indicates whether the log reader should poll the file for changes
	// or rather use inotify. Note that inotify can be unreliable on some systems,
	// so we set this flag in the unit tests to ensure that the tests are reliable.
	poll bool
}

// Options for the log reader capture.
type logReaderCaptureOption int

const (
	logReaderCaptureOptionReadFromEnd logReaderCaptureOption = iota
	logReaderCaptureOptionReadFollow
)

// Implements the log reader for text files.
type textFileLogReader struct {
	path   string
	config textLogReaderConfig
}

// Creates a new instance of the text file log reader. The path specifies the absolute
// location of the log file. The config can be used to enable or disable polling the file.
func newTextFileLogReader(path string, config textLogReaderConfig) *textFileLogReader {
	return &textFileLogReader{
		path:   path,
		config: config,
	}
}

// Returns true indicating that the text log file reader is supported on all platforms.
func (lc *textFileLogReader) isSupported() bool {
	return true
}

// Captures the log contents from the log source. By default, it returns the contents of the log
// file from the start without following. The options can be used to instruct the capture to follow
// the new lines or read from the end of the file.
func (lc *textFileLogReader) capture(ctx context.Context, options ...logReaderCaptureOption) (<-chan logReaderLine, error) {
	// Check if the capture should follow the new lines.
	follow := slices.Contains(options, logReaderCaptureOptionReadFollow)
	// Check if the capture should read from the start or the end of the file.
	whence := io.SeekStart
	if slices.Contains(options, logReaderCaptureOptionReadFromEnd) {
		whence = io.SeekEnd
		if !follow {
			// Reading from the end is only supported when following is enabled. Otherwise, it
			// would have to always return empty contents.
			return nil, errors.New("cannot read from the end of the file without following")
		}
	}
	// Read the log file using specified configuration and options.
	t, err := tail.TailFile(lc.path, tail.Config{
		Follow: slices.Contains(options, logReaderCaptureOptionReadFollow),
		ReOpen: follow,
		Poll:   lc.config.poll,
		Location: &tail.SeekInfo{
			Offset: 0,
			Whence: whence,
		},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to capture the file: %s", lc.path)
	}
	// Create a channel for returning captured lines.
	lines := make(chan logReaderLine)
	go func() {
		defer func() {
			// Cleanup the resources when we're done.
			close(lines)
			t.Cleanup()
		}()
		for {
			select {
			case <-ctx.Done():
				// Stop reading when the caller cancelled the context.
				_ = t.Stop()
				return
			case line, ok := <-t.Lines:
				var output logReaderLine
				switch {
				case (!ok || line == nil) && !follow:
					// If this is the end of stream and we're not following, there is nothing more to return.
					return
				case (!ok || line == nil) && follow:
					// If this is the end of stream and we're following, it is unexpected, and we need to
					// return an error to the caller, so the caller can restart the capture.
					output = logReaderLine{text: "", err: errors.New("log reader stopped unexpectedly")}
				default:
					// Otherwise, we're ok and the line is non-nil. Let's return the line to the caller.
					output = logReaderLine{text: line.Text, err: line.Err}
				}
				// Send the line to the caller or stop if the caller cancelled the context
				// in the meantime.
				select {
				case lines <- output:
				case <-ctx.Done():
					_ = t.Stop()
					return
				}
			}
		}
	}()
	return lines, nil
}
