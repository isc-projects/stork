//go:build ignore

package profiler

import "github.com/sirupsen/logrus"

func Start(port int) func() {
	logrus.Debug("Profiler is not available in this build")
	return func() {}
}
