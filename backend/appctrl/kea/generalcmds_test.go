package keactrl

import (
	"testing"

	"github.com/stretchr/testify/require"
	keaconfig "isc.org/stork/appcfg/kea"
)

// Test config-set command.
func TestNewCommandConfigSet(t *testing.T) {
	config, err := keaconfig.NewConfig(`{
		"Dhcp6": {
			"valid-lifetime": 1000,
			"preferred-lifetime": 500
		}
	}`)
	require.NoError(t, err)
	require.NotNil(t, config)

	command := NewCommandConfigSet(config, "dhcp6")
	require.NotNil(t, command)
	require.JSONEq(t, `{
		"command": "config-set",
		"service": [ "dhcp6" ],
		"arguments": {
			"Dhcp6": {
				"valid-lifetime": 1000,
				"preferred-lifetime": 500
			}
		}
	}`, command.Marshal())
}
