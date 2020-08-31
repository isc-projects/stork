package agent

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"

	keactrl "isc.org/stork/appctrl/kea"
)

// Test the case that the command is successfully sent to Kea.
func TestSendToKeaOverHTTPSuccess(t *testing.T) {
	sa, _ := setupAgentTest(nil)

	// Expect appropriate content type and the body. If they are not matched
	// an error will be raised.
	defer gock.Off()
	gock.New("http://localhost:45634").
		MatchHeader("Content-Type", "application/json").
		JSON(map[string]string{"command": "list-commands"}).
		Post("/").
		Reply(200).
		JSON([]map[string]int{{"result": 0}})

	command, err := keactrl.NewCommand("list-commands", nil, nil)
	require.NoError(t, err)

	responses := keactrl.ResponseList{}
	err = sendToKeaOverHTTP(sa, "localhost", 45634, command, &responses)
	require.NoError(t, err)

	require.Len(t, responses, 1)
}

// Test the case when Kea returns invalid response to the command.
func TestSendToKeaOverHTTPInvalidResponse(t *testing.T) {
	sa, _ := setupAgentTest(nil)

	// Return invalid response. Arguments must be a map not an integer.
	defer gock.Off()
	gock.New("http://localhost:45634").
		MatchHeader("Content-Type", "application/json").
		JSON(map[string]string{"command": "list-commands"}).
		Post("/").
		Reply(200).
		JSON([]map[string]int{{"result": 0, "arguments": 1}})

	command, err := keactrl.NewCommand("list-commands", nil, nil)
	require.NoError(t, err)

	responses := keactrl.ResponseList{}
	err = sendToKeaOverHTTP(sa, "localhost", 45634, command, &responses)
	require.Error(t, err)
}

// Test the case when Kea server is unreachable.
func TestSendToKeaOverHTTPNoKea(t *testing.T) {
	sa, _ := setupAgentTest(nil)

	command, err := keactrl.NewCommand("list-commands", nil, nil)
	require.NoError(t, err)

	responses := keactrl.ResponseList{}
	err = sendToKeaOverHTTP(sa, "localhost", 45634, command, &responses)
	require.Error(t, err)
}

// Test the function which extracts the list of log files from the Kea
// application by sending the request to the Kea Control Agent and the
// daemons behind it.
func TestKeaAllowedLogs(t *testing.T) {
	sa, _ := setupAgentTest(nil)

	// The first config-get command should go to the Kea Control Agent.
	// The logs should be extracted from there and the subsequent config-get
	// commands should be sent to the daemons with which the CA is configured
	// to communicate.
	defer gock.Off()
	gock.New("http://localhost:45634").
		MatchHeader("Content-Type", "application/json").
		JSON(map[string]string{"command": "config-get"}).
		Post("/").
		Reply(200).
		JSON([]interface{}{
			map[string]interface{}{
				"result": 0,
				"arguments": map[string]interface{}{
					"CtrlAgent": map[string]interface{}{
						"control-sockets": map[string]interface{}{
							"dhcp4": map[string]interface{}{
								"socket-name": "/tmp/dhcp4.sock",
							},
							"dhcp6": map[string]interface{}{
								"socket-name": "/tmp/dhcp6.sock",
							},
						},
						"loggers": []interface{}{
							map[string]interface{}{
								"output_options": []interface{}{
									map[string]interface{}{
										"output": "/tmp/kea-ctrl-agent.log",
									},
								},
							},
						},
					},
				},
			},
		})

	// The config-get command sent to the daemons behind CA should return
	// configurations of the DHCPv4 and DHCPv6 daemons.
	gock.New("http://localhost:45634").
		MatchHeader("Content-Type", "application/json").
		JSON(map[string]interface{}{"command": "config-get", "service": []string{"dhcp4", "dhcp6"}}).
		Post("/").
		Reply(200).
		JSON([]interface{}{
			map[string]interface{}{
				"result": 0,
				"arguments": map[string]interface{}{
					"Dhcp4": map[string]interface{}{
						"loggers": []interface{}{
							map[string]interface{}{
								"output_options": []interface{}{
									map[string]interface{}{
										"output": "/tmp/kea-dhcp4.log",
									},
								},
							},
						},
					},
				},
			},
			map[string]interface{}{
				"result": 0,
				"arguments": map[string]interface{}{
					"Dhcp6": map[string]interface{}{
						"loggers": []interface{}{
							map[string]interface{}{
								"output_options": []interface{}{
									map[string]interface{}{
										"output": "/tmp/kea-dhcp6.log",
									},
								},
							},
						},
					},
				},
			},
		})

	err := detectKeaAllowedLogs(sa, "localhost", 45634)
	require.NoError(t, err)

	// We should have three log files recorded from the returned configurations.
	// One from CA, one from DHCPv4 and one from DHCPv6.
	require.Len(t, sa.logTailer.allowedPaths, 3)
}
