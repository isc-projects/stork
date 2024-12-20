package agent

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	keactrl "isc.org/stork/appctrl/kea"
)

// Tests that config-get is intercepted and loggers found in the returned
// configuration are recorded. The log tailer is permitted to access only
// those log files.
func TestInterceptConfigGetLoggers(t *testing.T) {
	sa, _, teardown := setupAgentTest()
	defer teardown()

	responseArgsJSON := `{
        "Dhcp4": {
            "loggers": [
                {
                    "output_options": [
                        {
                            "output": "/tmp/kea-dhcp4.log"
                        },
                        {
                            "output": "stderr"
                        }
                    ]
                },
                {
                    "output_options": [
                        {
                            "output": "/tmp/kea-dhcp4.log"
                        }
                    ]
                },
                {
                    "output_options": [
                        {
                            "output": "stdout"
                        }
                    ]
                },
                {
                    "output_options": [
                        {
                            "output": "/tmp/kea-dhcp4-allocations.log"
                        },
                        {
                            "output": "syslog:1"
                        }
                    ]
                }
            ]
        }
    }`
	responseArgs := make(map[string]interface{})
	err := json.Unmarshal([]byte(responseArgsJSON), &responseArgs)
	require.NoError(t, err)

	response := &keactrl.Response{
		ResponseHeader: keactrl.ResponseHeader{
			Result: 0,
			Text:   "Everything is fine",
			Daemon: "dhcp4",
		},
		Arguments: &responseArgs,
	}
	err = interceptConfigGetLoggers(sa, response)
	require.NoError(t, err)
	require.NotNil(t, sa.logTailer)
	require.True(t, sa.logTailer.allowed("/tmp/kea-dhcp4.log"))
	require.True(t, sa.logTailer.allowed("/tmp/kea-dhcp4-allocations.log"))
	require.False(t, sa.logTailer.allowed("stdout"))
	require.False(t, sa.logTailer.allowed("stderr"))
	require.False(t, sa.logTailer.allowed("syslog:1"))
}

// Test that the result code is changed if the reservation-get-page command
// returns an unsupported error.
func TestReservationGetPageUnsupported(t *testing.T) {
	// Arrange
	sa, _, teardown := setupAgentTest()
	defer teardown()

	rsp := &keactrl.Response{
		ResponseHeader: keactrl.ResponseHeader{
			Result: keactrl.ResponseError,
			Text:   "not supported by the RADIUS backend",
		},
	}

	// Act
	err := reservationGetPageUnsupported(sa, rsp)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, keactrl.ResponseCommandUnsupported, rsp.Result)
}
