package testutil

import (
	"io"
	"net"
	"os"
	"strings"
	"time"

	errors "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Capture the stdout (including log output) and stderr content produced by
// a given function.
func CaptureOutput(f func()) (stdout []byte, stderr []byte, err error) {
	rescueStdout := os.Stdout
	rescueStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr
	rescueLogOutput := logrus.StandardLogger().Out
	logrus.StandardLogger().SetOutput(wOut)
	// Restore the standard pipelines at the end.
	defer func() {
		os.Stdout = rescueStdout
		os.Stderr = rescueStderr
		logrus.StandardLogger().SetOutput(rescueLogOutput)
	}()

	// Execute function
	f()

	// Close the internal pipelines.
	wOut.Close()
	wErr.Close()

	// Reads the stdout
	stdout, err = io.ReadAll(rOut)
	if err != nil {
		err = errors.Wrap(err, "cannot read stdout")
		return
	}

	stderr, err = io.ReadAll(rErr)
	err = errors.Wrap(err, "cannot read stderr")
	return stdout, stderr, err
}

// Parses the conventional filename with suffix.
// Returns a prefix of filename, parsed timestamp, and error (if failed).
func ParseTimestampFilename(filename string) (prefix string, timestamp time.Time, extension string, err error) {
	timestampStart := strings.LastIndex(filename, "_")
	if timestampStart <= 0 {
		err = errors.Errorf("missing prefix delimiter: %s", filename)
		return
	}
	timestampStart++
	prefix = filename[:timestampStart]

	timestampEnd := strings.Index(filename[timestampStart:], ".")
	if timestampEnd >= 0 {
		timestampEnd += timestampStart
		extension = filename[timestampEnd:]
	} else {
		timestampEnd = len(filename)
	}

	if timestampEnd-timestampStart < len(time.RFC3339)-5 { // Timezone is optional
		err = errors.Errorf("timestamp is too short: %s", filename)
		return
	}

	raw := filename[timestampStart:timestampEnd]
	raw = raw[:11] + strings.ReplaceAll(raw[11:], "-", ":")

	timestamp, err = time.Parse(time.RFC3339, raw)
	err = errors.Wrapf(err, "cannot parse a timestamp: %s for: %s", raw, filename)
	return
}

// Allows reverting the changes in the environment variables to a previous
// state. It remembers the current environment variables and returns a function
// that must be called to restore these values.
func CreateEnvironmentRestorePoint() func() {
	originalEnv := os.Environ()

	return func() {
		originalEnvDict := make(map[string]string, len(originalEnv))
		for _, pair := range originalEnv {
			key, value, _ := strings.Cut(pair, "=")
			originalEnvDict[key] = value
		}

		actualEnv := os.Environ()
		actualKeys := make(map[string]bool, len(actualEnv))
		for _, actualPair := range actualEnv {
			actualKey, actualValue, _ := strings.Cut(actualPair, "=")
			actualKeys[actualKey] = true
			originalValue, exist := originalEnvDict[actualKey]

			if !exist {
				// Environment variable was added.
				os.Unsetenv(actualKey)
			} else if actualValue != originalValue {
				// Environment variable was changed.
				os.Setenv(actualKey, originalValue)
			}
		}

		for originalKey, originalValue := range originalEnvDict {
			if _, exist := actualKeys[originalKey]; !exist {
				// Environment variable was removed.
				os.Setenv(originalKey, originalValue)
			}
		}
	}
}

// Allows reverting the changes in the os.Args variables to a previous
// state. It remembers the current os.Args and returns a function
// that must be called to restore these values.
func CreateOsArgsRestorePoint() func() {
	original := os.Args
	return func() {
		os.Args = original
	}
}

// Helper function that returns a free TCP port on localhost. Returns an error
// if no ports are available.
// Source: https://gist.github.com/sevkin/96bdae9274465b2d09191384f86ef39d
func GetFreeLocalTCPPort() (int, error) {
	if a, err := net.ResolveTCPAddr("tcp", "localhost:0"); err == nil {
		var l *net.TCPListener
		if l, err = net.ListenTCP("tcp", a); err == nil {
			defer l.Close()
			return l.Addr().(*net.TCPAddr).Port, nil
		}
	}
	return 0, errors.Errorf("none TCP port is available")
}
