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
