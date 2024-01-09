//go:build !profiler

package profiler

import "github.com/sirupsen/logrus"

// The stub is used when the profiler is not compiled in the build.
// It does nothing, just logs a debug message.
func Start(port int) func() {
	logrus.Debug("Profiler is not available in this build")
	return func() {}
}
