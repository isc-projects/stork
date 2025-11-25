package agent

import (
	"github.com/pkg/errors"
	keaconfig "isc.org/stork/daemoncfg/kea"
	keactrl "isc.org/stork/daemonctrl/kea"
)

// Intercept callback function for config-get. It records log files
// found in the daemon's configuration, making them accessible by the
// log viewer.
func interceptConfigGetLoggers(agent *StorkAgent, response *keactrl.Response) error {
	if err := response.GetError(); err != nil {
		err = errors.WithMessage(err, "skipped refreshing viewable log files because config-get returned unsuccessful result")
		return err
	}
	if response.Arguments == nil {
		err := errors.New("skipped refreshing viewable log files because config-get response has no arguments")
		return err
	}
	config, err := keaconfig.NewConfig(response.Arguments)
	if err != nil {
		err := errors.WithMessage(err, "skipped refreshing viewable log files because config-get response contains arguments which could not be parsed")
		return err
	}

	paths := collectKeaAllowedLogs(config)
	for _, p := range paths {
		agent.logTailer.allow(p)
	}
	return nil
}

// Change the reservation-get-page response status if unsupported error is
// returned.
//
// Kea 2.2 and below return a general error response if RADIUS is used as
// the host backend. It causes Stork to generate a false disconnect event
// and block pulling host reservations from other host backends.
// See: https://gitlab.isc.org/isc-projects/stork/-/issues/792 and
// https://gitlab.isc.org/isc-projects/kea/-/issues/2566 .
func reservationGetPageUnsupported(agent *StorkAgent, response *keactrl.Response) error {
	if response.Result == keactrl.ResponseError && response.Text == "not supported by the RADIUS backend" {
		response.Result = keactrl.ResponseCommandUnsupported
	}

	return nil
}

// Registers all intercept functions defined in this file. It should
// be extended every time a new intercept function is defined.
func registerKeaInterceptFns(agent *StorkAgent) {
	agent.keaInterceptor.registerAsync(interceptConfigGetLoggers, "config-get")
	agent.keaInterceptor.registerSync(reservationGetPageUnsupported, "reservation-get-page")
}
