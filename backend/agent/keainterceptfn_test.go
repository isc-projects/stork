package agent

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	keactrl "isc.org/stork/daemonctrl/kea"
)

// Tests that config-get is intercepted and loggers found in the returned
// configuration are recorded. The log tailer is permitted to access only
// those log files.
func TestInterceptConfigGetLoggers(t *testing.T) {
	// Arrange
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

	response := &keactrl.Response{
		ResponseHeader: keactrl.ResponseHeader{
			Result: 0,
			Text:   "Everything is fine",
		},
		Arguments: json.RawMessage(responseArgsJSON),
	}

	// Act
	err := interceptConfigGetLoggers(sa, response)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, sa.logTailer)
	require.True(t, sa.logTailer.allowed("/tmp/kea-dhcp4.log"))
	require.True(t, sa.logTailer.allowed("/tmp/kea-dhcp4-allocations.log"))
	require.False(t, sa.logTailer.allowed("stdout"))
	require.False(t, sa.logTailer.allowed("stderr"))
	require.False(t, sa.logTailer.allowed("syslog:1"))
}

// Tests that the error is returned if Kea responds with a non-success status.
func TestInterceptConfigGetLoggersErrorResponse(t *testing.T) {
	// Arrange
	sa, _, teardown := setupAgentTest()
	defer teardown()

	response := &keactrl.Response{
		ResponseHeader: keactrl.ResponseHeader{
			Result: 1,
			Text:   "error occurred",
		},
	}
	err := interceptConfigGetLoggers(sa, response)
	require.ErrorContains(t, err, "error occurred")
}

// Tests that the error is returned if Kea responds with no arguments.
func TestInterceptConfigGetLoggersNoArguments(t *testing.T) {
	// Arrange
	sa, _, teardown := setupAgentTest()
	defer teardown()

	response := &keactrl.Response{
		ResponseHeader: keactrl.ResponseHeader{
			Result: 0,
			Text:   "Everything is fine",
		},
		Arguments: nil,
	}

	// Act
	err := interceptConfigGetLoggers(sa, response)

	// Assert
	require.ErrorContains(t, err, "response has no arguments")
}

// Tests that the error is returned if Kea responds with unexpected arguments.
func TestInterceptConfigGetLoggersUnexpectedArguments(t *testing.T) {
	// Arrange
	sa, _, teardown := setupAgentTest()
	defer teardown()

	responseArgsJSON := `42`

	response := &keactrl.Response{
		ResponseHeader: keactrl.ResponseHeader{
			Result: 0,
			Text:   "Everything is fine",
		},
		Arguments: json.RawMessage(responseArgsJSON),
	}

	// Act
	err := interceptConfigGetLoggers(sa, response)

	// Assert
	require.ErrorContains(t, err, "arguments which could not be parsed")
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
