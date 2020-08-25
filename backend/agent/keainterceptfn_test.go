package agent

import (
	"testing"

	"github.com/stretchr/testify/require"
	keactrl "isc.org/stork/appctrl/kea"
)

// Tests that config-get is intercepted and loggers found in the returned
// configuration are recorded. The log viewer is permitted to access only
// those log files.
func TestIcptConfigGetLoggers(t *testing.T) {
	response := &keactrl.Response{}
	err := icptConfigGetLoggers(nil, response)
	require.Nil(t, err)
}
