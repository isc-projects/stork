package agent

import (
	"testing"

	"github.com/stretchr/testify/require"
	keactrl "isc.org/stork/appctrl/kea"
)

// Tests that config-get is intercepted and loggers found in the returned
// configuration are recorded. The log tailer is permitted to access only
// those log files.
func TestIcptConfigGetLoggers(t *testing.T) {
	sa, _ := setupAgentTest(nil)

	response := &keactrl.Response{
		ResponseHeader: keactrl.ResponseHeader{
			Result: 0,
			Text:   "Everything is fine",
			Daemon: "dhcp4",
		},
		Arguments: &map[string]interface{}{
			"Dhcp4": map[string]interface{}{
				"loggers": []interface{}{
					map[string]interface{}{
						"output_options": []interface{}{
							map[string]interface{}{
								"output": "/tmp/kea-dhcp4.log",
							},
							map[string]interface{}{
								"output": "stderr",
							},
						},
					},
					map[string]interface{}{
						"output_options": []interface{}{
							map[string]interface{}{
								"output": "/tmp/kea-dhcp4.log",
							},
						},
					},
					map[string]interface{}{
						"output_options": []interface{}{
							map[string]interface{}{
								"output": "stdout",
							},
						},
					},
					map[string]interface{}{
						"output_options": []interface{}{
							map[string]interface{}{
								"output": "/tmp/kea-dhcp4-allocations.log",
							},
							map[string]interface{}{
								"output": "syslog:1",
							},
						},
					},
				},
			},
		},
	}
	err := icptConfigGetLoggers(sa, response)
	require.NoError(t, err)
	require.NotNil(t, sa.logTailer)
	require.True(t, sa.logTailer.allowed("/tmp/kea-dhcp4.log"))
	require.True(t, sa.logTailer.allowed("/tmp/kea-dhcp4-allocations.log"))
	require.False(t, sa.logTailer.allowed("stdout"))
	require.False(t, sa.logTailer.allowed("stderr"))
	require.False(t, sa.logTailer.allowed("syslog:1"))
}
