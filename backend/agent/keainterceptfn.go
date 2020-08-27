package agent

import (
	"strings"

	log "github.com/sirupsen/logrus"

	keaconfig "isc.org/stork/appcfg/kea"
	keactrl "isc.org/stork/appctrl/kea"
)

// Intercept callback function for config-get. It records log files
// found in the daemon's configuration  making them accessible by the
// log viewer.
func icptConfigGetLoggers(agent *StorkAgent, response *keactrl.Response) error {
	if response.Result > 0 {
		log.Warn("skipped refreshing viewable log files because config-get returned non success result")
		return nil
	}
	if response.Arguments == nil {
		log.Warn("skipped refreshing viewable log files because config-get response has no arguments")
		return nil
	}
	cfg := keaconfig.New(response.Arguments)
	if cfg == nil {
		log.Warn("skipped refreshing viewable log files because config-get response contains arguments which could not be parsed")
		return nil
	}

	loggers := cfg.GetLoggers()
	if len(loggers) == 0 {
		log.Info("no loggers found in the returned configuration while trying to refresh the viewable log files")
		return nil
	}

	// Go over returned loggers and allow those found in the returned configuration.
	for _, l := range loggers {
		for _, o := range l.OutputOptions {
			if o.Output != "stdout" && o.Output != "stderr" && !strings.HasPrefix(o.Output, "syslog") {
				agent.logTailer.allow(o.Output)
			}
		}
	}

	return nil
}

// Registers all intercept functions defined in this file. It should
// be extended every time a new intercept function is defined.
func registerKeaInterceptFns(agent *StorkAgent) {
	agent.keaInterceptor.register(icptConfigGetLoggers, "config-get")
}
