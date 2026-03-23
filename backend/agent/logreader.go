package agent

import (
	"context"
	"fmt"
	"io"

	"github.com/nxadm/tail"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	storkutil "isc.org/stork/util"
)

var (
	_ logReader = (*textFileLogReader)(nil)
	_ logReader = (*systemdLogReader)(nil)
)

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

// Configuration used by the log reader capture() function. It is created from the
// logReaderCaptureOption arguments passed to the capture() function.
type logReaderCaptureConfig struct {
	follow       bool
	fromEnd      bool
	sinceDaysAgo int
	unitName     string
}

// A function representing a single option passed to the log reader capture() function.
// Various options are associated with different functions of this type to set
// the corresponding fields in the logReaderCaptureConfig structure. It follows the
// functional options pattern.
type logReaderCaptureOption func(*logReaderCaptureConfig)

// Returns the functional option instructing the log reader to read the log contents
// from the end of the log.
func logReaderCaptureOptionFromEnd() logReaderCaptureOption {
	return func(config *logReaderCaptureConfig) {
		config.fromEnd = true
	}
}

// Returns the functional option instructing the log reader to follow the new lines.
func logReaderCaptureOptionFollow() logReaderCaptureOption {
	return func(config *logReaderCaptureConfig) {
		config.follow = true
	}
}

// Returns the functional option instructing the log reader to read the log contents
// starting from the specified number of days ago.
func logReaderCaptureOptionSinceDaysAgo(days int) logReaderCaptureOption {
	return func(config *logReaderCaptureConfig) {
		config.sinceDaysAgo = days
	}
}

// Returns the functional option instructing the log reader to read the log contents
// from the specified systemd unit.
func logReaderCaptureOptionUnitName(unitName string) logReaderCaptureOption {
	return func(config *logReaderCaptureConfig) {
		config.unitName = unitName
	}
}

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
	// Map specified options to the configuration structure.
	config := logReaderCaptureConfig{}
	for _, option := range options {
		option(&config)
	}
	// Check if the capture should read from the start or the end of the file.
	whence := io.SeekStart
	if config.fromEnd {
		whence = io.SeekEnd
		if !config.follow {
			// Reading from the end is only supported when following is enabled. Otherwise, it
			// would have to always return empty contents.
			return nil, errors.New("cannot read from the end of the file without following")
		}
	}
	// Read the log file using specified configuration and options.
	t, err := tail.TailFile(lc.path, tail.Config{
		Follow: config.follow,
		ReOpen: config.follow,
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
				if err := t.Stop(); err != nil {
					log.WithError(err).Error("failed to stop the log reader")
				}
				return
			case line, ok := <-t.Lines:
				var output logReaderLine
				switch {
				case (!ok || line == nil) && !config.follow:
					// If this is the end of stream and we're not following, there is nothing more to return.
					return
				case (!ok || line == nil) && config.follow:
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
					if err := t.Stop(); err != nil {
						log.WithError(err).Error("failed to stop the log reader")
					}
					return
				}
			}
		}
	}()
	return lines, nil
}

// Implements the log reader for systemd logs.
type systemdLogReader struct {
	executor storkutil.CommandExecutor
}

// Creates a new instance of the systemd log reader. The executor is used to execute the journalctl command.
func newSystemdLogReader(executor storkutil.CommandExecutor) *systemdLogReader {
	return &systemdLogReader{
		executor: executor,
	}
}

// Returns true if the systemd log reader is supported on the current platform.
// it checks if the journalctl command is available.
func (lc *systemdLogReader) isSupported() bool {
	_, err := lc.executor.LookPath("journalctl")
	return err == nil
}

// Captures the log contents from systemd. By default, it returns the contents of the logs
// from the start without following. The options can be used to instruct the capture to follow
// the new lines or read from the end of the logs. The logReaderCaptureOptionUnitName option
// can be specified to narrow down the logs to a specific service. The logReaderCaptureOptionSinceDaysAgo
// option can be specified to read the logs starting from the specified number of days ago.
func (lc *systemdLogReader) capture(ctx context.Context, options ...logReaderCaptureOption) (<-chan logReaderLine, error) {
	// Map specified options to the configuration structure.
	config := logReaderCaptureConfig{}
	for _, option := range options {
		option(&config)
	}
	if config.fromEnd && !config.follow {
		// Reading from the end is only supported when following is enabled. Otherwise, it
		// would have to always return empty contents.
		return nil, errors.New("cannot read from the end of the systemd log without following")
	}
	// Build the journalctl command line arguments.
	var args []string
	if config.follow {
		// Follow the new lines.
		args = append(args, "-f")
	}
	if config.unitName != "" {
		// Track the log for the specified service.
		args = append(args, "-u", config.unitName)
	}
	switch {
	case config.fromEnd:
		// Start monitoring the logs from the end. It must be combined with
		// the follow option on.
		args = append(args, "-n", "0")
	case config.sinceDaysAgo > 0:
		// Read the logs starting from the specified number of days ago.
		args = append(args, "--since", fmt.Sprintf("%d days ago", config.sinceDaysAgo))
	default:
		// Read the logs from the start if neither from the end nor from a
		// specified date.
		args = append(args, "--no-tail", "-n", "+1")
	}
	// Run journalctl command with the arguments to start capturing the logs.
	// The pipe is used to make the logs accessible via the scanner.
	cmd, err := lc.executor.Start(ctx, "journalctl", args...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to start the journalctl command")
	}
	// Create a channel for returning captured lines.
	lines := make(chan logReaderLine)
	go func() {
		defer func() {
			// Cleanup the resources when we're done.
			close(lines)
			if err := cmd.Wait(); err != nil {
				log.WithError(err).Error("failed to wait for the journalctl command to complete")
			}
		}()
		// Read the logs from the command output using scanner.
		for cmd.GetScanner().Scan() {
			select {
			case lines <- logReaderLine{text: cmd.GetScanner().Text(), err: cmd.GetScanner().Err()}:
			case <-ctx.Done():
				// Stop reading when the caller cancelled the context.
				return
			}
		}
	}()
	return lines, nil
}
