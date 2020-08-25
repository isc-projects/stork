package agent

import keactrl "isc.org/stork/appctrl/kea"

// Intercept callback function for config-get. It records log files
// found in the daemon's configuration  making it accessible by the
// log viewer.
func icptConfigGetLoggers(agent *StorkAgent, response *keactrl.Response) error {
	return nil
}

// Registers all intercept functions defined in this file. It should
// be extended every time a new intercept function is defined.
func registerKeaInterceptFns(agent *StorkAgent) {
	agent.keaInterceptor.register(icptConfigGetLoggers, "config-get")
}
